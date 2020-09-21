package diversion

import (
	"DnsDiversion/common"
	"DnsDiversion/logger"
	"DnsDiversion/network"
	"golang.org/x/net/dns/dnsmessage"
	"net"
	"sync/atomic"
	"time"
)

var totalQueryCount uint64

func HandlePacket(bytes []byte, respCall func([]byte)) error {
	msg := dnsmessage.Message{}
	if err := msg.Unpack(bytes); err != nil {
		return err
	}
	if common.NeedDebug() {
		logger.Debug("Unpack DNS Message", msg.GoString())
	}
	answers := make([]dnsmessage.Resource, 0)
	numOfQueries := 0
	for _, question := range msg.Questions {
		queryType := dnsmessage.Type(0)
		if len(network.UpstreamsList[question.Type]) != 0 {
			queryType = question.Type
		}
		numOfQueries += len(network.UpstreamsList[queryType])
	}
	answerChan := make(chan []dnsmessage.Resource, numOfQueries)
	idChan := make(chan int, numOfQueries)
	receivedList := make([]bool, len(msg.Questions))
	for id, question := range msg.Questions {
		if common.NeedDebug() {
			logger.Debug("Question", question.Name, question.Type, question.Class)
		}
		queryType := dnsmessage.Type(0)
		if len(network.UpstreamsList[question.Type]) != 0 {
			queryType = question.Type
		}
		for _, upstream := range network.UpstreamsList[queryType] {
			newMsg := dnsmessage.Message{
				Header: dnsmessage.Header{
					ID:    uint16(atomic.AddUint64(&totalQueryCount, 1) % 65536),
					RCode: dnsmessage.RCodeSuccess,
				},
				Questions: make([]dnsmessage.Question, 1),
			}
			newMsg.Questions[0] = question
			if upstream.Network == "udp" {
				go requestUpstreamWithUDP(&newMsg, upstream.UDPAddr, answerChan, id, idChan)
			} else if upstream.Network == "tcp" {
				go requestUpstreamWithTCP(&newMsg, upstream.TCPAddr, answerChan, id, idChan)
			}
		}
	}
	timer := time.NewTimer(time.Duration(common.Config.Advanced.NSLookupTimeoutMs) * time.Millisecond)
loop:
	for {
		select {
		case id := <-idChan:
			receivedList[id] = true
			allReceived := true
			for _, received := range receivedList {
				if !received {
					allReceived = false
					break
				}
			}
			if allReceived {
				break loop
			}
		case myAnswers := <-answerChan:
			answers = append(answers, myAnswers...)
		case <-timer.C:
			break loop
		}
	}
	msg.Header.Response = true
	msg.Answers = answers
	bytes, err := msg.Pack()
	if err != nil {
		return err
	}
	respCall(bytes)
	return nil
}

func requestUpstreamWithUDP(msg *dnsmessage.Message, upstreamAddr *net.UDPAddr, answerChan chan []dnsmessage.Resource, questionId int, idChan chan int) {
	conn, err := net.DialUDP("udp", nil, upstreamAddr)
	if err != nil {
		logger.Warning("Dial UDP", upstreamAddr, err)
	}
	defer func() {
		_ = conn.Close()
		idChan <- questionId
	}()
	if err := conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.NSLookupTimeoutMs) * time.Millisecond)); err != nil {
		logger.Warning("Set UDP Timeout", upstreamAddr, err)
	}
	receivedMsg, err := requestDNS(msg, func() ([]byte, error) {
		buffer := make([]byte, common.Config.Advanced.MaxReceivedPacketSize)
		if _, err := conn.Read(buffer); err != nil {
			return nil, err
		}
		return buffer, nil
	}, func(bytes []byte) error {
		_, err := conn.Write(bytes)
		return err
	})
	if err != nil {
		logger.Warning("Request DNS", err)
		return
	}
	if common.NeedDebug() {
		logger.Debug("Unpack DNS Message", upstreamAddr.String(), receivedMsg.GoString())
	}
	answerChan <- receivedMsg.Answers
}
func requestUpstreamWithTCP(msg *dnsmessage.Message, upstreamAddr *net.TCPAddr, answerChan chan []dnsmessage.Resource, questionId int, idChan chan int) {
	conn, err := net.DialTCP("tcp", nil, upstreamAddr)
	if err != nil {
		logger.Warning("Dial TCP", upstreamAddr, err)
	}
	defer func() {
		_ = conn.Close()
		idChan <- questionId
	}()
	if err := conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.NSLookupTimeoutMs) * time.Millisecond)); err != nil {
		logger.Warning("Set TCP Timeout", upstreamAddr, err)
	}
	receivedMsg, err := requestDNS(msg, func() ([]byte, error) {
		readBytes, _, err := network.ReadPacketFromTCPConn(conn)
		return readBytes, err
	}, func(bytes []byte) error {
		_, err := network.WritePacketToTCPConn(bytes, conn)
		return err
	})
	if err != nil {
		logger.Warning("Request DNS", err)
		return
	}
	if common.NeedDebug() {
		logger.Debug("Unpack DNS Message", upstreamAddr.String(), receivedMsg.GoString())
	}
	answerChan <- receivedMsg.Answers
}

func requestDNS(msg *dnsmessage.Message, readFunc func() ([]byte, error), writeFunc func([]byte) error) (*dnsmessage.Message, error) {
	bytes, err := msg.Pack()
	if err != nil {
		return nil, err
	}
	if err := writeFunc(bytes); err != nil {
		return nil, err
	}
	readBytes, err := readFunc()
	if err != nil {
		return nil, err
	}
	receivedMsg := dnsmessage.Message{}
	if err := receivedMsg.Unpack(readBytes); err != nil {
		return nil, err
	}
	return &receivedMsg, nil
}

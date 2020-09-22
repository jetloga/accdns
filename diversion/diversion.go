package diversion

import (
	"DnsDiversion/common"
	"DnsDiversion/logger"
	"DnsDiversion/network"
	"golang.org/x/net/dns/dnsmessage"
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
	msgChan := make(chan *dnsmessage.Message, numOfQueries)
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
			go requestUpstreamDNS(&newMsg, upstream, msgChan, id, idChan)
		}
	}
	msg.Header.RCode = dnsmessage.RCodeServerFailure
	msg.Header.Response = true
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
		case myMsg := <-msgChan:
			if msg.Header.RCode != dnsmessage.RCodeSuccess {
				msg.Header.RCode = myMsg.Header.RCode
			}
			answers = append(answers, myMsg.Answers...)
		case <-timer.C:
			break loop
		}
	}
	msg.Answers = answers
	bytes, err := msg.Pack()
	if err != nil {
		return err
	}
	respCall(bytes)
	return nil
}

func requestUpstreamDNS(msg *dnsmessage.Message, upstreamAddr *network.SocketAddr, msgChan chan *dnsmessage.Message, questionId int, idChan chan int) {
	conn, err := network.GlobalConnPool.RequireConn(upstreamAddr)
	if err != nil {
		logger.Warning("Dial Socket Connection", upstreamAddr, err)
	}
	defer func() {
		_ = network.GlobalConnPool.ReleaseConn(conn)
		idChan <- questionId
	}()
	bytes, err := msg.Pack()
	if err != nil {
		logger.Warning("Pack DNS Packet", upstreamAddr, err)
	}
	if _, err := conn.WritePacket(bytes); err != nil {
		logger.Warning("Write DNS Packet", upstreamAddr, err)
	}
	readBytes, _, err := conn.ReadPacket()
	if err != nil {
		logger.Warning("Read DNS Packet", upstreamAddr, err)
	}
	receivedMsg := &dnsmessage.Message{}
	if err := receivedMsg.Unpack(readBytes); err != nil {
		logger.Warning("Unpack DNS Packet", upstreamAddr, err)
	}
	if err != nil {
		logger.Warning("Request DNS", err)
		return
	}
	if common.NeedDebug() {
		logger.Debug("Unpack DNS Message", upstreamAddr.String(), receivedMsg.GoString())
	}
	msgChan <- receivedMsg
}

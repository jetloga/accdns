package diversion

import (
	"DnsDiversion/common"
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
	common.Debug("[DIVERSION]", "{Unpack DNS Message}", msg.GoString())
	answers := make([]dnsmessage.Resource, 0)
	numOfQueries := 0
	for _, question := range msg.Questions {
		queryType := dnsmessage.Type(0)
		if len(common.UpstreamsList[question.Type]) != 0 {
			queryType = question.Type
		}
		numOfQueries += len(common.UpstreamsList[queryType])
	}
	answerChan := make(chan []dnsmessage.Resource, numOfQueries)
	idChan := make(chan int, len(msg.Questions))
	receivedList := make([]bool, len(msg.Questions))
	for id, question := range msg.Questions {
		common.Debug("[DIVERSION]", "{Question}", question.Name, question.Type, question.Class)
		queryType := dnsmessage.Type(0)
		if len(common.UpstreamsList[question.Type]) != 0 {
			queryType = question.Type
		} else {
			common.Debug("[DIVERSION]", "{Request Default Upstraeams}", question.Name, question.Type, question.Class)
		}
		for _, upstream := range common.UpstreamsList[queryType] {
			newMsg := dnsmessage.Message{
				Header: dnsmessage.Header{
					ID:    uint16(atomic.AddUint64(&totalQueryCount, 1) % 65536),
					RCode: dnsmessage.RCodeSuccess,
				},
				Questions: make([]dnsmessage.Question, 1),
			}
			newMsg.Questions[0] = question
			go requestUpstream(&newMsg, upstream, answerChan, id, idChan)
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

func requestUpstream(msg *dnsmessage.Message, upstream *common.SocketAddr, answerChan chan []dnsmessage.Resource, questionId int, idChan chan int) {
	if common.Config.Upstream.UseUDP {
		conn, err := net.DialUDP("udp", nil, upstream.UDPAddr)
		if err != nil {
			common.Warning("[DIVERSION]", "{Dial UDP}", upstream.UDPAddr, err)
		}
		defer func() { _ = conn.Close() }()
		if err := conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.NSLookupTimeoutMs) * time.Millisecond)); err != nil {
			common.Warning("[DIVERSION]", "{Set UDP Timeout}", upstream.UDPAddr, err)
		}
		bytes, err := msg.Pack()
		if err != nil {
			common.Warning("[DIVERSION]", "{DNS Message Pack}", upstream.UDPAddr, err)
		}
		n, err := conn.Write(bytes)
		if err != nil {
			common.Warning("[DIVERSION]", "{Write UDP Packet}", upstream.UDPAddr, err)
		}
		common.Debug("[DIVERSION]", "{Write UDP Packet}", "Write", n, "bytes to", upstream.UDPAddr)
		buffer := make([]byte, common.Config.Advanced.MaxReceivedPacketSize)
		n, err = conn.Read(buffer)
		if err != nil {
			common.Warning("[DIVERSION]", "{Read UDP Packet}", upstream.UDPAddr, err)
		}
		common.Debug("[DIVERSION]", "{Read UDP Packet}", "Read", n, "bytes from", upstream.UDPAddr)
		receivedMsg := dnsmessage.Message{}
		if err := receivedMsg.Unpack(buffer); err != nil {
			common.Warning("[DIVERSION]", "{DNS Unpack}", upstream.UDPAddr, err)
		}
		common.Debug("[DIVERSION]", "{Unpack DNS Message}", upstream.UDPAddr.String(), receivedMsg.GoString())
		answerChan <- receivedMsg.Answers
		idChan <- questionId
	}
}

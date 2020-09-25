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

	maxPacketSize := common.Config.Advanced.DefaultMaxPacketSize
	supportEDNS := false
	for _, res := range msg.Additionals {
		if res.Header.Type == dnsmessage.TypeOPT {
			supportEDNS = true
			maxPacketSize = common.IntMax(int(res.Header.Class), 512)
			break
		}
	}
	ednsRes := dnsmessage.Resource{
		Header: dnsmessage.ResourceHeader{
			Name:  dnsmessage.MustNewName("."),
			Type:  dnsmessage.TypeOPT,
			Class: dnsmessage.Class(maxPacketSize),
		},
		Body: &dnsmessage.OPTResource{},
	}

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
	retChan := make(chan bool, numOfQueries)
	receivedList := make([]bool, len(msg.Questions))
	for id, question := range msg.Questions {
		if common.NeedDebug() {
			logger.Debug("Question", question.Name, question.Type, question.Class)
		}
		queryType := dnsmessage.Type(0)
		if len(network.UpstreamsList[question.Type]) != 0 {
			queryType = question.Type
		}
		newMsg := dnsmessage.Message{
			Header: dnsmessage.Header{
				ID:               uint16(atomic.AddUint64(&totalQueryCount, 1) % 65536),
				OpCode:           msg.Header.OpCode,
				RCode:            dnsmessage.RCodeSuccess,
				RecursionDesired: msg.RecursionDesired,
			},
			Questions:   make([]dnsmessage.Question, 1),
			Additionals: make([]dnsmessage.Resource, 0),
		}
		newMsg.Questions[0] = question
		if maxPacketSize > common.StandardMaxDNSPacketSize {
			newMsg.Additionals = append(newMsg.Additionals, ednsRes)
		}
		for _, upstream := range network.UpstreamsList[queryType] {
			go requestUpstreamDNS(&newMsg, upstream, msgChan, maxPacketSize, id, idChan, retChan)
		}
	}

	respMsg := dnsmessage.Message{
		Header: dnsmessage.Header{
			ID:               msg.Header.ID,
			Response:         true,
			OpCode:           msg.Header.OpCode,
			RecursionDesired: msg.Header.RecursionDesired,
			RCode:            dnsmessage.RCodeServerFailure,
		},
		Questions:   msg.Questions,
		Answers:     make([]dnsmessage.Resource, 0),
		Authorities: make([]dnsmessage.Resource, 0),
		Additionals: make([]dnsmessage.Resource, 0),
	}

	timer := time.NewTimer(time.Duration(common.Config.Advanced.NSLookupTimeoutMs) * time.Millisecond)
	retServerCounter := 0
	appendMsgToResp := func(myMsg *dnsmessage.Message) {
		if respMsg.Header.RCode != dnsmessage.RCodeSuccess {
			respMsg.Header.RCode = myMsg.Header.RCode
		}
		if myMsg.Header.RecursionAvailable {
			respMsg.Header.RecursionAvailable = true
		}
		if myMsg.Header.Truncated {
			respMsg.Header.Truncated = true
		}
		if myMsg.Header.Authoritative {
			respMsg.Header.Authoritative = true
		}
		respMsg.Answers = append(respMsg.Answers, myMsg.Answers...)
		respMsg.Authorities = append(respMsg.Authorities, myMsg.Authorities...)
		for _, res := range myMsg.Additionals {
			if res.Header.Type != dnsmessage.TypeOPT {
				respMsg.Additionals = append(respMsg.Additionals, res)
			}
		}
	}
loop:
	for {
		select {
		case myMsg := <-msgChan:
			appendMsgToResp(myMsg)
			receivedList[<-idChan] = true
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
		case <-retChan:
			retServerCounter++
			if retServerCounter >= numOfQueries {
				for {
					select {
					case myMsg := <-msgChan:
						appendMsgToResp(myMsg)
					default:
						break loop
					}
				}
			}
		case <-timer.C:
			break loop
		}
	}

	if supportEDNS {
		respMsg.Additionals = append(respMsg.Additionals, ednsRes)
	}

	respBytes, err := respMsg.Pack()
	if err != nil {
		return err
	}
	if common.NeedDebug() {
		logger.Debug("Pack DNS Message", respMsg.GoString())
	}
	respCall(respBytes)
	return nil
}

func requestUpstreamDNS(msg *dnsmessage.Message, upstreamAddr *network.SocketAddr, msgChan chan *dnsmessage.Message, maxPacketSize int, questionId int, idChan chan int, retChan chan bool) {
	defer func() {
		retChan <- true
	}()
	conn, err := network.GlobalConnPool.RequireConn(upstreamAddr)
	if err != nil {
		logger.Warning("Dial Socket Connection", err)
		return
	}
	defer func() {
		if err := network.GlobalConnPool.ReleaseConn(conn); err != nil {
			logger.Warning("Release Connection", err)
		}
	}()
	bytes, err := msg.Pack()
	if err != nil {
		logger.Warning("Pack DNS Packet", err)
		return
	}
	if common.NeedDebug() {
		logger.Debug("Pack DNS Message", msg.GoString())
	}
	if _, err := conn.WritePacket(bytes); err != nil {
		logger.Warning("Write DNS Packet", err)
		return
	}
	readBytes, _, err := conn.ReadPacket(maxPacketSize)
	if err != nil {
		logger.Warning("Read DNS Packet", err)
		return
	}
	receivedMsg := &dnsmessage.Message{}
	if err := receivedMsg.Unpack(readBytes); err != nil {
		logger.Warning("Unpack DNS Packet", err)
		return
	}
	if common.NeedDebug() {
		logger.Debug("Unpack DNS Message", receivedMsg.GoString())
	}
	msgChan <- receivedMsg
	idChan <- questionId
}

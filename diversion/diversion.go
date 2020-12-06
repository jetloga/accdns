package diversion

import (
	"accdns/cache"
	"accdns/common"
	"accdns/logger"
	"accdns/network"
	"errors"
	"golang.org/x/net/dns/dnsmessage"
	"sync/atomic"
	"time"
)

var totalQueryCount uint64

func HandlePacket(bytes []byte, respCall func([]byte), dnsCache *cache.Cache) error {
	msg := dnsmessage.Message{}
	if err := msg.Unpack(bytes); err != nil {
		return err
	}
	if common.NeedDebug() {
		logger.Debug("Unpack DNS Message", msg.GoString())
	}

	maxPacketSize := common.StandardMaxDNSPacketSize
	supportEDNS := false
	for _, res := range msg.Additionals {
		if res.Header.Type == dnsmessage.TypeOPT {
			supportEDNS = true
			maxPacketSize = common.IntMax(int(res.Header.Class), common.StandardMaxDNSPacketSize)
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
			go func(upstream *network.SocketAddr) {
				defer func() {
					retChan <- true
				}()
				var receivedMsg *dnsmessage.Message
				var err error
				if dnsCache != nil {
					receivedMsg, err = dnsCache.QueryAndUpdate(&newMsg, upstream, requestUpstreamDNS)
				} else {
					receivedMsg, err = requestUpstreamDNS(&newMsg, upstream)
				}
				if err != nil {
					return
				}

				idChan <- id
				msgChan <- receivedMsg
			}(upstream)
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
			if myMsg.RCode == dnsmessage.RCodeSuccess {
				receivedList[<-idChan] = true
			}
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

func requestUpstreamDNS(msg *dnsmessage.Message, upstreamAddr *network.SocketAddr) (*dnsmessage.Message, error) {

	if common.NeedDebug() {
		logger.Debug("Request Upstream", upstreamAddr)
	}
	bytes, err := msg.Pack()
	if err != nil {
		logger.Warning("Pack DNS Packet", err)
		return nil, err
	}
	if common.NeedDebug() {
		logger.Debug("Pack DNS Message", msg.GoString())
	}
	var conn *network.SocketConn
	var readBytes []byte
	var networkErr error
	for i := 0; i < common.Config.Advanced.NetworkFailedRetries; i++ {
		func() {
			conn, networkErr = network.EstablishNewSocketConn(upstreamAddr)
			defer func() {
				_ = conn.Close()
			}()
			if networkErr != nil {
				logger.Warning("Dial Socket Connection", networkErr)
				return
			}
			_, networkErr = conn.WritePacket(bytes)
			if networkErr != nil {
				logger.Warning("Write DNS Packet", networkErr)
				return
			}
			readBytes, _, networkErr = conn.ReadPacket(common.Config.Advanced.MaxReceivedPacketSize)
			if networkErr != nil {
				logger.Warning("Read DNS Packet", networkErr)
				return
			}
		}()
	}
	if networkErr != nil {
		return nil, networkErr
	}
	receivedMsg := &dnsmessage.Message{}
	if err := receivedMsg.Unpack(readBytes); err != nil {
		logger.Warning("Unpack DNS Packet", err)
		return nil, err
	}
	if common.NeedDebug() {
		logger.Debug("Unpack DNS Message", receivedMsg.GoString())
	}
	if msg.ID != receivedMsg.ID {
		err = errors.New("response id is not match")
		logger.Warning("Check DNS Packet", err)
		return nil, err
	}
	return receivedMsg, nil
}

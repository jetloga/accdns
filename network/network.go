package network

import (
	"DnsDiversion/common"
	"DnsDiversion/logger"
	"bytes"
	"encoding/binary"
	"errors"
	"golang.org/x/net/dns/dnsmessage"
	"io"
	"net"
	"strconv"
	"strings"
)

var UpstreamsList [256][]*SocketAddr

func Init() error {
	for typeCode := range UpstreamsList {
		switch dnsmessage.Type(typeCode) {
		case dnsmessage.Type(0):
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.DefaultUpstreams))
			for i, upstreamStr := range common.Config.Upstream.DefaultUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Info("Load Default Upstream", socketAddr.Network, socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeA:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.ARecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.ARecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Info("Load Upstream For A Record", socketAddr.Network, socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeAAAA:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.AAAARecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.AAAARecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Info("Load Upstream For AAAA Record", socketAddr.Network, socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeCNAME:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.CNAMERecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.CNAMERecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Info("Load Upstream For CNAME Record", socketAddr.Network, socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeTXT:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.TXTRecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.TXTRecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Debug("Load Upstream For TXT Record", socketAddr.Network, socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypePTR:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.PTRRecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.PTRRecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Debug("Load Upstream For PTR Record", socketAddr.Network, socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		default:
			UpstreamsList[typeCode] = make([]*SocketAddr, 0)
		}

	}
	for _, kvPair := range common.Config.Upstream.CustomRecordUpstream {
		typeCodeStr, addr, err := common.ParseKVPair(kvPair)
		if err != nil {
			return err
		}
		typeCode, err := strconv.Atoi(typeCodeStr)
		if err != nil {
			return err
		}
		if typeCode < 0 || typeCode > 255 {
			return errors.New("type code is not correct")
		}
		socketAddr, err := ParseNewSocketAddr(addr)
		if err != nil {
			return err
		}
		UpstreamsList[typeCode] = append(UpstreamsList[typeCode], socketAddr)
	}
	return nil
}
func ParseNewSocketAddr(addrStr string) (*SocketAddr, error) {
	socketAddr := SocketAddr{
		Network: "udp",
	}
	if strings.HasPrefix(addrStr, "tcp:") {
		socketAddr.Network = "tcp"
		addrStr = addrStr[4:]
	} else if strings.HasPrefix(addrStr, "udp:") {
		addrStr = addrStr[4:]
	}
	ip := net.IP{}
	port := 53
	if strings.HasPrefix(addrStr, "[") {
		index := strings.Index(addrStr, "]")
		if index < 0 {
			return nil, errors.New("wrong socket address " + addrStr)
		}
		ip = net.ParseIP(addrStr[1:index])
		if ip == nil {
			return nil, errors.New("wrong socket address " + addrStr)
		}
		addrStr = addrStr[index+1:]
	} else {
		index := strings.Index(addrStr, ":")
		if index < 0 {
			ip = net.ParseIP(addrStr)
			if ip == nil {
				return nil, errors.New("wrong socket address " + addrStr)
			}
			addrStr = ""
		} else {
			ip = net.ParseIP(addrStr[:index])
			if ip == nil {
				return nil, errors.New("wrong socket address " + addrStr)
			}
			addrStr = addrStr[index:]
		}
	}
	if len(addrStr) > 0 {
		addrStr = addrStr[1:]
		myPort, err := strconv.Atoi(addrStr)
		if err != nil {
			return nil, err
		}
		if myPort < 1 || myPort > 65535 {
			return nil, errors.New("invalid port")
		}
		port = myPort
	}
	if socketAddr.Network == "tcp" {
		socketAddr.TCPAddr = &net.TCPAddr{
			IP:   ip,
			Port: port,
		}
	} else {
		socketAddr.UDPAddr = &net.UDPAddr{
			IP:   ip,
			Port: port,
		}
	}
	return &socketAddr, nil
}

func WritePacketToTCPConn(writeBytes []byte, conn *net.TCPConn) (int, error) {
	size := uint16(len(writeBytes))
	buffer := bytes.NewBuffer([]byte{})
	if err := binary.Write(buffer, binary.BigEndian, size); err != nil {
		return 0, err
	}
	completeBytes := buffer.Bytes()
	completeBytes = append(completeBytes, writeBytes...)
	n, err := conn.Write(completeBytes)
	return n, err
}

func ReadPacketFromTCPConn(conn *net.TCPConn) ([]byte, int, error) {
	bufferBytes := make([]byte, 2)
	n, err := io.ReadFull(conn, bufferBytes)
	if err != nil {
		return nil, n, err
	}
	size := uint16(0)
	if err := binary.Read(bytes.NewBuffer(bufferBytes), binary.BigEndian, &size); err != nil {
		return nil, n, err
	}
	bufferBytes = make([]byte, size)
	n, err = io.ReadFull(conn, bufferBytes)
	if err != nil {
		return nil, n, err
	}
	bufferBytes = bytes.TrimRight(bufferBytes, "\x00")
	return bufferBytes, n, nil
}

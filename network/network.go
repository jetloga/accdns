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
	"time"
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
				logger.Info("Load Default Upstream", socketAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeA:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.ARecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.ARecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Info("Load Upstream For A Record", socketAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeAAAA:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.AAAARecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.AAAARecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Info("Load Upstream For AAAA Record", socketAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeCNAME:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.CNAMERecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.CNAMERecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Info("Load Upstream For CNAME Record", socketAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeTXT:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.TXTRecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.TXTRecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Debug("Load Upstream For TXT Record", socketAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypePTR:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(common.Config.Upstream.PTRRecordUpstreams))
			for i, upstreamStr := range common.Config.Upstream.PTRRecordUpstreams {
				socketAddr, err := ParseNewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				logger.Debug("Load Upstream For PTR Record", socketAddr.String())
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
	socketAddr := &SocketAddr{
		UDPAddr: nil,
		TCPAddr: nil,
	}
	isTCP := false
	if strings.HasPrefix(addrStr, "tcp:") {
		isTCP = true
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
		ip = net.ParseIP(addrStr)
		if ip == nil {
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
		} else {
			addrStr = ""
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
	if isTCP {
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
	return socketAddr, nil
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

func (addr *SocketAddr) String() string {
	if addr.UDPAddr != nil {
		return "udp " + addr.UDPAddr.String()
	} else if addr.TCPAddr != nil {
		return "tcp " + addr.TCPAddr.String()
	}
	return "<nil>"
}

func EstablishNewSocketConn(addr *SocketAddr) (conn *SocketConn, err error) {
	conn = &SocketConn{
		UDPConn: nil,
		TCPConn: nil,
	}
	if addr.UDPAddr != nil {
		conn.UDPConn, err = net.DialUDP("udp", nil, addr.UDPAddr)
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.IdleConnectionTimeout) * time.Second))
	} else if addr.TCPAddr != nil {
		conn.TCPConn, err = net.DialTCP("tcp", nil, addr.TCPAddr)
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.IdleConnectionTimeout) * time.Second))
	} else {
		err = errors.New("socket address not initialize")
	}
	return
}

func (conn *SocketConn) ReadPacket() (readBytes []byte, n int, err error) {
	if conn.IsDead() {
		err = errors.New("connection is dead")
		return
	}
	if err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.RWTimeoutMs) * time.Millisecond)); err != nil {
		return
	}
	if conn.UDPConn != nil {
		readBytes = make([]byte, common.Config.Advanced.MaxReceivedPacketSize)
		n, err = conn.UDPConn.Read(readBytes)
		if err != nil {
			return
		}
		if err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.IdleConnectionTimeout) * time.Second)); err != nil {
			return
		}
		readBytes = bytes.TrimRight(readBytes, "\x00")
	} else if conn.TCPConn != nil {
		readBytes, n, err = ReadPacketFromTCPConn(conn.TCPConn)
		if err != nil {
			return
		}
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.IdleConnectionTimeout) * time.Second))
	} else {
		err = errors.New("socket connection not initialize")
	}
	return
}

func (conn *SocketConn) WritePacket(packetBytes []byte) (n int, err error) {
	if conn.IsDead() {
		err = errors.New("connection is dead")
		return
	}
	if err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.RWTimeoutMs) * time.Millisecond)); err != nil {
		return
	}
	if conn.UDPConn != nil {
		n, err = conn.UDPConn.Write(packetBytes)
		if err != nil {
			return
		}
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.IdleConnectionTimeout) * time.Second))
	} else if conn.TCPConn != nil {
		n, err = WritePacketToTCPConn(packetBytes, conn.TCPConn)
		if err != nil {
			return
		}
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.IdleConnectionTimeout) * time.Second))
	} else {
		err = errors.New("socket connection not initialize")
	}
	return
}

func (conn *SocketConn) SetDeadline(t time.Time) (err error) {
	if conn.IsDead() {
		err = errors.New("connection is dead")
		return
	}
	conn.deadTime = t.UnixNano()
	if conn.UDPConn != nil {
		err = conn.UDPConn.SetDeadline(t)
	} else if conn.TCPConn != nil {
		err = conn.UDPConn.SetDeadline(t)
	} else {
		err = errors.New("socket connection not initialize")
	}
	return
}

func (conn *SocketConn) IsDead() bool {
	if conn.deadTime != 0 {
		return time.Now().UnixNano() > conn.deadTime
	} else {
		return false
	}
}

func (conn *SocketConn) Close() (err error) {
	if conn.IsDead() {
		err = errors.New("connection is dead")
		return
	}
	conn.deadTime = -1
	if conn.UDPConn != nil {
		err = conn.UDPConn.Close()
	} else if conn.TCPConn != nil {
		err = conn.TCPConn.Close()
	}
	return
}

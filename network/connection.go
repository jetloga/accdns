package network

import (
	"DnsDiversion/common"
	"bytes"
	"errors"
	"net"
	"time"
)

func EstablishNewSocketConn(addr *SocketAddr) (conn *SocketConn, err error) {
	conn = &SocketConn{
		SocketAddr: addr,
		UDPConn:    nil,
		TCPConn:    nil,
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

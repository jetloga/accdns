package network

import (
	"accdns/common"
	"errors"
	"net"
	"time"
)

func EstablishNewSocketConn(addr *SocketAddr) (conn *SocketConn, err error) {
	conn = &SocketConn{
		SocketAddr: addr,
	}
	if addr.UDPAddr != nil {
		conn.UDPConn, err = net.DialUDP("udp", nil, addr.UDPAddr)
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.ConnectionTimeout) * time.Second))
	} else if addr.TCPAddr != nil {
		conn.TCPConn, err = net.DialTCP("tcp", nil, addr.TCPAddr)
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.ConnectionTimeout) * time.Second))
	} else {
		err = errors.New("socket address not initialize")
	}
	return
}

func (conn *SocketConn) ReadPacket(maxSize int) (readBytes []byte, n int, err error) {
	if conn.IsDead() {
		err = errors.New("connection is dead")
		return
	}
	if err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.RWTimeoutMs) * time.Millisecond)); err != nil {
		return
	}
	if conn.UDPConn != nil {
		readBytes = make([]byte, maxSize)
		n, err = conn.UDPConn.Read(readBytes)
		if err != nil {
			return
		}
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.ConnectionTimeout) * time.Second))
	} else if conn.TCPConn != nil {
		readBytes, n, err = ReadPacketFromTCPConn(conn.TCPConn)
		if err != nil {
			return
		}
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.ConnectionTimeout) * time.Second))
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
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.ConnectionTimeout) * time.Second))
	} else if conn.TCPConn != nil {
		n, err = WritePacketToTCPConn(packetBytes, conn.TCPConn)
		if err != nil {
			return
		}
		err = conn.SetDeadline(time.Now().Add(time.Duration(common.Config.Advanced.ConnectionTimeout) * time.Second))
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
		err = conn.TCPConn.SetDeadline(t)
	} else {
		err = errors.New("socket connection not initialize")
	}
	return
}

func (conn *SocketConn) IsDead() bool {
	if conn.closed || (conn.TCPConn == nil && conn.UDPConn == nil) {
		return true
	}
	if conn.deadTime != 0 && time.Now().UnixNano() > conn.deadTime {
		_ = conn.Close()
		return true
	}
	return false
}

func (conn *SocketConn) Close() (err error) {
	if conn.closed {
		err = errors.New("connection is dead")
		return
	}
	if conn.UDPConn != nil {
		err = conn.UDPConn.Close()
	} else if conn.TCPConn != nil {
		err = conn.TCPConn.Close()
	}
	conn.closed = true
	return
}

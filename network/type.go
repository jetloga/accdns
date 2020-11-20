package network

import (
	"net"
)

type SocketAddr struct {
	UDPAddr *net.UDPAddr
	TCPAddr *net.TCPAddr
}

type SocketConn struct {
	SocketAddr *SocketAddr
	UDPConn    *net.UDPConn
	TCPConn    *net.TCPConn
	deadTime   int64
	closed     bool
}

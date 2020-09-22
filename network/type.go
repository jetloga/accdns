package network

import (
	"net"
)

type SocketAddr struct {
	UDPAddr *net.UDPAddr
	TCPAddr *net.TCPAddr
}

type SocketConn struct {
	UDPConn  *net.UDPConn
	TCPConn  *net.TCPConn
	deadTime int64
}

type ConnPool struct {
	connectionMap map[string]*SocketConn
}

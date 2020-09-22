package network

import (
	"net"
	"sync"
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
}

type ConnPool struct {
	connChanMapLocker sync.RWMutex
	connChanMap       map[string]chan *SocketConn
}

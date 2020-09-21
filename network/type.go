package network

import "net"

type SocketAddr struct {
	*net.UDPAddr
	*net.TCPAddr
	Network string
}

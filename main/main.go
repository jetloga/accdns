package main

import (
	"DnsDiversion/common"
	"DnsDiversion/diversion"
	"flag"
	"net"
	"sync"
)

var configFilePath = flag.String("c", "", "Config File Path")

func main() {
	flag.Parse()
	if err := common.Init(*configFilePath); err != nil {
		common.Error(err)
		return
	}

	waitGroup := sync.WaitGroup{}
	if common.Config.Service.ListenUDP {
		func() {
			waitGroup.Add(1)
			defer waitGroup.Done()
			udpAddr, err := net.ResolveUDPAddr("udp", common.Config.Service.ListenAddr)
			listener, err := net.ListenUDP("udp", udpAddr)
			if err != nil {
				common.Error("[MAIN]", "{Listen UDP}", err)
				return
			}
			for true {
				buffer := make([]byte, common.Config.Advanced.MaxReceivedPacketSize)
				n, addr, err := listener.ReadFromUDP(buffer)
				if err != nil {
					common.Warning("[MAIN]", "{Read UDP Packet}", addr, err)
					continue
				}
				common.Debug("[MAIN]", "{Read UDP Packet}", "Read", n, "bytes from", addr)
				go func() {
					if err = diversion.HandlePacket(buffer, func(bytes []byte) {
						n, err := listener.WriteToUDP(bytes, addr)
						if err != nil {
							common.Warning("[MAIN]", "{Write UDP Packet}", addr, err)
						}
						common.Debug("[MAIN]", "{Write UDP Packet}", "Write", n, "bytes to", addr)
					}); err != nil {
						common.Warning("[MAIN]", "{Handle DNS Packet}", addr, err)
					}
				}()
			}
		}()
	}

	if common.Config.Service.ListenTCP {
		func() {
			waitGroup.Add(1)
			defer waitGroup.Done()
			tcpAddr, err := net.ResolveTCPAddr("tcp", common.Config.Service.ListenAddr)
			listener, err := net.ListenTCP("tcp", tcpAddr)
			if err != nil {
				common.Error(err)
				return
			}
			for true {
				conn, err := listener.AcceptTCP()
				if err != nil {
					common.Error(err)
					continue
				}
				go func() {
					buffer := make([]byte, common.Config.Advanced.MaxReceivedPacketSize)
					n, err := conn.Read(buffer)
					if err != nil {
						common.Error("[MAIN]", "{Read DNS Packet From TCP Connection}", conn.RemoteAddr(), err)
						if err := conn.Close(); err != nil {
							common.Error("[MAIN]", "{Close TCP Connection}", conn.RemoteAddr(), err)
						}
						return
					}
					common.Debug("[MAIN]", "{Read DNS Packet From TCP Connection}", "Read", n, "bytes from", conn.RemoteAddr())
					if err = diversion.HandlePacket(buffer, func(bytes []byte) {

					}); err != nil {
						common.Error(err)
					}
				}()
			}
		}()
	}
	waitGroup.Wait()
}

package main

import (
	"DnsDiversion/common"
	"DnsDiversion/diversion"
	"DnsDiversion/logger"
	"DnsDiversion/network"
	"flag"
	"net"
	"sync"
)

var configFilePath = flag.String("c", "", "Config File Path")
var newConfigFilePath = flag.String("n", "", "New Config File Path")

func main() {
	flag.Parse()
	if *newConfigFilePath != "" {
		if err := common.CreateConfigFile(*newConfigFilePath); err != nil {
			logger.Error("Create Configuration File", err)
		}
		return
	}
	if err := common.Init(*configFilePath); err != nil {
		logger.Error("Initialize", err)
		return
	}
	if err := logger.Init(); err != nil {
		logger.Error("Logger Initialize", err)
		return
	}
	if err := network.Init(); err != nil {
		logger.Error("Network Initialize", err)
		return
	}
	waitGroup := sync.WaitGroup{}
	if common.Config.Service.ListenUDP {
		udpAddr, err := net.ResolveUDPAddr("udp", common.Config.Service.ListenAddr)
		listener, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			logger.Error("Listen UDP", err)
			return
		}
		logger.Alert("Listen UDP", "listen on", common.Config.Service.ListenAddr)
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for true {
				bufferBytes := make([]byte, common.Config.Advanced.DefaultMaxPacketSize)
				n, addr, err := listener.ReadFromUDP(bufferBytes)
				if err != nil {
					logger.Warning("Read UDP Packet", addr, err)
					continue
				}
				if common.NeedDebug() {
					logger.Debug("Read UDP Packet", bufferBytes)
					logger.Debug("Read UDP Packet", "Read", n, "bytes from", addr)
				}
				go func() {
					if err = diversion.HandlePacket(bufferBytes, func(respBytes []byte) {
						n, err := listener.WriteToUDP(respBytes, addr)
						if err != nil {
							logger.Warning("Write UDP Packet", addr, err)
						}
						if common.NeedDebug() {
							logger.Debug("Write UDP Packet", respBytes)
							logger.Debug("Write UDP Packet", "Write", n, "bytes to", addr)
						}
					}); err != nil {
						logger.Warning("Handle DNS Packet", addr, err)
					}
				}()
			}
		}()
	}

	if common.Config.Service.ListenTCP {
		tcpAddr, err := net.ResolveTCPAddr("tcp", common.Config.Service.ListenAddr)
		listener, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			logger.Error("Listen TCP", err)
			return
		}
		logger.Alert("Listen TCP", "listen on", common.Config.Service.ListenAddr)
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for true {
				conn, err := listener.AcceptTCP()
				if err != nil {
					logger.Error("Establish TCP Connection", err)
					continue
				}
				go func() {
					defer func() {
						if err := conn.Close(); err != nil {
							logger.Warning("Close TCP Connection", conn.RemoteAddr(), err)
						}
					}()
					readBytes, n, err := network.ReadPacketFromTCPConn(conn)
					if err != nil {
						logger.Warning("Read DNS Packet from TCP Connection", conn.RemoteAddr(), err)
					}
					if common.NeedDebug() {
						logger.Debug("Read DNS Packet from TCP Connection", readBytes)
						logger.Debug("Read DNS Packet from TCP Connection", "Read", n, "bytes from", conn.RemoteAddr())
					}
					if err = diversion.HandlePacket(readBytes, func(respBytes []byte) {
						n, err := network.WritePacketToTCPConn(respBytes, conn)
						if err != nil {
							logger.Warning("Write DNS Packet to TCP Connection", conn.RemoteAddr(), err)
						}
						if common.NeedDebug() {
							logger.Debug("Write DNS Packet to TCP Connection", respBytes)
							logger.Debug("Write DNS Packet to TCP Connection", "Write", n, "bytes to", conn.RemoteAddr())
						}
					}); err != nil {
						logger.Warning("Handle DNS Packet", err)
					}
				}()
			}
		}()
	}
	logger.Alert("General", "DnsDiversion Started")
	waitGroup.Wait()
}

package main

import (
	"DnsDiversion/common"
	"DnsDiversion/diversion"
	"bytes"
	"encoding/binary"
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
			common.Error("MAIN", "{Create Configuration File}", err)
		}
		return
	}
	if err := common.Init(*configFilePath); err != nil {
		common.Error("MAIN", "{Initialize}", err)
		return
	}

	waitGroup := sync.WaitGroup{}
	if common.Config.Service.ListenUDP {
		udpAddr, err := net.ResolveUDPAddr("udp", common.Config.Service.ListenAddr)
		listener, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			common.Error("[MAIN]", "{Listen UDP}", err)
			return
		}
		common.Alert("[MAIN]", "{Listen UDP}", "listen on", common.Config.Service.ListenAddr)
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for true {
				bufferBytes := make([]byte, common.Config.Advanced.MaxReceivedPacketSize)
				n, addr, err := listener.ReadFromUDP(bufferBytes)
				bufferBytes = bytes.TrimRight(bufferBytes, "\x00")
				if err != nil {
					common.Warning("[MAIN]", "{Read UDP Packet}", addr, err)
					continue
				}
				if common.IfDebug() {
					common.Debug("[MAIN]", "{Read UDP Packet}", bufferBytes)
					common.Debug("[MAIN]", "{Read UDP Packet}", "Read", n, "bytes from", addr)
				}
				go func() {
					if err = diversion.HandlePacket(bufferBytes, func(respBytes []byte) {
						n, err := listener.WriteToUDP(respBytes, addr)
						if err != nil {
							common.Warning("[MAIN]", "{Write UDP Packet}", addr, err)
						}
						if common.IfDebug() {
							common.Debug("[MAIN]", "{Write UDP Packet}", respBytes)
							common.Debug("[MAIN]", "{Write UDP Packet}", "Write", n, "bytes to", addr)
						}
					}); err != nil {
						common.Warning("[MAIN]", "{Handle DNS Packet}", addr, err)
					}
				}()
			}
		}()
	}

	if common.Config.Service.ListenTCP {
		tcpAddr, err := net.ResolveTCPAddr("tcp", common.Config.Service.ListenAddr)
		listener, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			common.Error(err)
			return
		}
		common.Alert("[MAIN]", "{Listen TCP}", "listen on", common.Config.Service.ListenAddr)
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for true {
				conn, err := listener.AcceptTCP()
				if err != nil {
					common.Error(err)
					continue
				}
				go func() {
					defer func() {
						if err := conn.Close(); err != nil {
							common.Warning("[MAIN]", "{Close TCP Connection}", conn.RemoteAddr(), err)
						}
					}()
					bufferBytes := make([]byte, common.Config.Advanced.MaxReceivedPacketSize)
					n, err := conn.Read(bufferBytes)
					bufferBytes = bytes.TrimRight(bufferBytes, "\x00")
					if err != nil {
						common.Warning("[MAIN]", "{Read DNS Packet from TCP Connection}", conn.RemoteAddr(), err)
						return
					}
					if common.IfDebug() {
						common.Debug("[MAIN]", "{Read DNS Packet from TCP Connection}", bufferBytes)
						common.Debug("[MAIN]", "{Read DNS Packet from TCP Connection}", "Read", n, "bytes from", conn.RemoteAddr())
					}
					if n >= 2 {
						size := uint16(0)
						if err := binary.Read(bytes.NewBuffer(bufferBytes[:2]), binary.BigEndian, &size); err != nil {
							common.Warning("[MAIN]", "{Read Packet Size}", conn.RemoteAddr(), err)
							return
						}
						if n-int(size) != 2 {
							common.Warning("[MAIN]", "{Check Packet Size}", conn.RemoteAddr(), "packet size not match")
							return
						}
						if err = diversion.HandlePacket(bufferBytes[2:], func(respBytes []byte) {
							size := uint16(len(respBytes))
							buffer := bytes.NewBuffer([]byte{})
							if err := binary.Write(buffer, binary.BigEndian, size); err != nil {
								common.Warning("[MAIN]", "{Build Packet}", conn.RemoteAddr(), err)
								return
							}
							completeBytes := buffer.Bytes()
							completeBytes = append(completeBytes, respBytes...)
							n, err := conn.Write(completeBytes)
							if err != nil {
								common.Warning("[MAIN]", "{Write DNS Packet to TCP Connection}", conn.RemoteAddr(), err)
							}
							if common.IfDebug() {
								common.Debug("[MAIN]", "{Write DNS Packet to TCP Connection}", completeBytes)
								common.Debug("[MAIN]", "{Write DNS Packet to TCP Connection}", "Write", n, "bytes to", conn.RemoteAddr())
							}
						}); err != nil {
							common.Warning(err)
						}
					}

				}()
			}
		}()
	}
	common.Alert("[MAIN]", "{General}", "DnsDiversion Started")
	waitGroup.Wait()
}

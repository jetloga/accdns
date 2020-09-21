package common

import (
	"errors"
	"fmt"
	"github.com/phachon/go-logger"
	"golang.org/x/net/dns/dnsmessage"
	"gopkg.in/ini.v1"
	"net"
	"strconv"
	"strings"
)

var Logger = go_logger.NewLogger()
var Config = &ConfigStruct{
	Service: &ServiceConfig{
		ListenAddr: "127.0.0.1:53",
		ListenUDP:  true,
		ListenTCP:  false,
	},
	Upstream: &UpstreamConfig{
		UseUDP:               true,
		UseTCP:               false,
		DefaultUpstreams:     make([]string, 0),
		ARecordUpstreams:     make([]string, 0),
		AAAARecordUpstreams:  make([]string, 0),
		CNAMERecordUpstreams: make([]string, 0),
		TXTRecordUpstreams:   make([]string, 0),
		PTRRecordUpstreams:   make([]string, 0),
		CustomRecordUpstream: make([]string, 0),
	},
	Log: &LogConfig{
		LogFilePath:        "",
		LogFileMaxSizeKB:   16 * 1024,
		LogLevelForFile:    "info",
		LogLevelForConsole: "info",
	},
	Advanced: &AdvancedConfig{
		NSLookupTimeoutMs:     10000,
		MaxReceivedPacketSize: 512,
	},
}

var UpstreamsList [256][]*SocketAddr

func Init(configFilePath string) error {

	if configFilePath != "" {
		cfg, err := ini.Load(configFilePath)
		if err != nil {
			return err
		}
		if err := cfg.MapTo(Config); err != nil {
			return err
		}
	}
	switch Config.Log.LogLevelForConsole {
	case "debug":
		_ = Logger.Detach("console")
		_ = Logger.Attach("console", go_logger.LOGGER_LEVEL_DEBUG, &go_logger.ConsoleConfig{})
	case "info":
		_ = Logger.Detach("console")
		_ = Logger.Attach("console", go_logger.LOGGER_LEVEL_INFO, &go_logger.ConsoleConfig{})
	case "warning":
		_ = Logger.Detach("console")
		_ = Logger.Attach("console", go_logger.LOGGER_LEVEL_WARNING, &go_logger.ConsoleConfig{})
	case "error":
		_ = Logger.Detach("console")
		_ = Logger.Attach("console", go_logger.LOGGER_LEVEL_ERROR, &go_logger.ConsoleConfig{})
	case "none":
		_ = Logger.Detach("console")
	default:
		Error("[COMMON]", "{Set Log Level}", "unknown log level", Config.Log.LogLevelForConsole)
	}

	if Config.Log.LogFilePath != "" {
		logFileConfig := &go_logger.FileConfig{
			Filename:  Config.Log.LogFilePath,
			MaxSize:   Config.Log.LogFileMaxSizeKB,
			DateSlice: "d",
		}
		switch Config.Log.LogLevelForFile {
		case "debug":
			_ = Logger.Attach("file", go_logger.LOGGER_LEVEL_DEBUG, logFileConfig)
		case "info":
			_ = Logger.Attach("file", go_logger.LOGGER_LEVEL_INFO, logFileConfig)
		case "warning":
			_ = Logger.Attach("file", go_logger.LOGGER_LEVEL_WARNING, logFileConfig)
		case "error":
			_ = Logger.Attach("file", go_logger.LOGGER_LEVEL_ERROR, logFileConfig)
		case "none":
		default:
			Error("[COMMON]", "{Set Log Level}", "unknown log level", Config.Log.LogLevelForFile)
		}
	}
	for typeCode := range UpstreamsList {
		switch dnsmessage.Type(typeCode) {
		case dnsmessage.Type(0):
			UpstreamsList[typeCode] = make([]*SocketAddr, len(Config.Upstream.DefaultUpstreams))
			for i, upstreamStr := range Config.Upstream.DefaultUpstreams {
				socketAddr, err := NewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				Info("[COMMON]", "{Load Default Upstream}", socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeA:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(Config.Upstream.ARecordUpstreams))
			for i, upstreamStr := range Config.Upstream.ARecordUpstreams {
				socketAddr, err := NewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				Info("[COMMON]", "{Load Upstream For A Record}", socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeAAAA:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(Config.Upstream.AAAARecordUpstreams))
			for i, upstreamStr := range Config.Upstream.AAAARecordUpstreams {
				socketAddr, err := NewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				Info("[COMMON]", "{Load Upstream For AAAA Record}", socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeCNAME:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(Config.Upstream.CNAMERecordUpstreams))
			for i, upstreamStr := range Config.Upstream.CNAMERecordUpstreams {
				socketAddr, err := NewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				Info("[COMMON]", "{Load Upstream For CNAME Record}", socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypeTXT:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(Config.Upstream.TXTRecordUpstreams))
			for i, upstreamStr := range Config.Upstream.TXTRecordUpstreams {
				socketAddr, err := NewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				Debug("[COMMON]", "{Load Upstream For TXT Record}", socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		case dnsmessage.TypePTR:
			UpstreamsList[typeCode] = make([]*SocketAddr, len(Config.Upstream.PTRRecordUpstreams))
			for i, upstreamStr := range Config.Upstream.PTRRecordUpstreams {
				socketAddr, err := NewSocketAddr(upstreamStr)
				if err != nil {
					return err
				}
				Debug("[COMMON]", "{Load Upstream For PTR Record}", socketAddr.UDPAddr.String())
				UpstreamsList[typeCode][i] = socketAddr
			}
		default:
			UpstreamsList[typeCode] = make([]*SocketAddr, 0)
		}

	}
	for _, kvPair := range Config.Upstream.CustomRecordUpstream {
		typeCodeStr, addr, err := ParseKVPair(kvPair)
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
		socketAddr, err := NewSocketAddr(addr)
		if err != nil {
			return err
		}
		UpstreamsList[typeCode] = append(UpstreamsList[typeCode], socketAddr)
	}
	return nil
}
func CreateConfigFile(configFilePath string) error {
	cfg := ini.Empty()
	if err := cfg.ReflectFrom(Config); err != nil {
		return err
	}
	if err := cfg.SaveTo(configFilePath); err != nil {
		return err
	}
	return nil
}
func ParseKVPair(kvPair string) (key, value string, err error) {
	index := strings.Index(kvPair, ":")
	if index < 0 {
		return "", "", errors.New("key-value pair \"" + kvPair + "\" is not correct")
	}
	return kvPair[:index], kvPair[index+1:], nil
}

func NewSocketAddr(addr string) (*SocketAddr, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &SocketAddr{
		UDPAddr: udpAddr,
		TCPAddr: tcpAddr,
	}, nil
}

func Error(objs ...interface{}) {
	msg := ""
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Error(strings.TrimSpace(msg))
}
func Alert(objs ...interface{}) {
	msg := ""
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Alert(strings.TrimSpace(msg))
}
func Warning(objs ...interface{}) {
	msg := ""
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Warning(strings.TrimSpace(msg))
}
func Info(objs ...interface{}) {
	msg := ""
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Info(strings.TrimSpace(msg))
}
func Debug(objs ...interface{}) {
	msg := ""
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Debug(strings.TrimSpace(msg))
}
func IfDebug() bool {
	return Config.Log.LogLevelForFile == "debug" || Config.Log.LogLevelForConsole == "debug"
}

package common

import (
	"net"
	"sync"
)

type ConfigStruct struct {
	Service  *ServiceConfig
	Upstream *UpstreamConfig
	Log      *LogConfig
	Advanced *AdvancedConfig
}

type ServiceConfig struct {
	ListenAddr string `comment:"Listen Address (Example: [::]:53)"`
	ListenUDP  bool   `comment:"Listen on UDP Port"`
	ListenTCP  bool   `comment:"Listen on TCP Port"`
}

type UpstreamConfig struct {
	UseUDP               bool     `comment:"Use UDP Protocol to Access Upstream"`
	UseTCP               bool     `comment:"Use TCP Protocol to Access Upstream"`
	DefaultUpstreams     []string `comment:"Upstream List for Non-specific Record (Example: 223.5.5.5:53,223.6.6.6,[2001:da8::666]:53)"`
	ARecordUpstreams     []string `comment:"Upstream List for A Record"`
	AAAARecordUpstreams  []string `comment:"Upstream List for AAAA Record"`
	CNAMERecordUpstreams []string `comment:"Upstream List for CNAME Record"`
	TXTRecordUpstreams   []string `comment:"Upstream List for TXT Record"`
	PTRRecordUpstreams   []string `comment:"Upstream List for PTR Record"`
	CustomRecordUpstream []string `comment:"Upstream List for Custom Record (Example: 1:223.5.5.5:53,1:223.6.6.6,28:[2001:da8::666]:53)"`
}

type LogConfig struct {
	LogFilePath        string `comment:"Log File Path"`
	LogFileMaxSize     int    `comment:"Max Size of Log File"`
	LogLevelForFile    string `comment:"Log Level for Log File"`
	LogLevelForConsole string `comment:"Log Level for Console"`
}

type AdvancedConfig struct {
	NSLookupTimeoutMs     int
	MaxReceivedPacketSize int
}

type SocketAddr struct {
	*net.UDPAddr
	*net.TCPAddr
}

type SafeQueue struct {
	mutex    sync.Mutex
	contents []interface{}
}

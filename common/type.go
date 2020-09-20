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
	ListenUDP  bool
	ListenTCP  bool
}

type UpstreamConfig struct {
	UseUDP               bool
	UseTCP               bool
	DefaultUpstreams     []string `comment:"Upstream List for Non-specific Record (Example: 223.5.5.5:53,223.6.6.6,[2001:da8::666]:53)"`
	ARecordUpstreams     []string `comment:"Upstream List for A Record (Example: 223.5.5.5,223.6.6.6,2001:da8::666)"`
	AAAARecordUpstreams  []string `comment:"Upstream List for AAAA Record (Example: 223.5.5.5,223.6.6.6,2001:da8::666)"`
	CNAMERecordUpstreams []string `comment:"Upstream List for CNAME Record (Example: 223.5.5.5,223.6.6.6,2001:da8::666)"`
	TXTRecordUpstreams   []string `comment:"Upstream List for TXT Record (Example: 223.5.5.5,223.6.6.6,2001:da8::666)"`
	PTRRecordUpstreams   []string
	CustomRecordUpstream []string `comment:"Upstream List for Custom Record (Example: 1:223.5.5.5:53,1:223.6.6.6,28:[2001:da8::666]:53)"`
}

type LogConfig struct {
	LogFilePath        string
	LogFileMaxSize     int
	LogLevelForFile    string
	LogLevelForConsole string
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

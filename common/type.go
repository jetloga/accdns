package common

type ConfigStruct struct {
	Service  *ServiceConfig
	Upstream *UpstreamConfig
	Cache    *CacheConfig
	Log      *LogConfig
	Advanced *AdvancedConfig
}

type ServiceConfig struct {
	ListenAddr string `comment:"Listen Address (Example: [::]:53)"`
	ListenUDP  bool   `comment:"Listen on UDP Port"`
	ListenTCP  bool   `comment:"Listen on TCP Port"`
}

type UpstreamConfig struct {
	DefaultUpstreams     []string `comment:"Upstream List for Non-specific Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)"`
	ARecordUpstreams     []string `comment:"Upstream List for A Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)"`
	AAAARecordUpstreams  []string `comment:"Upstream List for AAAA Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)"`
	CNAMERecordUpstreams []string `comment:"Upstream List for CNAME Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)"`
	TXTRecordUpstreams   []string `comment:"Upstream List for TXT Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)"`
	PTRRecordUpstreams   []string `comment:"Upstream List for PTR Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)"`
	CustomRecordUpstream []string `comment:"Upstream List for Custom Record (Example: 1:223.5.5.5,1:udp:223.6.6.6:53,1:tcp:208.67.222.222,28:2001:da8::666,28:[2620:0:ccd::2]:53,28:tcp:2620:0:ccc::2)"`
}

type LogConfig struct {
	LogFilePath        string `comment:"Log File Path"`
	LogFileMaxSizeKB   int64  `comment:"Max Size of Log File (KB)"`
	LogLevelForFile    string `comment:"Log Level for Log File"`
	LogLevelForConsole string `comment:"Log Level for Console"`
}

type AdvancedConfig struct {
	NSLookupTimeoutMs            int
	RWTimeoutMs                  int
	DefaultMaxPacketSize         int
	IdleConnectionTimeout        int
	MaxIdleConnectionPerUpstream int
}

type CacheConfig struct {
	EnableCache bool
	MaxTTL      int
	MinTTL      int
}

package common

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
	DefaultUpstreams     []string `comment:"Upstream List for Non-specific Record (Example: 223.5.5.5:53,223.6.6.6:53,[2001:da8::666]:53)"`
	ARecordUpstreams     []string `comment:"Upstream List for A Record (Example: 223.5.5.5:53,223.6.6.6:53)"`
	AAAARecordUpstreams  []string `comment:"Upstream List for AAAA Record (Example: [2001:da8::666]:53)"`
	CNAMERecordUpstreams []string `comment:"Upstream List for CNAME Record"`
	TXTRecordUpstreams   []string `comment:"Upstream List for TXT Record"`
	PTRRecordUpstreams   []string `comment:"Upstream List for PTR Record"`
	CustomRecordUpstream []string `comment:"Upstream List for Custom Record (Example: 1:223.5.5.5:53,1:223.6.6.6,28:[2001:da8::666]:53)"`
}

type LogConfig struct {
	LogFilePath        string `comment:"Log File Path"`
	LogFileMaxSizeKB   int64  `comment:"Max Size of Log File (KB)"`
	LogLevelForFile    string `comment:"Log Level for Log File"`
	LogLevelForConsole string `comment:"Log Level for Console"`
}

type AdvancedConfig struct {
	NSLookupTimeoutMs      int
	RWTimeoutMs            int
	MaxReceivedPacketSize  int
	MaxNumOfIdleConnection int
	IdleConnectionTimeout  int
}

# AccDNS
A DNS forwarder that forwards according to the type of DNS message

### Usage
-c path &nbsp;&nbsp;&nbsp;&nbsp; Specify config file path

-n path &nbsp;&nbsp;&nbsp;&nbsp; Create config file template

### Configuration File
```ini
[Service]
; Listen Address (Example: [::]:53)
ListenAddr = [::]:53
; Listen on UDP Port
ListenUDP  = true
; Listen on TCP Port
ListenTCP  = false

[Upstream]
; Upstream List for Non-specific Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)
DefaultUpstreams     =
; Upstream List for A Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)
ARecordUpstreams     =
; Upstream List for AAAA Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)
AAAARecordUpstreams  =
; Upstream List for CNAME Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)
CNAMERecordUpstreams =
; Upstream List for TXT Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)
TXTRecordUpstreams   =
; Upstream List for PTR Record (Example: 223.5.5.5,udp:223.6.6.6:53,tcp:208.67.222.222,2001:da8::666,[2620:0:ccd::2]:53,tcp:2620:0:ccc::2)
PTRRecordUpstreams   =
; Upstream List for Custom Record (Example: 1:223.5.5.5,1:udp:223.6.6.6:53,1:tcp:208.67.222.222,28:2001:da8::666,28:[2620:0:ccd::2]:53,28:tcp:2620:0:ccc::2)
CustomRecordUpstream =

[Cache]
EnableCache = true
MaxTTL      = 3600
MinTTL      = 10

[Log]
; Log File Path
LogFilePath        = accdns.log
; Max Size of Log File (KB)
LogFileMaxSizeKB   = 16384
; Log Level for Log File
LogLevelForFile    = info
; Log Level for Console
LogLevelForConsole = info

[Advanced]
NSLookupTimeoutMs     = 20000
RWTimeoutMs           = 8000
MaxReceivedPacketSize = 4096
ConnectionTimeout     = 60
NetworkFailedRetries  = 3
```

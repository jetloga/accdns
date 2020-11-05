package common

import (
	"errors"
	"gopkg.in/ini.v1"
	"strings"
)

const StandardMaxDNSPacketSize = 512

var Config = &ConfigStruct{
	Service: &ServiceConfig{
		ListenAddr: "127.0.0.1:53",
		ListenUDP:  true,
		ListenTCP:  false,
	},
	Upstream: &UpstreamConfig{
		DefaultUpstreams:     make([]string, 0),
		ARecordUpstreams:     make([]string, 0),
		AAAARecordUpstreams:  make([]string, 0),
		CNAMERecordUpstreams: make([]string, 0),
		TXTRecordUpstreams:   make([]string, 0),
		PTRRecordUpstreams:   make([]string, 0),
		CustomRecordUpstream: make([]string, 0),
	},
	Cache: &CacheConfig{
		EnableCache:       true,
		MaxTTL:            3600,
		MinTTL:            10,
		MinLookupInterval: 10,
	},
	Log: &LogConfig{
		LogFilePath:        "",
		LogFileMaxSizeKB:   16 * 1024,
		LogLevelForFile:    "info",
		LogLevelForConsole: "info",
	},
	Advanced: &AdvancedConfig{
		NSLookupTimeoutMs:            10000,
		RWTimeoutMs:                  6000,
		DefaultMaxPacketSize:         512,
		MaxIdleConnectionPerUpstream: 32,
		IdleConnectionTimeout:        60,
	},
}

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

func NeedDebug() bool {
	return Config.Log.LogLevelForFile == "debug" || Config.Log.LogLevelForConsole == "debug"
}

func IntMin(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}
func IntMax(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

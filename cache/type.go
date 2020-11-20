package cache

import (
	"golang.org/x/net/dns/dnsmessage"
	"sync"
)

type Cache struct {
	cacheMap sync.Map
	MaxTTL   int
	MinTTL   int
}
type Item struct {
	UpdateAt int64
	TTL      int
	Msg      *dnsmessage.Message
}

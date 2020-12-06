package cache

import (
	"accdns/common"
	"accdns/logger"
	"accdns/network"
	"errors"
	"golang.org/x/net/dns/dnsmessage"
	"time"
)

func (dnsCache *Cache) UpdateItem(item *Item, msg *dnsmessage.Message) {
	if msg.Header.RCode == dnsmessage.RCodeSuccess && msg.Header.Truncated == false && len(msg.Answers) > 0 {
		itemTTL := dnsCache.MaxTTL
		for _, res := range msg.Answers {
			if int(res.Header.TTL) < itemTTL {
				if int(res.Header.TTL) > dnsCache.MinTTL {
					itemTTL = int(res.Header.TTL)
				} else {
					itemTTL = dnsCache.MinTTL
				}
			}
		}
		for _, res := range msg.Authorities {
			if int(res.Header.TTL) < itemTTL {
				if int(res.Header.TTL) > dnsCache.MinTTL {
					itemTTL = int(res.Header.TTL)
				} else {
					itemTTL = dnsCache.MinTTL
				}
			}
		}
		for _, res := range msg.Additionals {
			if int(res.Header.TTL) < itemTTL {
				if int(res.Header.TTL) > dnsCache.MinTTL {
					itemTTL = int(res.Header.TTL)
				} else {
					itemTTL = dnsCache.MinTTL
				}
			}
		}
		item.Msg = msg
		item.TTL = itemTTL
		item.UpdateAt = time.Now().UnixNano()
	}
}

func (dnsCache *Cache) QueryAndUpdate(queryMsg *dnsmessage.Message, upstream *network.SocketAddr, updateFunc func(*dnsmessage.Message, *network.SocketAddr) (*dnsmessage.Message, error)) (*dnsmessage.Message, error) {
	if queryMsg == nil || len(queryMsg.Questions) < 1 {
		return nil, errors.New("wrong dns message")
	}
	question := &queryMsg.Questions[0]
	key := question.Name.String() + "|" + question.Class.String() + "|" + question.Type.String()
	var item *Item
	rawItem, ok := dnsCache.cacheMap.Load(key)
	if ok && rawItem != nil {
		item = rawItem.(*Item)
		if item.Msg != nil && time.Now().UnixNano() < item.UpdateAt+(time.Duration(item.TTL)*time.Second).Nanoseconds() {
			if common.NeedDebug() {
				logger.Debug("Cache Hit", question.Name, question.Class, question.Type)
			}
			return item.Msg, nil
		}
		if common.NeedDebug() {
			logger.Debug("Cache Invalid", question.Name, question.Class, question.Type)
		}
	} else {
		item = &Item{}
		dnsCache.cacheMap.Store(key, item)
	}
	if common.NeedDebug() {
		logger.Debug("Cache Miss", question.Name, question.Class, question.Type)
	}
	msg, err := updateFunc(queryMsg, upstream)
	if err != nil {
		return nil, err
	}
	dnsCache.UpdateItem(item, msg)
	return msg, nil
}

package cache

import (
	"accdns/common"
	"accdns/logger"
	"golang.org/x/net/dns/dnsmessage"
	"sync"
	"time"
)

func (dnsCache *Cache) UpdateItem(item *Item, msg *dnsmessage.Message) {
	if msg.Header.RCode == dnsmessage.RCodeSuccess {
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

func (dnsCache *Cache) QueryAndUpdate(question *dnsmessage.Question, updateFunc func() (*dnsmessage.Message, error)) (*dnsmessage.Message, error) {
	key := Key{
		Name:  question.Name,
		Class: question.Class,
		Type:  question.Type,
	}
	var item *Item
	rawItem, ok := dnsCache.cacheMap.Load(key)
	if ok && rawItem != nil {
		item = rawItem.(*Item)
		if time.Now().UnixNano() < item.UpdateAt+(time.Duration(item.TTL)*time.Second).Nanoseconds() {
			if common.NeedDebug() {
				logger.Debug("Cache Hit", question.Name, question.Class, question.Type)
			}
			return item.Msg, nil
		}
	} else {
		item = &Item{Mutex: &sync.Mutex{}}
		dnsCache.cacheMap.Store(key, item)
	}
	item.Mutex.Lock()
	defer item.Mutex.Unlock()
	if time.Now().UnixNano() < item.UpdateAt+(time.Duration(item.TTL)*time.Second).Nanoseconds() {
		if common.NeedDebug() {
			logger.Debug("Cache Hit", question.Name, question.Class, question.Type)
		}
		return item.Msg, nil
	}
	if common.NeedDebug() {
		logger.Debug("Cache Miss", question.Name, question.Class, question.Type)
	}
	msg, err := updateFunc()
	if err != nil {
		return nil, err
	}
	dnsCache.UpdateItem(item, msg)
	return msg, nil
}

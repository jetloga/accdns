package network

import (
	"accdns/common"
	"errors"
	"sync"
)

func NewConnPool() *ConnPool {
	return &ConnPool{
		connChanMapLocker: sync.RWMutex{},
		connChanMap:       make(map[string]chan *SocketConn),
	}
}
func (connPool *ConnPool) RequireConn(addr *SocketAddr) (conn *SocketConn, err error) {
	connPool.connChanMapLocker.RLock()
	connChan := connPool.connChanMap[addr.String()]
	connPool.connChanMapLocker.RUnlock()
	if connChan == nil {
		connChan = make(chan *SocketConn, common.Config.Advanced.MaxIdleConnectionPerUpstream)
		connPool.connChanMapLocker.Lock()
		connPool.connChanMap[addr.String()] = connChan
		connPool.connChanMapLocker.Unlock()
	}
	for conn == nil || conn.IsDead() {
		select {
		case conn = <-connChan:
		default:
			conn, err = EstablishNewSocketConn(addr)
			return
		}
	}
	return
}

func (connPool *ConnPool) ReleaseConn(conn *SocketConn) error {
	if conn.IsDead() {
		return errors.New("connection is dead")
	}
	connPool.connChanMapLocker.RLock()
	connChan := connPool.connChanMap[conn.SocketAddr.String()]
	connPool.connChanMapLocker.RUnlock()
	if connChan == nil {
		connChan = make(chan *SocketConn, common.Config.Advanced.MaxIdleConnectionPerUpstream)
		connPool.connChanMapLocker.Lock()
		connPool.connChanMap[conn.SocketAddr.String()] = connChan
		connPool.connChanMapLocker.Unlock()
	}
	select {
	case connChan <- conn:
	default:
		if err := conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

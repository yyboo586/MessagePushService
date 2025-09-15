package logics

import (
	"MessagePushService/interfaces"
	"context"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	wsConnManagerOnce     sync.Once
	wsConnManagerInstance *wsConnManager
)

type wsConnManager struct {
	wsConns       map[string]interfaces.ILogicsWsConn // 用户ID -> 连接
	mu            sync.RWMutex
	logicsMessage interfaces.ILogicsMessage
}

func NewWsConnManager(logicsMessage interfaces.ILogicsMessage) interfaces.ILogicsWsConnManager {
	wsConnManagerOnce.Do(func() {
		wsConnManagerInstance = &wsConnManager{
			wsConns:       make(map[string]interfaces.ILogicsWsConn, 10000),
			logicsMessage: logicsMessage,
		}
	})

	return wsConnManagerInstance
}

func (manager *wsConnManager) Add(conn *websocket.Conn, userInfo *interfaces.UserInfo) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	newConn := NewWsConn(manager, conn, userInfo, manager.logicsMessage)
	manager.wsConns[userInfo.ID] = newConn
}

func (manager *wsConnManager) Get(ctx context.Context, userID string) (conn interfaces.ILogicsWsConn) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	conn = manager.wsConns[userID]
	return
}

func (manager *wsConnManager) Remove(userID string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	delete(manager.wsConns, userID)
}

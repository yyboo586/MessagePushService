package logics

import (
	"MessagePushService/interfaces"
	"context"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	wsConnManagerOnce     sync.Once
	wsConnManagerInstance *wsConnManager
)

type wsConnManager struct {
	wsConns            map[string]interfaces.LogicsWsConn // 用户ID -> 连接
	mu                 sync.RWMutex
	messagePersistence interfaces.MessagePersistence
}

func NewWsConnManager(messagePersistence interfaces.MessagePersistence) interfaces.WsConnManager {
	wsConnManagerOnce.Do(func() {
		wsConnManagerInstance = &wsConnManager{
			wsConns:            make(map[string]interfaces.LogicsWsConn, 10000),
			messagePersistence: messagePersistence,
		}
	})

	return wsConnManagerInstance
}

func (manager *wsConnManager) Add(conn *websocket.Conn, userInfo *interfaces.UserInfo) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if existingConn, exists := manager.wsConns[userInfo.ID]; exists {
		if existingConn.IsAlive() {
			defer conn.Close()

			err := conn.WriteMessage(interfaces.CloseMessage, []byte("用户已登录"))
			if err != nil {
				log.Println(err)
			}
			return
		} else {
			defer existingConn.Close() // 旧连接会在函数返回时关闭

			delete(manager.wsConns, userInfo.ID)
		}
	}

	newConn := NewWsConn(manager, conn, userInfo, manager.messagePersistence)
	manager.wsConns[userInfo.ID] = newConn
	log.Println("用户", userInfo.ID, "连接成功")
}

func (manager *wsConnManager) Get(ctx context.Context, userID string) (conn interfaces.LogicsWsConn, err error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	conn = manager.wsConns[userID]
	return
}

func (manager *wsConnManager) Remove(userID string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	delete(manager.wsConns, userID)
	log.Println("用户", userID, "连接断开")
}

package logics

import (
	"MessagePushService/interfaces"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WsConn struct {
	manager            *wsConnManager
	messagePersistence interfaces.MessagePersistence

	conn *websocket.Conn

	UserInfo *interfaces.UserInfo

	WriteChan chan *interfaces.WsConnMessage

	ticker            *time.Ticker
	heartBeatInterval time.Duration // 心跳间隔时间
	heartBeatTimeout  time.Duration // 心跳超时时间, protected by mu
	lastPongTime      time.Time     // 上一次收到pong消息的时间, protected by mu
	alive             bool          // protected by mu
	mu                sync.RWMutex

	wg sync.WaitGroup

	ctx    context.Context
	calcel context.CancelFunc
}

func NewWsConn(manager *wsConnManager, conn *websocket.Conn, userInfo *interfaces.UserInfo, messagePersistence interfaces.MessagePersistence) interfaces.LogicsWsConn {
	ctx, cancel := context.WithCancel(context.Background())
	wsConn := &WsConn{
		manager: manager,
		conn:    conn,

		UserInfo:  userInfo,
		WriteChan: make(chan *interfaces.WsConnMessage, 1000),

		heartBeatInterval: time.Second * 15,
		heartBeatTimeout:  time.Second * 31,
		lastPongTime:      time.Now(),
		alive:             true,

		ctx:    ctx,
		calcel: cancel,
	}

	wsConn.ticker = time.NewTicker(wsConn.heartBeatInterval)
	wsConn.SetPongHandler()
	wsConn.SetCloseHandler()

	go wsConn.readPump()
	go wsConn.writePump()
	go wsConn.heartbeat()

	return wsConn
}

func (wsConn *WsConn) Close() {
	log.Printf("[DEBUG] Close wsConn begin, %s", wsConn.UserInfo.ID)

	// 如果连接已经关闭，则直接返回
	if !wsConn.IsAlive() {
		return
	}

	wsConn.mu.Lock()
	wsConn.alive = false
	wsConn.calcel()
	wsConn.mu.Unlock()

	wsConn.wg.Wait() // 等待所有goroutine退出

	wsConn.mu.Lock()
	wsConn.conn.Close()
	wsConn.mu.Unlock()

	log.Printf("[DEBUG] close wsConn end, %s", wsConn.UserInfo.ID)

	wsConn.manager.Remove(wsConn.UserInfo.ID)
}

func (wsConn *WsConn) IsAlive() bool {
	wsConn.mu.RLock()
	defer wsConn.mu.RUnlock()

	return wsConn.alive
}

func (wsConn *WsConn) readPump() {
	wsConn.wg.Add(1)
	defer wsConn.calcel()
	defer wsConn.wg.Done()

	for {
		select {
		case <-wsConn.ctx.Done():
			log.Printf("[DEBUG] readPump receive close signal from %s", wsConn.conn.RemoteAddr().String())
			return
		default:
		}

		messageType, message, err := wsConn.conn.ReadMessage()
		if err != nil {
			log.Printf("[ERROR] read message error, %v", err)
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if wsConn.IsAlive() {
					continue
				}
			}
			return
		}
		log.Printf("[DEBUG] receive message, messageType: %d, message: %s", messageType, string(message))

		wsConn.handleMessage(messageType, message)
	}
}

func (wsConn *WsConn) handleMessage(messageType int, message []byte) {
	switch messageType {
	case websocket.TextMessage:
		log.Printf("[DEBUG] receive text message, %s", string(message))
		var i interface{}
		err := json.Unmarshal(message, &i)
		if err != nil {
			log.Printf("[ERROR] unmarshal message error, %v", err)
			return
		}

		messageType, ok := i.(map[string]interface{})["message_type"].(int)
		if !ok {
			log.Printf("[ERROR] message_type is not a int")
			return
		}

		switch interfaces.MessageType(messageType) {
		case interfaces.MessageTypeACK:
			msgID, ok := i.(map[string]interface{})["message_id"].(string)
			if !ok {
				log.Printf("[ERROR] message_id is required")
				return
			}
			wsConn.messagePersistence.UpdateMessageStatus(wsConn.ctx, wsConn.UserInfo.ID, msgID, interfaces.MessageStatusSentSuccess)
		default:
			log.Printf("[ERROR] receive unexpected message type, %d, %s", messageType, string(message))
		}

	default:
		log.Printf("[ERROR] receive unexpected message type, %d, %s", messageType, string(message))
	}
}

func (wsConn *WsConn) writePump() {
	wsConn.wg.Add(1)
	defer wsConn.calcel()
	defer wsConn.wg.Done()

	for {
		select {
		case <-wsConn.ctx.Done():
			log.Printf("[DEBUG] writePump receive close signal from %s", wsConn.conn.RemoteAddr().String())
			return
		case message := <-wsConn.WriteChan:
			var err error
			switch message.Type {
			case interfaces.PingMessage:
				err = wsConn.conn.WriteMessage(websocket.PingMessage, message.Data)
			case interfaces.TextMessage:
				err = wsConn.conn.WriteMessage(websocket.TextMessage, message.Data)
			case interfaces.CloseMessage:
				err = wsConn.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server close"))
			default:
				err = fmt.Errorf("unknown message type: %d", message.Type)
			}

			if err != nil {
				log.Printf("[ERROR] write message error, %v", err)
				return
			}
		}
	}
}

func (wsConn *WsConn) heartbeat() {
	wsConn.wg.Add(1)
	defer wsConn.calcel()
	defer wsConn.wg.Done()
	defer wsConn.ticker.Stop()

	for {
		select {
		case <-wsConn.ctx.Done():
			log.Printf("[DEBUG] heartbeat receive close signal from %s", wsConn.conn.RemoteAddr().String())
			return
		case <-wsConn.ticker.C:
			if wsConn.timeout() {
				log.Printf("[DEBUG] heartbeat timeout, lastPongTime: %v, now: %v", wsConn.getLastPongTime(), time.Now())
				return
			}
			log.Printf("[DEBUG] send ping message to %s, lastPongTime: %v", wsConn.conn.RemoteAddr().String(), wsConn.getLastPongTime())
			pingMessage := &interfaces.WsConnMessage{
				Type: interfaces.PingMessage,
				Data: nil,
			}
			wsConn.WriteChan <- pingMessage
		}
	}
}

func (wsConn *WsConn) timeout() bool {
	wsConn.mu.RLock()
	defer wsConn.mu.RUnlock()

	return time.Since(wsConn.lastPongTime) > wsConn.heartBeatTimeout
}

func (wsConn *WsConn) getLastPongTime() time.Time {
	wsConn.mu.RLock()
	defer wsConn.mu.RUnlock()

	return wsConn.lastPongTime
}

func (wsConn *WsConn) SetPongHandler() {
	wsConn.conn.SetPongHandler(func(appData string) error {
		wsConn.mu.Lock()
		defer wsConn.mu.Unlock()

		wsConn.lastPongTime = time.Now()

		return nil
	})
}

func (wsConn *WsConn) SetCloseHandler() {
	wsConn.conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("[DEBUG] receive close message, code: %d, text: %s", code, text)

		wsConn.WriteChan <- &interfaces.WsConnMessage{
			Type: interfaces.CloseMessage,
			Data: nil,
		}

		time.Sleep(time.Second * 1)
		// 必须返回错误，否则ReadMessage会阻塞
		return errClosedNormally
	})
}

func (wsConn *WsConn) SendMessage(message []byte) {
	wsConn.WriteChan <- &interfaces.WsConnMessage{
		Type: interfaces.TextMessage,
		Data: message,
	}
}

var errClosedNormally = errors.New("WebSocket closed normally")

package logics

import (
	"MessagePushService/interfaces"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WsConn struct {
	manager       interfaces.ILogicsWsConnManager
	logicsMessage interfaces.ILogicsMessage
	conn          *websocket.Conn
	UserInfo      *interfaces.UserInfo

	writeTimeout      time.Duration // time allowed to write a message to the peer
	readTimeout       time.Duration // time allowed to read the next pong message from the peer
	heartBeatInterval time.Duration // send pings to peer with this period. Must be less than pongWait

	bufferChan chan []byte // 发送消息缓冲区
	isAlive    bool        // 连接是否存活
	mu         sync.RWMutex
	closeOnce  sync.Once
	wg         sync.WaitGroup // 等待所有goroutine完成, 避免泄露

	ctx    context.Context
	cancel context.CancelFunc
}

func NewWsConn(manager interfaces.ILogicsWsConnManager, conn *websocket.Conn, userInfo *interfaces.UserInfo, logicsMessage interfaces.ILogicsMessage) interfaces.ILogicsWsConn {
	ctx, cancel := context.WithCancel(context.Background())
	wsConn := &WsConn{
		manager:       manager,
		logicsMessage: logicsMessage,
		conn:          conn,
		UserInfo:      userInfo,

		writeTimeout:      time.Second * 10,
		readTimeout:       time.Second * 6,
		heartBeatInterval: (time.Second * 6 * 9) / 10,

		bufferChan: make(chan []byte, 1000),
		isAlive:    true,

		ctx:    ctx,
		cancel: cancel,
	}

	wsConn.SetPongHandler()
	wsConn.SetCloseHandler()

	wsConn.wg.Add(2)
	go wsConn.readPump()
	go wsConn.writePump()

	return wsConn
}

func (wsConn *WsConn) SafeClose() {
	wsConn.closeOnce.Do(func() {
		wsConn.mu.Lock()
		wsConn.isAlive = false
		wsConn.mu.Unlock()

		wsConn.cancel()
		close(wsConn.bufferChan)

		wsConn.manager.Remove(wsConn.UserInfo.ID)

		wsConn.wg.Wait()
		// 等待所有goroutine退出后，再关闭连接
		wsConn.conn.Close()
	})
}

func (wsConn *WsConn) readPump() {
	defer wsConn.SafeClose()
	defer wsConn.wg.Done()

	wsConn.conn.SetReadDeadline(time.Now().Add(wsConn.readTimeout))
	for {
		select {
		case <-wsConn.ctx.Done():
			return
		default:
		}
		messageType, message, err := wsConn.conn.ReadMessage()
		if err != nil {
			// 非正常关闭连接
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("[ERROR] read message error, %v", err)
			}
			// 正常关闭连接
			return
		}
		log.Println("receive message from user: ", wsConn.UserInfo.ID, string(message))
		if messageType != websocket.TextMessage {
			log.Printf("[ERROR] receive unexpected message type, %d, %s", messageType, string(message))
			return
		}
		var i map[string]interface{}
		err = json.Unmarshal(message, &i)
		if err != nil {
			log.Printf("[ERROR] unmarshal message error, %v", err)
			return
		}
		err = wsConn.handleMessage(i)
		if err != nil {
			log.Printf("[ERROR] handle message error, %v", err)
			return
		}
	}
}

func (wsConn *WsConn) writePump() {
	defer wsConn.SafeClose()
	defer wsConn.wg.Done()

	ticker := time.NewTicker(wsConn.heartBeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wsConn.conn.SetWriteDeadline(time.Now().Add(wsConn.writeTimeout))
			err := wsConn.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Printf("[ERROR] write ping message error, %v", err)
				return
			}
		case <-wsConn.ctx.Done():
			err := wsConn.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server close"))
			if err != nil {
				log.Printf("[ERROR] write close message error, %v", err)
				return
			}
			return
		case data, ok := <-wsConn.bufferChan:
			if !ok {
				return
			}
			err := wsConn.conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Printf("[ERROR] write message error, %v", err)
				return
			}
		}
	}
}

func (wsConn *WsConn) handleMessage(msg map[string]interface{}) (err error) {
	id, ok := msg["id"].(string)
	if !ok {
		log.Printf("[ERROR] id is not a string")
		return
	}
	typ, ok := msg["type"].(float64)
	if !ok {
		log.Printf("[ERROR] type is not a float64")
		return
	}
	timestamp, ok := msg["timestamp"].(float64)
	if !ok {
		log.Printf("[ERROR] timestamp is not a float64")
		return
	}
	switch interfaces.MessageType(typ) {
	case interfaces.MessageTypeACK:
	case interfaces.MessageTypeChatRoom:
		body, ok := msg["body"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("body is not a map[string]interface{}")
		}
		from, ok := body["from"].(string)
		if !ok {
			return fmt.Errorf("from is not a string")
		}
		to, ok := body["to"].(string)
		if !ok {
			return fmt.Errorf("to is not a string")
		}

		bodyStr, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body error, %v", err)
		}

		err = wsConn.logicsMessage.Add(wsConn.ctx, interfaces.MessageTypeChatRoom, []string{from, to}, id, string(bodyStr), int64(timestamp))
		if err != nil {
			return fmt.Errorf("add message error, %v", err)
		}
		messagePushInstance.NotifyByNewMessage(id)
	default:
		return fmt.Errorf("receive unexpected message type, %v, %v", typ, msg)
	}

	return nil
}

func (wsConn *WsConn) SetPongHandler() {
	wsConn.conn.SetPongHandler(func(appData string) error {
		wsConn.conn.SetReadDeadline(time.Now().Add(wsConn.readTimeout))
		return nil
	})
}

func (wsConn *WsConn) SetCloseHandler() {
	wsConn.conn.SetCloseHandler(func(code int, text string) error {
		return &websocket.CloseError{Code: code, Text: text}
	})
}

func (wsConn *WsConn) Send(ctx context.Context, data []byte) {
	select {
	case <-wsConn.ctx.Done():
		return
	case <-ctx.Done():
		return
	default:
	}

	wsConn.mu.RLock()
	defer wsConn.mu.RUnlock()

	// 当 wsConn.isAlive 为 false 时，bufferChan 可能已经被关闭。
	if !wsConn.isAlive {
		return
	}

	wsConn.bufferChan <- data
}

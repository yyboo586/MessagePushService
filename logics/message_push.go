package logics

import (
	"MessagePushService/interfaces"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

var (
	messagePushOnce     sync.Once
	messagePushInstance *messagePush
)

type messagePush struct {
	wsConnManager interfaces.ILogicsWsConnManager
	logicsMessage interfaces.ILogicsMessage

	ctx context.Context

	newMessageSignal chan string
	userLoginSignal  chan string
}

func NewMessagePush(wsConnManager interfaces.ILogicsWsConnManager, logicsMessage interfaces.ILogicsMessage) interfaces.ILogicsMessagePush {
	messagePushOnce.Do(func() {
		messagePushInstance = &messagePush{
			wsConnManager:    wsConnManager,
			logicsMessage:    logicsMessage,
			ctx:              context.Background(),
			newMessageSignal: make(chan string, 1000),
			userLoginSignal:  make(chan string, 10),
		}

		go messagePushInstance.newMessageWorker()
		go messagePushInstance.userLoginWorker()
	})

	return messagePushInstance
}

func (messagePush *messagePush) NotifyByNewMessage(messageID string) {
	messagePush.newMessageSignal <- messageID
}

func (messagePush *messagePush) NotifyByUserLogin(userID string) {
	messagePush.userLoginSignal <- userID
}

func (messagePush *messagePush) newMessageWorker() {
	for {
		messageID := <-messagePush.newMessageSignal
		message, userIDs, err := messagePush.logicsMessage.GetByID(messagePush.ctx, messageID)
		if err != nil {
			log.Printf("[ERROR] get message error: %v", err)
			continue
		}

		if message == nil {
			continue
		}

		err = messagePush.pushMessageToUsers(messagePush.ctx, message, userIDs)
		if err != nil {
			log.Printf("[ERROR] push message to users error: %v", err)
			continue
		}
	}
}

// 这里的逻辑有问题: 可能会阻塞在第一个登录的用户那里
func (messagePush *messagePush) userLoginWorker() {
	for {
		select {
		case <-messagePush.ctx.Done():
			log.Printf("[DEBUG] userLoginWorker receive close signal")
			return
		case userID := <-messagePush.userLoginSignal:
			for {
				messages, err := messagePush.logicsMessage.GetByUserID(messagePush.ctx, userID)
				if err != nil {
					log.Printf("[ERROR] get pending message by user id error: %v", err)
					break
				}
				if len(messages) == 0 {
					break
				}

				err = messagePush.pushMessagesToUser(messagePush.ctx, messages, userID)
				if err != nil {
					log.Printf("[ERROR] push messages to user error: %v", err)
					break
				}
			}
		}
	}
}

func (messagePush *messagePush) pushMessageToUsers(ctx context.Context, message *interfaces.LogicsMessage, userIDs []string) (err error) {
	for _, userID := range userIDs {
		wsConn := messagePush.wsConnManager.Get(ctx, userID)
		if wsConn == nil {
			log.Printf("[WARN] 用户未上线, ws conn is nil, userID: %s", userID)
			continue
		}

		jsonData, err := json.Marshal(message)
		if err != nil {
			log.Printf("[ERROR] marshal message error: %v", err)
			continue
		}
		wsConn.Send(ctx, jsonData)
		err = messagePush.logicsMessage.UpdateStatus(ctx, userID, message.ID, interfaces.MessagePushStatusSuccess)
		if err != nil {
			log.Printf("[ERROR] update message status error: %v", err)
			return err
		}
	}
	return nil
}

func (messagePush *messagePush) pushMessagesToUser(ctx context.Context, messages []*interfaces.LogicsMessage, userID string) (err error) {
	wsConn := messagePush.wsConnManager.Get(ctx, userID)
	if wsConn == nil {
		err = fmt.Errorf("用户未上线, ws conn is nil, userID: %s", userID)
		return err
	}

	for _, message := range messages {
		jsonData, err := json.Marshal(message)
		if err != nil {
			log.Printf("[ERROR] marshal message error: %v", err)
			continue
		}
		wsConn.Send(ctx, jsonData)
		err = messagePush.logicsMessage.UpdateStatus(ctx, userID, message.ID, interfaces.MessagePushStatusSuccess)
		if err != nil {
			log.Printf("[ERROR] update message status error: %v", err)
			return err
		}
	}

	return nil
}

package logics

import (
	"MessagePushService/interfaces"
	"context"
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

	newMessageSignal chan struct{}
	userLoginSignal  chan string
}

func NewMessagePush(wsConnManager interfaces.ILogicsWsConnManager, logicsMessage interfaces.ILogicsMessage) interfaces.ILogicsMessagePush {
	messagePushOnce.Do(func() {
		messagePushInstance = &messagePush{
			wsConnManager:    wsConnManager,
			logicsMessage:    logicsMessage,
			ctx:              context.Background(),
			newMessageSignal: make(chan struct{}, 100),
			userLoginSignal:  make(chan string, 100),
		}

		go messagePushInstance.newMessageWorker()
		go messagePushInstance.userLoginWorker()
	})

	return messagePushInstance
}

func (messagePush *messagePush) NotifyByNewMessage() {
	messagePush.newMessageSignal <- struct{}{}
}

func (messagePush *messagePush) NotifyByUserLogin(userID string) {
	messagePush.userLoginSignal <- userID
}

func (messagePush *messagePush) userLoginWorker() {
	for {
		select {
		case <-messagePush.ctx.Done():
			log.Printf("[DEBUG] userLoginWorker receive close signal")
			return
		case userID := <-messagePush.userLoginSignal:
			for {
				messages, err := messagePush.logicsMessage.GetPendingByUserID(messagePush.ctx, userID)
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

func (messagePush *messagePush) newMessageWorker() {
	for {
		<-messagePush.newMessageSignal
		message, userIDs, err := messagePush.logicsMessage.GetPendingMessage(messagePush.ctx)
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

func (messagePush *messagePush) pushMessageToUsers(ctx context.Context, message *interfaces.LogicsMessage, userIDs []string) (err error) {
	for _, userID := range userIDs {
		wsConn := messagePush.wsConnManager.Get(ctx, userID)
		if wsConn == nil {
			log.Printf("[ERROR] 用户未上线, ws conn is nil, userID: %s", userID)
			continue
		}

		wsConn.Send(ctx, []byte(message.Content))
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
		log.Printf("[ERROR] 用户未上线, ws conn is nil, userID: %s", userID)
		return nil
	}

	for _, message := range messages {
		wsConn.Send(ctx, []byte(message.Content))
		err = messagePush.logicsMessage.UpdateStatus(ctx, userID, message.ID, interfaces.MessagePushStatusSuccess)
		if err != nil {
			log.Printf("[ERROR] update message status error: %v", err)
			return err
		}
	}

	return nil
}

package logics

import (
	"MessagePushService/interfaces"
	"context"
	"log"
	"sync"
	"time"
)

var (
	messagePushOnce     sync.Once
	messagePushInstance *messagePush
)

type messagePush struct {
	wsConnManager      interfaces.WsConnManager
	messagePersistence interfaces.MessagePersistence

	ctx context.Context

	sigChan       chan struct{}
	userLoginChan chan string
}

func NewMessagePush(wsConnManager interfaces.WsConnManager, messagePersistence interfaces.MessagePersistence) interfaces.MessagePush {
	messagePushOnce.Do(func() {
		messagePushInstance = &messagePush{
			wsConnManager:      wsConnManager,
			messagePersistence: messagePersistence,
			ctx:                context.Background(),
			sigChan:            make(chan struct{}, 100),
			userLoginChan:      make(chan string, 100),
		}

		// go messagePushInstance.run()
		go messagePushInstance.userLoginWorker()
	})

	return messagePushInstance
}

func (messagePush *messagePush) Notify() {
	messagePush.sigChan <- struct{}{}
}

func (messagePush *messagePush) NotifyByUserLogin(userID string) {
	messagePush.userLoginChan <- userID
}

func (messagePush *messagePush) userLoginWorker() {
	for {
		select {
		case <-messagePush.ctx.Done():
			log.Printf("[DEBUG] userLoginWorker receive close signal")
			return
		case userID := <-messagePush.userLoginChan:
			for {
				messages, err := messagePush.messagePersistence.GetPendingMessageByUserID(messagePush.ctx, userID)
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

func (messagePush *messagePush) NotifyByNewMessage(messageID string) {
	messagePush.sigChan <- struct{}{}
}

func (messagePush *messagePush) run() {
	for {
		select {
		case <-time.After(time.Second * 30):
		case <-messagePush.sigChan:
		}

		message, userIDs, err := messagePush.messagePersistence.GetPendingMessage(messagePush.ctx)
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

func (messagePush *messagePush) pushMessageToUsers(ctx context.Context, message *interfaces.LogicsMessage, userIDs []string) error {
	for _, userID := range userIDs {
		wsConn, err := messagePush.wsConnManager.Get(ctx, userID)
		if err != nil {
			log.Printf("[ERROR] get ws conn error: %v", err)
			return err
		}
		if wsConn == nil {
			log.Printf("[ERROR] 用户未上线, ws conn is nil, userID: %s", userID)
			continue
		}

		wsConn.SendMessage([]byte(message.Content))
		err = messagePush.messagePersistence.UpdateMessageStatus(ctx, userID, message.MessageID, interfaces.MessageStatusSentSuccess)
		if err != nil {
			log.Printf("[ERROR] update message status error: %v", err)
			return err
		}
	}
	return nil
}

func (messagePush *messagePush) pushMessagesToUser(ctx context.Context, messages []*interfaces.LogicsMessage, userID string) error {
	wsConn, err := messagePush.wsConnManager.Get(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] get ws conn error: %v", err)
		return err
	}
	if wsConn == nil {
		log.Printf("[ERROR] 用户未上线, ws conn is nil, userID: %s", userID)
		return nil
	}

	for _, message := range messages {
		wsConn.SendMessage([]byte(message.Content))
		err = messagePush.messagePersistence.UpdateMessageStatus(ctx, userID, message.MessageID, interfaces.MessageStatusSentSuccess)
		if err != nil {
			log.Printf("[ERROR] update message status error: %v", err)
			return err
		}
	}

	return nil
}

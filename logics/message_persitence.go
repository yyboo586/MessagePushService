package logics

import (
	"MessagePushService/interfaces"
	"context"
	"log"
	"sync"
)

var (
	messagePersistenceOnce     sync.Once
	messagePersistenceInstance *messagePersistence
)

type messagePersistence struct {
	userloginBatchLimit  int
	dbMessagePersistence interfaces.DBMessagePersistence
}

func NewMessagePersistence(dbMessagePersistence interfaces.DBMessagePersistence) interfaces.MessagePersistence {
	messagePersistenceOnce.Do(func() {
		messagePersistenceInstance = &messagePersistence{
			userloginBatchLimit:  10,
			dbMessagePersistence: dbMessagePersistence,
		}
	})

	return messagePersistenceInstance
}

func (messagePersistence *messagePersistence) SaveMessage(ctx context.Context, messageType interfaces.MessageType, userIDs []string, messageID string, content string) error {
	exists, err := messagePersistence.dbMessagePersistence.CheckExists(ctx, messageID)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("[DEBUG]message %s already exists", messageID)
		return nil
	}

	message := &interfaces.DBMessage{
		MessageID:   messageID,
		MessageType: int(messageType),
		Content:     content,
	}
	return messagePersistence.dbMessagePersistence.Add(ctx, userIDs, message)
}

func (messagePersistence *messagePersistence) GetPendingMessage(ctx context.Context) (out *interfaces.LogicsMessage, userIDs []string, err error) {
	message, userIDs, err := messagePersistence.dbMessagePersistence.GetPendingMessage(ctx)
	if err != nil {
		return
	}
	if message == nil {
		return
	}

	out = interfaces.ConvertDBMessageToModel(message)
	return
}

func (messagePersistence *messagePersistence) GetPendingMessageByUserID(ctx context.Context, userID string) (outs []*interfaces.LogicsMessage, err error) {
	messages, err := messagePersistence.dbMessagePersistence.GetPendingMessagesByUserID(ctx, userID, messagePersistence.userloginBatchLimit)
	if err != nil {
		return
	}

	for _, v := range messages {
		outs = append(outs, interfaces.ConvertDBMessageToModel(v))
	}
	return
}

func (messagePersistence *messagePersistence) UpdateMessageStatus(ctx context.Context, userID, msgID string, status interfaces.MessageStatus) error {
	return messagePersistence.dbMessagePersistence.UpdateStatus(ctx, userID, msgID, int(status))
}

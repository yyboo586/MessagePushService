package logics

import (
	"MessagePushService/interfaces"
	"context"
	"log"
	"strings"
	"sync"
)

var (
	logicsMessageOnce     sync.Once
	logicsMessageInstance *logicsMessage
)

type logicsMessage struct {
	userloginBatchLimit int
	dbMessage           interfaces.IDBMessage
}

func NewMessage(dbMessage interfaces.IDBMessage) interfaces.ILogicsMessage {
	logicsMessageOnce.Do(func() {
		logicsMessageInstance = &logicsMessage{
			userloginBatchLimit: 5,
			dbMessage:           dbMessage,
		}
	})

	return logicsMessageInstance
}

func (l *logicsMessage) Add(ctx context.Context, messageType interfaces.MessageType, userIDs []string, messageID string, content string, timestamp int64) (err error) {
	message := &interfaces.DBMessage{
		ID:        messageID,
		Type:      int(messageType),
		Content:   content,
		Timestamp: timestamp,
	}
	err = l.dbMessage.Add(ctx, userIDs, message)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			log.Printf("[DEBUG] message %s already exists", messageID)
			return nil
		}
		log.Println(err)
		return err
	}
	return
}

func (l *logicsMessage) GetByID(ctx context.Context, messageID string) (out *interfaces.LogicsMessage, userIDs []string, err error) {
	message, userIDs, err := l.dbMessage.GetByID(ctx, messageID)
	if err != nil {
		log.Println(err)
		return
	}
	return interfaces.ConvertDBMessageToModel(message), userIDs, nil
}

func (l *logicsMessage) GetByUserID(ctx context.Context, userID string) (outs []*interfaces.LogicsMessage, err error) {
	messages, err := l.dbMessage.GetByUserID(ctx, userID, interfaces.MessagePushStatusUnhandled, l.userloginBatchLimit)
	if err != nil {
		log.Println(err)
		return
	}

	for _, v := range messages {
		outs = append(outs, interfaces.ConvertDBMessageToModel(v))
	}
	return
}

func (logicsMessage *logicsMessage) UpdateStatus(ctx context.Context, userID, msgID string, status interfaces.MessagePushStatus) error {
	return logicsMessage.dbMessage.UpdateStatus(ctx, userID, msgID, status)
}

package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

type IDBMessage interface {
	// 添加消息
	Add(ctx context.Context, userIDs []string, message *DBMessage) error
	// 根据消息ID获取消息
	GetByID(ctx context.Context, messageID string) (out *DBMessage, userIDs []string, err error)
	// 根据消息状态获取消息（仅限一条，因为不同的消息推送的人群可能不一样）
	GetByPushStatus(ctx context.Context, status MessagePushStatus) (out *DBMessage, userIDs []string, err error)
	// 批量获取特定用户指定状态的消息
	GetByUserID(ctx context.Context, userID string, status MessagePushStatus, limit int) (out []*DBMessage, err error)
	// 更新消息状态
	UpdateStatus(ctx context.Context, userID, msgID string, status MessagePushStatus) error
}

type DBMessage struct {
	ID        string
	Type      int
	Content   string
	Timestamp int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func ConvertDBMessageToModel(message *DBMessage) *LogicsMessage {
	var i interface{}
	err := json.Unmarshal([]byte(message.Content), &i)
	if err != nil {
		log.Printf("[ERROR] unmarshal message content error: %v", err)
		return nil
	}
	return &LogicsMessage{
		ID:        message.ID,
		Type:      MessageType(message.Type),
		Content:   i,
		Timestamp: message.Timestamp,
		CreatedAt: message.CreatedAt,
		UpdatedAt: message.UpdatedAt,
	}
}

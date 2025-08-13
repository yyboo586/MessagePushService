package interfaces

import (
	"context"
	"time"
)

type DBMessagePersistence interface {
	Add(ctx context.Context, userIDs []string, message *DBMessage) error
	// 检查消息是否存在
	CheckExists(ctx context.Context, messageID string) (exists bool, err error)
	// 获取一条待推送的消息
	GetPendingMessage(ctx context.Context) (out *DBMessage, userIDs []string, err error)
	// 批量获取特定用户待推送的消息
	GetPendingMessagesByUserID(ctx context.Context, userID string, limit int) (out []*DBMessage, err error)
	// 更新消息状态
	UpdateStatus(ctx context.Context, userID, msgID string, status int) error
}

type DBMessage struct {
	MessageID   string
	MessageType int
	Content     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func ConvertDBMessageToModel(message *DBMessage) *LogicsMessage {
	return &LogicsMessage{
		MessageID:   message.MessageID,
		MessageType: MessageType(message.MessageType),
		Content:     message.Content,
		CreatedAt:   message.CreatedAt,
		UpdatedAt:   message.UpdatedAt,
	}
}

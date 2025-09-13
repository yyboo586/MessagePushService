package interfaces

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

//go:generate mockgen -source=./logics.go -destination=mock/logics_mock.go -package=mock

// MQMessageType 消息类型, 和 topic 对应，方便后续解析消息
type MessageType int

const (
	_                  MessageType = iota
	MessageTypeACK                 // 消息确认
	MessageTypeToUsers             // 推送给指定用户集合
)

const (
	MessageTypeToUsersTopic = "core.push.users"
)

type MessagePushStatus int

const (
	MessagePushStatusUnhandled MessagePushStatus = iota // 待处理
	MessagePushStatusSending                            // 推送中
	MessagePushStatusSuccess                            // 推送成功
	MessagePushStatusFailed                             // 推送失败
)

type UserInfo struct {
	ID    string // 用户ID
	OrgID string // 组织ID
	Name  string // 用户名
}

type ILogicsWsConn interface {
	SafeClose()
	Send(ctx context.Context, data []byte)
}

type ILogicsWsConnManager interface {
	Add(conn *websocket.Conn, userInfo *UserInfo)
	Get(ctx context.Context, userID string) ILogicsWsConn
	Remove(userID string)
}

type LogicsMessage struct {
	ID        string
	Type      int
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ILogicsMessage interface {
	// 添加消息
	Add(ctx context.Context, messageType MessageType, userIDs []string, messageID string, content string) error
	// 根据消息ID获取消息
	GetByID(ctx context.Context, messageID string) (out *LogicsMessage, userIDs []string, err error)
	// 根据用户ID获取待推送的消息
	GetByUserID(ctx context.Context, userID string) (outs []*LogicsMessage, err error)
	// 更新消息状态
	UpdateStatus(ctx context.Context, userID, msgID string, status MessagePushStatus) error
}

type ILogicsMessagePush interface {
	NotifyByNewMessage(messageID string)
	NotifyByUserLogin(userID string)
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, messageType MessageType, userIDs []string, content string) error
}

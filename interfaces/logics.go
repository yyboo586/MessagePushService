package interfaces

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

//go:generate mockgen -source=./logics.go -destination=mock/logics_mock.go -package=mock

const (
	// TextMessage denotes a text data message. The text message payload is
	// interpreted as UTF-8 encoded text data.
	TextMessage = 1

	// BinaryMessage denotes a binary data message.
	BinaryMessage = 2

	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text. Use the FormatCloseMessage
	// function to format a close message payload.
	CloseMessage = 8

	// PingMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PingMessage = 9

	// PongMessage denotes a pong control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage = 10
)

type LogicsWsConn interface {
	IsAlive() bool
	Close()
	SendMessage(message []byte)
}

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

type MessageStatus int

const (
	_ MessageStatus = iota
	MessageStatusUnsent
	MessageStatusSentSuccess
	MessageStatusSentFailed
)

type UserInfo struct {
	ID    string // 用户ID
	OrgID string // 组织ID
	Name  string // 用户名
}

type WsConnManager interface {
	Add(conn *websocket.Conn, userInfo *UserInfo)
	Get(ctx context.Context, userID string) (LogicsWsConn, error)
	Remove(userID string)
}

type LogicsMessage struct {
	MessageID   string
	MessageType MessageType
	Content     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WsConnMessage 推送给ws连接的消息
type WsConnMessage struct {
	Type int
	Data []byte
}

type MessagePush interface {
	Notify()
	NotifyByUserLogin(userID string)
}

type MessagePersistence interface {
	SaveMessage(ctx context.Context, messageType MessageType, userIDs []string, messageID string, content string) error
	GetPendingMessage(ctx context.Context) (out *LogicsMessage, userIDs []string, err error)
	GetPendingMessageByUserID(ctx context.Context, userID string) (outs []*LogicsMessage, err error)
	UpdateMessageStatus(ctx context.Context, userID, msgID string, status MessageStatus) error
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, messageType MessageType, userIDs []string, content string) error
}

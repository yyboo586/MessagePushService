package driveradapters

import (
	"MessagePushService/common"
	"MessagePushService/interfaces"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	mqsdk "github.com/yyboo586/MQSDK"
)

var (
	channel = "message_push_service"
)

var (
	mqHandlerOnce sync.Once
	mqHandler     *MQHandler
)

type MQHandler struct {
	consumer      mqsdk.Consumer
	logicsMessage interfaces.ILogicsMessage
	messagePush   interfaces.ILogicsMessagePush
}

func NewMQHandler(config *common.Config, logicsMessage interfaces.ILogicsMessage, messagePush interfaces.ILogicsMessagePush) *MQHandler {
	mqConfig := &mqsdk.NSQConfig{
		Type:      config.MQ.Type,
		NSQDAddr:  config.MQ.NSQDAddr,
		NSQLookup: []string{},
	}
	mqHandlerOnce.Do(func() {
		consumer, err := mqsdk.NewFactory().NewConsumer(mqConfig)
		if err != nil {
			log.Fatalf("Failed to create consumer: %v", err)
		}
		mqHandler = &MQHandler{
			consumer:      consumer,
			logicsMessage: logicsMessage,
			messagePush:   messagePush,
		}
	})

	mqHandler.Start(config)

	return mqHandler
}

func (mqHandler *MQHandler) Start(config *common.Config) {
	for _, topic := range config.Event.Topics {
		mqHandler.Register(topic, mqHandler.handleToUsers)
	}
}

func (mqHandler *MQHandler) Register(topic string, handler func(msg *mqsdk.Message) (err error)) {
	err := mqHandler.consumer.Subscribe(context.Background(), topic, channel, handler)
	if err != nil {
		log.Printf("Failed to subscribe to topic %s: %v", topic, err)
	}
}

func (mqHandler *MQHandler) handleToUsers(msg *mqsdk.Message) (err error) {
	userIDsInterface, ok := msg.Body.(map[string]interface{})["user_ids"]
	if !ok {
		return fmt.Errorf("body.user_ids is required")
	}

	userIDs, ok := userIDsInterface.([]interface{})
	if !ok {
		return fmt.Errorf("body.user_ids is not a []interface{}")
	}

	userIDsStr := make([]string, len(userIDs))
	for i, userID := range userIDs {
		userIDsStr[i], ok = userID.(string)
		if !ok {
			return fmt.Errorf("user_ids is not a []string")
		}
	}

	contentInterface, ok := msg.Body.(map[string]interface{})["content"]
	if !ok {
		return fmt.Errorf("body.content is required")
	}

	content, ok := contentInterface.(map[string]interface{})
	if !ok {
		return fmt.Errorf("body.content is not a map[string]interface{}")
	}

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %v", err)
	}

	err = mqHandler.logicsMessage.Add(context.Background(), interfaces.MessageTypeToUsers, userIDsStr, msg.ID, string(contentBytes))
	if err != nil {
		log.Printf("[ERROR] save message error: %v", err)
		return
	}

	mqHandler.messagePush.NotifyByNewMessage(msg.ID)
	return
}

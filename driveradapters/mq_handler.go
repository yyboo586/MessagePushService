package driveradapters

import (
	"MessagePushService/common"
	"MessagePushService/interfaces"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	mq "github.com/yyboo586/MQSDK"
)

var (
	channel = "message_push_service"
)

var (
	mqHandlerOnce sync.Once
	mqHandler     *MQHandler
)

type MQHandler struct {
	consumer           mq.Consumer
	messagePersistence interfaces.MessagePersistence
	messagePush        interfaces.MessagePush
}

func NewMQHandler(config *common.Config, messagePersistence interfaces.MessagePersistence, messagePush interfaces.MessagePush) *MQHandler {
	mqConfig := &mq.NSQConfig{
		Type:      config.MQ.Type,
		NSQDAddr:  config.MQ.NSQDAddr,
		NSQLookup: []string{},
	}
	mqHandlerOnce.Do(func() {
		consumer, err := mq.NewFactory().NewConsumer(mqConfig)
		if err != nil {
			log.Fatalf("Failed to create consumer: %v", err)
		}
		mqHandler = &MQHandler{
			consumer:           consumer,
			messagePersistence: messagePersistence,
			messagePush:        messagePush,
		}
	})

	mqHandler.start()

	return mqHandler
}

func (mqHandler *MQHandler) start() {
	mqHandler.Register(interfaces.MessageTypeToUsersTopic, mqHandler.handleToUsers)
}

func (mqHandler *MQHandler) Register(topic string, handler func(msg *mq.Message) (err error)) {
	err := mqHandler.consumer.Subscribe(context.Background(), topic, channel, handler)
	if err != nil {
		log.Printf("Failed to subscribe to topic %s: %v", topic, err)
	}
}

func (mqHandler *MQHandler) handleToUsers(msg *mq.Message) (err error) {
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

	err = mqHandler.messagePersistence.SaveMessage(context.Background(), interfaces.MessageTypeToUsers, userIDsStr, msg.ID, string(contentBytes))
	if err != nil {
		log.Printf("[ERROR] save message error: %v", err)
		return
	}

	mqHandler.messagePush.Notify()
	return
}

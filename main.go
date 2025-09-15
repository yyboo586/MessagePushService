package main

import (
	"MessagePushService/common"
	"MessagePushService/dbaccess"
	"MessagePushService/drivenadapters"
	"MessagePushService/driveradapters"
	"MessagePushService/interfaces"
	"MessagePushService/logics"
	"log"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config        *common.Config
	mqHandler     *driveradapters.MQHandler
	wsConnHandler interfaces.RESTHandler
}

func (s *Server) Start() {
	gin.SetMode(gin.DebugMode)

	go func() {
		server := gin.New()
		server.Use(gin.Recovery())
		server.Use(gin.Logger())

		s.wsConnHandler.RegisterPublic(server)

		if err := server.Run(s.config.Server.PublicAddr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	go func() {
		server := gin.New()
		server.Use(gin.Recovery())
		server.Use(gin.Logger())

		s.wsConnHandler.RegisterPrivate(server)

		if err := server.Run(s.config.Server.PrivateAddr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	go func() {
		for {
			<-time.After(time.Second * 30)
			nums := runtime.NumGoroutine()
			log.Println("Current NumGoroutine:", nums)
		}
	}()

}

func main() {
	config := common.NewConfig()
	dbPool, err := common.NewDB(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	httpClient := common.NewHTTPClient()

	drivenIdentifyService := drivenadapters.NewIdentifyService(config, httpClient)

	dbMessage := dbaccess.NewDBMessage(dbPool)

	logicsMessage := logics.NewMessage(dbMessage)
	logicsWsConnManager := logics.NewWsConnManager(logicsMessage)
	logicsMessagePush := logics.NewMessagePush(logicsWsConnManager, logicsMessage)

	server := &Server{
		config:        config,
		mqHandler:     driveradapters.NewMQHandler(config, logicsMessage, logicsMessagePush),
		wsConnHandler: driveradapters.NewWebsocketHandler(logicsWsConnManager, logicsMessagePush, drivenIdentifyService),
	}
	server.Start()

	select {}
}

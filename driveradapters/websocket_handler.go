package driveradapters

import (
	"MessagePushService/common"
	"MessagePushService/interfaces"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	websocketHandlerOnce     sync.Once
	websocketHandlerInstance *websocketHandler
)

type websocketHandler struct {
	upgrader        *websocket.Upgrader
	wsConnManager   interfaces.ILogicsWsConnManager
	messagePush     interfaces.ILogicsMessagePush
	identifyService interfaces.IDrivenIdentifyService
}

func NewWebsocketHandler(wsConnManager interfaces.ILogicsWsConnManager, messagePush interfaces.ILogicsMessagePush, identifyService interfaces.IDrivenIdentifyService) interfaces.RESTHandler {
	websocketHandlerOnce.Do(func() {
		websocketHandlerInstance = &websocketHandler{
			upgrader: &websocket.Upgrader{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				CheckOrigin: func(r *http.Request) bool {
					return true // 在生产环境中应该进行更严格的跨域检查
				},
			},
			wsConnManager:   wsConnManager,
			messagePush:     messagePush,
			identifyService: identifyService,
		}
	})

	return websocketHandlerInstance
}

func (handler *websocketHandler) RegisterPublic(engine *gin.Engine) {
	engine.GET("/ws/public", handler.upgradePublic)
}

func (handler *websocketHandler) RegisterPrivate(engine *gin.Engine) {
	engine.GET("/ws/private", handler.upgradePrivate)
}

func (handler *websocketHandler) upgradePrivate(ctx *gin.Context) {
	userID, ok := ctx.GetQuery("user_id")
	if !ok {
		common.ReplyError(ctx, common.NewHTTPError(http.StatusBadRequest, "user_id is required", nil))
		return
	}

	userInfo := &interfaces.UserInfo{
		ID:   userID,
		Name: fmt.Sprintf("private-%s", userID),
	}
	handler.upgrade(ctx, userInfo)
}

func (handler *websocketHandler) upgradePublic(c *gin.Context) {
	authorization := c.GetHeader("Authorization")
	if authorization == "" {
		common.ReplyError(c, common.NewHTTPError(http.StatusUnauthorized, "Authorization is required", nil))
		return
	}

	userInfo, err := handler.identifyService.Instrospect(c)
	if err != nil {
		common.ReplyError(c, err)
		return
	}
	if userInfo == nil {
		common.ReplyError(c, common.NewHTTPError(http.StatusUnauthorized, "用户未登录", nil))
		return
	}

	handler.upgrade(c, userInfo)
}

func (handler *websocketHandler) upgrade(c *gin.Context, userInfo *interfaces.UserInfo) {
	conn, err := handler.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		common.ReplyError(c, err)
		return
	}

	handler.wsConnManager.Add(conn, userInfo)
	handler.messagePush.NotifyByUserLogin(userInfo.ID)
}

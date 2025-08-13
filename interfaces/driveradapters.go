package interfaces

import "github.com/gin-gonic/gin"

type RESTHandler interface {
	// 注册公共路由
	RegisterPublic(engine *gin.Engine)
	// 注册私有路由
	RegisterPrivate(engine *gin.Engine)
}

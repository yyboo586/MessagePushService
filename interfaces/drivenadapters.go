package interfaces

import "github.com/gin-gonic/gin"

type IDrivenIdentifyService interface {
	// 令牌内省
	Instrospect(ctx *gin.Context) (*UserInfo, error)
}

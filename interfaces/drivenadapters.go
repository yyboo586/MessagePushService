package interfaces

import "github.com/gin-gonic/gin"

type IdentifyService interface {
	Instrospect(ctx *gin.Context) (*UserInfo, error)
}

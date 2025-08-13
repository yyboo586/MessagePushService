package drivenadapters

import (
	"MessagePushService/common"
	"MessagePushService/interfaces"
	"fmt"

	"github.com/gin-gonic/gin"
)

type identifyService struct {
	addr   string
	client common.HTTPClient
}

func NewIdentifyService(config *common.Config, client common.HTTPClient) interfaces.IDrivenIdentifyService {
	return &identifyService{
		addr:   config.ThirdService.IdentifyServiceAddr,
		client: client,
	}
}

func (s *identifyService) Instrospect(ctx *gin.Context) (userInfo *interfaces.UserInfo, err error) {
	url := fmt.Sprintf("%s/api/v1/identify-service/token/introspect", s.addr)

	_, resBody, err := s.client.POST(ctx, url, nil, map[string]interface{}{
		"Authorization": ctx.GetHeader("Authorization"),
	})
	if err != nil {
		return nil, err
	}

	userInfo = &interfaces.UserInfo{}
	userInfo.ID = resBody.(map[string]interface{})["user_id"].(string)
	userInfo.OrgID = resBody.(map[string]interface{})["org_id"].(string)
	userInfo.Name = resBody.(map[string]interface{})["user_name"].(string)

	return
}

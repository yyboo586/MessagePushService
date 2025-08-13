package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type HTTPClient interface {
	GET(ctx context.Context, url string, header map[string]interface{}) (status int, resBody interface{}, err error)
	POST(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, resBody interface{}, err error)
}

type httpClient struct {
	client *http.Client
}

var (
	cOnce sync.Once
	c     *httpClient
)

func NewHTTPClient() HTTPClient {
	cOnce.Do(func() {
		c = &httpClient{
			client: &http.Client{
				Timeout: time.Second * 10,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return nil
				},
			},
		}
	})
	return c
}

func (c *httpClient) GET(ctx context.Context, url string, header map[string]interface{}) (status int, resBody interface{}, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	c.setHeader(req, header)

	return c.do(req)
}

func (c *httpClient) POST(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, resBody interface{}, err error) {
	var dataByte []byte
	switch data := body.(type) {
	case []byte:
		dataByte = data
	case string:
		dataByte = []byte(data)
	default:
		dataByte, err = json.Marshal(data)
		if err != nil {
			return
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(dataByte))
	if err != nil {
		return
	}
	c.setHeader(req, header)

	return c.do(req)
}

func (c *httpClient) do(req *http.Request) (status int, resBody interface{}, err error) {
	response, err := c.client.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()

	status = response.StatusCode

	// 检查响应状态码
	if response.StatusCode != http.StatusOK {
		// 尝试读取错误响应体
		errorBody, _ := io.ReadAll(response.Body)
		if len(errorBody) > 0 {
			err = fmt.Errorf("http request failed, status code: %d, response: %s", response.StatusCode, string(errorBody))
		} else {
			err = fmt.Errorf("http request failed, status code: %d", response.StatusCode)
		}
		return
	}

	// 检查响应体是否为空
	if response.Body == nil {
		err = fmt.Errorf("response body is nil")
		return
	}

	// 读取响应体
	resBodyByte, err := io.ReadAll(response.Body)
	if err != nil {
		// 在读取响应体时遇到了意外的文件结束符
		// 1、响应体为空：服务器返回了空响应。
		// 2、连接被提前关闭：服务器在发送完响应头后，立即关闭了连接。
		// 3、响应体损坏：网络传输过程中数据丢失。
		// 4、Content-Length不匹配：实际响应体长度与Content-Length不一致。
		if err == io.EOF {
			// 如果是EOF，检查是否是因为响应体为空
			if len(resBodyByte) == 0 {
				// 响应体为空，这是正常情况
				return
			}
			err = fmt.Errorf("response body truncated, read %d bytes before EOF", len(resBodyByte))
		} else {
			err = fmt.Errorf("read response body failed, err: %w", err)
		}
		return
	}

	var i map[string]interface{}
	err = json.Unmarshal(resBodyByte, &i)
	if err != nil {
		return
	}

	code := int(i["code"].(float64))
	switch code {
	case 0:
		resBody = i["data"]
		return
	case http.StatusUnauthorized:
		err = NewHTTPError(code, i["message"].(string), nil)
		return
	default:
		err = NewHTTPError(http.StatusInternalServerError, i["message"].(string), nil)
		return
	}
}

func (c *httpClient) setHeader(req *http.Request, header map[string]interface{}) {
	for k, v := range header {
		req.Header.Set(k, v.(string))
	}
}

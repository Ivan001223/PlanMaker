package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 微信小程序 API 客户端
type Client struct {
	appID     string
	appSecret string
	client    *http.Client
	isDevMode bool
}

// NewClient 创建微信客户端
func NewClient(appID, appSecret string) *Client {
	// 检查是否配置了微信密钥
	isDevMode := appID == "" || appSecret == ""
	return &Client{
		appID:     appID,
		appSecret: appSecret,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		isDevMode: isDevMode,
	}
}

// IsDevMode 返回是否开发模式
func (c *Client) IsDevMode() bool {
	return c.isDevMode
}

// Code2SessionResponse code2session 响应
type Code2SessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// Code2Session 小程序登录凭证校验
func (c *Client) Code2Session(code string) (*Code2SessionResponse, error) {
	if c.isDevMode {
		// 开发模式：返回模拟数据
		return &Code2SessionResponse{
			OpenID:     "mock_openid_" + code,
			SessionKey: "mock_session_key",
		}, nil
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		c.appID, c.appSecret, code,
	)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求微信API失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result Code2SessionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("微信登录失败 (errcode=%d): %s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}

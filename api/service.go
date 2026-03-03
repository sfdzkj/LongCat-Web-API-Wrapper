package api

import (
	"bytes"
	"io"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/JessonChan/longcat-web-api/config"
)

type LongCatRequest struct {
	Content        string `json:"content"`
	ReasonEnabled  int    `json:"reasonEnabled"`
	SearchEnabled  int    `json:"searchEnabled"`
	Regenerate     int    `json:"regenerate"`
	ConversationId string `json:"conversationId,omitempty"`
	
	AgentId       string `json:"agentId,omitempty"`
	CreationParam any    `json:"creationParam,omitempty"`
	Files         []any  `json:"files"`
}

type LongCatClient struct {
	baseHeaders map[string]string
}

func NewLongCatClient() *LongCatClient {
	return &LongCatClient{baseHeaders: map[string]string{
		"accept":             "text/event-stream,application/json",
		"accept-language":    "en,zh-Hans-CN;q=0.9,zh-CN;q=0.8,zh;q=0.7,en-GB;q=0.6,en-US;q=0.5,zh-TW;q=0.4",
		"content-type":       "application/json",
		"m-appkey":           "fe_com.sankuai.friday.fe.longcat",
		"origin":             "https://longcat.chat",
		"sec-fetch-dest":     "empty",
		"sec-fetch-mode":     "cors",
		"sec-fetch-site":     "same-origin",
		"user-agent":         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		"x-client-language":  "en",
		"x-requested-with":   "XMLHttpRequest",
		"referer":            "https://longcat.chat/t",
		"referrer-policy":    "strict-origin-when-cross-origin",
		"connection":         "keep-alive",
		"cache-control":      "no-cache",
		"pragma":             "no-cache",
		"sec-ch-ua":          `"Not(A:Brand";v="99", "Chromium";v="133"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": `"macOS"`,
	}}
}

func (c *LongCatClient) httpClient(timeoutSeconds int) *http.Client {
	if timeoutSeconds <= 0 { timeoutSeconds = 30 }
	return &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
}

func (c *LongCatClient) CreateSession(ctx context.Context, lc config.LongCatConfig, cookies config.CookieConfig) (string, error) {
	sessionReq := struct {
		Model   string `json:"model"`
		AgentID string `json:"agentId"`
	}{Model: lc.Model, AgentID: lc.AgentID}

	sessionLC := lc
	if sessionLC.TimeoutSeconds < 60 {
		sessionLC.TimeoutSeconds = 60
	}

	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		resp, err = c.sendRequest(ctx, sessionLC, cookies, sessionLC.SessionURL, sessionReq)
		if err == nil {
			break
		}
		if i < 2 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	if err != nil { return "", fmt.Errorf("failed to create session: %w", err) }
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	var sessionResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct{ ConversationID string `json:"conversationId"` } `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &sessionResp); err != nil { 
		snippet := string(bodyBytes)
		if len(snippet) > 100 { snippet = snippet[:100] + "..." }
		return "", fmt.Errorf("Status %d: %s", resp.StatusCode, snippet) 
	}
	if sessionResp.Code != 0 { return "", fmt.Errorf("session creation failed: %s", sessionResp.Message) }
	return sessionResp.Data.ConversationID, nil
}

func (c *LongCatClient) SendRequest(ctx context.Context, lc config.LongCatConfig, cookies config.CookieConfig, longCatReq LongCatRequest) (*http.Response, error) {
	return c.sendRequest(ctx, lc, cookies, lc.APIURL, longCatReq)
}

func (c *LongCatClient) sendRequest(ctx context.Context, lc config.LongCatConfig, cookies config.CookieConfig, url string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil { return nil, fmt.Errorf("failed to marshal request: %w", err) }
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil { return nil, fmt.Errorf("failed to create request: %w", err) }
	for k, v := range c.baseHeaders { req.Header.Set(k, v) }
	req.Header.Set("m-traceid", fmt.Sprintf("%d", time.Now().UnixNano()))
	if cookies.RawString != "" {
		req.Header.Set("Cookie", cookies.RawString)
	} else {
		var cookieParts []string
		if cookies.LxsdkCuid != "" { cookieParts = append(cookieParts, "_lxsdk_cuid="+cookies.LxsdkCuid) }
		if cookies.PassportToken != "" { cookieParts = append(cookieParts, "passport_token_key="+cookies.PassportToken) }
		if cookies.LxsdkS != "" { cookieParts = append(cookieParts, "_lxsdk_s="+cookies.LxsdkS) }
		if len(cookieParts) > 0 {
			req.Header.Set("Cookie", strings.Join(cookieParts, "; "))
		}
	}
	resp, err := c.httpClient(lc.TimeoutSeconds).Do(req)
	if err != nil { return nil, fmt.Errorf("failed to make request: %w", err) }
	return resp, nil
}


func (c *LongCatClient) Ping(ctx context.Context, lc config.LongCatConfig, cookies config.CookieConfig) error {
	// Instead of creating a session, we can just make a GET request to the main page
	// or another non-destructive endpoint to check if cookies are still valid.
	// Since we don't know a specific endpoint, we can do a lightweight request to the main page.
	req, err := http.NewRequestWithContext(ctx, "GET", "https://longcat.chat/t", nil)
	if err != nil { return err }
	
	for k, v := range c.baseHeaders { req.Header.Set(k, v) }
	
	if cookies.RawString != "" {
		req.Header.Set("Cookie", cookies.RawString)
	} else {
		var cookieParts []string
		if cookies.LxsdkCuid != "" { cookieParts = append(cookieParts, "_lxsdk_cuid="+cookies.LxsdkCuid) }
		if cookies.PassportToken != "" { cookieParts = append(cookieParts, "passport_token_key="+cookies.PassportToken) }
		if cookies.LxsdkS != "" { cookieParts = append(cookieParts, "_lxsdk_s="+cookies.LxsdkS) }
		if len(cookieParts) > 0 {
			req.Header.Set("Cookie", strings.Join(cookieParts, "; "))
		}
	}
	
	resp, err := c.httpClient(lc.TimeoutSeconds).Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Status %d", resp.StatusCode)
	}
	
	return nil
}

type APIService interface {
	ConvertResponse(resp *http.Response, stream bool) (<-chan interface{}, <-chan error)
	GetResponseContentType(stream bool) string
	HandleNonStreamingResponse(w http.ResponseWriter, chunks <-chan interface{}, errs <-chan error) error
	HandleStreamingResponse(w http.ResponseWriter, flusher http.Flusher, chunks <-chan interface{}, errs <-chan error) error
}

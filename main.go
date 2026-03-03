package main

import (
	"bytes"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"github.com/google/uuid"
	"regexp"
	"strings"
	"time"

	"github.com/JessonChan/longcat-web-api/admin"
	"github.com/JessonChan/longcat-web-api/api"
	"github.com/JessonChan/longcat-web-api/config"
	conversation "github.com/JessonChan/longcat-web-api/conversation"
	"github.com/JessonChan/longcat-web-api/logging"
	"github.com/JessonChan/longcat-web-api/types"
)

var openClawMetaRegex = regexp.MustCompile(`(?s)(?:\[Pasted ~[^\]]+\]\s*)?\.?Conversation info \(untrusted metadata\):\s*` + "```" + `(?:json)?.*?` + "```" + `\s*`)

var errVideoGenerationDisabled = errors.New("video generation is disabled")

type UnifiedHandler struct {
	store              *config.Store
	longCatClient      *api.LongCatClient
	openAIService      api.APIService
	claudeService      api.APIService
	conversationManager *conversation.ConversationManager
}

func NewUnifiedHandler(store *config.Store) *UnifiedHandler {
	client := api.NewLongCatClient()
	return &UnifiedHandler{
		store:              store,
		longCatClient:      client,
		openAIService:      api.NewOpenAIService(client),
		claudeService:      api.NewClaudeService(client),
		conversationManager: conversation.NewConversationManager(),
	}
}

func (h *UnifiedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	var promptTokens, completionTokens, totalTokens int
	var logErr string
	var logStatus = "success"

	cfg := h.store.Get()

	if r.Method == http.MethodOptions {
		writeCORS(w, cfg)
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key, anthropic-version")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.URL.Path != "/v1/chat/completions" && r.URL.Path != "/v1/messages" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !checkUpstreamAuth(r, cfg.UpstreamAPIKey) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"unauthorized"}}`))
		return
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(bs))
	logging.LogDebug("Request Body: %s %s", string(bs), r.URL.Path)

	var service api.APIService
	if r.URL.Path == "/v1/chat/completions" {
		service = h.openAIService
	} else {
		service = h.claudeService
	}

	messages, err := extractMessagesFromRequest(bs, r.URL.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse messages: %v", err), http.StatusBadRequest)
		return
	}

	model, _ := extractModelFromRequest(bs, r.URL.Path)
	streaming := isStreamingRequest(bs, r.URL.Path)

	cfg = h.store.Get()
	entry, exists := h.conversationManager.FindConversation(messages)

	var convID string
	var accountIdx int = -1
	var account config.AccountConfig = config.AccountConfig{}

	if exists {
		convID = entry.ConversationID
		for i := range cfg.Accounts {
			if cfg.Accounts[i].ID == entry.AccountID {
				accountIdx = i
				account = cfg.Accounts[i]
				break
			}
		}
		if accountIdx == -1 {
			exists = false
		}
	}

	attempt := func(cid string, acc config.AccountConfig) (*http.Response, error) {
		req, err := createLongCatRequest(messages, cid, model)
		if err != nil {
			return nil, err
		}
		return h.longCatClient.SendRequest(r.Context(), cfg.LongCat, acc.Cookies, req)
	}

	if !exists {
		idx, acc, selErr := admin.SelectAccount(h.store, cfg, h.conversationManager)
		if selErr != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"no_accounts","type":"bad_response_status_code"}}`))
			return
		}
		accountIdx = idx
		account = acc

		var lastErr error
		for tries := 0; tries < len(cfg.Accounts); tries++ {
			newCID, err := h.longCatClient.CreateSession(r.Context(), cfg.LongCat, account.Cookies)
			if err == nil {
				convID = newCID
				h.conversationManager.SetConversation(convID, account.ID, messages)
				break
			}
			lastErr = err
			logging.LogDebug("Failed to create session for account %s: %v", account.ID, err)

			if strings.ToLower(strings.TrimSpace(cfg.Strategy.Type)) == "sequential" {
				// Disable failing account in sequential mode
				accID := account.ID
				_, _ = h.store.Update(func(c *config.Config) error {
					for i := range c.Accounts {
						if c.Accounts[i].ID == accID {
							// c.Accounts[i].Enabled = false // DO NOT AUTO-DISABLE
							c.Accounts[i].LastTest = &config.AccountTest{
								At:     time.Now(),
								OK:     false,
								Detail: fmt.Sprintf("Auto-disabled due to failure: %v", err),
							}
							break
						}
					}
					return nil
				})

				// Refresh config and try next
				cfg = h.store.Get()
				next, ok := admin.NextEnabledAfter(cfg, accountIdx)
				if !ok {
					break
				}
				accountIdx = next
				account = cfg.Accounts[accountIdx]
			} else {
				break
			}
		}

		if convID == "" {
			logging.LogDebug("session-create failed, fallback to direct request without conversationId: %v", lastErr)
		}
	} else {
		h.conversationManager.UpdateConversation(convID, messages)
	}

	var resp *http.Response
	if convID != "" {
		resp, err = attempt(convID, account)
	if err != nil {
		if errors.Is(err, errVideoGenerationDisabled) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"video generation is disabled","type":"invalid_request_error"}}`))
			return
		}
		http.Error(w, fmt.Sprintf("Upstream request failed: %v", err), http.StatusBadGateway)
		return
	}
	} else {
		resp, err = attempt("", account)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create session and fallback request failed: %v", err), http.StatusBadGateway)
			return
		}
	}

	writeCORS(w, cfg)
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		// Map upstream error to OpenAI error format
		var errorMsg string
		if len(bodyBytes) > 0 {
			errorMsg = string(bodyBytes)
		} else {
			errorMsg = fmt.Sprintf("Upstream returned status: %d", resp.StatusCode)
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		
		errJSON := fmt.Sprintf(`{"error": {"message": %q, "type": "upstream_error", "param": null, "code": %d}}`, errorMsg, resp.StatusCode)
		w.Write([]byte(errJSON))
		
		// Log the error
		logging.GlobalLogStore.Add(logging.RequestLog{
			ID:               uuid.New().String(),
			Timestamp:        startTime,
			Model:            model,
			AccountID:        account.ID,
			AccountName:      account.Name,
			Status:           "error",
			Error:            fmt.Sprintf("HTTP %d: %s", resp.StatusCode, errorMsg),
		})
		return
	}

	if r.URL.Path == "/v1/chat/completions" {
		service = api.NewOpenAIServiceWithContext(h.longCatClient, cfg.LongCat, account.Cookies)
	}

	chunks, errs := service.ConvertResponse(resp, true)

	// Wrap chunks to capture usage info
	wrappedChunks := make(chan interface{}, 10)
	go func() {
		defer close(wrappedChunks)
		for chunk := range chunks {
			// Extract usage from chunk if available
			if c, ok := chunk.(api.ChatCompletionChunk); ok {
				if c.Usage != nil {
					promptTokens = c.Usage.PromptTokens
					completionTokens = c.Usage.CompletionTokens
					totalTokens = c.Usage.TotalTokens
				}
			} else if c, ok := chunk.(api.ClaudeStreamChunk); ok {
				if c.Usage != nil {
					promptTokens = c.Usage.InputTokens
					completionTokens = c.Usage.OutputTokens
					totalTokens = c.Usage.InputTokens + c.Usage.OutputTokens
				} else if c.MessageDelta != nil {
					promptTokens = c.MessageDelta.Usage.InputTokens
					completionTokens = c.MessageDelta.Usage.OutputTokens
					totalTokens = c.MessageDelta.Usage.InputTokens + c.MessageDelta.Usage.OutputTokens
				}
			}
			wrappedChunks <- chunk
		}
	}()

	if streaming {
		w.Header().Set("Content-Type", service.GetResponseContentType(true))
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}
		err = service.HandleStreamingResponse(w, flusher, wrappedChunks, errs)
	} else {
		err = service.HandleNonStreamingResponse(w, wrappedChunks, errs)
	}

	if err != nil {
		logStatus = "error"
		logErr = err.Error()
		if !streaming { // Only error if not streaming, as streaming might have already sent headers
			http.Error(w, fmt.Sprintf("Failed to handle response: %v", err), http.StatusInternalServerError)
		}
	}

	// Log the request metrics
	logging.GlobalLogStore.Add(logging.RequestLog{
		ID:               uuid.New().String(),
		Timestamp:        startTime,
		Model:            model,
		AccountID:        account.ID,
		AccountName:      account.Name,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		Latency:          time.Since(startTime).Seconds(),
		Status:           logStatus,
		Error:            logErr,
	})
}

func writeCORS(w http.ResponseWriter, cfg *config.Config) {
	origins := strings.TrimSpace(cfg.CorsAllowOrigins)
	if origins == "" {
		origins = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origins)
}

func checkUpstreamAuth(r *http.Request, key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return true
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	xkey := strings.TrimSpace(r.Header.Get("X-Api-Key"))
	if xkey == key {
		return true
	}
	low := strings.ToLower(auth)
	if strings.HasPrefix(low, "bearer ") {
		return strings.TrimSpace(auth[7:]) == key
	}
	return false
}

func main() {
	logging.SetVerboseMode(strings.TrimSpace(strings.ToLower(os.Getenv("VERBOSE"))) == "true")
	store := config.NewStore(config.DefaultConfigPath)
	cfg, err := store.Init()
	if err != nil {
		log.Fatalf("failed to init config store: %v", err)
	}
	addr := config.ServerAddress(cfg)

	mux := http.NewServeMux()
	unified := NewUnifiedHandler(store)
	adminH := admin.NewHandler(store, api.NewLongCatClient())
	adminH.StartHealthCheck(30 * time.Minute)

	mux.Handle("/v1/chat/completions", unified)
	mux.Handle("/v1/messages", unified)
	mux.Handle("/admin", adminH)
	mux.Handle("/admin/", adminH)
	mux.Handle("/admin/login", adminH)
	mux.Handle("/admin/logout", adminH)
	mux.Handle("/admin/api/", adminH)

	fmt.Println("\n=== LongCat API Wrapper (Admin Full) ===")
	fmt.Printf("Config: %s (hot reload)\n", store.Path())
	fmt.Printf("Admin:  http://%s/admin\n", addr)
	fmt.Printf("OpenAI: POST http://%s/v1/chat/completions\n", addr)
	fmt.Printf("Claude: POST http://%s/v1/messages\n\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ---- request helpers ----

func extractMessagesFromRequest(requestBody []byte, path string) ([]types.Message, error) {
	var messages []types.Message
	var rawMessages []struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}

	var tools []any
	if path == "/v1/chat/completions" {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"messages"`
			Tools []any `json:"tools"`
		}
		if err := json.Unmarshal(requestBody, &req); err != nil {
			return nil, err
		}
		rawMessages = req.Messages
		tools = req.Tools
	} else if path == "/v1/messages" {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"messages"`
			System any   `json:"system,omitempty"`
			Tools  []any `json:"tools"`
		}
		if err := json.Unmarshal(requestBody, &req); err != nil {
			return nil, err
		}
		if req.System != nil {
			rawMessages = append(rawMessages, struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			}{Role: "system", Content: req.System})
		}
		rawMessages = append(rawMessages, req.Messages...)
		tools = req.Tools
	}

	for _, m := range rawMessages {
		content := ""
		switch v := m.Content.(type) {
		case string:
			content = v
		case []interface{}:
			for _, item := range v {
				if mm, ok := item.(map[string]interface{}); ok {
					if t, ok2 := mm["text"].(string); ok2 {
						content += t
					}
				}
			}
		}
		if content != "" {
			if strings.Contains(content, "Conversation info (untrusted metadata):") {
				content = openClawMetaRegex.ReplaceAllString(content, "")
			}
			content = strings.TrimSpace(content)
		}

		if content != "" {
			messages = append(messages, types.Message{Role: m.Role, Content: content})
		}
	}

	if len(tools) > 0 && len(messages) > 0 {
		toolsJSON, _ := json.Marshal(tools)
		// Instead of a massive block, we make it an ultra-compact one-liner if possible
		// We only give minimal instructions to save tokens and UI clutter.
		// Compact but explicit instruction to ensure tool calling reliability
		sysPrompt := fmt.Sprintf(`[System: You have these tools: %s. To use them, you MUST reply with EXACTLY this JSON: {"tool_calls": [{"name": "tool_name", "arguments": {"arg1": "val"}}]}]`, string(toolsJSON))

		lastIdx := len(messages) - 1
		messages[lastIdx].Content = sysPrompt + "\n\n[用户输入]\n" + messages[lastIdx].Content
	}

	return messages, nil
}

func extractModelFromRequest(requestBody []byte, path string) (string, error) {
	switch path {
	case "/v1/chat/completions":
		var req api.ChatCompletionRequest
		if err := json.Unmarshal(requestBody, &req); err != nil {
			return "", err
		}
		return req.Model, nil
	case "/v1/messages":
		var req api.ClaudeAPIRequest
		if err := json.Unmarshal(requestBody, &req); err != nil {
			return "", err
		}
		return req.Model, nil
	default:
		return "", nil
	}
}

func isStreamingRequest(requestBody []byte, path string) bool {
	switch path {
	case "/v1/chat/completions":
		var req api.ChatCompletionRequest
		if err := json.Unmarshal(requestBody, &req); err == nil {
			return req.Stream
		}
	case "/v1/messages":
		var req api.ClaudeAPIRequest
		if err := json.Unmarshal(requestBody, &req); err == nil {
			return req.Stream
		}
	}
	return false
}

func createLongCatRequest(messages []types.Message, conversationID string, model string) (api.LongCatRequest, error) {
	content := ""
	if len(messages) > 0 {
		content = messages[len(messages)-1].Content
	}
	
	req := api.LongCatRequest{
		Content:        content,
		ConversationId: conversationID,
		Regenerate:     0,
		Files:          []any{},
	}
	
	modelName := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(modelName, "video") {
		return api.LongCatRequest{}, errVideoGenerationDisabled
	}
	
	// Basic flags
	if strings.Contains(modelName, "thinking") {
		req.ReasonEnabled = 1
	}
	if strings.Contains(modelName, "search") {
		req.SearchEnabled = 1
	}
	
	// Agent injections based on model name
	if strings.Contains(modelName, "deepresearch") || strings.Contains(modelName, "deep-research") {
		req.AgentId = "deepResearch"
		req.CreationParam = map[string]any{"width": 16, "height": 9, "style": ""}
	} else if strings.Contains(modelName, "image") || strings.Contains(modelName, "draw") {
		req.AgentId = "genImage"
		req.CreationParam = map[string]any{"width": 16, "height": 9, "style": ""}
	}

	return req, nil
}

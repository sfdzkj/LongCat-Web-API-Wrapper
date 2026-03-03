package api

import (
    "bufio"
    "encoding/json"
    "fmt"
    "net/http"
    "regexp"
    "strings"
    "time"

    "github.com/JessonChan/longcat-web-api/config"
    "github.com/JessonChan/longcat-web-api/logging"
    "github.com/google/uuid"
)

var mediaURLRegex = regexp.MustCompile(`https?://[^"'\s\\]+`)

// OpenAI compatible request structures
type ChatCompletionRequest struct {
    Model     string          `json:"model"`
    Messages  []OpenaiMessage `json:"messages"`
    Stream    bool            `json:"stream,omitempty"`
    MaxTokens int             `json:"max_tokens,omitempty"`
    Tools     []any           `json:"tools,omitempty"`
}

type OpenaiMessage struct {
    Role    string
    Content any // string or []ClaudeMessageContent
}

// OpenAI compatible response structures
type ChatCompletionChunk struct {
    ID      string   `json:"id"`
    Object  string   `json:"object"`
    Created int64    `json:"created"`
    Model   string   `json:"model"`
    Choices []Choice `json:"choices"`
    Usage   *Usage   `json:"usage,omitempty"`
}

type Choice struct {
    Delta        *Delta  `json:"delta,omitempty"`
    Message      *Delta  `json:"message,omitempty"`
    Index        int    `json:"index"`
    FinishReason string `json:"finish_reason,omitempty"`
}

type Delta struct {
    Role      string     `json:"role,omitempty"`
    Content   string     `json:"content,omitempty"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
    Index    *int          `json:"index,omitempty"`
    ID       string        `json:"id,omitempty"`
    Type     string        `json:"type,omitempty"`
    Function *FunctionCall `json:"function,omitempty"`
}

type FunctionCall struct {
    Name      string `json:"name,omitempty"`
    Arguments string `json:"arguments,omitempty"`
}

// For non-streaming responses
type ChatCompletionResponse struct {
    ID      string   `json:"id"`
    Object  string   `json:"object"`
    Created int64    `json:"created"`
    Model   string   `json:"model"`
    Choices []Choice `json:"choices"`
    Usage   Usage    `json:"usage"`
}

type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

// LongCat specific structures needed for OpenAI service.
type LongCatResponse struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversationId"`
	MessageID      int             `json:"messageId"`
	ParentID       int             `json:"parentId"`
	Object         string          `json:"object"`
	Created        int64           `json:"created"`
	Model          string          `json:"model"`
	Choices        []LongCatChoice `json:"choices"`

	// Old fields
	Content       string `json:"content"`
	ReasonContent string `json:"reasonContent"`
	SearchEnabled bool   `json:"searchEnabled"`
	ReasonEnabled bool   `json:"reasonEnabled"`
	Title         *string `json:"title"`
	ReasonStatus  *string `json:"reasonStatus"`
	SearchStatus  *string `json:"searchStatus"`
	ContentStatus string  `json:"contentStatus"`
	SearchResults *string `json:"searchResults"`
	TokenInfo     TokenInfo `json:"tokenInfo"`
	PluginInfo    *string   `json:"pluginInfo"`
	Event         *LongCatEvent `json:"event"`
	LoadingStatus bool `json:"loadingStatus"`
	Sensitive     bool `json:"sensitive"`
	LastOne       bool `json:"lastOne"`

	// Extended fields for image/deep-research.
	Data    json.RawMessage `json:"data"`
	Result  json.RawMessage `json:"result"`
	ImageURL string         `json:"imageUrl"`
	Image   string         `json:"image"`
	FileURL string         `json:"fileUrl"`
	URL     string         `json:"url"`
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Code    int            `json:"code"`
	Ext     json.RawMessage `json:"ext"`
	Extra   json.RawMessage `json:"extra"`
}

type LongCatEvent struct {
    EventID    string          `json:"eventId"`
    Type       string          `json:"type"`    // create | content | finish
    Content    json.RawMessage `json:"content"` // ✅ can be string OR object
    Status     *string         `json:"status"`  // PROCESSING | FINISHED
    FinishType *string         `json:"finishType"`
    Usage      *LongCatUsageV2 `json:"usage"` // appears on finish
}

type LongCatUsageV2 struct {
    InputTokens  int `json:"inputTokens"`
    OutputTokens int `json:"outputTokens"`
    TotalTokens  int `json:"totalTokens"`
}

type LongCatChoice struct {
    Delta struct {
        Role             string  `json:"role"`
        Content          string  `json:"content"`
        ReasoningContent *string `json:"reasoningContent"`
        FunctionCall     *string `json:"functionCall"`
    } `json:"delta"`
    Index        int    `json:"index"`
    FinishReason string `json:"finishReason"`
}

type TokenInfo struct {
    PromptTokens     int  `json:"promptTokens"`
    CompletionTokens int  `json:"completionTokens"`
    TotalTokens      int  `json:"totalTokens"`
    HasTokens        bool `json:"hasTokens"`
}

// StreamProcessor
type StreamProcessor struct {
    conversationID string
    messageID      int
    parentID       int
    responseID     string
    model          string

    accumulated  strings.Builder
    finishReason string
    tokenInfo    TokenInfo
}

func NewStreamProcessor() *StreamProcessor {
    return &StreamProcessor{
        responseID:  uuid.New().String(),
        model:       "LongCat-Flash",
        accumulated: strings.Builder{},
    }
}

// ContentAsText tries to extract assistant content from event.content (string or object).
func (e *LongCatEvent) ContentAsText() (content string, reason string, searchResults string) {
	if e == nil || len(e.Content) == 0 || string(e.Content) == "null" {
		return "", "", ""
	}
	// Case A: JSON string
	var s string
	if err := json.Unmarshal(e.Content, &s); err == nil {
		return formatURLIfMedia(s), "", ""
	}
	// Case C: JSON array (先检查数组)
	var arrObj []map[string]any
	if err := json.Unmarshal(e.Content, &arrObj); err == nil && len(arrObj) > 0 {
		var urls []string
		for _, item := range arrObj {
			if u, ok := item["url"].(string); ok && u != "" {
				urls = append(urls, u)
			}
		}
		if len(urls) > 0 {
			content = formatURLIfMedia(strings.Join(urls, "\n"))
			return content, "", ""
		}
	}
	// Case B: JSON object
	var obj map[string]any
	if err := json.Unmarshal(e.Content, &obj); err != nil {
		return "", "", ""
	}
	if media := extractMediaContent(obj); media != "" {
		content = media
	}
	// helper to get string field
	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := obj[k]; ok && v != nil {
				switch vv := v.(type) {
				case string:
					return vv
				case map[string]any:
					// sometimes nested like {text:"..."}
					if t, ok := vv["text"].(string); ok {
						return t
					}
				}
			}
		}
		return ""
	}

	if content == "" {
		content = getStr("content", "text", "answer", "final", "finalContent", "url", "image", "file")
	}


	// If it's a plugin response (like an image generation) and we couldn't find a standard string,
	// let's just dump the raw JSON so the user can see the image URL or the data instead of silently dropping it!
	// let's just dump the raw JSON so the user can see the image URL or the data instead of silently dropping it!
	if content == "" && len(obj) > 0 {
	    // Ignore if it only has status or irrelevant fields
	    hasData := false
	    for k := range obj {
	        if k != "status" && k != "finishType" && k != "pluginInfo" && k != "searchResults" && k != "reasonContent" {
	            hasData = true
	            break
	        }
	    }

	    // Suppress dumping if it's just an intermediate processing state
	    if status, ok := obj["status"].(string); ok && status == "PROCESSING" {
	        hasData = false
	    }

		if hasData {
			if b, err := json.MarshalIndent(obj, "", "  "); err == nil {
				content = "```json\n" + string(b) + "\n```"
			}
		}
	}
	reason = getStr("reasonContent", "reason", "thinking", "analysis")

	// searchResults may be string/object/array — keep raw json if present
	if v, ok := obj["searchResults"]; ok && v != nil {
		if b, err := json.Marshal(v); err == nil {
			searchResults = string(b)
		}
	}

	// Some variants might use nested fields
	// e.g. {"delta":{"content":"..."}} etc
	if content == "" {
		if delta, ok := obj["delta"].(map[string]any); ok {
			if c, ok := delta["content"].(string); ok {
				content = c
			}
		}
	}

	return formatURLIfMedia(content), reason, searchResults
}
func isMediaURL(s string) bool {
    if s == "" {
        return false
    }

    lowerAll := strings.ToLower(strings.TrimSpace(s))
	if strings.Contains(lowerAll, "![image](") {
		return true
	}

    lines := strings.Split(s, "\n")
    for _, line := range lines {
        trimmedLine := strings.TrimSpace(line)
        if trimmedLine == "" {
            continue
        }

        // Very basic URL check
        if !strings.HasPrefix(trimmedLine, "http://") && !strings.HasPrefix(trimmedLine, "https://") {
            continue // Not a URL, check next line
        }

		// Check for common image extensions or query parameters indicating a file.
        // This is not exhaustive, but good enough for typical media URLs
        lowerLine := strings.ToLower(trimmedLine)
		if strings.Contains(lowerLine, ".jpg") || strings.Contains(lowerLine, ".jpeg") ||
			strings.Contains(lowerLine, ".png") || strings.Contains(lowerLine, ".gif") ||
			strings.Contains(lowerLine, ".webp") ||
			strings.Contains(lowerLine, "?file=") || strings.Contains(lowerLine, "&file=") ||
			strings.Contains(lowerLine, "?image=") || strings.Contains(lowerLine, "&image=") {
			return true // Found at least one media URL
		}
	}

    return false // No media URLs found in any line
}

func formatURLIfMedia(s string) string {
    if s == "" {
        return s
    }

    lines := strings.Split(s, "\n")
    rawImageURLs := make([]string, 0, 4)
    nonEmptyLines := 0
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed == "" {
            continue
        }
        nonEmptyLines++
        lower := strings.ToLower(trimmed)
        if (strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://")) &&
            (strings.Contains(lower, ".jpg") || strings.Contains(lower, ".jpeg") || strings.Contains(lower, ".png") || strings.Contains(lower, ".gif") || strings.Contains(lower, ".webp") || strings.Contains(lower, "?image=") || strings.Contains(lower, "&image=")) {
            rawImageURLs = append(rawImageURLs, trimmed)
        }
    }
    if nonEmptyLines == 4 && len(rawImageURLs) == 4 {
        return formatImageGrid2x2(rawImageURLs)
    }

    out := make([]string, 0, len(lines))
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed == "" {
            out = append(out, line)
            continue
        }
        if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
            out = append(out, line)
            continue
        }

        lower := strings.ToLower(trimmed)
		if strings.Contains(lower, ".jpg") || strings.Contains(lower, ".jpeg") ||
			strings.Contains(lower, ".png") || strings.Contains(lower, ".gif") ||
			strings.Contains(lower, ".webp") || strings.Contains(lower, "?image=") ||
			strings.Contains(lower, "&image=") {
			out = append(out, fmt.Sprintf("![image](%s)", trimmed))
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func formatImageGrid2x2(urls []string) string {
	if len(urls) < 4 {
		rows := make([]string, 0, len(urls))
		for _, u := range urls {
			rows = append(rows, fmt.Sprintf("![image](%s)", u))
		}
		return strings.Join(rows, "\n")
	}
	return fmt.Sprintf("|  |  |\n|---|---|\n| ![image](%s) | ![image](%s) |\n| ![image](%s) | ![image](%s) |", urls[0], urls[1], urls[2], urls[3])
}

func extractMediaContent(v any) string {
    isURL := func(s string) bool {
        return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
    }
	isImageURL := func(s string) bool {
		lower := strings.ToLower(s)
		return strings.Contains(lower, ".jpg") || strings.Contains(lower, ".jpeg") || strings.Contains(lower, ".png") || strings.Contains(lower, ".gif") || strings.Contains(lower, ".webp") || strings.Contains(lower, "?image=") || strings.Contains(lower, "&image=")
	}

    urls := make([]string, 0, 8)
    seen := map[string]struct{}{}
    addURL := func(s string) {
        s = strings.TrimSpace(strings.ReplaceAll(s, `\/`, "/"))
        if !isURL(s) {
            return
        }
        if _, ok := seen[s]; ok {
            return
        }
        seen[s] = struct{}{}
        urls = append(urls, s)
    }

    var walk func(any)
    walk = func(node any) {
        switch n := node.(type) {
        case map[string]any:
			for _, key := range []string{"imageUrl", "image", "url", "fileUrl", "coverImage"} {
				if v, ok := n[key].(string); ok && strings.TrimSpace(v) != "" {
					addURL(v)
				}
			}
            for _, key := range []string{"data", "result", "extra", "content", "taskResult", "task_result", "payload", "output", "items"} {
                if child, ok := n[key]; ok {
                    walk(child)
                }
            }
        case []any:
            for _, item := range n {
                walk(item)
            }
        case string:
            trimmed := strings.TrimSpace(n)
            if trimmed == "" {
                return
            }
            if isURL(trimmed) {
                addURL(trimmed)
                return
            }
            if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
                var nested any
                if err := json.Unmarshal([]byte(trimmed), &nested); err == nil {
                    walk(nested)
                }
            }
        }
    }
    walk(v)

    if len(urls) == 0 {
        return ""
    }

    imageURLs := make([]string, 0, len(urls))
	for _, u := range urls {
		if isImageURL(u) {
			imageURLs = append(imageURLs, u)
		}
	}

    if len(imageURLs) >= 4 {
        return formatImageGrid2x2(imageURLs[:4])
    }
	if len(imageURLs) > 0 {
		rows := make([]string, 0, len(imageURLs))
		for _, u := range imageURLs {
			rows = append(rows, fmt.Sprintf("![image](%s)", u))
		}
		return strings.Join(rows, "\n")
	}
	return ""
}

func extractMediaFromRawMessage(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var obj any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	return extractMediaContent(obj)
}

func extractMediaFromJSONString(s string) string {
	var obj any
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return ""
	}
	return extractMediaContent(obj)
}

func mediaFromDirectFields(r LongCatResponse) string {
	if r.ImageURL != "" {
		return fmt.Sprintf("![image](%s)", r.ImageURL)
	}
	if r.Image != "" {
		return fmt.Sprintf("![image](%s)", r.Image)
	}
	if r.URL != "" {
		return r.URL
	}
	if r.FileURL != "" {
		return r.FileURL
	}
	return ""
}

func extractMediaFromRawText(raw string) string {
	matches := mediaURLRegex.FindAllString(raw, -1)
	if len(matches) == 0 {
		return ""
	}

	imageURLs := make([]string, 0, 8)
	seenImage := map[string]struct{}{}
	for _, m := range matches {
		u := strings.ReplaceAll(m, `\/`, "/")
		lu := strings.ToLower(u)
		isImage := strings.Contains(lu, ".jpg") || strings.Contains(lu, ".jpeg") || strings.Contains(lu, ".png") || strings.Contains(lu, ".gif") || strings.Contains(lu, ".webp") || strings.Contains(lu, "?image=") || strings.Contains(lu, "&image=")
		if isImage {
			if _, ok := seenImage[u]; !ok {
				seenImage[u] = struct{}{}
				imageURLs = append(imageURLs, u)
			}
		}
	}
	if len(imageURLs) >= 4 {
		return formatImageGrid2x2(imageURLs[:4])
	}
	if len(imageURLs) > 0 {
		rows := make([]string, 0, len(imageURLs))
		for _, u := range imageURLs {
			rows = append(rows, fmt.Sprintf("![image](%s)", u))
		}
		return strings.Join(rows, "\n")
	}
	return ""
}

func resolveContentUnified(r *LongCatResponse, rawSSE string) {
	if r.Content != "" {
		return
	}

	if media := extractMediaFromJSONString(rawSSE); media != "" {
		r.Content = media
		return
	}

	if media := mediaFromDirectFields(*r); media != "" {
		r.Content = media
		return
	}

	for _, raw := range []json.RawMessage{r.Data, r.Result, r.Ext, r.Extra} {
		if media := extractMediaFromRawMessage(raw); media != "" {
			r.Content = media
			return
		}
	}

	if r.Event != nil {
		if media := extractMediaFromRawMessage(r.Event.Content); media != "" {
			r.Content = media
			return
		}
	}

	if r.PluginInfo != nil {
		if plugin := strings.TrimSpace(*r.PluginInfo); plugin != "" {
			if media := extractMediaContent(plugin); media != "" {
				r.Content = media
				return
			}
			r.Content = plugin
			return
		}
	}

	if media := extractMediaFromRawText(rawSSE); media != "" {
		r.Content = media
	}
}

func (p *StreamProcessor) ProcessStream(resp *http.Response, stream bool) (<-chan ChatCompletionChunk, <-chan error) {
    chunks := make(chan ChatCompletionChunk)
    errs := make(chan error, 1)

    go func() {
        defer close(chunks)
        defer close(errs)
        defer resp.Body.Close()

        scanner := bufio.NewScanner(resp.Body)
        for scanner.Scan() {
            line := scanner.Text()
            if !strings.HasPrefix(line, "data:") {
                continue
            }
            data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
            if data == "[DONE]" {
                break
            }

            logging.LogDebug("LongCat Raw SSE: %s", data)

            var longCatResp LongCatResponse
            if err := json.Unmarshal([]byte(data), &longCatResp); err != nil {
                errs <- fmt.Errorf("failed to unmarshal response: %w", err)
                return
            }

			resolveContentUnified(&longCatResp, data)

			// DEBUG: Log all fields to understand response format.
			logging.LogDebug("=== LongCat Response Fields ===")
			logging.LogDebug("Content: %s", longCatResp.Content)
			logging.LogDebug("PluginInfo: %s", longCatResp.PluginInfo)
			logging.LogDebug("ImageURL: %s", longCatResp.ImageURL)
			logging.LogDebug("Image: %s", longCatResp.Image)
			logging.LogDebug("URL: %s", longCatResp.URL)
            logging.LogDebug("Data: %s", longCatResp.Data)
            logging.LogDebug("Result: %s", longCatResp.Result)
            logging.LogDebug("Status: %s", longCatResp.Status)
            logging.LogDebug("Message: %s", longCatResp.Message)
            logging.LogDebug("LastOne: %v", longCatResp.LastOne)
            logging.LogDebug("ContentStatus: %s", longCatResp.ContentStatus)
            if longCatResp.Event != nil {
                logging.LogDebug("Event.Type: %s", longCatResp.Event.Type)
                logging.LogDebug("Event.Content: %s", string(longCatResp.Event.Content))
            }
            logging.LogDebug("=== End Response Fields ===")

			// Unified extraction chain: raw payload -> top-level fields -> data/result/ext/extra -> event -> plugin
			resolveContentUnified(&longCatResp, data)
			if longCatResp.Event != nil {
				switch longCatResp.Event.Type {
				case "finish":
					if longCatResp.Event.Usage != nil {
						longCatResp.TokenInfo = TokenInfo{
							PromptTokens:     longCatResp.Event.Usage.InputTokens,
							CompletionTokens: longCatResp.Event.Usage.OutputTokens,
							TotalTokens:      longCatResp.Event.Usage.TotalTokens,
							HasTokens:        true,
						}
					}
				default:
					// Extract content regardless of whether it's "content", "plugin", "image", etc.
					c, rc, sr := longCatResp.Event.ContentAsText()
					if c != "" && longCatResp.Content == "" {
						longCatResp.Content = c
					}
					if rc != "" {
						longCatResp.ReasonContent = rc
					}
					if sr != "" {
						tmp := sr
						longCatResp.SearchResults = &tmp
					}
					if longCatResp.Event.Status != nil {
						longCatResp.ContentStatus = *longCatResp.Event.Status
					}
				}
			}

			// Re-apply once after event parsing in case event content is nested media JSON
			resolveContentUnified(&longCatResp, data)

            logging.LogDebug("LongCat Response: %+v", longCatResp)

            // Update processor state
            p.conversationID = longCatResp.ConversationID
            p.messageID = longCatResp.MessageID
            p.parentID = longCatResp.ParentID
            if p.model == "" && longCatResp.Model != "" {
                p.model = longCatResp.Model
            }
            if longCatResp.TokenInfo.HasTokens {
                p.tokenInfo = longCatResp.TokenInfo
            }

            // Determine finish reason (safe)
            finishReason := ""
            if len(longCatResp.Choices) > 0 {
                finishReason = longCatResp.Choices[0].FinishReason
            }
            if finishReason == "" && (longCatResp.LastOne || longCatResp.ContentStatus == "FINISHED") {
                finishReason = "stop"
            }
            if finishReason == "" && longCatResp.Event != nil && longCatResp.Event.Type == "finish" {
                finishReason = "stop"
            }
            if finishReason != "" {
                p.finishReason = finishReason
            }

            if longCatResp.Content != "" {
                longCatResp.Content = formatURLIfMedia(longCatResp.Content)
            }

            // Convert to OpenAI chunks
            chunk := p.convertToOpenAIFormat(longCatResp, true)
            if chunk != nil {
                logging.LogDebug("OpenAI Conversion Output: %+v", *chunk)
                chunks <- *chunk
            }

            if longCatResp.LastOne && p.finishReason == "stop" {
                break
            }
        }

        if err := scanner.Err(); err != nil {
            errs <- fmt.Errorf("scanner error: %w", err)
        }
    }()

    return chunks, errs
}

func (p *StreamProcessor) convertToOpenAIFormat(longCatResp LongCatResponse, stream bool) *ChatCompletionChunk {
    if !stream {
        return nil
    }

    // Send role once at the beginning
    role := ""
    if p.accumulated.Len() == 0 {
        role = "assistant"
    }

	// Calculate delta from cumulative content
	content := ""
	if longCatResp.Content != "" {
		acc := p.accumulated.String()
		if strings.HasPrefix(longCatResp.Content, acc) {
			content = strings.TrimPrefix(longCatResp.Content, acc)
		} else if longCatResp.Content != acc {
			content = longCatResp.Content
		}
	}

    var toolCalls []ToolCall
    
    fullContent := p.accumulated.String() + content
    // Also check if it contains the magic string
    isLikelyJSON := strings.HasPrefix(strings.TrimSpace(fullContent), "{") || strings.HasPrefix(strings.TrimSpace(fullContent), "[") || strings.HasPrefix(strings.TrimSpace(fullContent), "`" + "``" + "json") || strings.Contains(fullContent, "\"tool_calls\"")

    // If it looks like a tool call, we suppress outputting the content during streaming
    isMediaContent := isMediaURL(longCatResp.Content)

    // If it looks like a tool call OR generic JSON, AND it's NOT a media URL, we suppress outputting the content during streaming
    if isLikelyJSON && !isMediaContent {
        content = "" // Suppress intermediate JSON text streaming
    }

    // Check if the accumulated content looks like our requested JSON for tool calls at the end
    if (longCatResp.LastOne || p.finishReason == "stop") && isLikelyJSON {
        // Try to clean up markdown code block if present
        cleanContent := fullContent
        if strings.HasPrefix(cleanContent, "`" + "``" + "json") {
            cleanContent = strings.TrimPrefix(cleanContent, "`" + "``" + "json")
            cleanContent = strings.TrimSuffix(strings.TrimSpace(cleanContent), "`" + "``")
        }
        
        // Find the first { and last } to extract JSON
        firstBrace := strings.Index(cleanContent, "{")
        lastBrace := strings.LastIndex(cleanContent, "}")
        
        if firstBrace >= 0 && lastBrace > firstBrace {
            jsonStr := cleanContent[firstBrace : lastBrace+1]
            
            var parsedResp struct {
                ToolCalls []struct {
                    Name      string `json:"name"`
                    Arguments any    `json:"arguments"`
                } `json:"tool_calls"`
            }
            
            if err := json.Unmarshal([]byte(jsonStr), &parsedResp); err == nil && len(parsedResp.ToolCalls) > 0 {
                // It's a tool call! Convert it.
                for i, tc := range parsedResp.ToolCalls {
                    argsBytes, _ := json.Marshal(tc.Arguments)
                    idx := i
                    toolCalls = append(toolCalls, ToolCall{
                        Index: &idx,
                        ID:    "call_" + uuid.New().String()[:8],
                        Type:  "function",
                        Function: &FunctionCall{
                            Name:      tc.Name,
                            Arguments: string(argsBytes),
                        },
                    })
                }
                
                p.finishReason = "tool_calls"
            }
        }
        
        // If it was supposed to be a tool call but failed parsing, 
        // we should output the raw text so the user sees something went wrong
        if len(toolCalls) == 0 {
            content = fullContent // Dump everything at the end
        }
    }

    chunk := &ChatCompletionChunk{
        ID:      p.responseID,
        Object:  "chat.completion.chunk",
        Created: time.Now().Unix(),
        Model:   p.model,
        Choices: []Choice{{
            Delta: &Delta{
                Role:    role,
                Content: content,
                ToolCalls: toolCalls,
            },
            Index:        0,
            FinishReason: p.finishReason,
        }},
    }
    if p.tokenInfo.HasTokens {
        chunk.Usage = &Usage{
            PromptTokens:     p.tokenInfo.PromptTokens,
            CompletionTokens: p.tokenInfo.CompletionTokens,
            TotalTokens:      p.tokenInfo.TotalTokens,
        }
    }

	if content != "" || isLikelyJSON {
		// Calculate what we should append to p.accumulated
		// We need to keep fullContent correct
		if longCatResp.Content != "" {
			acc := p.accumulated.String()
			if strings.HasPrefix(longCatResp.Content, acc) {
				p.accumulated.WriteString(strings.TrimPrefix(longCatResp.Content, acc))
			} else if longCatResp.Content != acc {
				// Re-write accumulated if it's different (rare)
				p.accumulated.Reset()
				p.accumulated.WriteString(longCatResp.Content)
			}
        }
    }

    // Only return a chunk if we have something to say
    if content != "" || len(toolCalls) > 0 || p.finishReason != "" || role != "" {
        return chunk
    }
    return nil
}

// OpenAIService implements APIService for OpenAI compatibility
type OpenAIService struct {
    longCatClient *LongCatClient
    longCatConfig config.LongCatConfig
    cookies       config.CookieConfig
}

func NewOpenAIService(client *LongCatClient) *OpenAIService {
    return &OpenAIService{longCatClient: client}
}

func NewOpenAIServiceWithContext(client *LongCatClient, lc config.LongCatConfig, cookies config.CookieConfig) *OpenAIService {
	return &OpenAIService{
		longCatClient: client,
		longCatConfig: lc,
		cookies:       cookies,
	}
}

func (s *OpenAIService) ConvertResponse(resp *http.Response, stream bool) (<-chan interface{}, <-chan error) {
    chunks := make(chan interface{}, 10)
    errs := make(chan error, 1)

    go func() {
        defer close(chunks)
		defer close(errs)
		defer resp.Body.Close()

		processor := NewStreamProcessor()
		rawChunks, rawErrs := processor.ProcessStream(resp, true)

        for {
            select {
            case chunk, ok := <-rawChunks:
                if !ok {
                    return
                }
                chunks <- chunk
            case err := <-rawErrs:
                if err != nil {
                    errs <- err
                }
                return
            }
        }
    }()

    return chunks, errs
}

func (s *OpenAIService) GetResponseContentType(stream bool) string {
    if stream {
        return "text/event-stream"
    }
    return "application/json"
}

func (s *OpenAIService) HandleNonStreamingResponse(w http.ResponseWriter, chunks <-chan interface{}, errs <-chan error) error {
    var fullContent strings.Builder
    var finishReason string
    responseID := uuid.New().String()
    model := "LongCat-Flash"
    tokenInfo := TokenInfo{}

    for {
        select {
        case chunk, ok := <-chunks:
            if !ok {
                var toolCalls []ToolCall
                var finalContent = fullContent.String()
                
                isMediaContent := isMediaURL(finalContent)

                // If it looks like a tool call, we extract it for the non-streaming response as well
                isLikelyJSON := strings.HasPrefix(strings.TrimSpace(finalContent), "{") || strings.HasPrefix(strings.TrimSpace(finalContent), "[") || strings.HasPrefix(strings.TrimSpace(finalContent), "`" + "``" + "json") || strings.Contains(finalContent, "\"tool_calls\"")
                if isLikelyJSON && !isMediaContent {
                    cleanContent := finalContent
                    if strings.HasPrefix(cleanContent, "`" + "``" + "json") {
                        cleanContent = strings.TrimPrefix(cleanContent, "`" + "``" + "json")
                        cleanContent = strings.TrimSuffix(strings.TrimSpace(cleanContent), "`" + "``")
                    }
                    
                    firstBrace := strings.Index(cleanContent, "{")
                    lastBrace := strings.LastIndex(cleanContent, "}")
                    
                    if firstBrace >= 0 && lastBrace > firstBrace {
                        jsonStr := cleanContent[firstBrace : lastBrace+1]
                        
                        var parsedResp struct {
                            ToolCalls []struct {
                                Name      string `json:"name"`
                                Arguments any    `json:"arguments"`
                            } `json:"tool_calls"`
                        }
                        
                        if err := json.Unmarshal([]byte(jsonStr), &parsedResp); err == nil && len(parsedResp.ToolCalls) > 0 {
                            finalContent = "" // Suppress text if it's a tool call
                            finishReason = "tool_calls"
                            
                            for i, tc := range parsedResp.ToolCalls {
                                argsBytes, _ := json.Marshal(tc.Arguments)
                                idx := i
                                toolCalls = append(toolCalls, ToolCall{
                                    Index: &idx,
                                    ID:    "call_" + uuid.New().String()[:8],
                                    Type:  "function",
                                    Function: &FunctionCall{
                                        Name:      tc.Name,
                                        Arguments: string(argsBytes),
                                    },
                                })
                            }
                        }
                    }
                }

                resp := ChatCompletionResponse{
                    ID:      responseID,
                    Object:  "chat.completion",
                    Created: time.Now().Unix(),
                    Model:   model,
                    Choices: []Choice{{
                        Message: &Delta{
                            Role:    "assistant",
                            Content: finalContent,
                            ToolCalls: toolCalls,
                        },
                        Index:        0,
                        FinishReason: finishReason,
                    }},
                    Usage: Usage{
                        PromptTokens:     tokenInfo.PromptTokens,
                        CompletionTokens: tokenInfo.CompletionTokens,
                        TotalTokens:      tokenInfo.TotalTokens,
                    },
                }
                w.Header().Set("Content-Type", "application/json")
                return json.NewEncoder(w).Encode(resp)
            }

            if openAIChunk, ok := chunk.(ChatCompletionChunk); ok {
                if len(openAIChunk.Choices) > 0 {
                    choice := openAIChunk.Choices[0]
                    if choice.Delta != nil {
                        fullContent.WriteString(choice.Delta.Content)
                    }
                    if choice.FinishReason != "" {
                        finishReason = choice.FinishReason
                    }
                }
                model = openAIChunk.Model
                responseID = openAIChunk.ID
                if openAIChunk.Usage != nil {
                    if openAIChunk.Usage.PromptTokens > tokenInfo.PromptTokens {
                        tokenInfo.PromptTokens = openAIChunk.Usage.PromptTokens
                    }
                    if openAIChunk.Usage.CompletionTokens > tokenInfo.CompletionTokens {
                        tokenInfo.CompletionTokens = openAIChunk.Usage.CompletionTokens
                    }
                    if openAIChunk.Usage.TotalTokens > tokenInfo.TotalTokens {
                        tokenInfo.TotalTokens = openAIChunk.Usage.TotalTokens
                    }
                    tokenInfo.HasTokens = true
                }
            }
        case err := <-errs:
            if err != nil {
                return fmt.Errorf("error processing chunks: %w", err)
            }
        }
    }
}

func (s *OpenAIService) HandleStreamingResponse(w http.ResponseWriter, flusher http.Flusher, chunks <-chan interface{}, errs <-chan error) error {
    hasReceivedContent := false
    keepAliveTicker := time.NewTicker(15 * time.Second)
    defer keepAliveTicker.Stop()

    // Send an early SSE comment so reverse proxies/CDN see activity quickly.
    fmt.Fprintf(w, ": keep-alive\n\n")
    flusher.Flush()

    for {
        select {
        case chunk, ok := <-chunks:
            if !ok {
                if !hasReceivedContent {
                    defaultChunk := ChatCompletionChunk{
                        ID:      uuid.New().String(),
                        Object:  "chat.completion.chunk",
                        Created: time.Now().Unix(),
                        Model:   "LongCat-Flash",
                        Choices: []Choice{{
                            Delta: &Delta{
                                Role:    "assistant",
                                Content: "I apologize, but I'm unable to process your request at the moment.",
                            },
                            Index:        0,
                            FinishReason: "stop",
                        }},
                    }
                    if data, err := json.Marshal(defaultChunk); err == nil {
                        fmt.Fprintf(w, "data: %s\n\n", data)
                        flusher.Flush()
                    }
                }
                fmt.Fprintf(w, "data: [DONE]\n\n")
                flusher.Flush()
                return nil
            }

            hasReceivedContent = true
            if data, err := json.Marshal(chunk); err == nil {
                fmt.Fprintf(w, "data: %s\n\n", data)
                flusher.Flush()
            }

        case <-keepAliveTicker.C:
            // Keep connection active during long-running upstream tasks.
            fmt.Fprintf(w, ": keep-alive\n\n")
            flusher.Flush()

        case err := <-errs:
            if err != nil {
                return fmt.Errorf("error processing stream: %w", err)
            }
        }
    }
}

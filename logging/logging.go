package logging

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// VerboseMode controls logging output - will be set by main package
var VerboseMode bool

// LogDebug prints debug messages only in verbose mode
func LogDebug(format string, args ...interface{}) {
	if VerboseMode {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// LogInfo prints info messages only in verbose mode
func LogInfo(format string, args ...interface{}) {
	if VerboseMode {
		fmt.Printf("[INFO] "+format+"\n", args...)
	}
}

// LogError prints error messages (always shown)
func LogError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

// SetVerboseMode sets the global verbose logging mode
func SetVerboseMode(verbose bool) {
	VerboseMode = verbose
}
type RequestLog struct {
	ID               string    `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	Model            string    `json:"model"`
	AccountID        string    `json:"account_id"`
	AccountName      string    `json:"account_name"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	Latency          float64   `json:"latency"` // in seconds
	Status           string    `json:"status"`  // success, error
	Error            string    `json:"error,omitempty"`
}

type LogStore struct {
	mu   sync.RWMutex
	logs []RequestLog
	max  int
}

func NewLogStore(max int) *LogStore {
	if max <= 0 {
		max = 1000
	}
	return &LogStore{
		logs: make([]RequestLog, 0, max),
		max:  max,
	}
}

func (s *LogStore) Add(log RequestLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.logs) >= s.max {
		s.logs = s.logs[1:]
	}
	s.logs = append(s.logs, log)
}

func (s *LogStore) GetLogs() []RequestLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to avoid race conditions and modification of the original slice
	out := make([]RequestLog, len(s.logs))
	copy(out, s.logs)
	// Reverse order to show latest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

var GlobalLogStore = NewLogStore(1000)
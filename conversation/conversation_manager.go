package conversation

import (
	"strings"
	"sync"
	"time"

	"github.com/JessonChan/longcat-web-api/types"
)

type ConversationEntry struct {
	ConversationID string
	AccountID      string
	Messages       []types.Message
	LastAccessed   time.Time
	CreatedAt      time.Time
}

type ConversationManager struct {
	mu            sync.RWMutex
	conversations map[string]*ConversationEntry
	maxAge        time.Duration
}

func NewConversationManager() *ConversationManager {
	cm := &ConversationManager{
		conversations: make(map[string]*ConversationEntry),
		maxAge:        24 * time.Hour,
	}
	go cm.cleanupExpired()
	return cm
}

func (cm *ConversationManager) FindConversation(messages []types.Message) (*ConversationEntry, bool) {
	if len(messages) == 0 {
		return nil, false
	}
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	var best *ConversationEntry
	bestLen := -1
	
	for _, entry := range cm.conversations {
		if cm.isMatch(entry.Messages, messages) {
			l := len(entry.Messages)
			if l > bestLen {
				best = entry
				bestLen = l
			} else if l == bestLen && best != nil && entry.LastAccessed.After(best.LastAccessed) {
				best = entry
			}
		}
	}
	if best != nil {
		best.LastAccessed = time.Now()
		return best, true
	}
	return nil, false
}

func (cm *ConversationManager) isMatch(stored []types.Message, incoming []types.Message) bool {
	// Filter out system messages
	var sFiltered []types.Message
	for _, m := range stored {
		if m.Role != "system" {
			sFiltered = append(sFiltered, m)
		}
	}
	
	var iFiltered []types.Message
	if len(incoming) > 1 {
		for i := 0; i < len(incoming)-1; i++ {
			if incoming[i].Role != "system" {
				iFiltered = append(iFiltered, incoming[i])
			}
		}
	}

	if len(sFiltered) == 0 || len(iFiltered) == 0 {
		return false
	}

	// Bind if at least one user/assistant message matches
	for _, sm := range sFiltered {
		for _, im := range iFiltered {
			if cm.messagesEqual(sm, im) {
				return true
			}
		}
	}
	return false
}

func (cm *ConversationManager) messagesEqual(m1, m2 types.Message) bool {
	if m1.Role != m2.Role {
		return false
	}
	s1 := strings.TrimSpace(m1.Content)
	s2 := strings.TrimSpace(m2.Content)
	
	if s1 == s2 {
		return true
	}
	// Extremely fuzzy matching: one contains the other
	if len(s1) > 10 && len(s2) > 10 {
		if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
			return true
		}
	}
	// Extremely fuzzy matching: significant prefix (first 50 chars)
	const prefixLen = 50
	if len(s1) >= prefixLen && len(s2) >= prefixLen {
		if s1[:prefixLen] == s2[:prefixLen] {
			return true
		}
	}
	return false
}

func (cm *ConversationManager) SetConversation(conversationID, accountID string, messages []types.Message) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.conversations[conversationID] = &ConversationEntry{
		ConversationID: conversationID, 
		AccountID: accountID, 
		Messages: messages, 
		LastAccessed: time.Now(), 
		CreatedAt: time.Now(),
	}
}

func (cm *ConversationManager) UpdateConversation(conversationID string, messages []types.Message) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if entry, ok := cm.conversations[conversationID]; ok {
		entry.Messages = messages
		entry.LastAccessed = time.Now()
	}
}

func (cm *ConversationManager) ActiveCounts() map[string]int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	out := map[string]int{}
	for _, entry := range cm.conversations {
		out[entry.AccountID]++
	}
	return out
}

func (cm *ConversationManager) cleanupExpired() {
	t := time.NewTicker(1 * time.Hour)
	defer t.Stop()
	for range t.C {
		cm.mu.Lock()
		now := time.Now()
		for id, entry := range cm.conversations {
			if now.Sub(entry.LastAccessed) > cm.maxAge {
				delete(cm.conversations, id)
			}
		}
		cm.mu.Unlock()
	}
}

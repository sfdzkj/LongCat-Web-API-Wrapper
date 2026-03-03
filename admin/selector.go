package admin

import (
	"errors"
	"sort"
	"strings"

	"github.com/JessonChan/longcat-web-api/config"
	conversation "github.com/JessonChan/longcat-web-api/conversation"
)

var ErrNoEnabledAccounts = errors.New("no enabled accounts")

func SelectAccount(store *config.Store, cfg *config.Config, cm *conversation.ConversationManager) (int, config.AccountConfig, error) {
	enabled := make([]int, 0)
	for i, a := range cfg.Accounts {
		if a.Enabled && (strings.TrimSpace(a.Cookies.PassportToken) != "" || strings.TrimSpace(a.Cookies.RawString) != "") {
			enabled = append(enabled, i)
		}
	}
	if len(enabled) == 0 {
		return -1, config.AccountConfig{}, ErrNoEnabledAccounts
	}

	st := strings.ToLower(strings.TrimSpace(cfg.Strategy.Type))
	if st == "sequential" {
		// Sequential: use NextIndex to pick, then increment
		start := cfg.Strategy.NextIndex
		idx := enabled[start%len(enabled)]
		_, _ = store.Update(func(c *config.Config) error {
			c.Strategy.Type = st
			c.Strategy.NextIndex = (start + 1) % len(enabled)
			return nil
		})
		return idx, cfg.Accounts[idx], nil
	}

	// Average strategy: pick account with least active conversations
	counts := cm.ActiveCounts()
	sort.Slice(enabled, func(i, j int) bool {
		ai := cfg.Accounts[enabled[i]].ID
		aj := cfg.Accounts[enabled[j]].ID
		ci := counts[ai]
		cj := counts[aj]
		if ci != cj {
			return ci < cj
		}
		return enabled[i] < enabled[j]
	})
	idx := enabled[0]
	return idx, cfg.Accounts[idx], nil
}

func NextEnabledAfter(cfg *config.Config, currentIdx int) (int, bool) {
	if len(cfg.Accounts) == 0 {
		return -1, false
	}
	for step := 1; step <= len(cfg.Accounts); step++ {
		i := (currentIdx + step) % len(cfg.Accounts)
		a := cfg.Accounts[i]
		if a.Enabled && (strings.TrimSpace(a.Cookies.PassportToken) != "" || strings.TrimSpace(a.Cookies.RawString) != "") {
			return i, true
		}
	}
	return -1, false
}

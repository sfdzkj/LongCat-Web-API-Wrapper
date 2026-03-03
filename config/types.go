package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	Version          int            `json:"version"`
	ServerPort       string         `json:"serverPort"`
	BindAddr         string         `json:"bindAddr"`
	CorsAllowOrigins string         `json:"corsAllowOrigins"`
	UpstreamAPIKey   string         `json:"upstreamApiKey"`
	Admin            AdminConfig    `json:"admin"`
	LongCat          LongCatConfig  `json:"longcat"`
	Accounts         []AccountConfig `json:"accounts"`
	Strategy         StrategyConfig  `json:"strategy"`
}

type AdminConfig struct {
	Password               string `json:"password"`
	MustChangePassword     bool   `json:"mustChangePassword"`
	IsDefaultPassword      bool   `json:"isDefaultPassword"`
	DefaultPasswordPrinted bool   `json:"defaultPasswordPrinted"`
	SessionVersion         int    `json:"sessionVersion"`
}

type LongCatConfig struct {
	APIURL         string `json:"apiUrl"`
	SessionURL     string `json:"sessionUrl"`
	Model          string `json:"model"`
	AgentID        string `json:"agentId"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

type StrategyConfig struct {
	Type      string `json:"type"`
	NextIndex int    `json:"nextIndex"`
}

type AccountConfig struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Note     string       `json:"note"`
	Enabled  bool         `json:"enabled"`
	Cookies  CookieConfig `json:"cookies"`
	LastTest *AccountTest `json:"lastTest,omitempty"`
}

type AccountTest struct {
	At     time.Time `json:"at"`
	OK     bool      `json:"ok"`
	Detail string    `json:"detail"`
}

type CookieConfig struct {
	LxsdkCuid     string `json:"_lxsdk_cuid"`
	PassportToken string `json:"passport_token_key"`
	LxsdkS        string `json:"_lxsdk_s"`
	RawString     string `json:"raw_string,omitempty"`
}


func DefaultConfig() *Config {
	return &Config{
		Version:          1,
		ServerPort:       "8082",
		BindAddr:         "0.0.0.0",
		CorsAllowOrigins: "*",
		UpstreamAPIKey:   "",
		Admin:            AdminConfig{Password: "", MustChangePassword: true, IsDefaultPassword: true, DefaultPasswordPrinted: false, SessionVersion: 1},
		LongCat: LongCatConfig{
			APIURL:         getenv("LONGCAT_API_URL", "https://longcat.chat/api/v1/chat-completion-V2"),
			SessionURL:     getenv("LONGCAT_SESSION_URL", "https://longcat.chat/api/v1/session-create"),
			Model:          getenv("LONGCAT_MODEL", ""),
			AgentID:        getenv("LONGCAT_AGENT_ID", ""),
			TimeoutSeconds: getenvInt("TIMEOUT_SECONDS", 30),
		},
		Accounts: []AccountConfig{},
		Strategy: StrategyConfig{Type: "average", NextIndex: 0},
	}
}

func (c *Config) Normalize() {
	if c.Version == 0 { c.Version = 1 }
	if c.ServerPort == "" { c.ServerPort = "8082" }
	if c.BindAddr == "" { c.BindAddr = "0.0.0.0" }
	if c.CorsAllowOrigins == "" { c.CorsAllowOrigins = "*" }
	if c.LongCat.TimeoutSeconds <= 0 { c.LongCat.TimeoutSeconds = 30 }
	if c.LongCat.APIURL == "" { c.LongCat.APIURL = "https://longcat.chat/api/v1/chat-completion-V2" }
	if c.LongCat.SessionURL == "" { c.LongCat.SessionURL = "https://longcat.chat/api/v1/session-create" }
	st := strings.ToLower(strings.TrimSpace(c.Strategy.Type))
	if st == "" { st = "average" }
	if st != "average" && st != "sequential" { st = "average" }
	c.Strategy.Type = st
	if c.Admin.SessionVersion <= 0 { c.Admin.SessionVersion = 1 }
	for i := range c.Accounts {
		if c.Accounts[i].Name == "" { c.Accounts[i].Name = c.Accounts[i].ID }
	}
}

func getenv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" { return def }
	return v
}

func getenvInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" { return def }
	n := 0
	for _, ch := range v {
		if ch < '0' || ch > '9' { return def }
		n = n*10 + int(ch-'0')
	}
	if n <= 0 { return def }
	return n
}

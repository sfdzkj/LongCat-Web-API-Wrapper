package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/JessonChan/longcat-web-api/api"
	"github.com/JessonChan/longcat-web-api/config"
	"github.com/JessonChan/longcat-web-api/logging"
)

type session struct {
	Token   string
	Expires time.Time
	Ver     int
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]session
}

func NewSessionManager() *SessionManager {
	m := &SessionManager{sessions: map[string]session{}}
	go func() {
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for range t.C {
			m.mu.Lock()
			now := time.Now()
			for k, s := range m.sessions {
				if now.After(s.Expires) {
					delete(m.sessions, k)
				}
			}
			m.mu.Unlock()
		}
	}()
	return m
}

func (m *SessionManager) New(ver int) session {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	tok := hex.EncodeToString(b)
	s := session{Token: tok, Expires: time.Now().Add(config.AdminSessionTTL), Ver: ver}
	m.mu.Lock()
	m.sessions[tok] = s
	m.mu.Unlock()
	return s
}

func (m *SessionManager) Valid(tok string, ver int) bool {
	m.mu.RLock()
	s, ok := m.sessions[tok]
	m.mu.RUnlock()
	if !ok || time.Now().After(s.Expires) || s.Ver != ver {
		return false
	}
	return true
}

func (m *SessionManager) InvalidateAll() {
	m.mu.Lock()
	m.sessions = map[string]session{}
	m.mu.Unlock()
}

type Handler struct {
	Store  *config.Store
	SM     *SessionManager
	Client *api.LongCatClient
	LoginT *template.Template
	DashT  *template.Template
}

func NewHandler(store *config.Store, client *api.LongCatClient) *Handler {
	return &Handler{
		Store:  store,
		SM:     NewSessionManager(),
		Client: client,
		LoginT: template.Must(template.New("login").Parse(loginHTML)),
		DashT:  template.Must(template.New("dash").Parse(dashHTML)),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if p == "/admin" || p == "/admin/" {
		if !h.requireAuth(r) {
			h.renderLogin(w)
			return
		}
		h.renderDash(w, r)
		return
	}
	if p == "/admin/login" {
		if r.Method == http.MethodGet {
			h.renderLogin(w)
			return
		}
		if r.Method == http.MethodPost {
			h.handleLogin(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if p == "/admin/logout" {
		http.SetCookie(w, &http.Cookie{Name: "lc_admin", Value: "", Path: "/admin", HttpOnly: true, MaxAge: -1})
		w.Header().Set("Location", "/admin")
		w.WriteHeader(http.StatusFound)
		return
	}
	if strings.HasPrefix(p, "/admin/api/") {
		if !h.requireAuth(r) {
			jsonErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		h.api(w, r)
		return
	}
	http.NotFound(w, r)
}

func (h *Handler) requireAuth(r *http.Request) bool {
	cfg := h.Store.Get()
	c, err := r.Cookie("lc_admin")
	if err != nil {
		return false
	}
	return h.SM.Valid(c.Value, cfg.Admin.SessionVersion)
}

func (h *Handler) renderLogin(w http.ResponseWriter) {
	cfg := h.Store.Get()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := map[string]any{
		"IsDefault": cfg.Admin.IsDefaultPassword,
	}
	if cfg.Admin.IsDefaultPassword {
		data["DefaultPwd"] = cfg.Admin.Password
	}
	_ = h.LoginT.Execute(w, data)
}

func (h *Handler) renderDash(w http.ResponseWriter, r *http.Request) {
	cfg := h.Store.Get()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	scheme := requestScheme(r)
	host := requestHost(r)
	_ = h.DashT.Execute(w, map[string]any{
		"MustChange": cfg.Admin.MustChangePassword,
		"Host":       host,
		"Scheme":     scheme,
	})
}

func firstHeaderValue(h string) string {
	if h == "" {
		return ""
	}
	v := strings.TrimSpace(strings.Split(h, ",")[0])
	return strings.Trim(v, "\"")
}

func parseForwardedField(forwarded, key string) string {
	if forwarded == "" || key == "" {
		return ""
	}
	parts := strings.Split(forwarded, ";")
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(kv[0]), key) {
			return strings.Trim(strings.TrimSpace(kv[1]), "\"")
		}
	}
	return ""
}

func requestScheme(r *http.Request) string {
	if v := firstHeaderValue(r.Header.Get("X-Forwarded-Proto")); v != "" {
		return strings.ToLower(v)
	}
	if v := firstHeaderValue(r.Header.Get("X-Forwarded-Scheme")); v != "" {
		return strings.ToLower(v)
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-SSL")), "on") {
		return "https"
	}
	if fwd := firstHeaderValue(r.Header.Get("Forwarded")); fwd != "" {
		if proto := parseForwardedField(fwd, "proto"); proto != "" {
			return strings.ToLower(proto)
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func requestHost(r *http.Request) string {
	if v := firstHeaderValue(r.Header.Get("X-Forwarded-Host")); v != "" {
		return v
	}
	if fwd := firstHeaderValue(r.Header.Get("Forwarded")); fwd != "" {
		if host := parseForwardedField(fwd, "host"); host != "" {
			return host
		}
	}
	return r.Host
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	cfg := h.Store.Get()
	body, _ := io.ReadAll(r.Body)
	var in struct {
		Password string `json:"password"`
	}
	_ = json.Unmarshal(body, &in)
	pwd := strings.TrimSpace(in.Password)
	if !config.ComparePassword(cfg.Admin.Password, pwd) {
		jsonErr(w, http.StatusUnauthorized, "invalid password")
		return
	}
	s := h.SM.New(cfg.Admin.SessionVersion)
	http.SetCookie(w, &http.Cookie{Name: "lc_admin", Value: s.Token, Path: "/admin", HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: s.Expires})
	jsonOK(w, map[string]any{"ok": true})
}

func (h *Handler) api(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/admin/api/config" && r.Method == http.MethodGet:
		h.getConfig(w)
		return
	case r.URL.Path == "/admin/api/upstream-key" && r.Method == http.MethodPost:
		h.setKey(w, r)
		return
	case r.URL.Path == "/admin/api/strategy" && r.Method == http.MethodPost:
		h.setStrategy(w, r)
		return
	case r.URL.Path == "/admin/api/password" && r.Method == http.MethodPost:
		h.changePwd(w, r)
		return
	case r.URL.Path == "/admin/api/account" && r.Method == http.MethodPost:
		h.addAccount(w, r)
		return
	case r.URL.Path == "/admin/api/export" && r.Method == http.MethodGet:
		h.exportConfig(w)
		return
	case r.URL.Path == "/admin/api/import" && r.Method == http.MethodPost:
		h.importConfig(w, r)
		return
	case r.URL.Path == "/admin/api/records" && r.Method == http.MethodGet:
		h.getLogs(w)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/admin/api/account/") {
		rest := strings.TrimPrefix(r.URL.Path, "/admin/api/account/")
		parts := strings.Split(rest, "/")
		id := parts[0]
		remain := ""
		if len(parts) > 1 {
			remain = parts[1]
		}
		if remain == "" && r.Method == http.MethodPut {
			h.updateAccount(w, r, id)
			return
		}
		if remain == "" && r.Method == http.MethodDelete {
			h.deleteAccount(w, id)
			return
		}
		if remain == "test" && r.Method == http.MethodPost {
			h.testAccount(w, r, id)
			return
		}
	}
	jsonErr(w, http.StatusNotFound, "not found")
}

func (h *Handler) getConfig(w http.ResponseWriter) {
	cfg := h.Store.Get()
	jsonOK(w, map[string]any{
		"strategy":       cfg.Strategy,
		"accounts":       sanitize(cfg.Accounts),
		"upstreamApiKey": cfg.UpstreamAPIKey,
	})
}

func sanitize(accs []config.AccountConfig) []map[string]any {
	mask := func(s string) string {
		if s == "" {
			return ""
		}
		if len(s) <= 8 {
			return "****"
		}
		return s[:4] + "****" + s[len(s)-4:]
	}
	out := make([]map[string]any, 0, len(accs))
	for _, a := range accs {
		out = append(out, map[string]any{
			"id":      a.ID,
			"name":    a.Name,
			"note":    a.Note,
			"enabled": a.Enabled,
			"cookies": map[string]any{
				"_lxsdk_cuid":        mask(a.Cookies.LxsdkCuid),
				"passport_token_key": mask(a.Cookies.PassportToken),
				"_lxsdk_s":           mask(a.Cookies.LxsdkS),
			},
			"lastTest": a.LastTest,
		})
	}
	return out
}

func (h *Handler) setKey(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var in struct {
		Key string `json:"key"`
	}
	_ = json.Unmarshal(body, &in)
	key := strings.TrimSpace(in.Key)
	_, err := h.Store.Update(func(c *config.Config) error {
		c.UpstreamAPIKey = key
		return nil
	})
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]any{"ok": true})
}

func (h *Handler) setStrategy(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var in struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(body, &in)
	st := strings.ToLower(strings.TrimSpace(in.Type))
	if st != "average" && st != "sequential" {
		jsonErr(w, 400, "type must be average or sequential")
		return
	}
	_, err := h.Store.Update(func(c *config.Config) error {
		c.Strategy.Type = st
		return nil
	})
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]any{"ok": true})
}

func (h *Handler) changePwd(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var in struct {
		Old string `json:"old"`
		New string `json:"new"`
	}
	_ = json.Unmarshal(body, &in)
	oldPwd := strings.TrimSpace(in.Old)
	newPwd := strings.TrimSpace(in.New)
	cfg := h.Store.Get()
	if !config.ComparePassword(cfg.Admin.Password, oldPwd) {
		jsonErr(w, 400, "old password incorrect")
		return
	}
	hash, err := config.HashPassword(newPwd)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	_, err = h.Store.Update(func(c *config.Config) error {
		c.Admin.Password = hash
		c.Admin.IsDefaultPassword = false
		c.Admin.MustChangePassword = false
		c.Admin.SessionVersion++
		return nil
	})
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	h.SM.InvalidateAll()
	http.SetCookie(w, &http.Cookie{Name: "lc_admin", Value: "", Path: "/admin", HttpOnly: true, MaxAge: -1})
	jsonOK(w, map[string]any{"ok": true})
}

func (h *Handler) addAccount(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var in struct {
		Name    string `json:"name"`
		Note    string `json:"note"`
		Cookies any    `json:"cookies"` // can be string or map
	}
	_ = json.Unmarshal(body, &in)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "account"
	}
	note := strings.TrimSpace(in.Note)

	var ck config.CookieConfig
	var err error

	switch v := in.Cookies.(type) {
	case string:
		ck, err = config.ParseRawCookies(v)
	case map[string]any:
		if s, ok := v["_lxsdk_cuid"].(string); ok {
			ck.LxsdkCuid = s
		}
		if s, ok := v["passport_token_key"].(string); ok {
			ck.PassportToken = s
		}
		if s, ok := v["_lxsdk_s"].(string); ok {
			ck.LxsdkS = s
		}
		if ck.PassportToken == "" {
			err = errors.New("missing required cookie: passport_token_key")
		}
	default:
		err = errors.New("invalid cookies format")
	}

	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	_, err = h.Store.Update(func(c *config.Config) error {
		id := randID("acc-", 8)
		c.Accounts = append(c.Accounts, config.AccountConfig{ID: id, Name: name, Note: note, Enabled: true, Cookies: ck})
		return nil
	})
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]any{"ok": true})
}

func (h *Handler) updateAccount(w http.ResponseWriter, r *http.Request, id string) {
	body, _ := io.ReadAll(r.Body)
	var in struct {
		Name    *string `json:"name"`
		Note    *string `json:"note"`
		Enabled *bool   `json:"enabled"`
	}
	_ = json.Unmarshal(body, &in)
	_, err := h.Store.Update(func(c *config.Config) error {
		for i := range c.Accounts {
			if c.Accounts[i].ID == id {
				if in.Enabled != nil {
					c.Accounts[i].Enabled = *in.Enabled
				}
				if in.Name != nil {
					c.Accounts[i].Name = *in.Name
				}
				if in.Note != nil {
					c.Accounts[i].Note = *in.Note
				}
				return nil
			}
		}
		return errors.New("account not found")
	})
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonOK(w, map[string]any{"ok": true})
}

func (h *Handler) deleteAccount(w http.ResponseWriter, id string) {
	_, err := h.Store.Update(func(c *config.Config) error {
		out := make([]config.AccountConfig, 0, len(c.Accounts))
		found := false
		for _, a := range c.Accounts {
			if a.ID == id {
				found = true
				continue
			}
			out = append(out, a)
		}
		if !found {
			return errors.New("account not found")
		}
		c.Accounts = out
		return nil
	})
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonOK(w, map[string]any{"ok": true})
}

func (h *Handler) testAccount(w http.ResponseWriter, r *http.Request, id string) {
	cfg := h.Store.Get()
	var acc *config.AccountConfig
	for i := range cfg.Accounts {
		if cfg.Accounts[i].ID == id {
			acc = &cfg.Accounts[i]
			break
		}
	}
	if acc == nil {
		jsonErr(w, 400, "account not found")
		return
	}
	err := h.Client.Ping(r.Context(), cfg.LongCat, acc.Cookies)
	ok := err == nil
	detail := "ok"
	if err != nil {
		detail = err.Error()
	}
	_, _ = h.Store.Update(func(c *config.Config) error {
		for i := range c.Accounts {
			if c.Accounts[i].ID == id {
				c.Accounts[i].LastTest = &config.AccountTest{At: time.Now(), OK: ok, Detail: detail}
			}
		}
		return nil
	})
	jsonOK(w, map[string]any{"ok": ok, "detail": detail})
}

func (h *Handler) StartHealthCheck(interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			cfg := h.Store.Get()
			for _, acc := range cfg.Accounts {
				if !acc.Enabled {
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				err := h.Client.Ping(ctx, cfg.LongCat, acc.Cookies)
				cancel()

				ok := err == nil
				detail := "ok"
				if err != nil {
					detail = err.Error()
				}

				accID := acc.ID
				_, _ = h.Store.Update(func(c *config.Config) error {
					for i := range c.Accounts {
						if c.Accounts[i].ID == accID {
							c.Accounts[i].LastTest = &config.AccountTest{At: time.Now(), OK: ok, Detail: detail}
							// If sequential strategy and failed, disable it
							if !ok && strings.ToLower(c.Strategy.Type) == "sequential" {
								c.Accounts[i].Enabled = false
							}
						}
					}
					return nil
				})
			}
		}
	}()
}

func (h *Handler) exportConfig(w http.ResponseWriter) {
	cfg := h.Store.Get()
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=longcat_config.json")
	_, _ = w.Write(b)
}

func (h *Handler) importConfig(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		jsonErr(w, 400, "failed to get file")
		return
	}
	defer file.Close()
	b, err := io.ReadAll(file)
	if err != nil {
		jsonErr(w, 400, "failed to read file")
		return
	}
	var cfg config.Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		jsonErr(w, 400, "invalid config format")
		return
	}
	_, err = h.Store.Update(func(c *config.Config) error {
		*c = cfg
		return nil
	})
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]any{"ok": true})
}
func (h *Handler) getLogs(w http.ResponseWriter) {
	logs := logging.GlobalLogStore.GetLogs()
	jsonOK(w, logs)
}


func randID(prefix string, n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return prefix + string(out)
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": msg})
}

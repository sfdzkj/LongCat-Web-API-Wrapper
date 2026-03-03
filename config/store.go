package config

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultConfigPath = "./data/config.json"
	AdminSessionTTL   = 24 * time.Hour
)

type Store struct {
	path        string
	mu          sync.RWMutex
	cfg         atomic.Value
	lastModTime time.Time
	stopCh      chan struct{}
}

func NewStore(path string) *Store {
	if path == "" { path = DefaultConfigPath }
	return &Store{path: path, stopCh: make(chan struct{})}
}

func (s *Store) Path() string { return s.path }

func (s *Store) Init() (*Config, error) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0777); err != nil { return nil, err }
	cfg, err := s.loadOrCreateRepair()
	if err != nil { return nil, err }
	s.cfg.Store(cfg)
	if st, err := os.Stat(s.path); err == nil { s.lastModTime = st.ModTime() }
	go s.pollReload(2 * time.Second)
	return cfg, nil
}

func (s *Store) Get() *Config {
	v := s.cfg.Load()
	if v == nil { return DefaultConfig() }
	cfg := v.(*Config)
	b, _ := json.Marshal(cfg)
	var cp Config
	_ = json.Unmarshal(b, &cp)
	return &cp
}

func (s *Store) Update(mutator func(c *Config) error) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := s.readFromDisk()
	if err != nil {
		cfg, err = s.loadOrCreateRepair()
		if err != nil { return nil, err }
	}
	if err := mutator(cfg); err != nil { return nil, err }
	cfg.Normalize()
	if err := s.writeAtomic(cfg); err != nil { return nil, err }
	s.cfg.Store(cfg)
	if st, err := os.Stat(s.path); err == nil { s.lastModTime = st.ModTime() }
	return cfg, nil
}

func (s *Store) pollReload(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-t.C:
			s.mu.RLock()
			lastMod := s.lastModTime
			s.mu.RUnlock()

			st, err := os.Stat(s.path)
			if err != nil { continue }
			mt := st.ModTime()
			if mt.After(lastMod) {
				cfg, err := s.readFromDisk()
				if err == nil {
					cfg.Normalize()
					s.cfg.Store(cfg)
					s.mu.Lock()
					s.lastModTime = mt
					s.mu.Unlock()
				}
			}

		}
	}
}

func (s *Store) readFromDisk() (*Config, error) {
	f, err := os.Open(s.path)
	if err != nil { return nil, err }
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil { return nil, err }
	if len(b) == 0 { return nil, fmt.Errorf("empty config file") }
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil { return nil, err }
	cfg.Normalize()
	return &cfg, nil
}

func (s *Store) loadOrCreateRepair() (*Config, error) {
	cfg, err := s.readFromDisk()
	if err == nil { return cfg, nil }
	if errors.Is(err, os.ErrNotExist) { return s.createDefault() }
	_ = s.backupCorrupted()
	return s.createDefault()
}

func (s *Store) backupCorrupted() error {
	b, err := os.ReadFile(s.path)
	if err != nil { return err }
	bak := fmt.Sprintf("%s.bak.%d", s.path, time.Now().Unix())
	return os.WriteFile(bak, b, 0666)
}

func (s *Store) createDefault() (*Config, error) {
	cfg := DefaultConfig()
	pwd := randPassword(16)
	cfg.Admin.Password = pwd
	cfg.Admin.IsDefaultPassword = true
	cfg.Admin.MustChangePassword = true
	cfg.Admin.DefaultPasswordPrinted = true
	cfg.Admin.SessionVersion = 1
	cfg.Normalize()
	if err := s.writeAtomic(cfg); err != nil { return nil, err }
	fmt.Printf("\n======================================================\n")
	fmt.Printf("[LongCat-Wrapper] Admin default password: %s\n", pwd)
	fmt.Printf("======================================================\n\n")
	return cfg, nil
}

func (s *Store) writeAtomic(cfg *Config) error {
	tmp := s.path + ".tmp"
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil { return err }
	if err := os.WriteFile(tmp, b, 0666); err != nil { return err }
	return os.Rename(tmp, s.path)
}

func ComparePassword(stored string, password string) bool {
	if stored == "" { return false }
	return stored == password
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 { return "", fmt.Errorf("password too short (min 8)") }
	return password, nil
}

func randPassword(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	out := make([]byte, n)
	for i := 0; i < n; i++ { out[i] = alphabet[int(b[i])%len(alphabet)] }
	return string(out)
}

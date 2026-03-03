package config

import "net"

func ServerAddress(cfg *Config) string {
	addr := cfg.BindAddr
	if addr == "" { addr = "0.0.0.0" }
	port := cfg.ServerPort
	if port == "" { port = "8082" }
	return net.JoinHostPort(addr, port)
}

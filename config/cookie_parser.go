package config

import (
	"fmt"
	"strings"
)

func ParseRawCookies(raw string) (CookieConfig, error) {
	cookies := CookieConfig{}
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "\n", "")
	raw = strings.ReplaceAll(raw, "\r", "")
	if raw == "" { return cookies, fmt.Errorf("empty cookie string") }
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" { continue }
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 { continue }
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch k {
		case "_lxsdk_cuid": cookies.LxsdkCuid = v
		case "passport_token_key": cookies.PassportToken = v
		case "_lxsdk_s": cookies.LxsdkS = v
		}
	}
	cookies.RawString = raw
	if cookies.PassportToken == "" {
		// Just store the raw string, don't fail completely
		return cookies, nil
	}
	return cookies, nil
	return cookies, nil
}

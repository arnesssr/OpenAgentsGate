package main

import "strings"

var sensitiveArgParts = []string{
	"authorization",
	"api-key",
	"api_key",
	"apikey",
	"access-token",
	"access_token",
	"password",
	"passwd",
	"secret",
	"token",
	"cookie",
	"private-key",
	"private_key",
}

func commandInput(argv []string, cwd string) map[string]any {
	return map[string]any{
		"argv": redactArgv(argv),
		"cwd":  cwd,
	}
}

func redactArgv(argv []string) []string {
	out := make([]string, len(argv))
	redactNext := false
	for i, arg := range argv {
		if redactNext {
			out[i] = "[REDACTED]"
			redactNext = false
			continue
		}
		if key, value, ok := strings.Cut(arg, "="); ok && sensitiveArgKey(key) {
			out[i] = key + "=[REDACTED]"
			if value == "" {
				redactNext = false
			}
			continue
		}
		out[i] = arg
		if sensitiveArgKey(arg) {
			redactNext = true
		}
	}
	return out
}

func sensitiveArgKey(arg string) bool {
	key := strings.ToLower(strings.TrimLeft(strings.TrimSpace(arg), "-"))
	for _, part := range sensitiveArgParts {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

package redact

import "strings"

const replacement = "[REDACTED]"

var sensitiveKeyParts = []string{
	"authorization",
	"api_key",
	"apikey",
	"access_token",
	"refresh_token",
	"password",
	"passwd",
	"secret",
	"token",
	"cookie",
	"private_key",
}

func Map(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		if sensitiveKey(key) {
			out[key] = replacement
			continue
		}
		out[key] = Value(value)
	}
	return out
}

func Value(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return Map(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = Value(item)
		}
		return out
	default:
		return value
	}
}

func sensitiveKey(key string) bool {
	normalized := normalizeKey(key)
	for _, part := range sensitiveKeyParts {
		if strings.Contains(normalized, normalizeKey(part)) {
			return true
		}
	}
	return false
}

func normalizeKey(key string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '-', '_', '.', ' ', '\t', '\n', '\r':
			return -1
		default:
			return r
		}
	}, strings.ToLower(strings.TrimSpace(key)))
}

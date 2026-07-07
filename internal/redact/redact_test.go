package redact

import "testing"

func TestMapRedactsCommonSecretKeys(t *testing.T) {
	input := map[string]any{
		"Authorization":  "Bearer secret",
		"X-Api-Key":      "api-secret",
		"OPENAI_API_KEY": "sk-secret",
		"client-secret":  "client-secret",
		"private-key":    "private-key",
		"refresh-token":  "refresh-token",
		"session_cookie": "cookie",
		"safe":           "visible",
	}

	got := Map(input)
	for _, key := range []string{
		"Authorization",
		"X-Api-Key",
		"OPENAI_API_KEY",
		"client-secret",
		"private-key",
		"refresh-token",
		"session_cookie",
	} {
		if got[key] != replacement {
			t.Fatalf("%s = %q, want redacted; full=%#v", key, got[key], got)
		}
	}
	if got["safe"] != "visible" {
		t.Fatalf("safe = %q, want visible", got["safe"])
	}
}

func TestMapRedactsNestedInputs(t *testing.T) {
	input := map[string]any{
		"headers": map[string]any{
			"Authorization": "Bearer secret",
			"Content-Type":  "application/json",
		},
		"steps": []any{
			map[string]any{
				"api-key": "abc",
				"path":    "README.md",
			},
		},
	}

	got := Map(input)
	headers := got["headers"].(map[string]any)
	if headers["Authorization"] != replacement {
		t.Fatalf("authorization = %q, want redacted", headers["Authorization"])
	}
	if headers["Content-Type"] != "application/json" {
		t.Fatalf("content type = %q, want application/json", headers["Content-Type"])
	}
	steps := got["steps"].([]any)
	first := steps[0].(map[string]any)
	if first["api-key"] != replacement {
		t.Fatalf("api-key = %q, want redacted", first["api-key"])
	}
	if first["path"] != "README.md" {
		t.Fatalf("path = %q, want README.md", first["path"])
	}
}

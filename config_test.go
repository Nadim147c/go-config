package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"reflect"
	"testing"
)

func Json(v any) string {
	return string(should(json.MarshalIndent(v, "", "   ")))
}

func TestReadConfigWithIncludes(t *testing.T) {
	// Setup
	c := New()
	c.paths = []string{"./test/config.json"}
	c.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	c.defaultFormat = "json"

	// Execute
	c.ReadConfig()

	// Verify structure
	expected := map[string]any{
		"app": map[string]any{
			"name":  "MyApp",
			"port":  "8080",
			"env":   "production",
			"debug": true,
		},
		"database": map[string]any{
			"host": "db.example.com",
			"port": "5432",
		},
		"logging": map[string]any{
			"level":  "debug",
			"format": "json",
		},
		"feature_flags": map[string]any{
			"new_ui": true,
		},
	}

	if !reflect.DeepEqual(c.config, expected) {
		t.Errorf("Config does not match expected structure:\nGot: %s\nWant: %s",
			Json(c.config),
			Json(expected))
	}

	// Verify no include keys remain
	if containsIncludeKey(c.config) {
		t.Error("Final config should not contain any 'include' keys")
	}
}

func containsIncludeKey(m map[string]any) bool {
	for k, v := range m {
		if k == "include" {
			return true
		}
		if nested, ok := v.(map[string]any); ok {
			if containsIncludeKey(nested) {
				return true
			}
		}
	}
	return false
}

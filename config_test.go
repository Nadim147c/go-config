package config_test

import (
	"encoding/json"
	"log/slog"
	"reflect"
	"testing"

	"github.com/Nadim147c/go-config"
)

// must indicates that there must not be any error; it panics if an error occurs.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func JSON(v any) string {
	return string(must(json.MarshalIndent(v, "", "   ")))
}

func TestReadConfigWithIncludes(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	c := config.New()
	c.AddFile("./test/config.json")
	c.SetFormat("json")

	// Execute
	c.ReadConfig()

	// Verify structure
	expected := map[string]any{
		"app": map[string]any{
			"name":  "MyApp",
			"port":  "8080",
			"env":   "production",
			"debug": "true",
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

	if !reflect.DeepEqual(c.Settings(), expected) {
		t.Fatalf("Config does not match expected structure:\nGot: %s\nWant: %s",
			JSON(c.Settings()),
			JSON(expected))
	}

	// Verify no include keys remain
	if containsIncludeKey(c.Settings()) {
		t.Fatal("Final config should not contain any 'include' keys")
	}

	dbPort := c.GetIntMust("database.port")
	if dbPort != 5432 {
		t.Fatalf("c.GetInt(\"database.port\") = %d, want = %d", dbPort, 5432)
	}

	debug := c.GetBoolMust("app.debug")
	if !debug {
		t.Fatalf("c.GetBool(\"app.debug\") = %v, want = %v", debug, true)
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

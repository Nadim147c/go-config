package config_test

import (
	"encoding/json"
	"log/slog"
	"os"
	"reflect"
	"testing"

	"github.com/Nadim147c/go-config"
	"github.com/spf13/pflag"
)

func JSON(v any) string {
	return string(config.Must(json.MarshalIndent(v, "", "   ")))
}

func TestConfig(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	tests := []struct {
		name     string
		setup    func() *config.Config
		validate func(t *testing.T, c *config.Config)
	}{
		{
			name: "config matches expected structure",
			setup: func() *config.Config {
				c := config.New()
				c.AddFile("./test/config.json")
				c.SetFormat("json")
				c.ReadConfig()
				return c
			},
			validate: func(t *testing.T, c *config.Config) {
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
			},
		},
		{
			name: "no include keys remain",
			setup: func() *config.Config {
				c := config.New()
				c.AddFile("./test/config.json")
				c.SetFormat("json")
				c.ReadConfig()
				return c
			},
			validate: func(t *testing.T, c *config.Config) {
				if containsIncludeKey(c.Settings()) {
					t.Fatal("Final config should not contain any 'include' keys")
				}
			},
		},
		{
			name: "nested default value works",
			setup: func() *config.Config {
				c := config.New()
				c.ReadConfig()
				c.SetDefault("a.b.c.d.e.f", "nested-value")
				return c
			},
			validate: func(t *testing.T, c *config.Config) {
				if v := c.GetMust("a.b.c.d.e.f"); v != "nested-value" {
					t.Fatalf("GetMust(\"a.b.c.d.e.f\") = %s, want = %s", v, "nested-value")
				}
			},
		},
		{
			name: "int value retrieval works",
			setup: func() *config.Config {
				c := config.New()
				c.AddFile("./test/config.json")
				c.SetFormat("json")
				c.ReadConfig()
				return c
			},
			validate: func(t *testing.T, c *config.Config) {
				dbPort := c.GetIntMust("database.port")
				if dbPort != 5432 {
					t.Fatalf("c.GetInt(\"database.port\") = %d, want = %d", dbPort, 5432)
				}
			},
		},
		{
			name: "bool value retrieval works",
			setup: func() *config.Config {
				c := config.New()
				c.AddFile("./test/config.json")
				c.SetFormat("json")
				c.ReadConfig()
				return c
			},
			validate: func(t *testing.T, c *config.Config) {
				debug := c.GetBoolMust("app.debug")
				if !debug {
					t.Fatalf("c.GetBool(\"app.debug\") = %v, want = %v", debug, true)
				}
			},
		},
		{
			name: "env overrides config",
			setup: func() *config.Config {
				c := config.New()
				c.AddFile("./test/config.json")
				c.SetFormat("json")
				c.ReadConfig()
				c.SetEnvPrefix("CONFIG")
				_ = os.Setenv("CONFIG_ENV", "prod")
				return c
			},
			validate: func(t *testing.T, c *config.Config) {
				env := c.GetStringMust("env")
				if env != "prod" {
					t.Fatalf("c.GetStringMust(\"env\") = %v, want = %v", env, "prod")
				}
			},
		},
		{
			name: "pflag set overrides config",
			setup: func() *config.Config {
				c := config.New()
				set := pflag.NewFlagSet("app", pflag.ContinueOnError)
				set.String("mode", "", "set mode")
				_ = set.Parse([]string{})
				_ = set.Set("mode", "test")
				c.SetPflagSet(set)
				return c
			},
			validate: func(t *testing.T, c *config.Config) {
				mode := c.GetStringMust("mode")
				if mode != "test" {
					t.Fatalf("c.GetStringMust(\"mode\") = %v, want = %v", mode, "test")
				}
			},
		},
		{
			name: "add single pflag overrides existing",
			setup: func() *config.Config {
				c := config.New()
				set2 := pflag.NewFlagSet("app2", pflag.ContinueOnError)
				set2.String("mode", "", "set mode")
				_ = set2.Parse([]string{})
				_ = set2.Set("mode", "test2")
				c.AddPflag("", set2.Lookup("mode"))
				return c
			},
			validate: func(t *testing.T, c *config.Config) {
				mode2 := c.GetStringMust("mode")
				if mode2 != "test2" {
					t.Fatalf("c.GetStringMust(\"mode\") = %v, want = %v", mode2, "test2")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.setup()
			tt.validate(t, c)
		})
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

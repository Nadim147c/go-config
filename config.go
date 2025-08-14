package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cast"
)

// must indicates that there must not be any error; it panics if an error occurs.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

type (
	DecodeFunc func([]byte) (map[string]any, error)
	EncodeFunc func(map[string]any) ([]byte, error)
)

type (
	UnmarshalFunc func([]byte, any) error
	MarshalFunc   func(any) ([]byte, error)
)

func DecoderFromUnmarshal(unmarshall UnmarshalFunc) DecodeFunc {
	return func(b []byte) (map[string]any, error) {
		v := map[string]any{}
		err := unmarshall(b, &v)
		return v, err
	}
}

func EncoderFromMarshal(marshall MarshalFunc) EncodeFunc {
	return func(m map[string]any) ([]byte, error) {
		return marshall(m)
	}
}

var cfg = New()

type Config struct {
	config        map[string]any
	envPrefix     string
	paths         []string
	logger        *slog.Logger
	defaultFormat string
	decoders      map[string]DecodeFunc
	encoders      map[string]EncodeFunc
}

func New() *Config {
	return &Config{
		encoders: map[string]EncodeFunc{
			"json": EncoderFromMarshal(json.Marshal),
			"yaml": EncoderFromMarshal(yaml.Marshal),
			"yml":  EncoderFromMarshal(yaml.Marshal),
			"toml": EncoderFromMarshal(toml.Marshal),
		},
		decoders: map[string]DecodeFunc{
			"json": DecoderFromUnmarshal(json.Unmarshal),
			"yaml": DecoderFromUnmarshal(yaml.Unmarshal),
			"yml":  DecoderFromUnmarshal(yaml.Unmarshal),
			"toml": DecoderFromUnmarshal(toml.Unmarshal),
		},
	}
}

func (c *Config) SetEnvPrefix(p string) {
	c.envPrefix = strings.ReplaceAll(p, "_", "")
}

func (c *Config) ReadConfig() {
	config := map[string]any{}
	for _, path := range c.paths {
		visited := make(map[string]bool)
		m, err := c.readConfigFile(path, visited)
		if err != nil {
			if os.IsNotExist(err) {
				c.logger.Debug("Config path doesn't exist", "path", path)
			} else {
				c.logger.Warn("Failed to load config", "error", err)
			}
			continue
		}
		DeepMerge(config, m)
	}
	c.config = config
}

func (c *Config) readConfigFile(path string, visited map[string]bool) (map[string]any, error) {
	if visited[path] {
		return nil, fmt.Errorf("cycle import detected: %s", path)
	}
	visited[path] = true
	defer delete(visited, path)

	m, err := c.parse(path)
	if err != nil {
		return nil, err
	}

	base := make(map[string]any)
	dir := filepath.Dir(path)

	if includeVal, ok := m["include"]; ok {
		delete(m, "include")
		switch v := includeVal.(type) {
		case string:
			included, err := c.resolveInclude(dir, v, visited)
			if err != nil {
				c.logger.Warn("Failed to load included config", "path", v, "error", err)
			} else {
				DeepMerge(base, included)
			}
		case []any:
			for _, item := range v {
				if inc, ok := item.(string); ok {
					included, err := c.resolveInclude(dir, inc, visited)
					if err != nil {
						c.logger.Warn("Failed to load included config", "path", inc, "error", err)
					} else {
						DeepMerge(base, included)
					}
				}
			}
		}
	}

	DeepMerge(base, m)
	return base, nil
}

func (c *Config) resolveInclude(baseDir, include string, visited map[string]bool) (map[string]any, error) {
	includePath, err := FindPath(baseDir, include)
	if err != nil {
		return nil, err
	}
	return c.readConfigFile(includePath, visited)
}

func (c *Config) parse(path string) (m map[string]any, err error) {
	ext := filepath.Ext(path)[1:]

	if _, err := os.Stat(path); err != nil {
		return m, err
	}

	decoder, ok := c.decoders[ext]
	if !ok {
		decoder, ok = c.decoders[c.defaultFormat]
		if !ok {
			return m, fmt.Errorf("decoder not found for format: %v", ext)
		}
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return m, fmt.Errorf("failed to read file: %v", err)
	}

	m, err = decoder(b)
	if err != nil {
		return m, fmt.Errorf("%s: %v", ext, err)
	}
	return m, nil
}

func (c *Config) Set(key string, v any) error {
	nested, err := KeySplit(key)
	if err != nil {
		return err
	}

	m := c.config
	for i := 0; i < len(nested)-1; i++ {
		part := nested[i]
		if next, ok := m[part]; ok {
			if subMap, ok := next.(map[string]any); ok {
				m = subMap
			} else {
				// Overwrite non-map value with a new map
				newMap := make(map[string]any)
				m[part] = newMap
				m = newMap
			}
		} else {
			newMap := make(map[string]any)
			m[part] = newMap
			m = newMap
		}
	}
	m[nested[len(nested)-1]] = v
	return nil
}

// Get returns the value for the key, or error if missing/invalid.
func GetE(key string) (any, error) { return cfg.GetE(key) }

// GetE returns the value for the key, or an error if missing/invalid.
func (c *Config) GetE(key string) (any, error) {
	nested, err := KeySplit(key)
	if err != nil {
		return nil, err
	}

	var prefix strings.Builder

	m := c.config
	for i := 0; i < len(nested)-1; i++ {
		part := nested[i]
		prefix.WriteString(part)

		next, ok := m[part]
		if !ok {
			return nil, fmt.Errorf("key not found: %s", prefix.String())
		}

		subMap, ok := next.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid type for key: %s (expected map)", prefix.String())
		}
		m = subMap

		prefix.WriteByte('.')
	}

	val, ok := m[nested[len(nested)-1]]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return val, nil
}

// GetInt returns the int value for the key, or false if missing/invalid.
func GetIntE(key string) (int, error) { return cfg.GetIntE(key) }

// GetInt returns the int value for the key, or false if missing/invalid.
func (c *Config) GetIntE(key string) (int, error) {
	return getValueE(c, key, cast.ToIntE)
}

// GetInt returns the int value for the key. Panics if missing/invalid.
func GetIntMust(key string) int { return cfg.GetIntMust(key) }

// GetInt returns the int value for the key. Panics if missing/invalid.
func (c *Config) GetIntMust(key string) int {
	return getValueMust(c, key, cast.ToIntE)
}

// GetInt returns the int value for the key. Returns default if missing/invalid.
func GetInt(key string) int { return cfg.GetInt(key) }

// GetInt returns the int value for the key. Returns default if missing/invalid.
func (c *Config) GetInt(key string) int {
	return getValue(c, key, cast.ToIntE)
}

// GetInt64 returns the int64 value for the key, or false if missing/invalid.
func GetInt64E(key string) (int64, error) { return cfg.GetInt64E(key) }

// GetInt64 returns the int64 value for the key, or false if missing/invalid.
func (c *Config) GetInt64E(key string) (int64, error) {
	return getValueE(c, key, cast.ToInt64E)
}

// GetInt64Must returns the int64 value for the key. Panics if missing/invalid.
func GetInt64Must(key string) int64 { return cfg.GetInt64Must(key) }

// GetInt64Must returns the int64 value for the key. Panics if missing/invalid.
func (c *Config) GetInt64Must(key string) int64 {
	return getValueMust(c, key, cast.ToInt64E)
}

// GetInt64 returns the int64 value for the key. Returns default if missing/invalid.
func GetInt64(key string) int64 { return cfg.GetInt64(key) }

// GetInt64 returns the int64 value for the key. Returns default if missing/invalid.
func (c *Config) GetInt64(key string) int64 {
	return getValue(c, key, cast.ToInt64E)
}

// GetUint returns the uint value for the key, or false if missing/invalid.
func GetUintE(key string) (uint, error) { return cfg.GetUintE(key) }

// GetUint returns the uint value for the key, or false if missing/invalid.
func (c *Config) GetUintE(key string) (uint, error) {
	return getValueE(c, key, cast.ToUintE)
}

// GetUintMust returns the uint value for the key. Panics if missing/invalid.
func GetUintMust(key string) uint { return cfg.GetUintMust(key) }

// GetUintMust returns the uint value for the key. Panics if missing/invalid.
func (c *Config) GetUintMust(key string) uint {
	return getValueMust(c, key, cast.ToUintE)
}

// GetUint returns the uint value for the key. Returns default if missing/invalid.
func GetUint(key string) uint { return cfg.GetUint(key) }

// GetUint returns the uint value for the key. Returns default if missing/invalid.
func (c *Config) GetUint(key string) uint {
	return getValue(c, key, cast.ToUintE)
}

// GetUint64 returns the uint64 value for the key, or false if missing/invalid.
func GetUint64E(key string) (uint64, error) { return cfg.GetUint64E(key) }

// GetUint64 returns the uint64 value for the key, or false if missing/invalid.
func (c *Config) GetUint64E(key string) (uint64, error) {
	return getValueE(c, key, cast.ToUint64E)
}

// GetUint64Must returns the uint64 value for the key. Panics if missing/invalid.
func GetUint64Must(key string) uint64 { return cfg.GetUint64Must(key) }

// GetUint64Must returns the uint64 value for the key. Panics if missing/invalid.
func (c *Config) GetUint64Must(key string) uint64 {
	return getValueMust(c, key, cast.ToUint64E)
}

// GetUint64 returns the uint64 value for the key. Returns default if missing/invalid.
func GetUint64(key string) uint64 { return cfg.GetUint64(key) }

// GetUint64 returns the uint64 value for the key. Returns default if missing/invalid.
func (c *Config) GetUint64(key string) uint64 {
	return getValue(c, key, cast.ToUint64E)
}

// GetString returns the string value for the key, or false if missing/invalid.
func GetStringE(key string) (string, error) { return cfg.GetStringE(key) }

// GetString returns the string value for the key, or false if missing/invalid.
func (c *Config) GetStringE(key string) (string, error) {
	return getValueE(c, key, cast.ToStringE)
}

// GetStringMust returns the string value for the key. Panics if missing/invalid.
func GetStringMust(key string) string { return cfg.GetStringMust(key) }

// GetStringMust returns the string value for the key. Panics if missing/invalid.
func (c *Config) GetStringMust(key string) string {
	return getValueMust(c, key, cast.ToStringE)
}

// GetString returns the string value for the key. Returns default if missing/invalid.
func GetString(key string) string { return cfg.GetString(key) }

// GetString returns the string value for the key. Returns default if missing/invalid.
func (c *Config) GetString(key string) string {
	return getValue(c, key, cast.ToStringE)
}

// GetBool returns the bool value for the key, or false if missing/invalid.
func GetBoolE(key string) (bool, error) { return cfg.GetBoolE(key) }

// GetBool returns the bool value for the key, or false if missing/invalid.
func (c *Config) GetBoolE(key string) (bool, error) {
	return getValueE(c, key, cast.ToBoolE)
}

// GetBoolMust returns the bool value for the key. Panics if missing/invalid.
func GetBoolMust(key string) bool { return cfg.GetBoolMust(key) }

// GetBoolMust returns the bool value for the key. Panics if missing/invalid.
func (c *Config) GetBoolMust(key string) bool {
	return getValueMust(c, key, cast.ToBoolE)
}

// GetBool returns the bool value for the key. Returns default if missing/invalid.
func GetBool(key string) bool { return cfg.GetBool(key) }

// GetBool returns the bool value for the key. Returns default if missing/invalid.
func (c *Config) GetBool(key string) bool {
	return getValue(c, key, cast.ToBoolE)
}

// Generic helper for type-safe get with casting
func getValueE[T any](c *Config, key string, conv func(any) (T, error)) (T, error) {
	var zero T
	v, err := c.GetE(key)
	if err != nil {
		return zero, err
	}
	t, err := conv(v)
	if err != nil {
		return zero, err
	}
	return t, nil
}

// Generic helper for type-safe get with casting
func getValue[T any](c *Config, key string, conv func(any) (T, error)) T {
	var zero T
	v, err := c.GetE(key)
	if err != nil {
		return zero
	}
	t, err := conv(v)
	if err != nil {
		return zero
	}
	return t
}

// Generic helper for type-safe get with casting
func getValueMust[T any](c *Config, key string, conv func(any) (T, error)) T {
	v, err := c.GetE(key)
	if err != nil {
		panic(err)
	}
	t, err := conv(v)
	if err != nil {
		panic(err)
	}
	return t
}

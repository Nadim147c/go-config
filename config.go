package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cast"
)

// Must indicates that there Must not be any error; it panics if an error occurs.
func Must[T any](v T, err error) T {
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
	config    map[string]any
	envPrefix string
	logger    *slog.Logger

	// paths is slice all paths to look for config
	paths []string
	// fullPath indicates if the config path is fullPath or directory
	fullPath      map[string]bool
	defaultFormat string
	fileName      string

	decoders map[string]DecodeFunc
	encoders map[string]EncodeFunc
}

func New() *Config {
	return &Config{
		logger:   slog.Default(),
		config:   make(map[string]any),
		fullPath: make(map[string]bool),
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

// basenameWithoutExt return filename without extension
func basenameWithoutExt(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}

// GetConfigFiles resolves and returns all configuration files that should be read by ReadConfig.
// It processes all registered paths (added via AddPath/AddFile) and returns a consolidated list
// of valid configuration files according to these rules:
//
// 1. For paths marked as "full" (added via AddFile):
//   - Verifies the exact file path exists
//   - Includes it directly if found
//
// 2. For regular directory paths (added via AddPath):
//   - Scans the directory for files matching the Config's filename (without extension)
//   - Skips subdirectories and non-matching files
//   - Includes matching files with their full paths
//
// The method handles path resolution (using FindPath) and silently skips invalid paths,
// logging debug information for troubleshooting. The returned paths are ordered according
// to the original registration order of their parent directories.
//
// Example: For fileName "config" and path "/etc/app", it would match:
//
//	"/etc/app/config.json", "/etc/app/config.yaml", etc.
//
// Returns: A slice of absolute file paths ready for configuration loading
func (c *Config) GetConfigFiles() []string {
	paths := make([]string, 0)

	for path := range slices.Values(c.paths) {
		full := c.fullPath[path]
		path, err := FindPath("", path)
		if err != nil {
			continue
		}

		if full {
			if _, err := os.Stat(path); err == nil {
				paths = append(paths, path)
			}
			continue
		}

		dir, err := os.ReadDir(path)
		if err != nil {
			c.logger.Debug("Failed to read directory", "path", path, "error", err)
			continue
		}
		for entry := range slices.Values(dir) {
			name := entry.Name()
			if entry.IsDir() {
				c.logger.Debug("Skip directory", "path", path)
				continue
			}
			if basenameWithoutExt(name) == c.fileName {
				paths = append(paths, filepath.Join(path, name))
			}
		}
	}

	return paths
}

// SetFormat sets the default configuration format to be used when there isn't
// any encoder or decoder available for a specific format. The format string
// should specify the configuration format (e.g., "json", "yaml", "toml").
// This is a global convenience function that delegates to the default config instance.
func SetFormat(f string) { cfg.SetFormat(f) }

// SetFormat sets the default configuration format for this Config instance.
// The format will be used when no specific encoder/decoder is available for
// a requested format. Typical formats include "json", "yaml", "toml", etc.
func (c *Config) SetFormat(f string) {
	c.defaultFormat = f
}

// AddPath adds a file path to the list of paths that will be searched for
// configuration files. This is a global convenience function that delegates
// to the default config instance.
func AddPath(p string) { cfg.AddPath(p) }

// AddPath adds a file path to the Config instance's list of search paths.
// These paths will be used when looking for configuration files to load.
// Duplicate paths may be added.
func (c *Config) AddPath(p string) {
	c.paths = append(c.paths, p)
}

// AddFile adds a specific file path to be loaded as a configuration file.
// This is a global convenience function that delegates to the default config instance.
// The file will be marked for loading and added to the search paths.
func AddFile(p string) { cfg.AddFile(p) }

// AddFile adds a specific file path to the Config instance, marking it to be
// loaded as a configuration file. The file is added to both the fullPath map
// (to track specific files) and the general paths list (for search purposes).
// This allows for both explicit file loading and path-based searching.
func (c *Config) AddFile(p string) {
	c.fullPath[p] = true
	c.paths = append(c.paths, p)
}

func (c *Config) ReadConfig() {
	config := map[string]any{}
	paths := c.GetConfigFiles()
	for path := range slices.Values(paths) {
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
	if key == "." {
		m, ok := v.(map[string]any)
		if !ok {
			return errors.New("global config must be a map[string]any")
		}
		c.config = m
	}

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

// Keps returns top-level keys of config
func (c *Config) Keys() []string {
	if c.config == nil {
		return make([]string, 0)
	}
	keys := make([]string, 0, len(c.config))
	for k := range c.config {
		keys = append(keys, k)
	}
	return keys
}

// Settings returns the settings map
func (c *Config) Settings() map[string]any {
	return c.config
}

// Get returns the value for the key, or error if missing/invalid.
func GetE(key string) (any, error) { return cfg.GetE(key) }

// GetE returns the value for the key, or an error if missing/invalid.
func (c *Config) GetE(key string) (any, error) {
	if key == "." {
		return c.config, nil
	}
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

// GetValueE returns the reflect.Value for the key, or error if missing/invalid.
func GetValueE(key string) (reflect.Value, error) { return cfg.GetValueE(key) }

// GetValueE returns the reflect.Value for the key, or error if missing/invalid.
func (c *Config) GetValueE(key string) (reflect.Value, error) {
	v, err := c.GetE(key)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(v), nil
}

// GetValueE returns the reflect.Value for the key. Returns default value if missing/invalid.
func GetValue(key string) reflect.Value { return cfg.GetValue(key) }

// GetValueE returns the reflect.Value for the key. Returns default value if missing/invalid.
func (c *Config) GetValue(key string) reflect.Value {
	v, err := c.GetE(key)
	if err != nil {
		return reflect.Value{}
	}
	return reflect.ValueOf(v)
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

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
	"github.com/spf13/pflag"
)

// Must indicates that there Must not be any error; it panics if an error
// occurs.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// DecodeFunc decodes raw bytes into a generic map representation of a config
// file.
type DecodeFunc func([]byte) (map[string]any, error)

// EncodeFunc encodes a generic config map into raw bytes for storage or
// transmission.
type EncodeFunc func(map[string]any) ([]byte, error)

// UnmarshalFunc decodes raw bytes into a provided Go value
// (like json.Unmarshal).
type UnmarshalFunc func([]byte, any) error

// MarshalFunc encodes a Go value into raw bytes (like json.Marshal).
type MarshalFunc func(any) ([]byte, error)

// DecoderFromUnmarshal wraps a standard UnmarshalFunc (e.g., YAML, JSON) into
// a DecodeFunc that produces a map[string]any suitable for generic config
// handling.
func DecoderFromUnmarshal(unmarshall UnmarshalFunc) DecodeFunc {
	return func(b []byte) (map[string]any, error) {
		v := map[string]any{}
		err := unmarshall(b, &v)
		return v, err
	}
}

// EncoderFromMarshal wraps a standard MarshalFunc into an EncodeFunc that
// serializes a map[string]any into raw bytes.
func EncoderFromMarshal(marshall MarshalFunc) EncodeFunc {
	return func(m map[string]any) ([]byte, error) {
		return marshall(m)
	}
}

var cfg = New()

// Default returns default *Config
func Default() *Config {
	if cfg == nil {
		cfg = New()
	}
	return cfg
}

// Config represents an application configuration container. It holds
// configuration values loaded from files, environment variables, or other
// sources. The struct also manages metadata and encoding/decoding behavior for
// configuration data.
type Config struct {
	defaults map[string]any
	config   map[string]any

	pflagSet *pflag.FlagSet
	pflags   map[string]*pflag.Flag

	envPrefix string
	logger    *slog.Logger

	paths         []string
	fullPath      map[string]bool
	defaultFormat string
	fileName      string

	decoders map[string]DecodeFunc
	encoders map[string]EncodeFunc
}

// New creates Config instance.
func New() *Config {
	return &Config{
		logger:   slog.Default(),
		defaults: map[string]any{},
		config:   map[string]any{},
		fullPath: map[string]bool{},
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

// SetPflagSet adds *pflag.FlagSet
func (c *Config) SetPflagSet(fs *pflag.FlagSet) {
	c.pflagSet = fs
}

// AddPflag adds *pflag.FlagSet
func (c *Config) AddPflag(name string, f *pflag.Flag) {
	if c.pflags == nil {
		c.pflags = map[string]*pflag.Flag{}
	}
	if name == "" {
		name = f.Name
	}
	c.pflags[name] = f
}

// SetEnvPrefix sets the environment variable prefix for the configuration.
// All underscores in the provided string are removed before assignment.
//
// For example, calling SetEnvPrefix("APP_") will set the prefix to "APP".
func (c *Config) SetEnvPrefix(p string) {
	c.envPrefix = strings.TrimSuffix(p, "_")
}

// basenameWithoutExt return filename without extension
func basenameWithoutExt(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}

// GetConfigFiles returns all config file paths to be loaded by ReadConfig. It
// resolves registered files (AddFile) and directories (AddPath), matching the
// config filename across supported extensions. Missing or invalid paths are
// skipped with debug logs. Paths are returned in registration order.
//
// Example: fileName "config", path "/etc/app" → matches "/etc/app/config.json",
// "/etc/app/config.yaml", etc.
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

// SetFormat sets the default configuration format for this Config instance.
// The format will be used when no specific encoder/decoder is available for
// a requested format. Typical formats include "json", "yaml", "toml", etc.
func (c *Config) SetFormat(f string) {
	c.defaultFormat = f
}

// AddPath adds a file path to the Config instance's list of search paths.
// These paths will be used when looking for configuration files to load.
// Duplicate paths may be added.
func (c *Config) AddPath(p string) {
	c.paths = append(c.paths, p)
}

// AddFile adds a specific file path to the Config instance, marking it to be
// loaded as a configuration file. The file is added to both the fullPath map
// (to track specific files) and the general paths list (for search purposes).
// This allows for both explicit file loading and path-based searching.
func (c *Config) AddFile(p string) {
	c.fullPath[p] = true
	c.paths = append(c.paths, p)
}

// ReadConfig loads config files from GetConfigFiles(), following any "include"
// directives to merge additional files recursively. Later values override
// earlier ones.
//
// Example:
//
//	main.json:
//	  { "include": ["a.yaml"], "app": { "port": "8080" } }
//	a.yaml:
//	  app: { "port": 9000, "env": "prod" }
//
// Result:
//
//	app.port = "8080"   // overridden by main.json
//	app.env  = "prod"   // merged from a.yaml
func (c *Config) ReadConfig() error {
	config := map[string]any{}
	paths := c.GetConfigFiles()
	for path := range slices.Values(paths) {
		visited := map[string]bool{}
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
	if len(config) == 0 {
		return errors.New("No configuration found")
	}
	return nil
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

	base := map[string]any{}
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

// Set sets a value in the configuration under the specified key.
func (c *Config) Set(key string, v any) error {
	return c.setValue(&c.config, key, v)
}

// SetDefault sets a value in the configuration's default values under the
// specified key.
func (c *Config) SetDefault(key string, v any) error {
	return c.setValue(&c.defaults, key, v)
}

// setValue sets a value in the provided map for a specific key. Nested keys
// can be specified using dot notation (e.g., "database.host"). If the key is
// ".", the entire map is replaced with the provided value.
func (c *Config) setValue(in *map[string]any, key string, v any) error {
	if key == "." {
		vm, ok := v.(map[string]any)
		if !ok {
			return errors.New("global config must be a map[string]any")
		}
		*in = vm
	}

	parsed, err := KeySplit(key)
	if err != nil {
		return err
	}

	m := *in
	for i := range parsed.LastIndex() {
		part := parsed.Parts[i].String()
		if next, ok := m[part]; ok {
			if subMap, ok := next.(map[string]any); ok {
				m = subMap
			} else {
				// Overwrite non-map value with a new map
				newMap := map[string]any{}
				m[part] = newMap
				m = newMap
			}
		} else {
			newMap := map[string]any{}
			m[part] = newMap
			m = newMap
		}
	}

	m[parsed.Parts[parsed.LastIndex()].String()] = v
	return nil
}

// Keys returns top-level keys of config
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

// GetE returns the value for the key, or error if missing/invalid.
func (c *Config) GetE(key string) (any, error) {
	if c.pflags != nil {
		if flag, ok := c.pflags[key]; ok && flag.Changed {
			return flag.Value.String(), nil
		}
	}

	if c.pflagSet != nil && c.pflagSet.Parsed() && c.pflagSet.Changed(key) {
		return c.pflagSet.GetString(key)
	}

	parsed, err := KeySplit(key)
	if err != nil {
		return nil, err
	}

	env := parsed.EnvKey(c.envPrefix)
	if v, ok := os.LookupEnv(env); ok {
		return v, nil
	}
	c.logger.Debug("Couldn't find value in env", "env_name", env, "error", err)

	v, err := c.getValue(c.config, parsed)
	if err != nil {
		c.logger.Debug("Failed to find value", "key", key, "error", err)
		v, err := c.getValue(c.defaults, parsed)
		if err == nil {
			return v, nil
		}
		c.logger.Debug("Failed to find default value", "key", key, "error", err)
	}
	return v, err
}

// GetE returns the value for the key, or an error if missing/invalid.
func (c *Config) getValue(m map[string]any, key Key) (any, error) {
	if key.Raw == "." {
		return m, nil
	}

	var prefix strings.Builder

	for i := range key.LastIndex() {
		part := key.Parts[i].String()
		prefix.WriteString(part)

		next, ok := m[part]
		if !ok {
			return nil, fmt.Errorf("key not found: %s", prefix.String())
		}

		subMap, ok := next.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid type for key: %s (expected map)",
				prefix.String())
		}
		m = subMap

		prefix.WriteByte('.')
	}

	val, ok := m[key.Parts[key.LastIndex()].String()]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return val, nil
}

// GetValueE returns the reflect.Value for the key, or error if missing/invalid.
func (c *Config) GetValueE(key string) (reflect.Value, error) {
	v, err := c.GetE(key)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(v), nil
}

// GetIntE returns the int value for the key, or error if missing/invalid.
func (c *Config) GetIntE(key string) (int, error) {
	return getValueE(c, key, cast.ToIntE)
}

// GetInt64E returns the int64 value for the key, or error if missing/invalid.
func (c *Config) GetInt64E(key string) (int64, error) {
	return getValueE(c, key, cast.ToInt64E)
}

// GetUintE returns the uint value for the key, or error if missing/invalid.
func (c *Config) GetUintE(key string) (uint, error) {
	return getValueE(c, key, cast.ToUintE)
}

// GetUint64E returns the uint64 value for the key, or error if missing/invalid.
func (c *Config) GetUint64E(key string) (uint64, error) {
	return getValueE(c, key, cast.ToUint64E)
}

// GetStringE returns the string value for the key, or error if missing/invalid.
func (c *Config) GetStringE(key string) (string, error) {
	return getValueE(c, key, cast.ToStringE)
}

// GetBoolE returns the bool value for the key, or error if missing/invalid.
func (c *Config) GetBoolE(key string) (bool, error) {
	return getValueE(c, key, cast.ToBoolE)
}

// GetStringMapE returns the map[string]any value for the key, or error if
// missing/invalid.
func (c *Config) GetStringMapE(key string) (map[string]any, error) {
	return getValueE(c, key, cast.ToStringMapE)
}

// GetStringMapIntE returns the map[string]any value for the key, or error if
// missing/invalid.
func (c *Config) GetStringMapIntE(key string) (map[string]int, error) {
	return getValueE(c, key, cast.ToStringMapIntE)
}

// GetStringMapInt64E returns the map[string]int64 value for the key, or error if
// missing/invalid.
func (c *Config) GetStringMapInt64E(key string) (map[string]int64, error) {
	return getValueE(c, key, cast.ToStringMapInt64E)
}

// GetStringMapUintE returns the map[string]uint value for the key, or error if
// missing/invalid.
func (c *Config) GetStringMapUintE(key string) (map[string]uint, error) {
	return toStringMapAny(c, key, cast.ToUintE)
}

// GetStringMapUint64E returns the map[string]uint64 value for the key, or error if
// missing/invalid.
func (c *Config) GetStringMapUint64E(key string) (map[string]uint64, error) {
	return toStringMapAny(c, key, cast.ToUint64E)
}

// GetStringMapStringE returns the map[string]string value for the key, or error if
// missing/invalid.
func (c *Config) GetStringMapStringE(key string) (map[string]string, error) {
	return getValueE(c, key, cast.ToStringMapStringE)
}

// GetStringMapBoolE returns the map[string]bool value for the key, or error if
// missing/invalid.
func (c *Config) GetStringMapBoolE(key string) (map[string]bool, error) {
	return getValueE(c, key, cast.ToStringMapBoolE)
}

// GetStringMapStringSliceE returns the map[string][]string value for the key, or error if
// missing/invalid.
func (c *Config) GetStringMapStringSliceE(key string) (map[string][]string, error) {
	return getValueE(c, key, cast.ToStringMapStringSliceE)
}

func toStringMapAny[T any](c *Config, key string, conv func(any) (T, error)) (map[string]T, error) {
	var zero map[string]T
	v, err := c.GetE(key)
	if err != nil {
		return zero, err
	}
	m, err := cast.ToStringMapE(v)
	if err != nil {
		return zero, err
	}
	out := map[string]T{}
	for k, v := range m {
		converted, err := conv(v)
		if err != nil {
			return out, err
		}
		out[k] = converted
	}
	return out, nil
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

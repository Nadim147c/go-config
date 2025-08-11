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
)

// must indicates that there must not be any error; it panics if an error occurs.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// should indicates that any returned error will be ignored.
func should[T any](v T, _ error) T {
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

func (c *Config) Set(key string, v any) {
	key = strings.ToLower(key)
	nested := strings.Split(key, ".")

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
}

func (c *Config) SetEnvPrefix(p string) {
	c.envPrefix = strings.ReplaceAll(p, "_", "")
}

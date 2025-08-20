# Go Config Library

> [!CAUTION]
> This is highly experimental 🧪 and expect breaking changes

[![Go Reference](https://pkg.go.dev/badge/github.com/Nadim147c/go-config.svg)](https://pkg.go.dev/github.com/Nadim147c/go-config)

A comprehensive, feature-rich configuration management library for Go applications that supports
multiple file formats, environment variables, command-line flags, and advanced validation.

## Overview

This library provides a unified interface for managing application configuration from multiple
sources with intelligent merging, validation, and binding capabilities. It's designed to handle
complex configuration scenarios while maintaining simplicity for basic use cases.

## Key Features

- **Multi-format support**: JSON, YAML, TOML, and more via extensible codecs
- **Multiple sources**: Files, environment variables, command-line flags
- **Intelligent merging**: Deep merging of configuration with override support
- **Validation**: Comprehensive validation with built-in and custom rules
- **Struct binding**: Automatic mapping to Go structs with tags
- **XDG compliance**: Proper support for XDG Base Directory Specification
- **Recursive includes**: Configuration files can include other files
- **Type-safe access**: Extensive getter methods with error handling variants

## Installation

```bash
go get github.com/Nadim147c/go-config
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/Nadim147c/go-config"
)

type AppConfig struct {
    Port     int    `config:"port" check:"required,min=1024,max=65535"`
    Env      string `config:"env" check:"default=production"`
    Database struct {
        Host     string `config:"host" check:"required"`
        Port     int    `config:"port" check:"default=5432"`
        SSL      bool   `config:"ssl" check:"default=true"`
    } `config:"database"`
}

func main() {
    cfg := config.New()

    // Add configuration paths
    cfg.AddPath("/etc/myapp")
    cfg.AddPath("$XDG_CONFIG_HOME/myapp")
    cfg.AddPath("./config")

    // Load configuration
    if err := cfg.ReadConfig(); err != nil {
        panic(err)
    }

    // Bind to struct
    var appConfig AppConfig
    if err := cfg.Bind("app", &appConfig); err != nil {
        panic(err)
    }

    fmt.Printf("Server running on port %d in %s environment\n",
        appConfig.Port, appConfig.Env)
}
```

## Configuration Sources

### File-based Configuration

```go
cfg := config.New()

// Add specific files
cfg.AddFile("/etc/app/config.yaml")
cfg.AddFile("~/.config/app/config.json")

// Add search directories
cfg.AddPath("/etc/app/conf.d")
cfg.AddPath("$XDG_CONFIG_HOME/app")
cfg.AddPath("./config")

// Load all configuration
err := cfg.ReadConfig()
```

Supported file extensions: `.json`, `.yaml`, `.yml`, `.toml`, `.ini`

### Environment Variables

```go
cfg := config.New()
cfg.SetEnvPrefix("APP_") // Sets prefix for environment variables

// Environment variable APP_DATABASE__HOST maps to database.host
```

### Command-line Flags

```go
import "github.com/spf13/pflag"

func main() {
    flags := pflag.NewFlagSet("app", pflag.ContinueOnError)
    flags.Int("port", 8080, "server port")
    flags.String("env", "development", "environment")

    cfg := config.New()
    cfg.SetPflagSet(flags)

    // Now --port and --env flags will be available in configuration
}
```

## Configuration Structure

### File Format Examples

**JSON:**

```json
{
  "app": {
    "port": 8080,
    "env": "production",
    "include": ["database.json", "cache.yaml"]
  }
}
```

**YAML:**

```yaml
app:
  port: 8080
  env: production
  include:
    - database.json
    - cache.yaml

database:
  host: localhost
  port: 5432
  ssl: true
```

### Recursive Includes

Configuration files can include other files using the `include` key:

```json
{
  "include": ["base-config.yaml", "secrets.json"],
  "app": {
    "port": 8080
  }
}
```

Files are loaded in order, with later files overriding earlier ones.

## Accessing Configuration

### Basic Access Methods

```go
// Get values with different error handling approaches
port := cfg.GetInt("app.port")                    // Returns 0 if missing
port := cfg.GetIntMust("app.port")                // Panics if missing
port, err := cfg.GetIntE("app.port")              // Returns error if missing

// Type-specific getters
str := cfg.GetString("app.name")
num := cfg.GetInt("app.workers")
flag := cfg.GetBool("app.debug")
```

### Map Access

```go
// Get maps of various types
settings := cfg.GetStringMap("app.settings")
users := cfg.GetStringMapString("app.users")
counts := cfg.GetStringMapInt("app.counts")
flags := cfg.GetStringMapBool("app.features")
```

### Advanced Access

```go
// Get reflect.Value for advanced manipulation
value := cfg.GetReflection("app.complexSetting")

// Check if key exists
if _, err := cfg.GetE("app.optional"); err != nil {
    // Key doesn't exist
}

// Get all top-level keys
keys := cfg.Keys()
```

## Struct Binding

### Basic Binding

```go
type Config struct {
    Port    int    `config:"port"`
    Env     string `config:"env"`
    Debug   bool   `config:"debug"`
    Timeout int    `config:"timeout"`
}

var appConfig Config
err := cfg.Bind("app", &appConfig)
```

### Nested Structures

```go
type DatabaseConfig struct {
    Host     string `config:"host"`
    Port     int    `config:"port"`
    Username string `config:"username"`
    Password string `config:"password"`
}

type AppConfig struct {
    Name     string        `config:"name"`
    Port     int           `config:"port"`
    Database DatabaseConfig `config:"database"`
}

var config AppConfig
err := cfg.Bind("", &config) // Bind to root
```

### Validation with Tags

```go
type UserConfig struct {
    Email    string `config:"email" check:"required,email"`
    Age      int    `config:"age" check:"min=18,max=120"`
    Password string `config:"password" check:"required,min=8"`
    APIKey   string `config:"api_key" check:"uuid"`
    Theme    string `config:"theme" check:"default=light"`
}
```

Available validation rules:

- `required` - Field must be non-zero
- `default=value` - Set default value if empty
- `min`, `max` - Numeric/string length bounds
- `email` - Valid email format
- `uuid` - Valid UUID format
- `alpha` - Alphabetic characters only
- `alphanumeric` - Alphanumeric characters only
- `number` - Digits only
- `base64` - Valid base64 encoding
- `match=regex` - Custom regex pattern

## Advanced Features

### Deep Merging

```go
// Manual deep merging of maps
result := config.DeepMerge(
    map[string]any{"a": 1, "b": map[string]any{"x": 1}},
    map[string]any{"b": map[string]any{"y": 2}, "c": 3}
)
// Result: {"a": 1, "b": {"x": 1, "y": 2}, "c": 3}
```

### Path Resolution

```go
// Resolve paths with environment variables
path, err := config.FindPath("/base/path", "$HOME/config/app.yaml")
// Expands to: /home/user/config/app.yaml

// Supported variables:
// - $HOME, ~
// - $XDG_CONFIG_HOME, $XDG_CACHE_HOME, $XDG_DATA_HOME
// - $TMPDIR, $PWD
// - XDG user directories: Desktop, Documents, Downloads, etc.
```

## Error Handling

The library provides multiple error handling patterns:

```go
// Method variants for different error handling needs
val := cfg.GetString("key")          // Returns zero value on error
val := cfg.GetStringMust("key")      // Panics on error
val, err := cfg.GetStringE("key")    // Returns error

// Helper functions for concise error handling
val := config.Must(cfg.GetStringE("key"))     // Panics on error
val := config.Should(cfg.GetStringE("key"))   // Ignores error, returns zero value
```

## Best Practices

### Configuration Organization

1. **Leverage includes** for environment-specific configurations
2. **Use XDG directories** for proper filesystem organization
3. **Validate early** with comprehensive validation rules
4. **Provide sensible defaults** for all configuration options

### Performance

- Load configuration once at application startup
- Use struct binding for frequently accessed values
- Cache computed configuration values if needed

## Examples

### Web Server Configuration

```go
type ServerConfig struct {
    Addr         string        `config:"addr" check:"default=:8080"`
    ReadTimeout  time.Duration `config:"read_timeout" check:"default=30s"`
    WriteTimeout time.Duration `config:"write_timeout" check:"default=30s"`
    TLS          struct {
        Enabled bool   `config:"enabled" check:"default=false"`
        Cert    string `config:"cert"`
        Key     string `config:"key"`
    } `config:"tls"`
}

func LoadConfig() (*ServerConfig, error) {
    cfg := config.New()
    cfg.AddPath("/etc/myserver")
    cfg.AddPath("$XDG_CONFIG_HOME/myserver")

    if err := cfg.ReadConfig(); err != nil {
        return nil, err
    }

    var serverConfig ServerConfig
    if err := cfg.Bind("server", &serverConfig); err != nil {
        return nil, err
    }

    return &serverConfig, nil
}
```

### Database Configuration with Validation

```go
type DBConfig struct {
    Host     string `config:"host" check:"required"`
    Port     int    `config:"port" check:"default=5432,min=1,max=65535"`
    Name     string `config:"name" check:"required"`
    User     string `config:"user" check:"required"`
    Password string `config:"password" check:"required"`
    SSLMode  string `config:"ssl_mode" check:"default=require"`
    Pool     struct {
        MaxConns    int           `config:"max_conns" check:"default=10,min=1"`
        MaxIdleTime time.Duration `config:"max_idle_time" check:"default=5m"`
    } `config:"pool"`
}
```

## API Reference

See the generated documentation at the top of this file for complete API details. The library
provides extensive methods for configuration access, manipulation, and validation.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the [GNU-LGPL-3.0 License](./LICENSE).

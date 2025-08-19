// This file is auto generated; DO NOT EDIT IT.
package config

import (
	"reflect"

	"github.com/spf13/pflag"
)

// Bind maps the configuration values from the Config instance into a structured
// Go type. It uses struct tags to determine how to bind the data and can also
// perform validation.
//
// Example:
//
//	type MyConfig struct {
//	    Port int    `config:"port" validate:"min=1000,max=9999"`
//	    Key  string `config:"key"`
//	}
//
//	var cfg MyConfig
//	c := config.New()
//	c.ReadConfig()
//	c.Bind(&cfg)
//
// Parameters:
//   - v: A pointer to a struct where the configuration values will be
//     populated.
//
// Returns:
//   - error: If the input is not a non-nil pointer to a struct, or if binding
//     fails.
func Bind(v any) error { return Default().Bind(v) }

// SetPflagSet adds *pflag.FlagSet
func SetPflagSet(fs *pflag.FlagSet) { Default().SetPflagSet(fs) }

// AddPflag adds *pflag.FlagSet
func AddPflag(name string, f *pflag.Flag) { Default().AddPflag(name, f) }

// SetEnvPrefix sets the environment variable prefix for the configuration.
// All underscores in the provided string are removed before assignment.
//
// For example, calling SetEnvPrefix("APP_") will set the prefix to "APP".
func SetEnvPrefix(p string) { Default().SetEnvPrefix(p) }

// GetConfigFiles returns all config file paths to be loaded by ReadConfig. It
// resolves registered files (AddFile) and directories (AddPath), matching the
// config filename across supported extensions. Missing or invalid paths are
// skipped with debug logs. Paths are returned in registration order.
//
// Example: fileName "config", path "/etc/app" → matches "/etc/app/config.json",
// "/etc/app/config.yaml", etc.
func GetConfigFiles() []string { return Default().GetConfigFiles() }

// SetFormat sets the default configuration format for this Config instance.
// The format will be used when no specific encoder/decoder is available for
// a requested format. Typical formats include "json", "yaml", "toml", etc.
func SetFormat(f string) { Default().SetFormat(f) }

// AddPath adds a file path to the Config instance's list of search paths.
// These paths will be used when looking for configuration files to load.
// Duplicate paths may be added.
func AddPath(p string) { Default().AddPath(p) }

// AddFile adds a specific file path to the Config instance, marking it to be
// loaded as a configuration file. The file is added to both the fullPath map
// (to track specific files) and the general paths list (for search purposes).
// This allows for both explicit file loading and path-based searching.
func AddFile(p string) { Default().AddFile(p) }

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
func ReadConfig() error { return Default().ReadConfig() }

// Set sets a value in the configuration under the specified key.
func Set(key string, v any) error { return Default().Set(key, v) }

// SetDefault sets a value in the configuration's default values under the
// specified key.
func SetDefault(key string, v any) error { return Default().SetDefault(key, v) }

// Keys returns top-level keys of config
func Keys() []string { return Default().Keys() }

// Settings returns the settings map
func Settings() map[string]any { return Default().Settings() }

// GetE returns the  value for the key, or error if missing/invalid.
func GetE(key string) (any, error) { return Default().GetE(key) }

// GetMust returns the  value for the key. Panics if missing/invalid.
func GetMust(key string) any { return Default().GetMust(key) }

// GetMust returns the  value for the key. Panics if missing/invalid.
func (c *Config) GetMust(key string) any {
	return Must(c.GetE(key))
}

// Get returns the  value for the key. Returns default if missing/invalid.
func Get(key string) any { return Default().Get(key) }

// Get returns the  value for the key. Returns default if missing/invalid.
func (c *Config) Get(key string) any {
	return Should(c.GetE(key))
}

// GetValueE returns the value value for the key, or error if missing/invalid.
func GetValueE(key string) (reflect.Value, error) { return Default().GetValueE(key) }

// GetValueMust returns the value value for the key. Panics if missing/invalid.
func GetValueMust(key string) reflect.Value { return Default().GetValueMust(key) }

// GetValueMust returns the value value for the key. Panics if missing/invalid.
func (c *Config) GetValueMust(key string) reflect.Value {
	return Must(c.GetValueE(key))
}

// GetValue returns the value value for the key. Returns default if missing/invalid.
func GetValue(key string) reflect.Value { return Default().GetValue(key) }

// GetValue returns the value value for the key. Returns default if missing/invalid.
func (c *Config) GetValue(key string) reflect.Value {
	return Should(c.GetValueE(key))
}

// GetIntE returns the int value for the key, or error if missing/invalid.
func GetIntE(key string) (int, error) { return Default().GetIntE(key) }

// GetIntMust returns the int value for the key. Panics if missing/invalid.
func GetIntMust(key string) int { return Default().GetIntMust(key) }

// GetIntMust returns the int value for the key. Panics if missing/invalid.
func (c *Config) GetIntMust(key string) int {
	return Must(c.GetIntE(key))
}

// GetInt returns the int value for the key. Returns default if missing/invalid.
func GetInt(key string) int { return Default().GetInt(key) }

// GetInt returns the int value for the key. Returns default if missing/invalid.
func (c *Config) GetInt(key string) int {
	return Should(c.GetIntE(key))
}

// GetInt64E returns the int64 value for the key, or error if missing/invalid.
func GetInt64E(key string) (int64, error) { return Default().GetInt64E(key) }

// GetInt64Must returns the int64 value for the key. Panics if missing/invalid.
func GetInt64Must(key string) int64 { return Default().GetInt64Must(key) }

// GetInt64Must returns the int64 value for the key. Panics if missing/invalid.
func (c *Config) GetInt64Must(key string) int64 {
	return Must(c.GetInt64E(key))
}

// GetInt64 returns the int64 value for the key. Returns default if missing/invalid.
func GetInt64(key string) int64 { return Default().GetInt64(key) }

// GetInt64 returns the int64 value for the key. Returns default if missing/invalid.
func (c *Config) GetInt64(key string) int64 {
	return Should(c.GetInt64E(key))
}

// GetUintE returns the uint value for the key, or error if missing/invalid.
func GetUintE(key string) (uint, error) { return Default().GetUintE(key) }

// GetUintMust returns the uint value for the key. Panics if missing/invalid.
func GetUintMust(key string) uint { return Default().GetUintMust(key) }

// GetUintMust returns the uint value for the key. Panics if missing/invalid.
func (c *Config) GetUintMust(key string) uint {
	return Must(c.GetUintE(key))
}

// GetUint returns the uint value for the key. Returns default if missing/invalid.
func GetUint(key string) uint { return Default().GetUint(key) }

// GetUint returns the uint value for the key. Returns default if missing/invalid.
func (c *Config) GetUint(key string) uint {
	return Should(c.GetUintE(key))
}

// GetUint64E returns the uint64 value for the key, or error if missing/invalid.
func GetUint64E(key string) (uint64, error) { return Default().GetUint64E(key) }

// GetUint64Must returns the uint64 value for the key. Panics if missing/invalid.
func GetUint64Must(key string) uint64 { return Default().GetUint64Must(key) }

// GetUint64Must returns the uint64 value for the key. Panics if missing/invalid.
func (c *Config) GetUint64Must(key string) uint64 {
	return Must(c.GetUint64E(key))
}

// GetUint64 returns the uint64 value for the key. Returns default if missing/invalid.
func GetUint64(key string) uint64 { return Default().GetUint64(key) }

// GetUint64 returns the uint64 value for the key. Returns default if missing/invalid.
func (c *Config) GetUint64(key string) uint64 {
	return Should(c.GetUint64E(key))
}

// GetStringE returns the string value for the key, or error if missing/invalid.
func GetStringE(key string) (string, error) { return Default().GetStringE(key) }

// GetStringMust returns the string value for the key. Panics if missing/invalid.
func GetStringMust(key string) string { return Default().GetStringMust(key) }

// GetStringMust returns the string value for the key. Panics if missing/invalid.
func (c *Config) GetStringMust(key string) string {
	return Must(c.GetStringE(key))
}

// GetString returns the string value for the key. Returns default if missing/invalid.
func GetString(key string) string { return Default().GetString(key) }

// GetString returns the string value for the key. Returns default if missing/invalid.
func (c *Config) GetString(key string) string {
	return Should(c.GetStringE(key))
}

// GetBoolE returns the bool value for the key, or error if missing/invalid.
func GetBoolE(key string) (bool, error) { return Default().GetBoolE(key) }

// GetBoolMust returns the bool value for the key. Panics if missing/invalid.
func GetBoolMust(key string) bool { return Default().GetBoolMust(key) }

// GetBoolMust returns the bool value for the key. Panics if missing/invalid.
func (c *Config) GetBoolMust(key string) bool {
	return Must(c.GetBoolE(key))
}

// GetBool returns the bool value for the key. Returns default if missing/invalid.
func GetBool(key string) bool { return Default().GetBool(key) }

// GetBool returns the bool value for the key. Returns default if missing/invalid.
func (c *Config) GetBool(key string) bool {
	return Should(c.GetBoolE(key))
}

// GetStringMapE returns the stringmap value for the key, or error if missing/invalid.
func GetStringMapE(key string) (map[string]any, error) { return Default().GetStringMapE(key) }

// GetStringMapMust returns the stringmap value for the key. Panics if missing/invalid.
func GetStringMapMust(key string) map[string]any { return Default().GetStringMapMust(key) }

// GetStringMapMust returns the stringmap value for the key. Panics if missing/invalid.
func (c *Config) GetStringMapMust(key string) map[string]any {
	return Must(c.GetStringMapE(key))
}

// GetStringMap returns the stringmap value for the key. Returns default if missing/invalid.
func GetStringMap(key string) map[string]any { return Default().GetStringMap(key) }

// GetStringMap returns the stringmap value for the key. Returns default if missing/invalid.
func (c *Config) GetStringMap(key string) map[string]any {
	return Should(c.GetStringMapE(key))
}

// GetStringMapIntE returns the stringmapint value for the key, or error if missing/invalid.
func GetStringMapIntE(key string) (map[string]int, error) { return Default().GetStringMapIntE(key) }

// GetStringMapIntMust returns the stringmapint value for the key. Panics if missing/invalid.
func GetStringMapIntMust(key string) map[string]int { return Default().GetStringMapIntMust(key) }

// GetStringMapIntMust returns the stringmapint value for the key. Panics if missing/invalid.
func (c *Config) GetStringMapIntMust(key string) map[string]int {
	return Must(c.GetStringMapIntE(key))
}

// GetStringMapInt returns the stringmapint value for the key. Returns default if missing/invalid.
func GetStringMapInt(key string) map[string]int { return Default().GetStringMapInt(key) }

// GetStringMapInt returns the stringmapint value for the key. Returns default if missing/invalid.
func (c *Config) GetStringMapInt(key string) map[string]int {
	return Should(c.GetStringMapIntE(key))
}

// GetStringMapInt64E returns the stringmapint64 value for the key, or error if missing/invalid.
func GetStringMapInt64E(key string) (map[string]int64, error) {
	return Default().GetStringMapInt64E(key)
}

// GetStringMapInt64Must returns the stringmapint64 value for the key. Panics if missing/invalid.
func GetStringMapInt64Must(key string) map[string]int64 { return Default().GetStringMapInt64Must(key) }

// GetStringMapInt64Must returns the stringmapint64 value for the key. Panics if missing/invalid.
func (c *Config) GetStringMapInt64Must(key string) map[string]int64 {
	return Must(c.GetStringMapInt64E(key))
}

// GetStringMapInt64 returns the stringmapint64 value for the key. Returns default if missing/invalid.
func GetStringMapInt64(key string) map[string]int64 { return Default().GetStringMapInt64(key) }

// GetStringMapInt64 returns the stringmapint64 value for the key. Returns default if missing/invalid.
func (c *Config) GetStringMapInt64(key string) map[string]int64 {
	return Should(c.GetStringMapInt64E(key))
}

// GetStringMapUintE returns the stringmapuint value for the key, or error if missing/invalid.
func GetStringMapUintE(key string) (map[string]uint, error) { return Default().GetStringMapUintE(key) }

// GetStringMapUintMust returns the stringmapuint value for the key. Panics if missing/invalid.
func GetStringMapUintMust(key string) map[string]uint { return Default().GetStringMapUintMust(key) }

// GetStringMapUintMust returns the stringmapuint value for the key. Panics if missing/invalid.
func (c *Config) GetStringMapUintMust(key string) map[string]uint {
	return Must(c.GetStringMapUintE(key))
}

// GetStringMapUint returns the stringmapuint value for the key. Returns default if missing/invalid.
func GetStringMapUint(key string) map[string]uint { return Default().GetStringMapUint(key) }

// GetStringMapUint returns the stringmapuint value for the key. Returns default if missing/invalid.
func (c *Config) GetStringMapUint(key string) map[string]uint {
	return Should(c.GetStringMapUintE(key))
}

// GetStringMapUint64E returns the stringmapuint64 value for the key, or error if missing/invalid.
func GetStringMapUint64E(key string) (map[string]uint64, error) {
	return Default().GetStringMapUint64E(key)
}

// GetStringMapUint64Must returns the stringmapuint64 value for the key. Panics if missing/invalid.
func GetStringMapUint64Must(key string) map[string]uint64 {
	return Default().GetStringMapUint64Must(key)
}

// GetStringMapUint64Must returns the stringmapuint64 value for the key. Panics if missing/invalid.
func (c *Config) GetStringMapUint64Must(key string) map[string]uint64 {
	return Must(c.GetStringMapUint64E(key))
}

// GetStringMapUint64 returns the stringmapuint64 value for the key. Returns default if missing/invalid.
func GetStringMapUint64(key string) map[string]uint64 { return Default().GetStringMapUint64(key) }

// GetStringMapUint64 returns the stringmapuint64 value for the key. Returns default if missing/invalid.
func (c *Config) GetStringMapUint64(key string) map[string]uint64 {
	return Should(c.GetStringMapUint64E(key))
}

// GetStringMapStringE returns the stringmapstring value for the key, or error if missing/invalid.
func GetStringMapStringE(key string) (map[string]string, error) {
	return Default().GetStringMapStringE(key)
}

// GetStringMapStringMust returns the stringmapstring value for the key. Panics if missing/invalid.
func GetStringMapStringMust(key string) map[string]string {
	return Default().GetStringMapStringMust(key)
}

// GetStringMapStringMust returns the stringmapstring value for the key. Panics if missing/invalid.
func (c *Config) GetStringMapStringMust(key string) map[string]string {
	return Must(c.GetStringMapStringE(key))
}

// GetStringMapString returns the stringmapstring value for the key. Returns default if missing/invalid.
func GetStringMapString(key string) map[string]string { return Default().GetStringMapString(key) }

// GetStringMapString returns the stringmapstring value for the key. Returns default if missing/invalid.
func (c *Config) GetStringMapString(key string) map[string]string {
	return Should(c.GetStringMapStringE(key))
}

// GetStringMapBoolE returns the stringmapbool value for the key, or error if missing/invalid.
func GetStringMapBoolE(key string) (map[string]bool, error) { return Default().GetStringMapBoolE(key) }

// GetStringMapBoolMust returns the stringmapbool value for the key. Panics if missing/invalid.
func GetStringMapBoolMust(key string) map[string]bool { return Default().GetStringMapBoolMust(key) }

// GetStringMapBoolMust returns the stringmapbool value for the key. Panics if missing/invalid.
func (c *Config) GetStringMapBoolMust(key string) map[string]bool {
	return Must(c.GetStringMapBoolE(key))
}

// GetStringMapBool returns the stringmapbool value for the key. Returns default if missing/invalid.
func GetStringMapBool(key string) map[string]bool { return Default().GetStringMapBool(key) }

// GetStringMapBool returns the stringmapbool value for the key. Returns default if missing/invalid.
func (c *Config) GetStringMapBool(key string) map[string]bool {
	return Should(c.GetStringMapBoolE(key))
}

// GetStringMapStringSliceE returns the stringmapstringslice value for the key, or error if missing/invalid.
func GetStringMapStringSliceE(key string) (map[string][]string, error) {
	return Default().GetStringMapStringSliceE(key)
}

// GetStringMapStringSliceMust returns the stringmapstringslice value for the key. Panics if missing/invalid.
func GetStringMapStringSliceMust(key string) map[string][]string {
	return Default().GetStringMapStringSliceMust(key)
}

// GetStringMapStringSliceMust returns the stringmapstringslice value for the key. Panics if missing/invalid.
func (c *Config) GetStringMapStringSliceMust(key string) map[string][]string {
	return Must(c.GetStringMapStringSliceE(key))
}

// GetStringMapStringSlice returns the stringmapstringslice value for the key. Returns default if missing/invalid.
func GetStringMapStringSlice(key string) map[string][]string {
	return Default().GetStringMapStringSlice(key)
}

// GetStringMapStringSlice returns the stringmapstringslice value for the key. Returns default if missing/invalid.
func (c *Config) GetStringMapStringSlice(key string) map[string][]string {
	return Should(c.GetStringMapStringSliceE(key))
}

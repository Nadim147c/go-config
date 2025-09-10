package config

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// Bind maps configuration values from the Config instance into a structured
// Go type. It uses struct tags to determine how to bind the data and can also
// perform validation.
//
// Parameters:
//   - prefix: The prefix to prepend to all configuration keys
//   - v: A pointer to a struct where the configuration values will be populated
//
// Returns:
//   - error: If the input is not a non-nil pointer, or if binding fails
func (c *Config) Bind(prefix string, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("input type must be a non-nil pointer")
	}

	if prefix != "" {
		prefix = strings.Trim(prefix, ".")
	}

	return c.bindValue(rv.Elem(), prefix)
}

func (c *Config) bindValue(rv reflect.Value, key string) error {
	// Dereference pointers
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	if rv.CanInterface() {
		if text, ok := rv.Interface().(encoding.TextUnmarshaler); ok {
			return text.UnmarshalText([]byte(c.GetString(key)))
		}
	}

	switch rv.Kind() {
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			_, err := c.bindPrimitive(rv, key)
			if err != nil {
				return err
			}
			// For time.Time, we don't have a struct field to validate
			return nil
		}
		return c.bindStruct(rv, key)
	case reflect.Slice, reflect.Array:
		_, err := c.bindSliceOrArray(rv, key)
		return err
	case reflect.Map:
		_, err := c.bindMap(rv, key)
		return err
	default:
		_, err := c.bindPrimitive(rv, key)
		if err != nil {
			return err
		}
		return nil
	}
}

func (c *Config) bindStruct(rv reflect.Value, prefix string) error {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		field := rv.Field(i)
		cfgTag := strings.TrimSpace(sf.Tag.Get("config"))
		ignoreTag := strings.TrimSpace(sf.Tag.Get("config")) == "-"

		if ignoreTag {
			continue
		}

		// Build key path
		key := cfgTag
		if key == "" {
			key = sf.Name
		}
		key = strings.Trim(key, ".")
		if prefix != "" {
			if key != "" {
				key = prefix + "." + key
			} else {
				key = prefix
			}
		}

		// Handle embedded structs
		if sf.Anonymous {
			if err := c.bindValue(field, prefix); err != nil {
				return err
			}
			continue
		}

		// Track if the field was changed
		var changed bool
		var err error

		// Handle different field types
		switch field.Kind() {
		case reflect.Struct:
			if field.Type() == reflect.TypeOf(time.Time{}) {
				changed, err = c.bindPrimitive(field, key)
			} else {
				if c.Changed(key) {
					err = c.bindStruct(field, key)
					changed = true
				} else {
					changed = false
				}
			}
		case reflect.Slice, reflect.Array:
			changed, err = c.bindSliceOrArray(field, key)
		case reflect.Map:
			changed, err = c.bindMap(field, key)
		default:
			changed, err = c.bindPrimitive(field, key)
		}

		if err != nil {
			if _, ok := err.(KeyError); !ok {
				return err
			}
			changed = false
		}

		// Validate the field with the correct changed status
		if err := Validate(sf, field, changed); err != nil {
			return fmt.Errorf("%s: %v", key, err)
		}
	}
	return nil
}

func (c *Config) bindSliceOrArray(rv reflect.Value, key string) (bool, error) {
	configVal, err := c.GetReflectionE(key)
	if err != nil {
		return false, err
	}

	if configVal.Kind() != reflect.Slice && configVal.Kind() != reflect.Array {
		return false, fmt.Errorf("config value for %s is not a slice or array", key)
	}

	length := configVal.Len()
	elemType := rv.Type().Elem()

	if rv.Kind() == reflect.Array && rv.Len() != length {
		return false, fmt.Errorf("array size mismatch for %s: expected %d, got %d",
			key, rv.Len(), length)
	}

	var container reflect.Value
	if rv.Kind() == reflect.Slice {
		container = reflect.MakeSlice(rv.Type(), length, length)
	} else {
		container = rv
	}

	for i := range length {
		elemVal := configVal.Index(i)
		if elemVal.Kind() == reflect.Interface {
			elemVal = reflect.ValueOf(elemVal.Interface())
		}

		elem := container.Index(i)
		elemKey := fmt.Sprintf("%s.%d", key, i)

		if elem.Kind() == reflect.Pointer && elem.IsNil() {
			elem.Set(reflect.New(elem.Type().Elem()))
		}

		if elemVal.IsValid() && (elem.Kind() == reflect.Struct ||
			elem.Kind() == reflect.Slice || elem.Kind() == reflect.Array ||
			elem.Kind() == reflect.Map) {
			if err := c.bindValue(elem, elemKey); err != nil {
				return true, err
			}
		} else {
			converted, err := c.convertValue(elemVal.Interface(), elemType)
			if err != nil {
				return true, fmt.Errorf("%s[%d]: %v", key, i, err)
			}
			elem.Set(reflect.ValueOf(converted))
		}
	}

	if rv.Kind() == reflect.Slice {
		rv.Set(container)
	}

	return true, nil // Changed because we set the value
}

func (c *Config) bindMap(rv reflect.Value, key string) (bool, error) {
	configVal, err := c.GetReflectionE(key)
	if err != nil {
		return false, err
	}

	if configVal.Kind() != reflect.Map {
		return false, fmt.Errorf("config value for %s is not a map", key)
	}

	mapType := rv.Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()
	newMap := reflect.MakeMap(mapType)

	for _, k := range configVal.MapKeys() {
		v := configVal.MapIndex(k)
		if v.Kind() == reflect.Interface {
			v = reflect.ValueOf(v.Interface())
		}

		keyVal, err := c.convertValue(k.Interface(), keyType)
		if err != nil {
			return true, fmt.Errorf("%s: key conversion error: %v", key, err)
		}

		valueVal := reflect.New(valueType).Elem()
		elemKey := fmt.Sprintf("%s.%s", key, cast.ToString(k.Interface()))

		if valueVal.Kind() == reflect.Pointer && valueVal.IsNil() {
			valueVal.Set(reflect.New(valueType.Elem()))
		}

		if err := c.bindValue(valueVal, elemKey); err != nil {
			converted, err := c.convertValue(v.Interface(), valueType)
			if err != nil {
				return true, fmt.Errorf("%s[%v]: %v", key, k.Interface(), err)
			}
			valueVal = reflect.ValueOf(converted)
		}

		newMap.SetMapIndex(reflect.ValueOf(keyVal), valueVal)
	}

	rv.Set(newMap)
	return true, nil // Changed because we set the value
}

func (c *Config) bindPrimitive(rv reflect.Value, key string) (bool, error) {
	got, err := c.GetReflectionE(key)
	if err != nil {
		return false, err
	}

	converted, err := c.convertValue(got.Interface(), rv.Type())
	if err != nil {
		return false, fmt.Errorf("%s: cannot convert %v to %v: %v",
			key, got.Type(), rv.Type(), err)
	}

	if rv.CanSet() {
		cv := reflect.ValueOf(converted)

		if cv.Type().AssignableTo(rv.Type()) {
			rv.Set(cv)
		} else if cv.Type().ConvertibleTo(rv.Type()) {
			rv.Set(cv.Convert(rv.Type()))
		} else {
			return false, fmt.Errorf("%s: %v is not assignable to %v", key, got.Type(), rv.Type())
		}
	}

	return true, nil // Changed because we set the value
}

// convertValue function remains the same as in the previous implementation
func (c *Config) convertValue(in any, targetType reflect.Type) (any, error) {
	if targetType == reflect.TypeOf(time.Second) {
		return cast.ToDurationE(in)
	}
	switch targetType.Kind() {
	case reflect.String:
		return cast.ToStringE(in)
	case reflect.Int:
		return cast.ToIntE(in)
	case reflect.Int8:
		return cast.ToInt8E(in)
	case reflect.Int16:
		return cast.ToInt16E(in)
	case reflect.Int32:
		return cast.ToInt32E(in)
	case reflect.Int64:
		return cast.ToInt64E(in)
	case reflect.Uint:
		return cast.ToUintE(in)
	case reflect.Uint8:
		return cast.ToUint8E(in)
	case reflect.Uint16:
		return cast.ToUint16E(in)
	case reflect.Uint32:
		return cast.ToUint32E(in)
	case reflect.Uint64:
		return cast.ToUint64E(in)
	case reflect.Float32:
		return cast.ToFloat32E(in)
	case reflect.Float64:
		return cast.ToFloat64E(in)
	case reflect.Bool:
		return cast.ToBoolE(in)
	case reflect.Slice:
		return c.convertSlice(in, targetType)
	default:
		val := reflect.ValueOf(in)
		if val.Type().AssignableTo(targetType) {
			return in, nil
		}
		return nil, fmt.Errorf("unsupported type: %v", targetType)
	}
}

// convertSlice function remains the same as in the previous implementation
func (c *Config) convertSlice(in any, targetType reflect.Type) (any, error) {
	elemType := targetType.Elem()

	switch elemType.Kind() {
	case reflect.String:
		return cast.ToStringSliceE(in)
	case reflect.Int:
		return cast.ToIntSliceE(in)
	case reflect.Int64:
		return cast.ToInt64SliceE(in)
	case reflect.Uint:
		return cast.ToUintSliceE(in)
	case reflect.Uint64:
		return cast.ToUint64SliceE(in)
	case reflect.Float64:
		return cast.ToFloat64SliceE(in)
	case reflect.Bool:
		return cast.ToBoolSliceE(in)
	default:
		inSlice := reflect.ValueOf(in)
		if inSlice.Kind() != reflect.Slice {
			return nil, errors.New("input is not a slice")
		}

		length := inSlice.Len()
		outSlice := reflect.MakeSlice(targetType, length, length)

		for i := range length {
			elem := inSlice.Index(i)
			if elem.Kind() == reflect.Interface {
				elem = reflect.ValueOf(elem.Interface())
			}

			converted, err := c.convertValue(elem.Interface(), elemType)
			if err != nil {
				return nil, fmt.Errorf("element %d: %v", i, err)
			}

			outSlice.Index(i).Set(reflect.ValueOf(converted))
		}

		return outSlice.Interface(), nil
	}
}

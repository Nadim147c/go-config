package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cast"
)

func (c *Config) Bind(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("input type must be a non-nil pointer")
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.New("input must point to a struct")
	}
	return c.bindInto(elem, "")
}

func (c *Config) bindInto(rv reflect.Value, prefix string) error {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}

		field := rv.Field(i)
		cfgTag := strings.TrimSpace(sf.Tag.Get("config"))

		// build key path
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

		// Recurse for structs
		if field.Kind() == reflect.Struct && sf.Type.Kind() == reflect.Struct {
			newPrefix := cfgTag
			if newPrefix == "" {
				newPrefix = sf.Name
			}
			newPrefix = strings.Trim(newPrefix, ".")
			if prefix != "" {
				if newPrefix != "" {
					newPrefix = prefix + "." + newPrefix
				} else {
					newPrefix = prefix
				}
			}
			if err := c.bindInto(field, newPrefix); err != nil {
				return err
			}
			continue
		}

		// Pointers to structs -> initialize and recurse
		if field.Kind() == reflect.Pointer && field.Type().Elem().Kind() == reflect.Struct {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			newPrefix := cfgTag
			if newPrefix == "" {
				newPrefix = sf.Name
			}
			newPrefix = strings.Trim(newPrefix, ".")
			if prefix != "" {
				if newPrefix != "" {
					newPrefix = prefix + "." + newPrefix
				} else {
					newPrefix = prefix
				}
			}
			if err := c.bindInto(field.Elem(), newPrefix); err != nil {
				return err
			}

			continue
		}

		changed := true
		// 1) Try fetching from config into a temporary reflect.Value
		got, err := c.GetValueE(key)
		if err != nil {
			changed = false
			got = reflect.Zero(field.Type())
		}

		// Ensure type compatibility
		if !got.Type().AssignableTo(field.Type()) {
			var err error
			var v any
			switch field.Kind() {
			case reflect.String:
				v, err = cast.ToStringE(got.Interface())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				v, err = cast.ToInt64E(got.Interface())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				v, err = cast.ToUint64E(got.Interface())
			case reflect.Float32, reflect.Float64:
				v, err = cast.ToFloat64E(got.Interface())
			case reflect.Bool:
				v, err = cast.ToBoolE(got.Interface())
			}
			if err != nil {
				return fmt.Errorf("%s: cannot assign %v to %v", key, got.Type(), field.Type())
			}
			got = reflect.ValueOf(v)
		}

		if field.CanSet() {
			field.Set(got)
		}

		if err := Validate(sf, field, changed); err != nil {
			return fmt.Errorf("%s: %v", key, err)
		}
	}
	return nil
}

func resolvePointer(sv reflect.Value) reflect.Value {
	var i int
	for sv.Kind() == reflect.Pointer && i < 50 {
		sv = sv.Elem()
		i++
	}
	return sv
}

func tryConvertBasic(v reflect.Value, dst reflect.Type) (reflect.Value, bool) {
	if !v.IsValid() {
		return reflect.Value{}, false
	}
	// Allow numeric width adjustments
	switch {
	case v.Kind() == reflect.Int && dst.Kind() >= reflect.Int && dst.Kind() <= reflect.Int64:
		return v.Convert(dst), true
	case v.Kind() == reflect.Int64 && dst.Kind() >= reflect.Int && dst.Kind() <= reflect.Int64:
		return v.Convert(dst), true
	case v.Kind() == reflect.Uint && dst.Kind() >= reflect.Uint && dst.Kind() <= reflect.Uint64:
		return v.Convert(dst), true
	case v.Kind() == reflect.Uint64 && dst.Kind() >= reflect.Uint && dst.Kind() <= reflect.Uint64:
		return v.Convert(dst), true
	case v.Kind() == reflect.Bool && dst.Kind() == reflect.Bool:
		return v.Convert(dst), true
	case v.Kind() == reflect.String && dst.Kind() == reflect.String:
		return v.Convert(dst), true
	default:
		return reflect.Value{}, false
	}
}

func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return v.IsNil()
	default:
		zero := reflect.Zero(v.Type())
		return reflect.DeepEqual(v.Interface(), zero.Interface())
	}
}

package config

import (
	"errors"
	"fmt"
	"net/mail"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/uuid"
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
		valTag := strings.TrimSpace(sf.Tag.Get("validate"))

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
			// also validate struct field rules on the struct itself if present
			if valTag != "" {
				rules, err := parseValidateTag(valTag)
				if err != nil {
					panic(fmt.Sprintf("invalid validate tag on field %s: %v", sf.Name, err))
				}
				if err := validateValue(field, rules, sf); err != nil {
					return fmt.Errorf("%s: %w", key, err)
				}
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
			// validate pointer-to-struct if rules exist
			if valTag != "" {
				rules, err := parseValidateTag(valTag)
				if err != nil {
					panic(fmt.Sprintf("invalid validate tag on field %s: %v", sf.Name, err))
				}
				if err := validateValue(field.Elem(), rules, sf); err != nil {
					return fmt.Errorf("%s: %w", key, err)
				}
			}
			continue
		}

		// Leaf fields: fetch, apply defaults, set, validate
		rules := map[string]string{}
		if valTag != "" {
			m, err := parseValidateTag(valTag)
			if err != nil {
				panic(fmt.Sprintf("invalid validate tag on field %s: %v", sf.Name, err))
			}
			rules = m
		}

		// 1) Try fetching from config into a temporary reflect.Value
		got, found, err := getFromConfig(c, key, field.Kind())
		if err != nil {
			// Not found or wrong type -> we'll consider found=false and let default/required handle
			found = false
		}
		// 2) Apply default if needed
		if (!found || isZeroValue(got)) && rules["default"] != "" {
			got = convertDefault(rules["default"], field.Kind())
			found = true
		}

		// If still not found, set zero (later 'required' may fail)
		if !found {
			got = reflect.Zero(field.Type())
		}

		// 3) Assign (with necessary conversion for compatible kinds)
		if !got.IsValid() {
			got = reflect.Zero(field.Type())
		}
		// Ensure type compatibility
		if !got.Type().AssignableTo(field.Type()) {
			// attempt conversion for common int/uint sizes
			cv, ok := tryConvertBasic(got, field.Type())
			if !ok {
				return fmt.Errorf("%s: cannot assign %v to %v", key, got.Type(), field.Type())
			}
			got = cv
		}
		if field.CanSet() {
			field.Set(got)
		}

		// 4) Validate rules
		if len(rules) > 0 {
			if err := validateValue(field, rules, sf); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
	}
	return nil
}

func getFromConfig(c *Config, key string, kind reflect.Kind) (reflect.Value, bool, error) {
	switch kind {
	case reflect.String:
		s, err := c.GetStringE(key)
		if err != nil {
			return reflect.Value{}, false, err
		}
		return reflect.ValueOf(s), true, nil
	case reflect.Bool:
		b, err := c.GetBoolE(key)
		if err != nil {
			return reflect.Value{}, false, err
		}
		return reflect.ValueOf(b), true, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		i, err := c.GetIntE(key)
		if err != nil {
			return reflect.Value{}, false, err
		}
		return reflect.ValueOf(int(i)).Convert(reflect.TypeOf(int(0))), true, nil
	case reflect.Int64:
		i, err := c.GetInt64E(key)
		if err != nil {
			return reflect.Value{}, false, err
		}
		return reflect.ValueOf(i), true, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		u, err := c.GetUintE(key)
		if err != nil {
			return reflect.Value{}, false, err
		}
		return reflect.ValueOf(uint(u)).Convert(reflect.TypeOf(uint(0))), true, nil
	case reflect.Uint64:
		u, err := c.GetUint64E(key)
		if err != nil {
			return reflect.Value{}, false, err
		}
		return reflect.ValueOf(u), true, nil
	default:
		av, err := c.GetE(key)
		if err != nil {
			return reflect.Value{}, false, err
		}
		return reflect.ValueOf(av), true, nil
	}
}

func convertDefault(def string, kind reflect.Kind) reflect.Value {
	switch kind {
	case reflect.String:
		return reflect.ValueOf(def)
	case reflect.Bool:
		return reflect.ValueOf(cast.ToBool(def))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(cast.ToInt64(def)).Convert(reflect.TypeOf(int64(0)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflect.ValueOf(cast.ToUint64(def)).Convert(reflect.TypeOf(uint64(0)))
	default:
		// best-effort: leave zero for unsupported kinds
		return reflect.Value{}
	}
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

func parseValidateTag(tag string) (map[string]string, error) {
	out := map[string]string{}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return out, nil
	}

	parts, err := splitCSVRespectQuotes(tag)
	if err != nil {
		return nil, err
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.Contains(p, "=") {
			// flags like "required"
			out[p] = ""
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("malformed rule: %q", p)
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		// strip single or double quotes if surrounding
		if len(v) >= 2 && ((v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"')) {
			v = v[1 : len(v)-1]
		}
		if k == "" {
			return nil, fmt.Errorf("empty key in rule: %q", p)
		}
		out[k] = v
	}
	return out, nil
}

func splitCSVRespectQuotes(s string) ([]string, error) {
	var parts []string
	var b strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
			b.WriteByte(ch)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			b.WriteByte(ch)
		case ',':
			if inSingle || inDouble {
				b.WriteByte(ch)
			} else {
				parts = append(parts, b.String())
				b.Reset()
			}
		default:
			b.WriteByte(ch)
		}
	}
	if inSingle || inDouble {
		return nil, errors.New("unbalanced quotes in validate tag")
	}
	parts = append(parts, b.String())
	return parts, nil
}

func validateValue(v reflect.Value, rules map[string]string, sf reflect.StructField) error {
	// required
	if _, ok := rules["required"]; ok {
		if isZeroValue(v) {
			return errors.New("validation failed: required")
		}
	}

	// string-specific checks
	if v.Kind() == reflect.String {
		str := v.String()

		// email
		if _, ok := rules["email"]; ok {
			if _, err := mail.ParseAddress(str); err != nil {
				return errors.New("validation failed: email")
			}
		}

		// uuid
		if _, ok := rules["uuid"]; ok {
			if _, err := uuid.Parse(str); err != nil {
				return errors.New("validation failed: uuid")
			}
		}

		// regex / re
		if pattern, ok := rules["regex"]; ok {
			re, err := regexp.Compile(pattern)
			if err != nil {
				panic(fmt.Sprintf("invalid regex in validate tag on field %s: %v", sf.Name, err))
			}
			if !re.MatchString(str) {
				return errors.New("validation failed: re")
			}
		}

		// min/max for string length
		if s, ok := rules["min"]; ok {
			min := cast.ToInt64(s)
			if int64(len(str)) < min {
				return fmt.Errorf("validation failed: min length %d", min)
			}
		}
		if s, ok := rules["max"]; ok {
			max := cast.ToInt64(s)
			if int64(len(str)) > max {
				return fmt.Errorf("validation failed: max length %d", max)
			}
		}
		return nil
	}

	// numeric min/max
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val := v.Int()
		if s, ok := rules["min"]; ok {
			min := cast.ToInt64(s)
			if val < min {
				return fmt.Errorf("validation failed: min %d", min)
			}
		}
		if s, ok := rules["max"]; ok {
			max := cast.ToInt64(s)
			if val > max {
				return fmt.Errorf("validation failed: max %d", max)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val := v.Uint()
		if s, ok := rules["min"]; ok {
			min := cast.ToUint64(s)
			if val < min {
				return fmt.Errorf("validation failed: min %d", min)
			}
		}
		if s, ok := rules["max"]; ok {
			max := cast.ToUint64(s)
			if val > max {
				return fmt.Errorf("validation failed: max %d", max)
			}
		}
	}
	return nil
}

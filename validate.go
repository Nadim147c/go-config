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

func Validate(sf reflect.StructField, sfv reflect.Value, changed bool) error {
	ruleTag, ok := sf.Tag.Lookup("validate")
	if !ok {
		return nil
	}

	rules := Must(parseValidateTag(ruleTag))
	exclusive(rules, "required", "default")

	for name, rule := range rules {
		switch name {
		default:
			panic(fmt.Sprintf("unknown validation rule %q", name))
		case "required":
			if !changed {
				return errors.New("value is not changed")
			}
		case "default":
			value := resolvePointer(sfv)
			if isZeroValue(value) {
				switch value.Kind() {
				case reflect.String:
					sfv.SetString(Must(cast.ToStringE(rule)))
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					sfv.SetInt(Must(cast.ToInt64E(rule)))
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					sfv.SetUint(Must(cast.ToUint64E(rule)))
				case reflect.Float32, reflect.Float64:
					sfv.SetFloat(Must(cast.ToFloat64E(rule)))
				case reflect.Bool:
					sfv.SetBool(Must(cast.ToBoolE(rule)))
				default:
					panic(fmt.Sprintf("%s does not support default value assignment", value.Kind()))
				}
			}
		case "base64":
			value := resolvePointer(sfv)
			if value.Kind() != reflect.String {
				panic("base64 must be a string")
			}
			str := value.String()

			// Quick base64 structural validation
			if len(str)%4 != 0 {
				return fmt.Errorf("invalid base64 length")
			}
			for i := 0; i < len(str); i++ {
				c := str[i]
				if !(c >= 'A' && c <= 'Z' ||
					c >= 'a' && c <= 'z' ||
					c >= '0' && c <= '9' ||
					c == '+' || c == '/' || c == '=') {
					return fmt.Errorf("invalid base64 character at position %d", i)
				}
			}
			// Padding check
			if pad := strings.Count(str, "="); pad > 2 ||
				(pad > 0 && !strings.HasSuffix(str, strings.Repeat("=", pad))) {
				return fmt.Errorf("invalid base64 padding")
			}
		case "email":
			value := resolvePointer(sfv)
			if value.Kind() != reflect.String {
				panic("email must be a string")
			}
			str := value.String()
			addr, err := mail.ParseAddress(str)
			if err != nil {
				return fmt.Errorf("invalid email: %w", err)
			}
			if addr.Name != "" {
				return fmt.Errorf("email must not contain a display name")
			}
			sfv.SetString(addr.Address)
		case "uuid":
			value := resolvePointer(sfv)
			if value.Kind() != reflect.String {
				panic("uuid must be a string")
			}
			str := value.String()
			_, err := uuid.Parse(str)
			if err != nil {
				return fmt.Errorf("%q is not valid uuid", str)
			}
		case "alpha":
			value := resolvePointer(sfv)
			if value.Kind() != reflect.String {
				panic("alpha must be a string")
			}
			str := value.String()
			for i := 0; i < len(str); i++ {
				c := str[i]
				if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z') {
					return fmt.Errorf("alpha must contain only letters, found '%c' at position %d", c, i)
				}
			}

		case "alphanumeric":
			value := resolvePointer(sfv)
			if value.Kind() != reflect.String {
				panic("alphanumeric must be a string")
			}
			str := value.String()
			for i := 0; i < len(str); i++ {
				c := str[i]
				if !(c >= 'A' && c <= 'Z' ||
					c >= 'a' && c <= 'z' ||
					c >= '0' && c <= '9') {
					return fmt.Errorf("alphanumeric must contain only letters or digits, found '%c' at position %d", c, i)
				}
			}

		case "number":
			value := resolvePointer(sfv)
			if value.Kind() != reflect.String {
				panic("number must be a string")
			}
			str := value.String()
			if len(str) == 0 {
				return fmt.Errorf("number must not be empty")
			}
			for i := 0; i < len(str); i++ {
				if str[i] < '0' || str[i] > '9' {
					return fmt.Errorf("number must contain only digits, found '%c' at position %d", str[i], i)
				}
			}
		case "match":
			re := regexp.MustCompile(Must(cast.ToStringE(rule)))
			value := resolvePointer(sfv)
			if value.Kind() != reflect.String {
				panic("match pattern must be a string")
			}
			if !re.MatchString(value.String()) {
				return fmt.Errorf("string must match following the pattern: %s", rule)
			}
		case "min":
			value := resolvePointer(sfv)
			kind := sf.Type.Kind()
			switch kind {
			case reflect.String, reflect.Array, reflect.Slice, reflect.Chan, reflect.Map:
				limit := Must(cast.ToIntE(rule))
				if value.Len() < limit {
					return fmt.Errorf("%s len (%d) is less the minimum len (%d)", kind, value.Len(), limit)
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				limit := Must(cast.ToInt64E(rule))
				value := value.Int()
				if value < limit {
					return fmt.Errorf("%d is less the minimum (%d)", value, limit)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				limit := Must(cast.ToUint64E(rule))
				value := value.Uint()
				if value < limit {
					return fmt.Errorf("%d is less the minimum (%d)", value, limit)
				}
			default:
				panic(fmt.Sprintf("%s does not support min value", kind))
			}
		case "max":
			value := resolvePointer(sfv)
			kind := sf.Type.Kind()
			switch kind {
			case reflect.String, reflect.Array, reflect.Slice, reflect.Chan, reflect.Map:
				limit := Must(cast.ToIntE(rule))
				if value.Len() > limit {
					return fmt.Errorf("%s len (%d) is less the maximum len (%d)", kind, value.Len(), limit)
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				limit := Must(cast.ToInt64E(rule))
				value := value.Int()
				if value > limit {
					return fmt.Errorf("%d is less the maximum (%d)", value, limit)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				limit := Must(cast.ToUint64E(rule))
				value := value.Uint()
				if value > limit {
					return fmt.Errorf("%d is less the maximum (%d)", value, limit)
				}
			default:
				panic(fmt.Sprintf("%s does not support min value", kind))
			}
		}
	}
	return nil
}

func exclusive(rules map[string]any, a, b string) {
	_, okA := rules[a]
	_, okB := rules[b]
	if okA && okB {
		panic("%q and %q are mutually exclusive")
	}
}

func parseValidateTag(tag string) (map[string]any, error) {
	out := map[string]any{}
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
			out[p] = true
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

package config

import "reflect"

// DeepMerge recursively merges src into dst, combining nested maps rather than
// replacing them. Non-map values in src overwrite those in dst. Both maps must
// have string keys. Returns the updated dst.
//
// Example:
//
//	dst = { "a": { "x": 1 }, "b": 2 }
//	src = { "a": { "y": 3 }, "b": 4 }
//	DeepMerge(dst, src) → { "a": { "x": 1, "y": 3 }, "b": 4 }
func DeepMerge(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	if src == nil {
		return dst
	}
	for k, sv := range src {
		if dv, ok := dst[k]; ok {
			dst[k] = mergeValues(dv, sv)
		} else {
			dst[k] = cloneIfMap(sv)
		}
	}
	return dst
}

func mergeValues(dstVal, srcVal any) any {
	rd := reflect.ValueOf(dstVal)
	rs := reflect.ValueOf(srcVal)
	if isStringKeyMap(rd) && isStringKeyMap(rs) {
		md := toStringAnyMap(rd)
		ms := toStringAnyMap(rs)
		return DeepMerge(md, ms)
	}
	return cloneIfMap(srcVal)
}

func cloneIfMap(v any) any {
	rv := reflect.ValueOf(v)
	if !isStringKeyMap(rv) {
		return v
	}
	m := toStringAnyMap(rv)
	out := map[string]any{}
	for kk, vv := range m {
		if isStringKeyMap(reflect.ValueOf(vv)) {
			out[kk] = cloneIfMap(vv)
		} else {
			out[kk] = vv
		}
	}
	return out
}

func isStringKeyMap(rv reflect.Value) bool {
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return false
	}
	return rv.Type().Key().Kind() == reflect.String
}

func toStringAnyMap(rv reflect.Value) map[string]any {
	out := map[string]any{}
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return out
	}
	for _, key := range rv.MapKeys() {
		if key.Kind() != reflect.String {
			continue
		}
		val := rv.MapIndex(key)
		if !val.IsValid() {
			out[key.String()] = nil
			continue
		}
		out[key.String()] = val.Interface()
	}
	return out
}

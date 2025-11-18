package gowmbus

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FieldSet offers typed helpers on top of a dynamic field map.
type FieldSet struct {
	data map[string]any
}

// FieldSet returns a FieldSet wrapper for the result's fields.
func (r Result) FieldSet() FieldSet {
	return FieldSet{data: r.Fields}
}

// Map exposes the underlying map for callers that still need raw access.
func (fs FieldSet) Map() map[string]any {
	return fs.data
}

// Raw returns the stored value without conversions.
func (fs FieldSet) Raw(key string) (any, bool) {
	if fs.data == nil {
		return nil, false
	}
	v, ok := fs.data[key]
	return v, ok
}

// Float returns the field coerced to float64.
func (fs FieldSet) Float(key string) (float64, error) {
	v, ok := fs.Raw(key)
	if !ok {
		return 0, fmt.Errorf("field %q missing", key)
	}
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case uint:
		return float64(n), nil
	case uint32:
		return float64(n), nil
	case uint64:
		return float64(n), nil
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, fmt.Errorf("field %q is not numeric: %w", key, err)
		}
		return f, nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, fmt.Errorf("field %q is not numeric: %w", key, err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("field %q has unsupported type %T", key, v)
	}
}

// Int returns the field coerced to int64.
func (fs FieldSet) Int(key string) (int64, error) {
	v, ok := fs.Raw(key)
	if !ok {
		return 0, fmt.Errorf("field %q missing", key)
	}
	switch n := v.(type) {
	case int:
		return int64(n), nil
	case int32:
		return int64(n), nil
	case int64:
		return n, nil
	case uint:
		return int64(n), nil
	case uint32:
		return int64(n), nil
	case uint64:
		return int64(n), nil
	case float32:
		return int64(n), nil
	case float64:
		return int64(n), nil
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, fmt.Errorf("field %q is not integer: %w", key, err)
		}
		return i, nil
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("field %q is not integer: %w", key, err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("field %q has unsupported type %T", key, v)
	}
}

// String returns the field as a string.
func (fs FieldSet) String(key string) (string, error) {
	v, ok := fs.Raw(key)
	if !ok {
		return "", fmt.Errorf("field %q missing", key)
	}
	switch s := v.(type) {
	case string:
		return s, nil
	case fmt.Stringer:
		return s.String(), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// Bool returns the field coerced to bool.
func (fs FieldSet) Bool(key string) (bool, error) {
	v, ok := fs.Raw(key)
	if !ok {
		return false, fmt.Errorf("field %q missing", key)
	}
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		parsed, err := strconv.ParseBool(b)
		if err != nil {
			return false, fmt.Errorf("field %q is not bool: %w", key, err)
		}
		return parsed, nil
	default:
		return false, fmt.Errorf("field %q has unsupported type %T", key, v)
	}
}

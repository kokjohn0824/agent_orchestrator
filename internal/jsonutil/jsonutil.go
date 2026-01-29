// Package jsonutil provides helper functions for working with JSON data,
// particularly for extracting values from map[string]interface{} structures.
package jsonutil

// GetString extracts a string value from a map by key.
// Returns empty string if the key doesn't exist or the value is not a string.
func GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// GetInt extracts an integer value from a map by key.
// Handles both int and float64 (JSON numbers are decoded as float64).
// Returns 0 if the key doesn't exist or the value is not a number.
func GetInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

// GetFloat extracts a float64 value from a map by key.
// Returns 0 if the key doesn't exist or the value is not a number.
func GetFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

// GetBool extracts a boolean value from a map by key.
// Returns false if the key doesn't exist or the value is not a boolean.
func GetBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// ToStringSlice converts a slice of interface{} to a slice of strings.
// Non-string values are skipped.
func ToStringSlice(slice []interface{}) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// GetStringSlice extracts a string slice from a map by key.
// Returns nil if the key doesn't exist or the value is not a slice.
func GetStringSlice(m map[string]interface{}, key string) []string {
	if slice, ok := m[key].([]interface{}); ok {
		return ToStringSlice(slice)
	}
	return nil
}

package jsonutil

import (
	"reflect"
	"testing"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "existing string key",
			m:    map[string]interface{}{"key": "value"},
			key:  "key",
			want: "value",
		},
		{
			name: "missing key",
			m:    map[string]interface{}{"other": "value"},
			key:  "key",
			want: "",
		},
		{
			name: "non-string value",
			m:    map[string]interface{}{"key": 123},
			key:  "key",
			want: "",
		},
		{
			name: "empty map",
			m:    map[string]interface{}{},
			key:  "key",
			want: "",
		},
		{
			name: "nil map",
			m:    nil,
			key:  "key",
			want: "",
		},
		{
			name: "empty string value",
			m:    map[string]interface{}{"key": ""},
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetString(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want int
	}{
		{
			name: "int value",
			m:    map[string]interface{}{"key": 42},
			key:  "key",
			want: 42,
		},
		{
			name: "float64 value (JSON number)",
			m:    map[string]interface{}{"key": float64(42)},
			key:  "key",
			want: 42,
		},
		{
			name: "float64 with decimals",
			m:    map[string]interface{}{"key": float64(42.9)},
			key:  "key",
			want: 42,
		},
		{
			name: "missing key",
			m:    map[string]interface{}{"other": 42},
			key:  "key",
			want: 0,
		},
		{
			name: "non-numeric value",
			m:    map[string]interface{}{"key": "42"},
			key:  "key",
			want: 0,
		},
		{
			name: "nil map",
			m:    nil,
			key:  "key",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetInt(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("GetInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFloat(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want float64
	}{
		{
			name: "float64 value",
			m:    map[string]interface{}{"key": 3.14},
			key:  "key",
			want: 3.14,
		},
		{
			name: "missing key",
			m:    map[string]interface{}{"other": 3.14},
			key:  "key",
			want: 0,
		},
		{
			name: "non-float value",
			m:    map[string]interface{}{"key": "3.14"},
			key:  "key",
			want: 0,
		},
		{
			name: "nil map",
			m:    nil,
			key:  "key",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFloat(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("GetFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want bool
	}{
		{
			name: "true value",
			m:    map[string]interface{}{"key": true},
			key:  "key",
			want: true,
		},
		{
			name: "false value",
			m:    map[string]interface{}{"key": false},
			key:  "key",
			want: false,
		},
		{
			name: "missing key",
			m:    map[string]interface{}{"other": true},
			key:  "key",
			want: false,
		},
		{
			name: "non-bool value",
			m:    map[string]interface{}{"key": "true"},
			key:  "key",
			want: false,
		},
		{
			name: "nil map",
			m:    nil,
			key:  "key",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBool(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		slice []interface{}
		want  []string
	}{
		{
			name:  "all strings",
			slice: []interface{}{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "mixed types",
			slice: []interface{}{"a", 1, "b", true, "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "no strings",
			slice: []interface{}{1, 2, 3},
			want:  []string{},
		},
		{
			name:  "empty slice",
			slice: []interface{}{},
			want:  []string{},
		},
		{
			name:  "nil slice",
			slice: nil,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToStringSlice(tt.slice)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStringSlice(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want []string
	}{
		{
			name: "existing slice",
			m:    map[string]interface{}{"key": []interface{}{"a", "b", "c"}},
			key:  "key",
			want: []string{"a", "b", "c"},
		},
		{
			name: "missing key",
			m:    map[string]interface{}{"other": []interface{}{"a"}},
			key:  "key",
			want: nil,
		},
		{
			name: "non-slice value",
			m:    map[string]interface{}{"key": "not a slice"},
			key:  "key",
			want: nil,
		},
		{
			name: "mixed types in slice",
			m:    map[string]interface{}{"key": []interface{}{"a", 1, "b"}},
			key:  "key",
			want: []string{"a", "b"},
		},
		{
			name: "empty slice",
			m:    map[string]interface{}{"key": []interface{}{}},
			key:  "key",
			want: []string{},
		},
		{
			name: "nil map",
			m:    nil,
			key:  "key",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetStringSlice(tt.m, tt.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

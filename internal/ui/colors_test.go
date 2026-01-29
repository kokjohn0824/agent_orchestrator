package ui

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "短字串不截斷",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "剛好等於最大長度",
			input:    "Hello",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "長字串需要截斷",
			input:    "Hello World",
			maxLen:   8,
			expected: "Hello...",
		},
		{
			name:     "空字串",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "maxLen 為零",
			input:    "Hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "maxLen 為負數",
			input:    "Hello",
			maxLen:   -1,
			expected: "",
		},
		{
			name:     "maxLen 小於等於 3",
			input:    "Hello",
			maxLen:   3,
			expected: "Hel",
		},
		{
			name:     "maxLen 為 1",
			input:    "Hello",
			maxLen:   1,
			expected: "H",
		},
		{
			name:     "ASCII 長文字截斷",
			input:    "This is a very long title that needs truncation",
			maxLen:   20,
			expected: "This is a very lo...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestTruncateLength(t *testing.T) {
	// 確保截斷後的字串長度不超過 maxLen
	testCases := []struct {
		input  string
		maxLen int
	}{
		{"Hello World!", 5},
		{"Hello World!", 10},
		{"Short", 10},
		{"A very long string that needs truncation", 20},
	}

	for _, tc := range testCases {
		result := Truncate(tc.input, tc.maxLen)
		if len(result) > tc.maxLen {
			t.Errorf("Truncate(%q, %d) returned string of length %d, expected <= %d",
				tc.input, tc.maxLen, len(result), tc.maxLen)
		}
	}
}

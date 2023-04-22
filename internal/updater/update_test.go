package updater_test

import (
	"testing"

	"github.com/soulteary/nginx-formatter/internal/updater"
)

func TestEncodeEscapeChars(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "",
		},
		{
			input:    `Hello\tworld`,
			expected: `Hello【\\】tworld`,
		},
		{
			input:    `This is a test\nwith new line`,
			expected: `This is a test【\\】nwith new line`,
		},
		{
			input:    `Some\rtext`,
			expected: `Some【\\】rtext`,
		},
		{
			input:    `\t\s\r\n`,
			expected: `【\\】t【\\】s【\\】r【\\】n`,
		},
	}

	for _, tc := range testCases {
		output := updater.EncodeEscapeChars(tc.input)
		if output != tc.expected {
			t.Errorf("Unexpected output. Input: %s, Expected: %s, Output: %s", tc.input, tc.expected, output)
		}
	}
}

func TestDecodeEscapeChars(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "",
		},
		{
			input:    `Hello【\】tworld`,
			expected: `Hello\tworld`,
		},
		{
			input:    `This is a test【\】nwith new line`,
			expected: `This is a test\nwith new line`,
		},
		{
			input:    `Some【\】rtext`,
			expected: `Some\rtext`,
		},
		{
			input:    `【\】t【\】r【\】n【\】s`,
			expected: `\t\r\n\s`,
		},
	}

	for _, tc := range testCases {
		output := updater.DecodeEscapeChars(tc.input)
		if output != tc.expected {
			t.Errorf("Unexpected output. Input: %s, Expected: %s, Output: %s", tc.input, tc.expected, output)
		}
	}
}

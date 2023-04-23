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
			expected: `Hello【&】tworld`,
		},
		{
			input:    `This is a test\nwith new line`,
			expected: `This is a test【&】nwith new line`,
		},
		{
			input:    `Some\rtext`,
			expected: `Some【&】rtext`,
		},
		{
			input:    `\t\s\r\n`,
			expected: `【&】t【&】s【&】r【&】n`,
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
			input:    `Hello【&】tworld`,
			expected: `Hello\tworld`,
		},
		{
			input:    `This is a test【&】nwith new line`,
			expected: `This is a test\nwith new line`,
		},
		{
			input:    `Some【&】rtext`,
			expected: `Some\rtext`,
		},
		{
			input:    `【&】t【&】r【&】n【&】s`,
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

func TestFixReturn(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{ // empty
			input:    "",
			expected: "",
		},
		{ // add quotes to strings
			input:    "return 200 ok;",
			expected: "return 200 \"ok\";",
		},
		{ // add quotes to strings
			input:    "return 200 $content;",
			expected: "return 200 \"$content\";",
		},
		{ // keep it as it is
			input:    "return BACKEND\n;",
			expected: "return BACKEND;",
		},
		{ // keep it as it is
			input:    "return 200 \"BACKEND B:$uri\n\";",
			expected: "return 200 \"BACKEND B:$uri\n\";",
		},
		{ // keep it as it is
			input:    "return 200;",
			expected: "return 200;",
		},
		{ // trim
			input:    "return   200   ;",
			expected: "return 200;",
		},
		{ // trim
			input:    "return    \"ok\"  ; ",
			expected: "return \"ok\";",
		},
		{ // trim
			input:    "return   200  \"ok\" ;  ",
			expected: "return 200 \"ok\";",
		},
		{
			input:    "return   200        \"1 1 1\"    ;",
			expected: "return 200 \"1 1 1\";",
		},
		{
			input:    "return   \"1\n\"    ;",
			expected: "return \"1\n\";",
		},
		{
			input:    "return   \"1\n  11\"    ;",
			expected: "return \"1\n  11\";",
		},
		{
			input:    "return   \"1\n\t\r  11\n\"    ;",
			expected: "return \"1\n\t\r  11\n\";",
		},
	}

	for _, tc := range testCases {
		output := updater.FixReturn(updater.FixVars(updater.EncodeEscapeChars(tc.input)))
		if output != tc.expected {
			t.Errorf("Unexpected output. Input: %s, Expected: %s, Output: %s", tc.input, tc.expected, (updater.DecodeEscapeChars(output)))
		}
	}
}

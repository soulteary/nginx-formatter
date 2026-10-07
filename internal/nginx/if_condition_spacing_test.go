package nginx

import "testing"

func TestIfConditionClosingParenthesisSpacing(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"double quoted empty", `if ($secure_link = "")`, `if ($secure_link = "")`},
		{"single quoted empty", `if ($secure_link = '')`, `if ($secure_link = '')`},
		{"double quoted value", `if ($secure_link = "value")`, `if ($secure_link = "value")`},
		{"single quoted value", `if ($secure_link = 'value')`, `if ($secure_link = 'value')`},
		{"existing closing space", `if ($secure_link = "value"   )`, `if ($secure_link = "value")`},
		{"unquoted value", `if ($secure_link = value)`, `if ($secure_link = value)`},
		{"bare regex parentheses", `if ($uri ~ ^/(foo(bar)|baz)$)`, `if ($uri ~ ^/(foo(bar)|baz)$)`},
		{"escaped regex parenthesis", `if ($uri ~ ^/literal\)$ )`, `if ($uri ~ ^/literal\)$)`},
		{"quoted regex parentheses", `if ($uri ~ '^/(foo(bar)|baz)\)$')`, `if ($uri ~ '^/(foo(bar)|baz)\)$')`},
		{"quoted content stays verbatim", `if ($value = "contains ) and \"quotes\"  ")`, `if ($value = "contains ) and \"quotes\"  ")`},
		{"opening space stays intact", `if ( $secure_link = "" )`, `if ( $secure_link = "")`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := format(t, tc.src+" { return 403; }")
			want := tc.want + " {\n  return 403;\n}\n"
			if got != want {
				t.Errorf("condition spacing changed:\n got: %q\nwant: %q", got, want)
			}
			if again := format(t, got); again != got {
				t.Errorf("not idempotent:\npass1: %q\npass2: %q", got, again)
			}
		})
	}
}

func TestIfConditionSpacingLeavesDirectiveArgumentsIntact(t *testing.T) {
	for _, src := range []string{
		`log_format paren "$request" );`,
		`return 200 'if ($value = "x" )';`,
	} {
		t.Run(src, func(t *testing.T) {
			got := format(t, src)
			if want := src + "\n"; got != want {
				t.Errorf("directive arguments changed:\n got: %q\nwant: %q", got, want)
			}
			if again := format(t, got); again != got {
				t.Errorf("not idempotent:\npass1: %q\npass2: %q", got, again)
			}
		})
	}
}

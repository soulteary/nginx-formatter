package nginx

import "testing"

// TestBlockSpacing pins the layout as well as its fixed point: an unnecessary
// blank line before a parent closing brace is already idempotent.
func TestBlockSpacing(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "last location",
			src:  "server {\n  location / {\n    js_content main.whoami;\n  }\n}\n",
			want: "server {\n  location / {\n    js_content main.whoami;\n  }\n}\n",
		},
		{
			name: "sibling locations",
			src:  "server {\n  location /a {\n    return 200;\n  }\n  location /b {\n    return 204;\n  }\n}\n",
			want: "server {\n  location /a {\n    return 200;\n  }\n\n  location /b {\n    return 204;\n  }\n}\n",
		},
		{
			name: "author blank line before parent close",
			src:  "server {\n  location / {\n    return 200;\n  }\n\n}\n",
			want: "server {\n  location / {\n    return 200;\n  }\n\n}\n",
		},
		{
			name: "nested closing brace comments",
			src:  "http {\n  server {\n    location / {\n      return 200;\n    } # location\n  } # server\n} # http\n",
			want: "http {\n  server {\n    location / {\n      return 200;\n    } # location\n  } # server\n} # http\n",
		},
		{
			name: "raw Lua body",
			src: "server {\n  content_by_lua_block {\n    local t = {\n      value = 1,\n    }\n" +
				"    local s = [[\nkeep\n\n\nspaces  \n]]\n    ngx.say(s)\n  }\n}\n",
			want: "server {\n  content_by_lua_block {\n    local t = {\n      value = 1,\n    }\n" +
				"    local s = [[\nkeep\n\n\nspaces  \n]]\n    ngx.say(s)\n  }\n}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := format(t, tc.src)
			if got != tc.want {
				t.Errorf("unexpected layout:\n got: %q\nwant: %q", got, tc.want)
			}
			if again := format(t, got); again != got {
				t.Errorf("not idempotent:\npass1: %q\npass2: %q", got, again)
			}
		})
	}
}

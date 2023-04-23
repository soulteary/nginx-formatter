package formatter_test

import (
	"testing"

	"github.com/soulteary/nginx-formatter/internal/formatter"
)

func TestFormatter(t *testing.T) {

	const TestData = `
load_module modules/ngx_http_js_module.so;

events {  }

http {
js_path "/etc/nginx/njs/";

js_import main from http/api/set_keyval.js;

keyval_zone zone=foo:10m;

server {
listen 80;

location /keyval {
js_content main.set_keyval;
}
location /api {
internal;
api write=on;
}
location /api/ro {
api;
}
}
}`

	const TestExpected = `
load_module modules/ngx_http_js_module.so;

events {  }

http {
    js_path "/etc/nginx/njs/";

    js_import main from http/api/set_keyval.js;

    keyval_zone zone=foo:10m;

    server {
        listen 80;

        location /keyval {
            js_content main.set_keyval;
        }

        location /api {
            internal;
            api write=on;
        }

        location /api/ro {
            api;
        }

    }
}`

	result, err := formatter.Formatter(TestData, 4, " ")
	if err != nil {
		t.Errorf("formatter error: %v\n", err)
	}

	if result != TestExpected {
		t.Error("formatter result not expected", result, TestExpected)
	}
}

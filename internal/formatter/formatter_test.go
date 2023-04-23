package formatter_test

import (
	"testing"

	"github.com/soulteary/nginx-formatter/internal/formatter"
)

func TestFormatter(t *testing.T) {

	const TestData = `
  load_module modules/ngx_http_js_module.so;
  
  events { }
  
  http {
        js_path "/etc/nginx/njs/";
  
        js_import utils.js;
        js_import main from http/hello.js;
  
        server {
              listen 80;
  
              location = /version {
                  js_content utils.version;
              }
  
              location / {
                  js_content main.hello;
              }
        }
  }`

	const TestExpected = `
load_module modules/ngx_http_js_module.so;

events {  }

http {
  js_path "/etc/nginx/njs/";

  js_import utils.js;
  js_import main from http/hello.js;

  server {
    listen 80;

    location = /version {
      js_content utils.version;
    }

    location / {
      js_content main.hello;
    }

  }
}`

	result, err := formatter.Formatter(TestData, 2, " ")
	if err != nil {
		t.Errorf("formatter error: %v\n", err)
	}

	if result != TestExpected {
		t.Error("formatter result not expected")
	}
}

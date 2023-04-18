package formatter

import (
	"fmt"

	"github.com/dop251/goja"
)

func Formatter(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	vm := goja.New()
	v, err := vm.RunString(fmt.Sprintf("%s;FormatNginxConf(`%s`)", JS_FORMATTER, s))
	if err != nil {
		return "", err
	}
	return v.String(), nil
}

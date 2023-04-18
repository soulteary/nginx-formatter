package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func EncodeEscapeChars(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, `\t`, `{{\\}}t`), `\s`, `{{\\}}s`), `\r`, `{{\\}}r`), `\n`, `{{\\}}n`)
}

func DecodeEscapeChars(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, `{{\}}t`, `\t`), `{{\}}s`, `\s`), `{{\}}r`, `\r`), `{{\}}n`, `\n`)
}

func FixVars(s string) string {
	s = regexp.MustCompile(`(\$)(\{\S+?\})`).ReplaceAllString(s, "[dollar]$2")
	return regexp.MustCompile(`(return\s+\d+\s+?)([\s\S]+?);`).ReplaceAllString(s, "$1\"$2\";")
}

func UpdateConfInDir(rootDir string, fn func(s string) (string, error)) error {
	if rootDir == "" {
		return fmt.Errorf("scandir is empty")
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".conf") {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			modifiedData, err := fn(FixVars(EncodeEscapeChars(string(data))))
			if err != nil {
				return err
			}

			err = os.WriteFile(path, []byte(DecodeEscapeChars(modifiedData)), info.Mode())
			if err != nil {
				return err
			}

			rel, err := filepath.Rel(rootDir, path)
			if err != nil {
				fmt.Printf("Formatter Nginx Conf %s Successed\n", path)
			} else {
				fmt.Printf("Formatter Nginx Conf %s Successed\n", rel)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

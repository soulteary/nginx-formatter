package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func EncodeEscapeChars(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, `\t`, `【\\】t`), `\s`, `【\\】s`), `\r`, `【\\】r`), `\n`, `【\\】n`)
}

func DecodeEscapeChars(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, `【\】t`, `\t`), `【\】s`, `\s`), `【\】r`, `\r`), `【\】n`, `\n`)
}

// scenes:
//
//		return 200 "1" ;
//		return 200 "1";
//		return   "1" ;
//		return  200;
//		return  "aaa\naa";
//		return  200 "a\n"   ;
//	 	return BACKEND\n;
func FixReturn(s string) string {
	var scene1 = regexp.MustCompile(`return\s+(\d+)\s(\S+)\s*;`)
	var scene2 = regexp.MustCompile(`return\s+(\d+)\s"(\S+)"\s*;`)
	var scene3 = regexp.MustCompile(`return\s+(\S+)\s*;`)
	var scene4 = regexp.MustCompile(`return\s+"(\S+)"\s*;`)
	var scene5 = regexp.MustCompile(`return\s+(\d+)\s*;`)

	if scene1.MatchString(s) {
		if scene2.MatchString(s) { // eg: `return 200 "ok";`
			s = scene2.ReplaceAllString(s, "return $1 \"$2\";")
		} else { // eg: `return 200 $content;`
			s = scene1.ReplaceAllString(s, "return $1 \"$2\";")
		}
	} else if scene3.MatchString(s) {
		if scene5.MatchString(s) { // eg: `return 200;`
			s = scene5.ReplaceAllString(s, "return $1;")
		} else if scene4.MatchString(s) { // eg: `return "ok";`
			s = scene4.ReplaceAllString(s, "return \"$1\";")
		} else { // eg: `return BACKEND\n;`
			found := scene3.FindString(s)
			if !(strings.HasPrefix(found, `"`) && strings.HasSuffix(found, `"`)) {
				s = scene3.ReplaceAllString(s, "return $1;")
			} else {
				s = scene3.ReplaceAllString(s, "return \"$1\";")
			}
		}
	}
	return s
}

func FixVars(s string) string {
	s = regexp.MustCompile(`(\$)(\{\S+?\})`).ReplaceAllString(s, "[dollar]$2")
	return s
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

			modifiedData, err := fn(FixVars(FixReturn(EncodeEscapeChars(string(data)))))
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

package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func EncodeEscapeChars(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, `\t`, `【&】t`), `\s`, `【&】s`), `\r`, `【&】r`), `\n`, `【&】n`)
}

func DecodeEscapeChars(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, `【&】t`, `\t`), `【&】s`, `\s`), `【&】r`, `\r`), `【&】n`, `\n`)
}

func FixReturn(s string) string {

	var scene1 = regexp.MustCompile(`return\s+(\d+)\s+(\S+)\s*;`)
	var scene2 = regexp.MustCompile(`return\s+(\d+)\s+"(\S+)"\s*;`)
	var scene3 = regexp.MustCompile(`return\s+(\S+)\s*;`)
	var scene4 = regexp.MustCompile(`return\s+"(\S+)"\s*;`)
	var scene5 = regexp.MustCompile(`return\s+(\d+)\s*;`)
	var scene6 = regexp.MustCompile(`return\s+"(.+)"\s*;`)
	var scene7 = regexp.MustCompile(`return\s+(\d+)\s+"(.+)"\s*;`)
	var scene8 = regexp.MustCompile(`return\s+(\d+)\s+"([\s|\S|\n|\r|\t]+)"\s*;`)
	var scene9 = regexp.MustCompile(`return\s+"([\s|\S|\n|\r|\t]+)"\s*;`)

	if scene1.MatchString(s) {
		if scene2.MatchString(s) { // eg: `return 200 "ok";`
			return strings.TrimSpace(scene2.ReplaceAllString(s, "return $1 \"$2\";"))
		} else { // eg: `return 200 $content;`
			return strings.TrimSpace(scene1.ReplaceAllString(s, "return $1 \"$2\";"))
		}
	} else if scene3.MatchString(s) {
		if scene5.MatchString(s) { // eg: `return 200;`
			return strings.TrimSpace(scene5.ReplaceAllString(s, "return $1;"))
		} else if scene6.MatchString(s) { // eg: `return "ok";`
			if scene4.MatchString(s) {
				return strings.TrimSpace(scene4.ReplaceAllString(s, "return \"$1\";"))
			} else {
				return strings.TrimSpace(scene6.ReplaceAllString(s, "return \"$1\";"))
			}
		} else { // eg: `return BACKEND\n;`
			found := scene3.FindString(s)
			if !(strings.HasPrefix(found, `"`) && strings.HasSuffix(found, `"`)) {
				return strings.TrimSpace(scene3.ReplaceAllString(s, "return $1;"))
			} else {
				return strings.TrimSpace(scene3.ReplaceAllString(s, "return \"$1\";"))
			}
		}
	} else {
		if scene7.MatchString(s) {
			return strings.TrimSpace(scene7.ReplaceAllString(s, "return $1 \"$2\";"))
		} else if scene8.MatchString(s) {
			return strings.TrimSpace(scene8.ReplaceAllString(s, "return $1 \"$2\";"))
		} else if scene9.MatchString(s) {
			return strings.TrimSpace(scene9.ReplaceAllString(s, "return \"$1\";"))
		}
	}
	return s
}

func FixVars(s string) string {
	s = regexp.MustCompile(`(\$)(\{\S+?\})`).ReplaceAllString(s, "[dollar]$2")
	return s
}

func ScanFiles(rootDir string) ([]string, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("scandir is empty")
	}
	var files []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".conf") {
			_, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func UpdateConfInDir(rootDir string, outputDir string, indent int, indentChar string, fn func(s string, indent int, char string) (string, error)) error {
	files, err := ScanFiles(rootDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		buf, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Formatter Nginx Conf %s failed, can not open the file\n", err)
			return err
		}
		modifiedData, err := fn(FixVars(FixReturn(EncodeEscapeChars(string(buf)))), indent, indentChar)
		if err != nil {
			fmt.Printf("Formatter Nginx Conf %s failed, can not format the file\n", err)
			return err
		}

		output := ""
		relPath, err := filepath.Rel(rootDir, file)
		if err != nil {
			output = filepath.Join(outputDir, file)
		} else {
			output = filepath.Join(outputDir, relPath)
		}

		err = os.WriteFile(output, []byte(DecodeEscapeChars(modifiedData)), 0644)
		if err != nil {
			fmt.Printf("Formatter Nginx Conf %s failed, can not save the file\n", output)
			return err
		}

		relPath, err = filepath.Rel(rootDir, output)
		if err != nil {
			fmt.Printf("Formatter Nginx Conf %s Successed\n", output)
		} else {
			fmt.Printf("Formatter Nginx Conf %s Successed\n", relPath)
		}
	}
	return nil
}

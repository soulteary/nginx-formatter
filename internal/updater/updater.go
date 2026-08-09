package updater

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FixReturn normalizes "return" directives so a bare argument is wrapped in
// double quotes, matching the historical updater behavior. Examples:
//
//	return 200 $content;  -> return 200 "$content";
//	return 200 "ok";      -> return 200 "ok"; (unchanged)
//	return 200;           -> return 200; (unchanged)
//	return BACKEND;       -> return BACKEND; (unchanged)
//	return "ok";          -> return "ok"; (unchanged)
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
			if !strings.HasPrefix(found, `"`) || !strings.HasSuffix(found, `"`) {
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

func ScanFiles(rootDir string) ([]string, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("scandir is empty")
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	var files []string
	err = fs.WalkDir(root.FS(), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(rel, ".conf") {
			return nil
		}
		if _, err := root.ReadFile(rel); err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// resolveTarget decides where the formatted single-file output should be
// written based on the -output value:
//   - output == ""          -> overwrite inputFile in place
//   - output is a directory -> filepath.Join(output, filepath.Base(inputFile))
//   - otherwise             -> treat output as a file path, creating its parent
//     directory when needed
func resolveTarget(inputFile string, output string) (string, error) {
	if output == "" {
		return inputFile, nil
	}

	if info, err := os.Stat(output); err == nil && info.IsDir() {
		return filepath.Join(output, filepath.Base(inputFile)), nil
	}

	if dir := filepath.Dir(output); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", err
		}
	}
	return output, nil
}

// UpdateConfFile formats a single file. Unlike UpdateConfInDir it does not
// filter by the .conf suffix, so any file can be formatted.
func UpdateConfFile(inputFile string, output string, indent int, indentChar string, fn func(s string, indent int, char string) (string, error)) error {
	buf, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Formatter Nginx Conf %s failed, can not open the file\n", err)
		return err
	}

	modifiedData, err := fn(FixReturn(string(buf)), indent, indentChar)
	if err != nil {
		fmt.Printf("Formatter Nginx Conf %s failed, can not format the file\n", err)
		return err
	}

	target, err := resolveTarget(inputFile, output)
	if err != nil {
		fmt.Printf("Formatter Nginx Conf %s failed, can not prepare the save dir\n", err)
		return err
	}

	if err := os.WriteFile(target, []byte(modifiedData), 0600); err != nil {
		fmt.Printf("Formatter Nginx Conf %s failed, can not save the file\n", err)
		return err
	}

	fmt.Printf("Formatter Nginx Conf %s Successed\n", target)
	return nil
}

func UpdateConfInDir(rootDir string, outputDir string, indent int, indentChar string, fn func(s string, indent int, char string) (string, error)) error {
	files, err := ScanFiles(rootDir)
	if err != nil {
		return err
	}

	inRoot, err := os.OpenRoot(rootDir)
	if err != nil {
		return err
	}
	defer func() { _ = inRoot.Close() }()

	if err := os.MkdirAll(outputDir, 0700); err != nil {
		fmt.Printf("Formatter Nginx Conf %s failed, can not prepare the save dir\n", err)
		return err
	}
	outRoot, err := os.OpenRoot(outputDir)
	if err != nil {
		return err
	}
	defer func() { _ = outRoot.Close() }()

	for _, rel := range files {
		buf, err := inRoot.ReadFile(rel)
		if err != nil {
			fmt.Printf("Formatter Nginx Conf %s failed, can not open the file\n", err)
			return err
		}

		modifiedData, err := fn(FixReturn(string(buf)), indent, indentChar)
		if err != nil {
			fmt.Printf("Formatter Nginx Conf %s failed, can not format the file\n", err)
			return err
		}

		if dir := filepath.Dir(rel); dir != "." {
			if err := outRoot.MkdirAll(dir, 0700); err != nil {
				fmt.Printf("Formatter Nginx Conf %s failed, can not prepare the save dir\n", err)
				return err
			}
		}

		if err := outRoot.WriteFile(rel, []byte(modifiedData), 0600); err != nil {
			fmt.Printf("Formatter Nginx Conf %s failed, can not save the file\n", err)
			return err
		}

		fmt.Printf("Formatter Nginx Conf %s Successed\n", rel)
	}
	return nil
}

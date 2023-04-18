package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

			modifiedData, err := fn(string(data))
			if err != nil {
				return err
			}

			err = os.WriteFile(path, []byte(modifiedData), info.Mode())
			if err != nil {
				return err
			}

			fmt.Printf("Formatter Nginx Conf %s Successed\n", path)
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

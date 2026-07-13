package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	sanitizeRe = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
)

func uniquePath(dir, file, ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	fname := file + "." + ext

	count := 0

	for {
		if _, err := os.Stat(filepath.Join(dir, fname)); os.IsNotExist(err) {
			break
		} else {
			count++
			fname = fmt.Sprintf("%s (%d).%s", file, count, ext)
		}
	}

	return filepath.Join(dir, fname)
}

func sanitizeFilename(filename string) string {
	v := sanitizeRe.ReplaceAllString(filename, "_")
	return strings.TrimSpace(v)
}

func validateInputs(dir, name, ext string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("dir path not provided")
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("file name not provided")
	}
	if strings.TrimSpace(ext) == "" {
		return fmt.Errorf("file ext not provided")
	}
	return nil
}

func ConstructPathname(dir, name, ext string) (string, error) {
	if err := validateInputs(dir, name, ext); err != nil {
		return "", err
	}

	name = sanitizeFilename(name)

	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", dir)
	}

	return uniquePath(dir, name, ext), nil
}

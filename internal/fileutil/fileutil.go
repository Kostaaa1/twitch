package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func walkDir(dstpath, filename, ext string) string {
	var fname string
	if strings.HasPrefix(ext, ".") {
		fname = fmt.Sprintf("%s%s", filename, ext)
	} else {
		fname = fmt.Sprintf("%s.%s", filename, ext)
	}

	// ext := filepath.Ext(filename)
	// if ext == "" {
	// 	panic("ext cannot be empty")
	// }

	count := 0

	for {
		if _, err := os.Stat(filepath.Join(dstpath, fname)); os.IsNotExist(err) {
			break
		} else {
			count++
			fname = fmt.Sprintf("%s (%d).%s", filename, count, ext)
		}
	}

	return filepath.Join(dstpath, fname)
}

func sanitizeFilename(filename string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	v := re.ReplaceAllString(filename, "_")
	return strings.TrimSpace(v)
}

func ConstructPathname(dstPath, filename, ext string) (string, error) {
	if dstPath == "" {
		return "", fmt.Errorf("the output path was not provided. Add output either by -output flag or add it via twitch_config.json (outputPath)")
	}

	filename = sanitizeFilename(filename)

	info, err := os.Stat(dstPath)
	if os.IsNotExist(err) {
		if filepath.Ext(dstPath) != "" {
			dir := filepath.Dir(dstPath)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return "", fmt.Errorf("directory does not exist: %s", dir)
			}
			dir, fname := filepath.Split(dstPath)
			return walkDir(dir, fname, ext), nil
		}
		return "", fmt.Errorf("path does not exist: %s", dstPath)
	}

	if info.IsDir() {
		return walkDir(dstPath, filename, ext), nil
	}

	return "", fmt.Errorf("this path already exists %s: ", dstPath)
}

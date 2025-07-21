package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func newWalkPath(dstpath, filename, ext string) string {
	fname := fmt.Sprintf("%s.%s", filename, ext)
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

func constructPathname(dstPath, filename, ext string) (string, error) {
	if dstPath == "" {
		return "", fmt.Errorf("the output path was not provided. Add output either by -output flag or add it via twitch_config.json (outputPath)")
	}

	info, err := os.Stat(dstPath)

	if os.IsNotExist(err) {
		if filepath.Ext(dstPath) != "" {
			dir := filepath.Dir(dstPath)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return "", fmt.Errorf("directory does not exist: %s", dir)
			}
			dir, fname := filepath.Split(dstPath)
			ext := filepath.Ext(fname)
			return newWalkPath(dir, fname, ext), nil
		}
		return "", fmt.Errorf("path does not exist: %s", dstPath)
	}

	if info.IsDir() {
		return newWalkPath(dstPath, filename, ext), nil
	}

	return "", fmt.Errorf("this path already exists %s: ", dstPath)
}

func sanitizeFilename(filename string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	return re.ReplaceAllString(filename, "_")
}

func CreateFile(dir, filename, ext string) (*os.File, error) {
	path, err := constructPathname(dir, sanitizeFilename(filename), ext)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return f, nil
}

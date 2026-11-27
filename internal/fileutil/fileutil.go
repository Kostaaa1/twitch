package fileutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	sanitizeRe = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
)

type PathError fs.PathError

func (e PathError) Error() string {
	if e.Op == "" {
		e.Op = "failed to parse provided pathname"
	}
	return e.Op + " " + e.Path + ": " + e.Err.Error()
}

func sanitizeFilename(filename string) string {
	v := sanitizeRe.ReplaceAllString(filename, "_")
	return strings.TrimSpace(v)
}

func validateInputs(dir, name, ext string) error {
	if dir == "" {
		return fmt.Errorf("dir path not provided")
	}
	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		return err
	}
	if name == "" {
		return fmt.Errorf("file name not provided")
	}
	if ext == "" {
		return fmt.Errorf("file ext not provided")
	}
	return nil
}

func open(dir, fname, ext string) (*os.File, error) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	flag := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	var mode os.FileMode = 0644
	count := 0

	fname, _ = strings.CutSuffix(fname, filepath.Ext(fname))

	for {
		var checkname string
		if count > 0 {
			checkname = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", fname, count, ext))
		} else {
			checkname = filepath.Join(dir, fname+ext)
		}

		f, err := os.OpenFile(checkname, flag, mode)
		if err != nil {
			if os.IsExist(err) {
				count++
				continue
			}
			return nil, err
		}

		return f, nil
	}
}

func Open(dir, fname, ext string) (*os.File, error) {
	dir = strings.TrimSpace(dir)
	fname = strings.TrimSpace(fname)
	ext = strings.TrimSpace(ext)

	if err := validateInputs(dir, fname, ext); err != nil {
		return nil, err
	}

	fname = sanitizeFilename(fname)

	return open(dir, fname, ext)
}

// type TemplateData struct {
// 	Channel string
// 	Clipper string // clips only
// 	Name    string // title
// 	ID      string // VOD id or clip slug
// 	Quality string
// 	Date    time.Time // created at (VOD/clip) or start time (stream)
// }

func ParsePath(pathname string, forceDir bool) error {
	if pathname == "" {
		return PathError{Path: "", Err: errors.New("path is not provided")}
	}

	stat, err := os.Stat(pathname)
	if err != nil {
		if forceDir && os.IsNotExist(err) {
			return os.MkdirAll(pathname, 0755)
		}
		return err
	}

	if !stat.IsDir() {
		return PathError{Path: pathname, Err: errors.New("provided path must be dir")}
	}

	return nil
}

// type PathParts struct {
// 	Dir  string
// 	Name string
// 	Ext  string
// }

// func ParsePath(pathname string) (*PathParts, error) {
// 	if pathname == "" {
// 		return nil, PathError{Path: "", Err: errors.New("pathname is not provided")}
// 	}

// 	pp := new(PathParts)

// 	stat, err := os.Stat(pathname)
// 	if os.IsNotExist(err) {
// 		dir, filename := filepath.Split(pathname)
// 		if _, err := os.Stat(dir); err != nil {
// 			if os.IsNotExist(err) {
// 				return nil, PathError{Path: dir, Err: errors.New("dir does not exist")}
// 			}
// 			return nil, PathError{Path: dir, Err: err}
// 		}
// 		pp.Dir = dir
// 		pp.Name = filename
// 		return pp, nil
// 	}

// 	if err != nil {
// 		return nil, err
// 	}

// 	if stat.IsDir() {
// 		pp.Name = ""
// 		pp.Dir = pathname
// 		return pp, nil
// 	}

// 	dir, file := filepath.Split(pathname)
// 	pp.Dir = dir
// 	pp.Name = file

// 	return pp, nil
// }

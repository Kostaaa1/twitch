package utils

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func GetCurrentTimeFormatted() string {
	now := time.Now()
	timestamp := now.UnixNano() / int64(time.Millisecond)
	formattedTime := ParseTimestamp(fmt.Sprintf("%d", timestamp))
	return formattedTime
}

func ParseTimestamp(v string) string {
	timestamp, _ := strconv.ParseInt(v, 10, 64)
	seconds := timestamp / 1000
	t := time.Unix(seconds, 0)
	formatted := t.Format("03:04")
	return formatted
}

func Capitalize(v string) string {
	return strings.ToUpper(v[:1]) + v[1:]
}

func RemoveCursor() {
	fmt.Printf("\033[?25l")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Printf("\033[?25h")
		os.Exit(0)
	}()
}

func CreateServingID() string {
	w := strings.Split("0123456789abcdefghijklmnopqrstuvwxyz", "")
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	id := ""
	for i := 0; i < 32; i++ {
		id += w[r.Intn(len(w))]
	}
	return id
}

func ConstructPathname(dstPath, filename, quality string) (string, error) {
	if dstPath == "" {
		return "", fmt.Errorf("the output path was not provided. Add output either by -output flag or add it via config.json (outputPath)")
	}

	info, err := os.Stat(dstPath)
	if os.IsNotExist(err) {
		if filepath.Ext(dstPath) != "" {
			dir := filepath.Dir(dstPath)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return "", fmt.Errorf("directory does not exist: %s", dir)
			}
			return dstPath, nil
		}
		return "", fmt.Errorf("path does not exist: %s", dstPath)
	}

	if info.IsDir() {
		// fileID := CreateServingID()
		fname := fmt.Sprintf("%s.%s", filename, "mp4")
		newpath := filepath.Join(dstPath, fname)
		return newpath, nil
	}

	return "", fmt.Errorf("this path already exists %s: ", dstPath)
}

func ChangeImageResolution(imgURL string, w, h int) (string, error) {
	parsedURL, err := url.Parse(imgURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse the image URL: %s", err)
	}

	lastSlashIndex := strings.LastIndex(parsedURL.Path, "/")
	if lastSlashIndex == -1 {
		return "", fmt.Errorf("error while changing the image resolution")
	}

	baseURL := fmt.Sprintf("https://%s%s", parsedURL.Host, parsedURL.Path[:lastSlashIndex])
	imageFilename := parsedURL.Path[lastSlashIndex+1:]
	imgParts := strings.Split(imageFilename, "-")

	if len(imgParts) > 1 {
		fileName := imgParts[0]
		ext := filepath.Ext(imageFilename)
		f := fmt.Sprintf("%s/%s-%dx%d%s", baseURL, fileName, w, h, ext)
		return f, nil
	}

	return "", nil
}

func SegmentFileName(segmentURL string) string {
	parts := strings.Split(segmentURL, "/")
	return parts[len(parts)-1]
}

func ConcatenateSegments(outputFile io.Writer, segments []string, tempDir string) error {
	for _, tsFile := range segments {
		if !strings.HasSuffix(tsFile, ".ts") {
			continue
		}

		tempFilePath := fmt.Sprintf("%s/%s", tempDir, SegmentFileName(tsFile))

		tempFile, err := os.Open(tempFilePath)
		if err != nil {
			return fmt.Errorf("failed to open temp file %s: %w", tempFilePath, err)
		}

		if _, err := io.Copy(outputFile, tempFile); err != nil {
			tempFile.Close()
			return fmt.Errorf("failed to write segment to output file: %w", err)
		}
		tempFile.Close()
	}
	return nil
}

func DownloadSegmentToFile(segmentURL, tempFilePath string) (int64, error) {
	resp, err := http.Get(segmentURL)
	if err != nil {
		return 0, fmt.Errorf("failed to get response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-OK response: %s", resp.Status)
	}

	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	return io.Copy(tempFile, resp.Body)

}

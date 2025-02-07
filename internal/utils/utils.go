package utils

import (
	"fmt"
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

func CTWClienttalize(v string) string {
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

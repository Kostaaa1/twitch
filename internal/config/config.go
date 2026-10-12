package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix/auth"
)

type Config struct {
	ForceSecuredCreds bool            `json:"force_secure_credentials"`
	User              helix.User      `json:"user"`
	Downloader        Downloader      `json:"downloader"`
	CommandLineChat   CommandLineChat `json:"chat"`
	OAuthCreds        auth.OAuthCreds `json:"oauth_credentials"`
}

type CommandLineChat struct {
	OpenedChats    []string `json:"opened_chats"`
	ShowTimestamps bool     `json:"show_timestamps"`
	Colors         Colors   `json:"colors"`
}

type Downloader struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	Output          string `json:"output"`
}

type Messages struct {
	Announcement string `json:"announcement"`
	First        string `json:"first"`
	Original     string `json:"original"`
	Raid         string `json:"raid"`
	Sub          string `json:"sub"`
}

type Icons struct {
	Broadcaster string `json:"broadcaster"`
	Mod         string `json:"mod"`
	Staff       string `json:"staff"`
	Vip         string `json:"vip"`
}

type Colors struct {
	Primary   string   `json:"primary"`
	Secondary string   `json:"secondary"`
	Danger    string   `json:"danger"`
	Border    string   `json:"border"`
	Icons     Icons    `json:"icons"`
	Messages  Messages `json:"messages"`
	Timestamp string   `json:"timestamp"`
}

func Default() *Config {
	return &Config{
		ForceSecuredCreds: true,
		OAuthCreds: auth.OAuthCreds{
			ClientID:     "",
			ClientSecret: "",
			RedirectURL:  "",
			AppToken:     nil,
			UserToken:    nil,
		},
		User: helix.User{
			BroadcasterType: "",
			CreatedAt:       "",
			Description:     "",
			DisplayName:     "",
			ID:              "",
			Login:           "",
			ProfileImageURL: "",
			OfflineImageURL: "",
			Type:            "",
		},
		Downloader: Downloader{
			IsFFmpegEnabled: false,
			Output:          "",
		},
		CommandLineChat: CommandLineChat{
			OpenedChats:    []string{},
			ShowTimestamps: true,
			Colors: Colors{
				Primary:   "#8839ef",
				Secondary: "",
				Danger:    "#C92D05",
				Border:    "#8839ef",
				Icons: Icons{
					Broadcaster: "#d20f39",
					Mod:         "#29d416",
					Staff:       "#8839ef",
					Vip:         "#ea76cb",
				},
				Messages: Messages{
					Announcement: "#29d416",
					First:        "#ea76db",
					Original:     "#fff",
					Raid:         "#fe640b",
					Sub:          "#04a5e5",
				},
				Timestamp: "",
			},
		},
	}
}

var (
	outputDirName  = "twitch"
	outputFileName = "twitch_config.json"
)

func DefaultDirPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	if dir != "" {
		return filepath.Join(dir, outputDirName), nil
	}
	return "", errors.New("couldn't find the path for .config")
}

func getFullPath() (string, error) {
	dir, err := DefaultDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, outputFileName), nil
}

func Get() (*Config, error) {
	path, err := getFullPath()
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		// if errors.Is(err, os.ErrNotExist) {
		// 	cfg := defaultConfig()
		// 	if err := Save(cfg); err != nil {
		// 		return nil, err
		// 	}
		// 	return cfg, nil
		// }
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	if cfg.ForceSecuredCreds == true {
		cfg.OAuthCreds.Read()
	}

	return &cfg, nil
}

func Save(c *Config) error {
	p, err := getFullPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0644)
}

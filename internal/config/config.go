package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type User struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	Email           string `json:"email"`
	CreatedAt       string `json:"created_at"`
}

type Chat struct {
	OpenedChats    []string `json:"opened_chats"`
	ShowTimestamps bool     `json:"show_timestamps"`
	Colors         Colors   `json:"colors"`
}

type Downloader struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `json:"show_spinner"`
	Output          string `json:"output"`
	SpinnerModel    string `json:"spinner_model"`
	SkipAds         bool   `json:"skip_ads"`
}

type Creds struct {
	RefreshToken string   `json:"refresh_token"`
	AccessToken  string   `json:"access_token"`
	ClientID     string   `json:"client_id"`
	ExpiresIn    int      `json:"expires_in"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	TokenType    string   `json:"token_type"`
	Scope        []string `json:"scope"`
}

type Config struct {
	User       User       `json:"user"`
	Downloader Downloader `json:"downloader"`
	Chat       Chat       `json:"chat"`
	Creds      Creds      `json:"creds"`
}

type Colors struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Danger    string `json:"danger"`
	Border    string `json:"border"`
	Icons     struct {
		Broadcaster string `json:"broadcaster"`
		Mod         string `json:"mod"`
		Staff       string `json:"staff"`
		Vip         string `json:"vip"`
	} `json:"icons"`
	Messages struct {
		Announcement string `json:"announcement"`
		First        string `json:"first"`
		Original     string `json:"original"`
		Raid         string `json:"raid"`
		Sub          string `json:"sub"`
	} `json:"messages"`
	Timestamp string `json:"timestamp"`
}

func initConfigData() Config {
	return Config{
		User: User{
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
		Creds: Creds{
			AccessToken:  "",
			ClientID:     "",
			RefreshToken: "",
			RedirectURL:  "",
			ClientSecret: "",
			ExpiresIn:    0,
			TokenType:    "",
			Scope:        []string{},
		},
		Downloader: Downloader{
			IsFFmpegEnabled: false,
			ShowSpinner:     true,
			Output:          "",
			SpinnerModel:    "dot",
			SkipAds:         true,
		},
		Chat: Chat{
			OpenedChats:    []string{},
			ShowTimestamps: true,
			Colors: Colors{
				Primary:   "#8839ef",
				Secondary: "",
				Danger:    "#C92D05",
				Border:    "#8839ef",
				Icons: struct {
					Broadcaster string `json:"broadcaster"`
					Mod         string `json:"mod"`
					Staff       string `json:"staff"`
					Vip         string `json:"vip"`
				}{
					Broadcaster: "#d20f39",
					Mod:         "#40a02b",
					Staff:       "#8839ef",
					Vip:         "#ea76cb",
				},
				Messages: struct {
					Announcement string `json:"announcement"`
					First        string `json:"first"`
					Original     string `json:"original"`
					Raid         string `json:"raid"`
					Sub          string `json:"sub"`
				}{
					Announcement: "#40a02b",
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

// TODO: improve this
func GetConfigPath() (string, error) {
	configPath := os.Getenv("TWITCH_CONFIG_PATH")

	if configPath == "" {
		execPath, err := os.Executable()
		if err != nil || strings.HasPrefix(execPath, "/tmp") {
			wd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			return filepath.Join(wd, "twitch_config.json"), err
		}
		execDir := filepath.Dir(execPath)
		configPath = filepath.Join(execDir, "twitch_config.json")
	}

	return configPath, nil
}

func Save(fpath string, conf *Config) error {
	if _, err := os.Stat(fpath); err != nil {
		return err
	}
	b, err := json.MarshalIndent(conf, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal config bytes: %v\n", err)
	}
	return os.WriteFile(fpath, b, 0644)
}

func Get() (*Config, error) {
	var data Config

	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		data = initConfigData()
		b, err := json.MarshalIndent(data, "", " ")
		if err != nil {
			return nil, err
		}

		if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
			return nil, err
		}

		if err := os.WriteFile(configPath, b, 0644); err != nil {
			return nil, err
		}

		return &data, nil
	} else if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// func ValidateUserCreds() error {
// 	cfg, err := Get()
// 	if err != nil {
// 		return err
// 	}
// 	errors := []string{}
// 	if cfg.Creds.AccessToken == "" {
// 		errors = append(errors, "AccessToken")
// 	}
// 	if cfg.Creds.ClientSecret == "" {
// 		errors = append(errors, "ClientSecret")
// 	}
// 	if cfg.Creds.ClientID == "" {
// 		errors = append(errors, "ClientID")
// 	}
// 	if len(errors) > 0 {
// 		for _, err := range errors {
// 			msg := fmt.Sprintf("missing %s from twith_config.json", err)
// 			return fmt.Errorf(msg)
// 		}
// 	}
// 	return nil
// }

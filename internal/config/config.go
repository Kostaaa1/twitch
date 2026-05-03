package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "embed"

	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
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

type CommandLineChat struct {
	OpenedChats    []string `json:"opened_chats"`
	ShowTimestamps bool     `json:"show_timestamps"`
	Colors         Colors   `json:"colors"`
}

type Config struct {
	User            User              `json:"user"`
	Downloader      downloader.Config `json:"downloader"`
	CommandLineChat CommandLineChat   `json:"chat"`
	OAuthCreds      twitch.OAuthCreds `json:"creds"`
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
		OAuthCreds: twitch.OAuthCreds{
			AccessToken:  "",
			ClientID:     "",
			RefreshToken: "",
			RedirectURL:  "",
			ClientSecret: "",
			ExpiresIn:    0,
			TokenType:    "",
			Scope:        []string{},
		},
		Downloader: downloader.Config{
			IsFFmpegEnabled: false,
			ShowSpinner:     true,
			Output:          "",
			// Spinner: downloader.SpinnerConfig{
			// 	Model:       "dot",
			// 	TwitchColor: "#8839ef",
			// 	KickColor:   "#29d416",
			// },
		},
		CommandLineChat: CommandLineChat{
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
					Mod:         "#29d416",
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

func getConfigPath() (string, error) {
	filename := "config.json"

	// execPath, err := os.Executable()
	// if err == nil && strings.HasPrefix(execPath, os.TempDir()) {
	// if strings.HasPrefix(execPath, os.TempDir()) {
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, filename), nil
	}
	// }

	// configDir, err := os.UserConfigDir()
	// if err == nil {
	// 	cfgPath := filepath.Join(configDir, "twitch", filename)
	// 	return cfgPath, nil
	// }

	// if envPath := os.Getenv("TWITCH_CONFIG"); envPath != "" {
	// 	return envPath, nil
	// }

	return "", errors.New("config file path not found")
}

func (conf *Config) Save() error {
	fpath, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(fpath); err != nil {
		return err
	}

	b, err := json.MarshalIndent(conf, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal config bytes: %v\n", err)
	}

	return os.WriteFile(fpath, b, 0644)
}

func Read() (*Config, error) {
	var data Config

	configPath, err := getConfigPath()
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

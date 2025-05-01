package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type UserConfig struct {
	BroadcasterType string    `json:"broadcaster_type"`
	CreatedAt       time.Time `json:"created_at"`
	Description     string    `json:"description"`
	DisplayName     string    `json:"display_name"`
	ID              string    `json:"id"`
	Login           string    `json:"login"`
	OfflineImageUrl string    `json:"offline_image_url"`
	ProfileImageUrl string    `json:"profile_image_url"`
	Type            string    `json:"type"`
	Creds           Creds     `json:"creds"`
}

type Creds struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

type ChatConfig struct {
	OpenedChats    []string `json:"opened_chats"`
	ShowTimestamps bool     `json:"show_timestamps"`
	Colors         Colors   `json:"colors"`
}

type Downloader struct {
	IsFFmpegEnabled bool   `json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `json:"show_spinner"`
	Output          string `json:"output"`
	SpinnerModel    string `json:"spinner_model"`
}

type Data struct {
	User       UserConfig `json:"user"`
	Downloader Downloader `json:"downloader"`
	Chat       ChatConfig `json:"chat"`
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

func InitConfigData() Data {
	defaultCreatedAt, _ := time.Parse(time.RFC3339, "2023-10-18T21:12:53Z")
	return Data{
		User: UserConfig{
			BroadcasterType: "",
			CreatedAt:       defaultCreatedAt,
			Description:     "",
			DisplayName:     "",
			ID:              "",
			Login:           "",
			OfflineImageUrl: "",
			ProfileImageUrl: "",
			Type:            "",
			Creds: Creds{
				AccessToken:  "",
				ClientID:     "",
				ClientSecret: "",
				RefreshToken: "",
				RedirectURL:  "",
			},
		},
		Downloader: Downloader{
			IsFFmpegEnabled: false,
			ShowSpinner:     true,
			Output:          "",
			SpinnerModel:    "dot",
		},
		Chat: ChatConfig{
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

func getConfigPath() (string, error) {
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

func Get() (*Data, error) {
	var data Data

	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			data = InitConfigData()

			b, err := json.MarshalIndent(data, "", " ")
			if err != nil {
				return nil, err
			}

			f, err := os.Create(configPath)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			if _, err := f.Write(b); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	viper.SetConfigName("twitch_config")
	viper.SetConfigType("json")
	viper.AddConfigPath(filepath.Dir(configPath))

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	viper.Unmarshal(&data)

	return &data, nil
}

// func ValidateUserCreds() error {
// 	cfg, err := Get()
// 	if err != nil {
// 		return err
// 	}
// 	errors := []string{}
// 	if cfg.User.Creds.AccessToken == "" {
// 		errors = append(errors, "AccessToken")
// 	}
// 	if cfg.User.Creds.ClientSecret == "" {
// 		errors = append(errors, "ClientSecret")
// 	}
// 	if cfg.User.Creds.ClientID == "" {
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

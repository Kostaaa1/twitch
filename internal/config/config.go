package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
	"github.com/spf13/viper"
)

type User struct {
	ID              string `mapstructure:"id" json:"id"`
	Login           string `mapstructure:"login" json:"login"`
	DisplayName     string `mapstructure:"display_name" json:"display_name"`
	Type            string `mapstructure:"type" json:"type"`
	BroadcasterType string `mapstructure:"broadcaster_type" json:"broadcaster_type"`
	Description     string `mapstructure:"description" json:"description"`
	ProfileImageURL string `mapstructure:"profile_image_url" json:"profile_image_url"`
	OfflineImageURL string `mapstructure:"offline_image_url" json:"offline_image_url"`
	ViewCount       int    `mapstructure:"view_count" json:"view_count"`
	Email           string `mapstructure:"email" json:"email"`
	CreatedAt       string `mapstructure:"created_at" json:"created_at"`
}

type CommandLineChat struct {
	OpenedChats    []string `mapstructure:"opened_chats" json:"opened_chats"`
	ShowTimestamps bool     `mapstructure:"show_timestamps" json:"show_timestamps"`
	Colors         Colors   `mapstructure:"colors" json:"colors"`
}

type Downloader struct {
	IsFFmpegEnabled bool   `mapstructure:"is_ffmpeg_enabled" json:"is_ffmpeg_enabled"`
	ShowSpinner     bool   `mapstructure:"show_spinner" json:"show_spinner"`
	Output          string `mapstructure:"output" json:"output"`
}

type Config struct {
	User            User             `mapstructure:"user" json:"user"`
	Downloader      Downloader       `mapstructure:"downloader" json:"downloader"`
	CommandLineChat CommandLineChat  `mapstructure:"chat" json:"chat"`
	OAuthCreds      helix.OAuthCreds `mapstructure:"creds" json:"creds"`
}

type Messages struct {
	Announcement string `mapstructure:"announcement" json:"announcement"`
	First        string `mapstructure:"first" json:"first"`
	Original     string `mapstructure:"original" json:"original"`
	Raid         string `mapstructure:"raid" json:"raid"`
	Sub          string `mapstructure:"sub" json:"sub"`
}

type Icons struct {
	Broadcaster string `mapstructure:"broadcaster" json:"broadcaster"`
	Mod         string `mapstructure:"mod" json:"mod"`
	Staff       string `mapstructure:"staff" json:"staff"`
	Vip         string `mapstructure:"vip" json:"vip"`
}

type Colors struct {
	Primary   string   `mapstructure:"primary" json:"primary"`
	Secondary string   `mapstructure:"secondary" json:"secondary"`
	Danger    string   `mapstructure:"danger" json:"danger"`
	Border    string   `mapstructure:"border" json:"border"`
	Icons     Icons    `mapstructure:"icons" json:"icons"`
	Messages  Messages `mapstructure:"messages" json:"messages"`
	Timestamp string   `mapstructure:"timestamp" json:"timestamp"`
}

func defaultConfig() *Config {
	return &Config{
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
		Downloader: Downloader{
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

func Dir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	if dir != "" {
		return filepath.Join(dir, "twitch"), nil
	}

	return "", errors.New("couldn't find the path for .config")
}

func Set(c *Config) {
	viper.Set("user", c.User)
	viper.Set("downloader", c.Downloader)
	viper.Set("creds", c.OAuthCreds)
	viper.Set("chat", c.CommandLineChat)
}

func Save() error { return viper.WriteConfig() }

func Get() (*Config, error) {
	confDir, err := Dir()
	if err != nil {
		return nil, err
	}

	viper.SetConfigName("twitch_config")
	viper.SetConfigType("json")

	viper.AddConfigPath(confDir)
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			cfg := defaultConfig()
			Set(cfg)
			viper.SafeWriteConfig()
			return cfg, nil
		}
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	fmt.Println("CONFIG", config)

	return &config, nil
}

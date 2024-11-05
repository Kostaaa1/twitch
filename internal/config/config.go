package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type UserConfig struct {
	BroadcasterType string    `json:"broadcasterType"`
	CreatedAt       time.Time `json:"createdAt"`
	Description     string    `json:"description"`
	DisplayName     string    `json:"displayName"`
	ID              string    `json:"id"`
	Login           string    `json:"login"`
	OfflineImageUrl string    `json:"offlineImageUrl"`
	ProfileImageUrl string    `json:"profileImageUrl"`
	ShowTimestamps  bool      `json:"showTimestamps"`
	Type            string    `json:"type"`
	Creds           struct {
		AccessToken  string `json:"accessToken"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	} `json:"creds"`
}

type Config struct {
	OpenedChats []string `json:"openedChats"`
	Colors      Colors   `json:"colors"`
}

type Downloader struct {
	ShowSpinner bool   `json:"showSpinner"`
	Output      string `json:"output"`
}

type Data struct {
	User       UserConfig `json:"user"`
	Downloader Downloader `json:"downloader"`
	Chat       Config     `json:"chat"`
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

func InitData() Data {
	defaultCreatedAt, _ := time.Parse(time.RFC3339, "2023-10-18T21:12:53Z")
	return Data{
		User: struct {
			BroadcasterType string    `json:"broadcasterType"`
			CreatedAt       time.Time `json:"createdAt"`
			Description     string    `json:"description"`
			DisplayName     string    `json:"displayName"`
			ID              string    `json:"id"`
			Login           string    `json:"login"`
			OfflineImageUrl string    `json:"offlineImageUrl"`
			ProfileImageUrl string    `json:"profileImageUrl"`
			ShowTimestamps  bool      `json:"showTimestamps"`
			Type            string    `json:"type"`
			Creds           struct {
				AccessToken  string `json:"accessToken"`
				ClientID     string `json:"clientId"`
				ClientSecret string `json:"clientSecret"`
			} `json:"creds"`
		}{
			BroadcasterType: "",
			CreatedAt:       defaultCreatedAt,
			Description:     "",
			DisplayName:     "slorpglorpski",
			ID:              "972597583",
			Login:           "slorpglorpski",
			OfflineImageUrl: "",
			ProfileImageUrl: "https://static-cdn.jtvnw.net/user-default-pictures-uv/de130ab0-def7-11e9-b668-784f43822e80-profile_image-300x300.png",
			ShowTimestamps:  true,
			Type:            "",
			Creds: struct {
				AccessToken  string `json:"accessToken"`
				ClientID     string `json:"clientId"`
				ClientSecret string `json:"clientSecret"`
			}{
				AccessToken:  "5891t7kqiu6tfpws77082wmjbm7t5w",
				ClientID:     "xdovt7giiklpnx7bf0ewnh892ce8pf",
				ClientSecret: "4fi3gmc1nzigjii2lf7plwc2ogtd17",
			},
		},
		Downloader: struct {
			ShowSpinner bool   `json:"showSpinner"`
			Output      string `json:"output"`
		}{
			ShowSpinner: true,
			Output:      "/mnt/c/Users/Kosta/Clips",
		},
		Chat: struct {
			OpenedChats []string `json:"openedChats"`
			Colors      Colors   `json:"colors"`
		}{
			OpenedChats: []string{},
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

func Get() (*Data, error) {
	var data Data

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	p := filepath.Join(wd, "config.json")

	_, err = os.Stat(p)

	if os.IsNotExist(err) {
		data = InitData()

		b, err := json.MarshalIndent(data, "", " ")
		if err != nil {
			return nil, err
		}

		f, err := os.Create(p)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if _, err := f.Write(b); err != nil {
			return nil, err
		}
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("json")
		viper.AddConfigPath(".")

		err := viper.ReadInConfig()
		if err != nil {
			return nil, err
		}
		viper.Unmarshal(&data)
	}

	return &data, nil
}

func ValidateCreds() error {
	cfg, err := Get()
	if err != nil {
		return err
	}
	errors := []string{}

	if cfg.User.Creds.AccessToken == "" {
		errors = append(errors, "AccessToken")
	}

	if cfg.User.Creds.ClientSecret == "" {
		errors = append(errors, "ClientSecret")
	}

	if cfg.User.Creds.ClientID == "" {
		errors = append(errors, "ClientID")
	}

	if len(errors) > 0 {
		for _, err := range errors {
			msg := fmt.Sprintf("missing %s from config.json", err)
			return fmt.Errorf(msg)
		}
	}
	return nil
}

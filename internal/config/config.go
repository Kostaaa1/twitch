package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Data struct {
	BroadcasterType string    `json:"broadcasterType"`
	Colors          Colors    `json:"colors"`
	CreatedAt       time.Time `json:"createdAt"`
	Creds           struct {
		AccessToken  string `json:"accessToken"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	} `json:"creds"`
	Description     string   `json:"description"`
	DisplayName     string   `json:"displayName"`
	ID              string   `json:"id"`
	Login           string   `json:"login"`
	OfflineImageURL string   `json:"offlineImageUrl"`
	OpenedChats     []string `json:"openedChats"`
	Paths           struct {
		ChromePath string `json:"chromePath"`
		OutputPath string `json:"outputPath"`
	} `json:"paths"`
	ProfileImageURL string `json:"profileImageUrl"`
	ShowTimestamps  bool   `json:"showTimestamps"`
	Type            string `json:"type"`
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

func Get() (*Data, error) {
	var data Data

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	p := filepath.Join(wd, "config.json")

	_, err = os.Stat(p)
	if os.IsNotExist(err) {
		f, err := os.Create(p)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		b, err := json.MarshalIndent(data, "", " ")
		if err != nil {
			return nil, err
		}

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
	if cfg.Creds.AccessToken == "" {
		errors = append(errors, "AccessToken")
	}
	if cfg.Creds.ClientSecret == "" {
		errors = append(errors, "ClientSecret")
	}
	if cfg.Creds.ClientID == "" {
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

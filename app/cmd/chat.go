/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"log"
	"net/http"

	"github.com/Kostaaa1/twitch/internal/cli/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"

	"github.com/Kostaaa1/twitch/pkg/twitch/helix"

	"github.com/spf13/cobra"
)

func ensureUserData(ctx context.Context, c *helix.Client, conf *config.Config) error {
	if conf.OAuthCreds.UserToken == nil {
		if err := c.Authorize(ctx, helix.AuthOpts{
			ForceVerify: true,
			Scopes:      chat.DefaultScopes(),
		}); err != nil {
			return err
		}
	}

	if conf.OAuthCreds.UserToken.Expired() {
		if err := c.RefreshAccessToken(ctx); err != nil {
			return err
		}
	}

	if conf.User.ID == "" {
		users, err := c.Users().Run(ctx)
		if err != nil {
			return err
		}

		user := users.Data[0]

		conf.User = config.User{
			ID:              user.ID,
			Login:           user.Login,
			DisplayName:     user.DisplayName,
			Type:            user.Type,
			BroadcasterType: user.BroadcasterType,
			Description:     user.Description,
			ProfileImageURL: user.ProfileImageURL,
			OfflineImageURL: user.OfflineImageURL,
			ViewCount:       user.ViewCount,
			Email:           user.Email,
			CreatedAt:       user.CreatedAt,
		}
	}

	return nil
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.Get()
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := config.Save(); err != nil {
				log.Fatal(err)
			}
		}()

		c := helix.New(
			http.DefaultClient,
			helix.WithOAuthCreds(&conf.OAuthCreds),
		)

		ctx := context.Background()

		if err := ensureUserData(ctx, c, conf); err != nil {
			log.Fatal(err)
		}

		if err := chat.Open(ctx, conf); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// chatCmd.PersistentFlags().String("foo", "", "A help for foo")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// chatCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

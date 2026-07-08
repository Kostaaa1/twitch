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

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.Read()
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := conf.Save(); err != nil {
				log.Fatal(err)
			}
		}()

		ctx := context.Background()

		helix := helix.New(
			http.DefaultClient,
			helix.WithOAuthCreds(&conf.OAuthCreds),
		)

		if err := chat.Open(ctx, helix, conf); err != nil {
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

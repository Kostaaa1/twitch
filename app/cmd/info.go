/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"log"
	"net/http"

	"github.com/Kostaaa1/twitch/internal/cli"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/spf13/cobra"
)

func runInfoCommand(args []string) error {
	tw := &twitch.Client{Gql: gql.New(http.DefaultClient)}

	limit := 20

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, channel := range args {
		about, err := tw.Gql.ChannelRoot_AboutPanel(ctx, channel)
		if err != nil {
			return err
		}
		videos, err := tw.Gql.FilterableVideoTower_Videos(ctx, channel, limit)
		if err != nil {
			return err
		}
		clips, err := tw.Gql.ClipsCardsUser(ctx, channel, limit, gql.AllTime)
		if err != nil {
			return err
		}
		cli.PrintChannel(about, videos, clips)
	}

	return nil
}

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("invalid usage: info <channel_name>")
		}

		if err := runInfoCommand(args); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
	// infoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// infoCmd.PersistentFlags().String("foo", "", "A help for foo")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// infoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

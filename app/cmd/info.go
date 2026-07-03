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

type args struct {
	clipsLimit  int
	videosLimit int
	criteria    string
}

var infoArgs args

func runInfoCommand(args []string) error {
	tw := &twitch.Client{Gql: gql.New(http.DefaultClient)}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, channel := range args {
		about, err := tw.Gql.ChannelRoot_AboutPanel(ctx, channel)
		if err != nil {
			return err
		}
		videos, err := tw.Gql.FilterableVideoTower_Videos(ctx, channel, infoArgs.videosLimit)
		if err != nil {
			return err
		}
		clips, err := tw.Gql.ClipsCardsUser(ctx, channel, infoArgs.clipsLimit, gql.LastMonth)
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
	infoCmd.PersistentFlags().IntVar(&infoArgs.videosLimit, "clips_limit", 20, "")
	infoCmd.PersistentFlags().IntVar(&infoArgs.clipsLimit, "vods_limit", 20, "")
	infoCmd.PersistentFlags().StringVarP(&infoArgs.criteria, "filter", "f", "LAST_WEEK", "")
}

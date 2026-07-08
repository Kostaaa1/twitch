/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Kostaaa1/twitch/internal/cli"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/downloader"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix/eventsub"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type T struct {
	output      string
	quality     string
	threads     int
	watch       bool
	start, end  time.Duration
	showSpinner bool
	verbose     bool
}

var cmdArgs T

func runTwitchBatchDownload(ctx context.Context, dl *downloader.Downloader, units []*downloader.Unit) error {
	g, ctx := errgroup.WithContext(ctx)
	if cmdArgs.threads > 0 {
		g.SetLimit(cmdArgs.threads)
	}
	for _, unit := range units {
		g.Go(func() error {
			dl.Download(ctx, unit)
			return nil
		})
	}
	g.Wait()
	return nil
}

func runTwitchEventSub(
	ctx context.Context,
	helix *helix.Client,
	dl *downloader.Downloader,
	units []*downloader.Unit,
) error {
	e, err := eventsub.WithWebsocket(
		ctx,
		helix,
		eventsub.WebsocketConnArgs{
			KeepaliveSeconds: 30,
			OnNotification: func(msg eventsub.EventSubMessage) {
				for _, unit := range units {
					if msg.Payload.Event.BroadcasterUserLogin == unit.ID {
						switch msg.Metadata.SubscriptionType {
						case eventsub.StreamOnline:
							// start downloading
							go dl.Download(ctx, unit)
						case eventsub.StreamOffline:
							// cancel downloading
						}
					}
				}
			},
		},
	)

	if err != nil {
		return err
	}

	// ----- Delete previous subscriptions since we are using wss its session bound ------
	// currentSubs, err := e.Subscriptions().Get().Run(ctx)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// for _, sub := range currentSubs.Data {
	// 	if err := e.Subscriptions().Delete(sub.ID).Run(ctx); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	for _, unit := range units {
		user, err := helix.Users().UserLogin(unit.ID).Run(ctx)
		if err != nil {
			return err
		}
		id := user.Data[0].ID
		ev1 := e.StreamOnlineEvent(id)
		events := []eventsub.Event{ev1}
		for _, event := range events {
			resp, err := e.Subscriptions().Create(event).Run(ctx)
			if err != nil {
				return err
			}
			_ = resp
			// print(resp)
		}
	}

	// verify
	subs, err := e.Subscriptions().Get().Run(ctx)
	if err != nil {
		return err
	}

	b, _ := json.MarshalIndent(subs, "", " ")
	fmt.Println(string(b))

	return e.Wait()
}

func runDownloadCmd(args []string) error {
	// if err := godotenv.Load(); err != nil {
	// 	log.Fatal(err)
	// }

	conf, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	// defer func() {
	// 	if err := conf.Save(); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	httpClient := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	gql := gql.New(httpClient)
	dl := downloader.New(gql, httpClient)

	units, err := cli.ParseUnits(
		args,
		cmdArgs.quality,
		cmdArgs.start,
		cmdArgs.end,
		cmdArgs.output,
	)
	if err != nil {
		return err
	}

	var spin *spinner.Model
	if cmdArgs.showSpinner {
		spin = spinner.New(ctx, spinner.WithCancelFunc(cancel), spinner.WithUnits(units))
		g.Go(func() error {
			spin.Run()
			cancel()
			return nil
		})
	}

	g.Go(func() error {
		downloadGroup, ctx := errgroup.WithContext(ctx)

		if len(units) > 0 {
			if spin != nil {
				dl.SetProgressNotifier(func(pm downloader.Progress) {
					ctxErr := ctx.Err()
					if ctxErr != nil {
						if errors.Is(ctxErr, context.Canceled) {
							return
						}
						pm.Error = errors.Join(pm.Error, ctxErr)
					}
					spin.Send(spinner.Message{
						ID:    pm.ID,
						Label: pm.Label,
						Bytes: pm.Bytes,
						Error: pm.Error,
						Done:  pm.Done,
						Total: pm.Total,
					})
				})
			}

			downloadGroup.Go(func() error {
				if cmdArgs.watch {
					helix := helix.New(httpClient, helix.WithOAuthCreds(&conf.OAuthCreds))
					return runTwitchEventSub(ctx, helix, dl, units)
				} else {
					return runTwitchBatchDownload(ctx, dl, units)
				}
			})
		}

		downloadGroup.Wait()

		// cancel()

		return nil
	})

	g.Wait()

	return nil
}

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDownloadCmd(args); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	cobra.OnInitialize()
	// downloadCmd.PersistentFlags().BoolVarP(&cmdArgs.verbose, "verbose", "v", false, "")
	// downloadCmd.PersistentFlags().IntVarP(&cmdArgs.threads, "threads", "t", 0, "")
	downloadCmd.PersistentFlags().StringVarP(&cmdArgs.output, "output", "o", "", "")
	downloadCmd.PersistentFlags().BoolVarP(&cmdArgs.watch, "watch", "w", false, "")
	downloadCmd.PersistentFlags().BoolVar(&cmdArgs.showSpinner, "spinner", true, "")
	downloadCmd.PersistentFlags().StringVarP(&cmdArgs.quality, "quality", "q", "best", "")
	downloadCmd.PersistentFlags().DurationVarP(&cmdArgs.start, "start", "s", 0, " attribution")
	downloadCmd.PersistentFlags().DurationVarP(&cmdArgs.end, "end", "e", 0, " attribution")
	rootCmd.AddCommand(downloadCmd)
}

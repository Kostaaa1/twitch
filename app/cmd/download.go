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

var (
	output      string
	quality     string
	threads     int
	watch       bool
	start, end  time.Duration
	showSpinner bool
	verbose     bool
)

func runTwitchBatchDownload(
	ctx context.Context,
	dl *downloader.Downloader,
	units []*downloader.Unit,
) error {
	g, ctx := errgroup.WithContext(ctx)
	if threads > 0 {
		g.SetLimit(threads)
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

	subs, err := e.Subscriptions().Get().Run(ctx)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(subs, "", " ")
	fmt.Println(string(b))

	return e.Wait()
}

func runDownloadCmd(args []string) error {
	cfg, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := config.Save(cfg); err != nil {
			log.Fatal(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	httpClient := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			// total number of idle connections that can be reused
			MaxIdleConns: 32,
			// number of idle conns that are kept for host (optimal for segment downloading)
			MaxIdleConnsPerHost: 16,
			// how long an idle conn is allowed to stay idle before being closed
			IdleConnTimeout: 90 * time.Second,
			//
			ForceAttemptHTTP2: true,
			// sets the number of max connections per host, meaning it will block until connection becomes idle upon requesting
			// MaxConnsPerHost: 1,
			// timeout limit for TLS handshake to establish
			// TLSHandshakeTimeout: time.Second * 10,
		},
	}

	gql := gql.New(httpClient)
	dl := downloader.New(gql, httpClient)

	units, err := cli.ParseUnits(
		args,
		quality,
		start,
		end,
		output,
	)

	if err != nil {
		return err
	}

	var spin *spinner.Model
	if showSpinner {
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
				if watch {
					helix := helix.New(
						httpClient,
						helix.WithOAuthCreds(&cfg.OAuthCreds),
					)
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
	downloadCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "")
	downloadCmd.PersistentFlags().BoolVarP(&watch, "watch", "w", false, "")
	downloadCmd.PersistentFlags().BoolVar(&showSpinner, "spinner", true, "")
	downloadCmd.PersistentFlags().StringVarP(&quality, "quality", "q", "best", "")
	downloadCmd.PersistentFlags().DurationVarP(&start, "start", "s", 0, " attribution")
	downloadCmd.PersistentFlags().DurationVarP(&end, "end", "e", 0, " attribution")
	rootCmd.AddCommand(downloadCmd)
}

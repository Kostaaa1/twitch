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
	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix/auth"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix/eventsub"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	threads    int
	output     string
	quality    string
	watch      bool
	start, end time.Duration
	verbose    int

	channels                     []string
	limitVideos, limitClips      int
	sortVideosBy, filterVideosBy string
	filterClipsBy                string
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
			_ = dl.Download(ctx, unit)
			return nil
		})
	}
	g.Wait()
	return nil
}

func runTwitchEventSub(
	ctx context.Context,
	creds *auth.OAuthCreds,
	dl *downloader.Downloader,
	units []*downloader.Unit,
) error {
	helix, err := helix.New(creds)
	if err != nil {
		return err
	}

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
	// 	return err
	// }
	// for _, sub := range currentSubs.Data {
	// 	if err := e.Subscriptions().Delete(sub.ID).Run(ctx); err != nil {
	// 		return err
	// 	}
	// }

	for _, unit := range units {
		user, err := helix.Users().UserLogin(unit.ID).Run(ctx)
		if err != nil {
			return err
		}

		ev := e.StreamOnlineEvent(user.Data[0].ID)

		events := []eventsub.Event{ev}
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

func initSpinner(
	ctx context.Context,
	cancel context.CancelFunc,
	twitchDl *downloader.Downloader,
	kickDl *kick.Downloader,
	twitchUnits []*downloader.Unit,
	kickUnits []*kick.Unit,
) *spinner.Model {
	spin := spinner.New(
		ctx,
		spinner.WithCancelFunc(cancel),
		spinner.WithUnits(twitchUnits),
		spinner.WithUnits(kickUnits),
	)

	kickNotifyFn := func(pm kick.Progress) {
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
	}
	kickDl.SetProgressNotifier(kickNotifyFn)

	twitchNotifyFn := func(pm downloader.Progress) {
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
	}
	twitchDl.SetProgressNotifier(twitchNotifyFn)

	return spin
}

func runDownloadCmd(args []string) error {
	fmt.Println(channels, limitClips, limitVideos, sortVideosBy)

	cfg, err := config.Get()
	if err != nil {
		return err
	}
	defer func() {
		if err := config.Save(cfg); err != nil {
			log.Fatal(err)
		}
	}()

	// f, err := config.OpenFileLogger()
	// if err != nil {
	// 	return err
	// }
	// defer f.Close()

	// l := logger.New(f, verbose)
	// http := &http.Client{
	// 	Transport: logger.NewLoggingTransport(http.DefaultTransport, l),
	// 	Timeout:   10 * time.Second,
	// }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if output == "" {
		output = cfg.Downloader.Output
	}

	pu := cli.ParseUnit{
		Inputs:     args,
		Quality:    quality,
		Start:      start,
		End:        end,
		OutputPath: output,
	}

	twitchUnits, kickUnits, err := cli.ParseUnits(pu)
	if err != nil {
		return err
	}

	mainGroup, ctx := errgroup.WithContext(ctx)

	http := http.DefaultClient
	dl := downloader.New(gql.New(http), http)
	kickDl := kick.New()

	spin := initSpinner(ctx, cancel, dl, kickDl, twitchUnits, kickUnits)
	if spin != nil {
		mainGroup.Go(func() error {
			spin.Run()
			cancel()
			return nil
		})
	}

	if len(kickUnits) > 0 {
		mainGroup.Go(func() error {
			g, ctx := errgroup.WithContext(ctx)
			if threads > 0 {
				g.SetLimit(threads)
			}
			for _, unit := range kickUnits {
				g.Go(func() error {
					_ = kickDl.Download(ctx, unit)
					return nil
				})
			}
			g.Wait()
			return nil
		})
	}

	if len(twitchUnits) > 0 {
		mainGroup.Go(func() error {
			downloadGroup, ctx := errgroup.WithContext(ctx)
			downloadGroup.Go(func() error {
				if watch {
					return runTwitchEventSub(ctx, &cfg.OAuthCreds, dl, twitchUnits)
				}
				return runTwitchBatchDownload(ctx, dl, twitchUnits)
			})
			downloadGroup.Wait()
			return nil
		})
	}

	mainGroup.Wait()

	return nil
}

var cmd = &cobra.Command{
	Use:   "download",
	Short: "download twitch media [livestreams, videos, clips]",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDownloadCmd(args); err != nil {
			log.Fatal(err)
		}
	},
}

func testwht() {
	gql := gql.New(http.DefaultClient)
	ctx := context.Background()
	for _, channel := range channels {
		fmt.Println("CHANNEL:", channel)
		gql.FilterableVideoTower_Videos(ctx, channel, limitVideos)
	}
}

func init() {
	flagset := cmd.PersistentFlags()
	flagset.CountVar(&verbose, "v", "")
	flagset.IntVarP(&threads, "threads", "t", 0, "")

	flagset.StringVarP(&output, "output", "o", "", "download destination")
	flagset.StringVarP(&quality, "quality", "q", "best", "quality for units")
	flagset.DurationVarP(&start, "start", "s", 0, "start timestamp for VOD units")
	flagset.DurationVarP(&end, "end", "e", 0, "end timestamp for VOD units")
	flagset.BoolVarP(&watch, "watch", "w", false, "watch Twitch channels and automatically start recording when livestreams go online; can be used as a background process")

	// new flags: per channel download
	// ./twitch download --channels=channelname_1.channelnme_2 --limit-clips=20 --filter-clips-by=all --limit-videos --list
	flagset.StringSliceVar(&channels, "channels", []string{}, "channels")
	flagset.IntVar(&limitClips, "limit-clips", 5, "limit number of clips for specified channel: default 5")
	flagset.IntVar(&limitVideos, "limit-videos", 1, "limit number of videos for specified channel: default 1")
	flagset.StringVar(&filterClipsBy, "filter-clips-by", "all", "filter clips by: 24h, 7d, 30d, all")
	// TIME | VIEWS
	flagset.StringVar(&sortVideosBy, "sort-videos-by", "date", "sort videos by: date, popular")
	// ARCHIVE | HIGHLIGHT | CLIP | LIVE | UPLOAD
	flagset.StringVar(&filterVideosBy, "filter-videos-by", "default", "filter videos by: default, past_broadcasts, highlights, uploads, collections")

	rootCmd.AddCommand(cmd)
}

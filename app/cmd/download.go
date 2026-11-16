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
	"github.com/Kostaaa1/twitch/internal/httputil"
	"github.com/Kostaaa1/twitch/internal/logger"
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

func collectUnitsForChannels(ctx context.Context, gqlClient *gql.Client) ([]*downloader.Unit, error) {
	units := make([]*downloader.Unit, 0)

	for _, channel := range channels {
		if limitVideos > 0 {
			data, err := gqlClient.FilterableVideoTower_Videos(ctx, gql.FilterableVideoTowerArgs{
				Limit:           limitVideos,
				Channel:         channel,
				BroadcasterType: gql.VideoBroadcastType(filterVideosBy),
				VideoSort:       gql.VideoSort(sortVideosBy),
			})
			if err != nil {
				return nil, err
			}
			for _, vod := range data.User.Videos.Edges {
				units = append(units, downloader.NewUnit(
					vod.Node.ID,
					downloader.WithQuality(quality),
					downloader.WithTimestamps(start, end),
					downloader.WithPathname(output),
				))
			}
		}

		if limitClips > 0 {
			data, err := gqlClient.ClipsCardsUser(ctx, gql.ClipsCardsUserArgs{
				Channel: channel,
				Limit:   limitClips,
				Filter:  gql.CriteriaFilter(filterClipsBy),
			})
			if err != nil {
				return nil, err
			}
			for _, clip := range data.User.Clips.Edges {
				units = append(units, downloader.NewUnit(
					clip.Node.Slug,
					downloader.WithQuality(quality),
					downloader.WithPathname(output),
				))
			}
		}
	}

	return units, nil
}

func runProcess(
	cfg *config.Config,
	httpClient *http.Client,
	twitchUnits []*downloader.Unit,
	kickUnits []*kick.Unit,
) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gqlClient := gql.New(httpClient)
	twitchDl := downloader.New(gqlClient, httpClient)
	kickDl := kick.New()

	if len(channels) > 0 {
		collected, err := collectUnitsForChannels(ctx, gqlClient)
		if err != nil {
			return err
		}
		twitchUnits = append(twitchUnits, collected...)
	}

	mainGroup, ctx := errgroup.WithContext(ctx)

	if cfg.Downloader.Spinner.Enabled && (len(twitchUnits) > 0 || len(kickUnits) > 0) {
		spin := initSpinner(ctx, cancel, twitchDl, kickDl, twitchUnits, kickUnits)
		if spin != nil {
			mainGroup.Go(func() error {
				spin.Run()
				cancel()
				return nil
			})
		}
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
					return runTwitchEventSub(ctx, &cfg.OAuthCreds, twitchDl, twitchUnits)
				}
				return runTwitchBatchDownload(ctx, twitchDl, twitchUnits)
			})
			downloadGroup.Wait()
			return nil
		})
	}

	mainGroup.Wait()

	return nil
}

func runDownloadCmd(args []string) error {
	cfg, err := config.Get()
	if err != nil {
		return err
	}
	defer func() {
		// TODO: avoid saving when not changed
		if err := config.Save(cfg); err != nil {
			log.Fatal(err)
		}
	}()

	if output == "" {
		output = cfg.Downloader.Output
	}

	f, err := config.OpenFileLogger()
	if err != nil {
		return err
	}
	defer f.Close()

	log := logger.New(f, verbose)

	httpClient := &http.Client{
		Timeout: 0,
		Transport: httputil.NewBrowserTransport(
			http.DefaultTransport,
			httputil.WithLog(log),
		),
	}

	twitchUnits, kickUnits, err := cli.ParseUnits(cli.ParseUnit{
		Inputs:     args,
		Quality:    quality,
		Start:      start,
		End:        end,
		OutputPath: output,
	})
	if err != nil {
		return err
	}

	return runProcess(cfg, httpClient, twitchUnits, kickUnits)
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

func init() {
	flagset := cmd.PersistentFlags()
	flagset.CountVarP(&verbose, "v", "v", "")
	flagset.IntVarP(&threads, "threads", "t", 0, "")

	flagset.StringVarP(&output, "output", "o", "", "download destination")
	flagset.StringVarP(&quality, "quality", "q", "best", "quality for units")
	flagset.DurationVarP(&start, "start", "s", 0, "start timestamp for VOD units")
	flagset.DurationVarP(&end, "end", "e", 0, "end timestamp for VOD units")
	flagset.BoolVarP(&watch, "watch", "w", false, "watch Twitch channels and automatically start recording when livestreams go online; can be used as a background process")
	// new flags: per channel download
	// ./twitch download --channels=channelname_1.channelnme_2 --limit-clips=20 --filter-clips-by=all --limit-videos
	flagset.StringSliceVar(&channels, "channels", []string{}, "channels")
	flagset.IntVar(&limitClips, "limit-clips", 0, "limit number of clips for specified channel: default 0, max 100")
	flagset.IntVar(&limitVideos, "limit-videos", 0, "limit number of videos for specified channel: default 0, max 100")
	flagset.StringVar(&filterClipsBy, "filter-clips-by", string(gql.LastDay), "filter clips by: last_day, last_week, last_month, all_time. default is 'last_day'")
	flagset.StringVar(&sortVideosBy, "sort-videos-by", string(gql.SortTime), "sort videos by: time, views. default is 'time'")
	flagset.StringVar(&filterVideosBy, "filter-videos-by", string(gql.VideoTypeArchive), "filter videos by: archive (past broadcasts), highlight, upload. default is 'archive'")
	rootCmd.AddCommand(cmd)
}

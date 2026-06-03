package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"

	"github.com/Kostaaa1/twitch/internal/cli"
	"github.com/Kostaaa1/twitch/internal/cli/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/downloader"
	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	conf, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}
	defer conf.Save()

	ctx := context.Background()
	opt := cli.ParseFlags(*conf)
	tw := twitch.NewClient(twitch.WithOAuthCreds(&conf.OAuthCreds))

	switch {
	case opt.Authorize:
		if err := tw.Helix.Authorize(
			ctx,
			helix.AuthOpts{
				ResponseType: helix.TokenResponseType,
			}); err != nil {
			log.Fatal(err)
		}
	case opt.Print:
		runPrint(ctx, tw)
	case len(os.Args) == 1:
		runChat(ctx, tw, conf)
	default:
		runDownloader(ctx, tw, conf, opt)
	}
}

func runDownloader(ctx context.Context, tw *twitch.Client, conf *config.Config, opt cli.Flag) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	units := opt.UnitsFromInput()
	twitchUnits, kickUnits := cli.FilterUnits(units)

	var spin *spinner.Model
	if conf.Downloader.ShowSpinner {
		spin = spinner.New(ctx, units, spinner.WithCancelFunc(cancel))
		g.Go(func() error {
			spin.Run()
			return nil
		})
	}

	g.Go(func() error {
		downloadGroup, ctx := errgroup.WithContext(ctx)
		if len(twitchUnits) > 0 {
			startTwitchDownloader(ctx, tw, spin, conf, opt, twitchUnits, downloadGroup)
		}
		if len(kickUnits) > 0 {
			startKickDownloader(ctx, spin, opt.Threads, kickUnits, downloadGroup)
		}

		if err := downloadGroup.Wait(); err == nil {
			cancel()
		}

		return nil
	})

	g.Wait()
}

func startTwitchDownloader(
	ctx context.Context,
	tw *twitch.Client,
	spin *spinner.Model,
	conf *config.Config,
	option cli.Flag,
	twitchUnits []downloader.Unit,
	g *errgroup.Group,
) {
	dl := downloader.New(tw)

	if spin != nil {
		dl.SetProgressNotifier(func(pm downloader.Progress) {
			if ctx.Err() != nil {
				return
			}
			spin.C <- spinner.Message{
				ID:    pm.ID,
				Bytes: pm.Bytes,
				Err:   pm.Err,
				Done:  pm.Done,
			}
		})
	}

	g.Go(func() error {
		if option.Subscribe {
			return runTwitchEventSub(ctx, tw, dl, twitchUnits)
		} else {
			return batchDownloadTwitchUnits(ctx, option.Threads, twitchUnits, dl)
		}
	})
}

func batchDownloadTwitchUnits(
	ctx context.Context,
	threads int,
	units []downloader.Unit,
	dl *downloader.Downloader,
) error {
	g, ctx := errgroup.WithContext(ctx)
	if threads > 0 {
		g.SetLimit(threads)
	}

	var err error

	for _, unit := range units {
		g.Go(func() error {
			if e := dl.Download(ctx, unit); e != nil {
				err = errors.Join(err, e)
			}
			return nil
		})
	}

	g.Wait()

	return err
}

func startKickDownloader(
	ctx context.Context,
	spin *spinner.Model,
	threads int,
	kickUnits []kick.Unit,
	g *errgroup.Group,
) {
	c := kick.New()

	if spin != nil {
		c.SetProgressNotifier(func(pm kick.Progress) {
			if ctx.Err() != nil {
				return
			}
			spin.C <- spinner.Message{
				ID:    pm.ID,
				Bytes: pm.Bytes,
				Err:   pm.Error,
				Done:  pm.Done,
			}
		})
	}

	g.Go(func() error {
		g, ctx := errgroup.WithContext(ctx)
		if threads > 0 {
			g.SetLimit(threads)
		}

		var err error

		for _, unit := range kickUnits {
			g.Go(func() error {
				e := c.Download(ctx, unit)
				if e != nil {
					err = errors.Join(err, e)
				}
				return nil
			})
		}

		g.Wait()

		return err
	})
}

func runTwitchEventSub(
	ctx context.Context,
	tw *twitch.Client,
	dl *downloader.Downloader,
	units []downloader.Unit,
) error {
	return nil

	// ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	// defer cancel()

	// events, err := eventsub.FromUnits(units)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// event := eventsub.New(tw)

	// event.OnNotification = func(resp eventsub.ResponseBody) {
	// 	if resp.Payload.Subscription != nil {
	// 		condition := resp.Payload.Subscription.Condition

	// 		if userID, ok := condition["broadcaster_user_id"].(string); ok {
	// 			user, _ := tw.Helix.UserByID(ctx, userID)
	// 			unit := downloader.NewUnit(user.Login, downloader.WithQuality(""))

	// 			if unit.Error == nil {
	// 				go func() {
	// 					fmt.Println("Starting to record the stream for: ", unit.ID)

	// 					if err := dl.Download(ctx, *unit); err != nil {
	// 						fmt.Println("error occured: ", err)
	// 						isLive, _ := tw.IsChannelLive(ctx, user.Login)
	// 						if !isLive {
	// 							fmt.Println("Stream went offline!")
	// 						}
	// 						return
	// 					}

	// 					fmt.Println("Stream recording ended for: ", unit.ID)
	// 				}()
	// 			}
	// 		}
	// 	}
	// }

	// if err := event.DialWS(ctx, events); err != nil {
	// 	return err
	// }

	// return nil
}

func runChat(ctx context.Context, tw *twitch.Client, conf *config.Config) error {
	if err := tw.Helix.Authorize(ctx, helix.AuthOpts{ResponseType: helix.CodeResponseType}); err != nil {
		return err
	}

	userData, err := tw.Helix.Users().Run(ctx)
	if err != nil {
		return err
	}
	user := userData.Data[0]

	conf.User = config.User{
		BroadcasterType: user.BroadcasterType,
		CreatedAt:       user.CreatedAt,
		Description:     user.Description,
		DisplayName:     user.DisplayName,
		ID:              user.ID,
		Login:           user.Login,
		OfflineImageURL: user.OfflineImageURL,
		ProfileImageURL: user.ProfileImageURL,
		Type:            user.Type,
	}

	chat.Open(ctx, tw, conf)

	return nil
}

func runPrint(ctx context.Context, tw *twitch.Client) {
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("invalid usage: --print <channel_name>")
	}

	channel := args[0]

	about, err := tw.Gql.ChannelRoot_AboutPanel(ctx, channel)
	if err != nil {
		log.Fatal(err)
	}

	limit := 20

	videos, err := tw.Gql.FilterableVideoTower_Videos(ctx, channel, limit)
	if err != nil {
		log.Fatal(err)
	}

	clips, err := tw.Gql.ClipsCardsUser(ctx, channel, limit, gql.AllTime)
	if err != nil {
		log.Fatal(err)
	}

	PrintChannel(about, videos, clips)
}

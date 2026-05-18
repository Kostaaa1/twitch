package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kostaaa1/twitch/internal/cli"
	"github.com/Kostaaa1/twitch/internal/cli/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/kick"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
	"github.com/Kostaaa1/twitch/pkg/twitch/eventsub"
	"golang.org/x/sync/errgroup"
)

func main() {
	conf, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}
	defer conf.Save()

	ctx := context.Background()
	opt := cli.ParseFlags(*conf)
	tw := twitch.NewClient(twitch.WithOAuthCreds(&conf.OAuthCreds))

	switch {
	case opt.Authenticate:
		if err := tw.Helix.Authorize(ctx); err != nil {
			log.Fatal(err)
		}
	case opt.Print:
		args := flag.Args()

		if len(args) == 0 {
			log.Fatalln("wrong usage: --print <channel_name>")
			return
		}

		channel := args[0]
		_ = channel

		// limit := 20

		// about, err := tw.ChannelRoot_AboutPanel(ctx, channel)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// b, err := json.MarshalIndent(about, "", " ")
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Println(string(b))

		// videos, err := tw.FilterableVideoTower_Videos(ctx, channel, limit)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Println("VIDEOS:", videos)

		// clips, err := tw.ClipsCardsUser(ctx, channel, limit, "ALL_TIME")
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Println("CLIP{S:", clips)

		// fmt.Println(about)

		// fmt.Println(about.User.PrimaryColorHex)
		// primaryHex := fmt.Sprintf("#%s", about.User.PrimaryColorHex)
		// primary := lipgloss.NewStyle().Foreground(lipgloss.Color(primaryHex))
		// fmt.Println(primary.Render(about.User.DisplayName))

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

	for _, unit := range units {
		fmt.Println("UNITS:", unit)
	}

	g.Go(func() error {
		downloadGroup, ctx := errgroup.WithContext(ctx)

		if len(twitchUnits) > 0 {
			startTwitchDownloader(ctx, tw, spin, conf, opt, twitchUnits, downloadGroup)
		}

		if len(kickUnits) > 0 {
			startKickDownloader(ctx, spin, opt.Threads, kickUnits, downloadGroup)
		}

		// TODO: If error happens when downloading i want spinner to cancel the context, but if there is no errors, cancel needs to happen after downloading of batches finishes. Problem is that i cannot return
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
	dl := downloader.New(tw, &conf.Downloader)

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
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	events, err := eventsub.FromUnits(units)
	if err != nil {
		log.Fatal(err)
	}

	event := eventsub.New(tw)

	event.OnNotification = func(resp eventsub.ResponseBody) {
		if resp.Payload.Subscription != nil {
			condition := resp.Payload.Subscription.Condition

			if userID, ok := condition["broadcaster_user_id"].(string); ok {
				user, _ := tw.Helix.UserByID(ctx, userID)
				unit := downloader.NewUnit(user.Login, downloader.WithQuality(""))

				if unit.Error == nil {
					go func() {
						fmt.Println("Starting to record the stream for: ", unit.ID)

						if err := dl.Download(ctx, *unit); err != nil {
							fmt.Println("error occured: ", err)
							isLive, _ := tw.IsChannelLive(ctx, user.Login)
							if !isLive {
								fmt.Println("Stream went offline!")
							}
							return
						}

						fmt.Println("Stream recording ended for: ", unit.ID)
					}()
				}
			}
		}
	}

	if err := event.DialWS(ctx, events); err != nil {
		return err
	}

	return nil
}

func runChat(ctx context.Context, tw *twitch.Client, conf *config.Config) error {
	if err := tw.Helix.Authorize(ctx); err != nil {
		return err
	}

	user, err := tw.Helix.UserByChannelName(ctx, "")
	if err != nil {
		return fmt.Errorf("failed fetching user data for cli chat: %v", err)
	}

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

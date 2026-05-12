package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
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
	tw := twitch.NewClient()

	b, err := tw.VideoPlaylistBuilder(context.Background(), "2279431034")
	if err != nil {
		log.Fatal(err)
	}

	u := b.PlaylistURL()
	fmt.Println(u)

	resp, err := http.Get(u)
	if err != nil {
		log.Fatal(err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))

	// parsed, err := url.Parse("https://d3vd9lfkzbru3h.cloudfront.net/6d06e268d17b051dde79_sera_promisu_315855972593_1778166843/storyboards/2766330803-info.json")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// m3u8.MasterPlaylistMock(http.DefaultClient, "2766330803", parsed, "ARCHIVE")

	// conf, err := config.Read()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer conf.Save()

	// ctx := context.Background()

	// flag := cli.ParseFlags(*conf)
	// tw := twitch.NewClient(twitch.WithOAuthCreds(&conf.OAuthCreds))

	// switch {
	// case flag.Authenticate:
	// 	if err := tw.Authorize(ctx); err != nil {
	// 		log.Fatal(err)
	// 	}
	// case flag.Channel != "":
	// 	handlePrinting(ctx, tw, flag.Channel)
	// case len(os.Args) == 1:
	// 	initChat(ctx, tw, conf)
	// default:
	// 	initDownloader(ctx, tw, conf, flag)
	// }
}

func initDownloader(ctx context.Context, tw *twitch.Client, conf *config.Config, opt cli.Flag) {
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
			return initTwitchEventSub(ctx, tw, dl, twitchUnits)
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

func initTwitchEventSub(
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
				user, _ := tw.UserByID(ctx, userID)
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

func initChat(ctx context.Context, tw *twitch.Client, conf *config.Config) error {
	if err := tw.Authorize(ctx); err != nil {
		return err
	}

	user, err := tw.UserByChannelName(ctx, "")
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

func handlePrinting(ctx context.Context, tw *twitch.Client, input string) error {
	// videos, err := tw.ListVideosByChannelName(ctx, input, 100)
	// if err != nil {
	// 	return err
	// }
	// for _, vod := range videos {
	// 	b, err := json.MarshalIndent(vod, "", " ")
	// 	if err != nil {
	// 		return err
	// 	}
	// 	fmt.Println(b)
	// }
	return nil
}

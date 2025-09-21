package main

import (
	"context"
	"fmt"
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
	conf, err := config.Get()
	if err != nil {
		panic(err)
	}

	defer func() {
		conf.Save()
	}()

	option := ParseFlags(*conf)

	if option.Authorize {
		tw := twitch.NewClient(&conf.Creds)
		if err := tw.Authorize(); err != nil {
			panic(err)
		}
		return
	}

	if len(os.Args) == 1 {
		tw := twitch.NewClient(&conf.Creds)
		initChat(tw, conf)
		return
	}

	initDownloader(conf, option)
}

func initDownloader(conf *config.Config, option cli.Option) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	units := option.UnitsFromInput()
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
			startTwitchDownloader(ctx, spin, conf, option, twitchUnits, downloadGroup)
		}

		if len(kickUnits) > 0 {
			startKickDownloader(ctx, spin, option, kickUnits, downloadGroup)
		}

		err := downloadGroup.Wait()
		cancel()

		return err
	})

	g.Wait()
}

func startTwitchDownloader(
	ctx context.Context,
	spin *spinner.Model,
	conf *config.Config,
	option cli.Option,
	twitchUnits []downloader.Unit,
	g *errgroup.Group,
) {
	tw := twitch.NewClient(&conf.Creds)
	dl := downloader.New(tw, conf.Downloader)

	if spin != nil {
		dl.SetProgressNotifier(func(pm downloader.ProgressMessage) {
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
			return batchDownloadTwitchUnits(ctx, option.Threads, twitchUnits, dl, spin)
		}
	})
}

func batchDownloadTwitchUnits(
	ctx context.Context,
	threads int,
	units []downloader.Unit,
	dl *downloader.Downloader,
	spin *spinner.Model,
) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(threads)

	for _, unit := range units {
		g.Go(func() error {
			dl.Download(ctx, unit)
			return nil
		})
	}

	return g.Wait()
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
		panic(err)
	}

	event := eventsub.New(tw)

	event.OnNotification = func(resp eventsub.ResponseBody) {
		if resp.Payload.Subscription != nil {
			condition := resp.Payload.Subscription.Condition

			if userID, ok := condition["broadcaster_user_id"].(string); ok {
				user, _ := tw.UserByID(userID)
				unit := downloader.NewUnit(user.Login, downloader.Quality1080p60.String())

				if unit.Error == nil {
					go func() {
						fmt.Println("Starting to record the stream for: ", unit.ID)

						if err := dl.Download(ctx, *unit); err != nil {
							fmt.Println("error occured: ", err)
							isLive, _ := tw.IsChannelLive(user.Login)
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

func initChat(client *twitch.Client, conf *config.Config) {
	if err := conf.AuthorizeAndSaveUserData(client); err != nil {
		panic(err)
	}
	chat.Open(client, conf)
}

func startKickDownloader(
	ctx context.Context,
	spin *spinner.Model,
	option cli.Option,
	kickUnits []kick.Unit,
	g *errgroup.Group,
) {
	c := kick.New()

	if spin != nil {
		c.SetProgressNotifier(func(pm kick.ProgressMessage) {
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
		g.SetLimit(option.Threads)

		for _, unit := range kickUnits {
			g.Go(func() error {
				c.Download(ctx, unit)
				return nil
			})
		}

		return g.Wait()
	})
}

// TEST:
func getUnitTitles(client *twitch.Client, units []*downloader.Unit) {
	for _, u := range units {
		client.ClipMetadata()
	}
}

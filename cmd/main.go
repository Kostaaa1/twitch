package main

import (
	"context"
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
	conf, err := config.Get()
	if err != nil {
		panic(err)
	}

	defer func() {
		conf.Save()
	}()

	option := ParseFlags(*conf)

	var tw *twitch.Client

	if option.Authorize {
		if tw == nil {
			tw = twitch.NewClient(&conf.Creds)
		}
		if err := tw.Authorize(); err != nil {
			panic(err)
		}
	}

	if len(os.Args) == 1 {
		if tw == nil {
			tw = twitch.NewClient(&conf.Creds)
		}
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
		spin = spinner.New(units, conf.Downloader.SpinnerModel, cancel)
		defer close(spin.ProgressChan())

		g.Go(func() error {
			spin.Run()
			return nil
		})
	}

	if len(twitchUnits) > 0 {
		tw := twitch.NewClient(&conf.Creds)
		dl := downloader.New(ctx, tw, conf.Downloader)

		if spin != nil {
			dl.SetProgressChannel(spin.ProgressChan())
		}

		g.Go(func() error {
			if option.Subscribe {
				return initTwitchEventSub(ctx, tw, dl, twitchUnits)
			} else {
				return batchDownloadTwitchUnits(ctx, dl, option.Threads, twitchUnits)
			}
		})
	}

	if len(kickUnits) > 0 {
		kick := kick.New()
		if spin != nil {
			kick.SetProgressChannel(spin.ProgressChan())
		}

		g.Go(func() error {
			batchGroup, ctx := errgroup.WithContext(ctx)
			batchGroup.SetLimit(option.Threads)
			for _, unit := range kickUnits {
				batchGroup.Go(func() error {
					return kick.Download(ctx, unit)
				})
			}

			return batchGroup.Wait()
		})
	}

	if err := g.Wait(); err != nil {
		log.Println("Error while downloading: ", err)
	}
}

func batchDownloadTwitchUnits(
	ctx context.Context,
	dl *downloader.Downloader,
	threads int,
	units []downloader.Unit,
) error {
	batchGroup, ctx := errgroup.WithContext(ctx)
	batchGroup.SetLimit(threads)
	for _, unit := range units {
		batchGroup.Go(func() error {
			return dl.Download(unit)
		})
	}
	return batchGroup.Wait()
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

						if err := dl.Download(*unit); err != nil {
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

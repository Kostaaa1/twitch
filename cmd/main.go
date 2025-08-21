package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	twitch := twitch.NewClient(&conf.Creds)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	twitch.SetHTTPClient(httpClient)

	if option.Authorize {
		if err := twitch.Authorize(); err != nil {
			panic(err)
		}
	}

	// handle printify
	if option.Channel != "" {
		videos, err := twitch.GetVideosByChannelName(option.Channel, 100)
		if err != nil {
			panic(err)
		}
		b, _ := json.MarshalIndent(videos, "", "  ")
		fmt.Println(string(b))
		return
	}

	if len(os.Args) == 1 {
		initChat(twitch, conf)
		return
	}

	initDownloader(twitch, conf, option)
}

func initDownloader(tw *twitch.Client, conf *config.Config, option cli.Option) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	unitsTwitch, unitsKick := option.UnitsFromInput()

	var spin *spinner.Model

	if conf.Downloader.ShowSpinner && (len(unitsTwitch) > 0 || len(unitsKick) > 0) {
		spin = spinner.New(unitsTwitch, conf.Downloader.SpinnerModel, cancel)
		defer close(spin.ProgressChan())

		g.Go(func() error {
			spin.Run()
			return nil
		})
	}

	if len(unitsTwitch) > 0 {
		g.Go(func() error {
			dl := downloader.New(ctx, tw, conf.Downloader)
			if spin != nil {
				dl.SetProgressChannel(spin.ProgressChan())
			}

			if option.Subscribe {
				return initTwitchEventSub(ctx, tw, dl, unitsTwitch)
			} else {
				return batchDownloadTwitchUnits(ctx, dl, option.Threads, unitsTwitch)
			}
		})
	}

	if len(unitsKick) > 0 {
		g.Go(func() error {
			kick := kick.NewClient()
			if spin != nil {
				kick.SetProgressChannel(spin.ProgressChan())
			}
			return batchDownloadKickUnits(ctx, kick, option.Threads, unitsKick)
		})
	}

	if err := g.Wait(); err != nil {
		log.Println("error while downloading: ", err)
	}
}

func batchDownloadTwitchUnits(
	ctx context.Context,
	dl *downloader.Downloader,
	threads int,
	units []downloader.Unit,
) error {
	batchGroup, _ := errgroup.WithContext(ctx)
	batchGroup.SetLimit(threads)
	for _, unit := range units {
		batchGroup.Go(func() error {
			return dl.Download(unit)
		})
	}
	return batchGroup.Wait()
}

func batchDownloadKickUnits(
	ctx context.Context,
	kick *kick.Client,
	threads int,
	units []kick.Unit,
) error {
	batchGroup, _ := errgroup.WithContext(ctx)
	batchGroup.SetLimit(threads)
	for _, unit := range units {
		batchGroup.Go(func() error {
			return kick.Download(ctx, unit)
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

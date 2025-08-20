package main

import (
	"context"
	"encoding/json"
	"flag"
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
	"github.com/Kostaaa1/twitch/pkg/twitch/event"
	"golang.org/x/sync/errgroup"
)

var (
	option cli.Option
	conf   *config.Config
)

func init() {
	var err error

	conf, err = config.Get()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&option.Input, "input", "", "input can be twitch (URL, vod id or clip slug), kick (vod URL) or json file (check example.json). Multiple inputs can be comma-separated which will be downloaded concurrently")
	flag.StringVar(&option.Output, "output", conf.Downloader.Output, "Destination path for downloaded files")
	flag.StringVar(&option.Quality, "quality", "", "Video quality: best, 1080, 720, 480, 360, 160, worst, or audio")
	flag.DurationVar(&option.Start, "start", time.Duration(0), "Start time for VOD segment (e.g., 1h30m0s). Only for VODs")
	flag.DurationVar(&option.End, "end", time.Duration(0), "End time for VOD segment (e.g., 1h45m0s). Only for VODs")
	flag.IntVar(&option.Threads, "threads", 6, "Number of parallel downloads (batch mode only)")
	flag.BoolVar(&option.Set, "set", false, "Set a config field: key=value. (e.g. -set output=your_path")
	flag.StringVar(&option.Channel, "channel", "", "Twitch channel name")
	flag.BoolVar(&option.Subscribe, "subscribe", false, "Enable live stream monitoring: starts a websocket server and uses channel names from --input flag to automatically download streams when they go live. It could be used in combination with tools such as systemd, to auto-record the stream in the background.")
	flag.BoolVar(&option.Authorize, "auth", false, "Authorize with Twitch. It is mostly needed for CLI chat feature and Helix API. Downloader is not using authorization tokens")

	flag.Parse()
}

func main() {
	defer func() {
		conf.Save()
	}()

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

	twitch := twitch.NewClient(httpClient, &conf.Creds)

	if option.Set {
		conf.Downloader.Output = option.Output
	}

	if option.Authorize {
		twitch.Authorize()
	}

	if option.Channel != "" {
		videos, err := twitch.GetVideosByChannelName(option.Channel, 100)
		if err != nil {
			log.Fatal(err)
		}
		b, _ := json.MarshalIndent(videos, "", "  ")
		fmt.Println(string(b))
		return
	}

	if len(os.Args) == 1 {
		initChat(twitch, conf)
		return
	}

	initDownloader(twitch)
}

func initDownloader(tw *twitch.Client) {
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
				g.Go(func() error {
					return initEventSub(ctx, tw, dl)
				})
				return nil
			} else {
				batchGroup, _ := errgroup.WithContext(ctx)
				batchGroup.SetLimit(option.Threads)

				for _, unit := range unitsTwitch {
					batchGroup.Go(func() error {
						return dl.Download(unit)
					})
				}

				return batchGroup.Wait()
			}
		})
	}

	if len(unitsKick) > 0 {
		g.Go(func() error {
			batchGroup, _ := errgroup.WithContext(ctx)
			batchGroup.SetLimit(option.Threads)

			kick := kick.NewClient()
			if spin != nil {
				kick.SetProgressChannel(spin.ProgressChan())
			}

			for _, unit := range unitsKick {
				batchGroup.Go(func() error {
					return kick.Download(ctx, unit)
				})
			}

			return batchGroup.Wait()
		})
	}

	if err := g.Wait(); err != nil {
		log.Println("error while downloading: ", err)
	}
}

func initEventSub(ctx context.Context, tw *twitch.Client, dl *downloader.Downloader) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	units, _ := option.UnitsFromInput()
	events, err := event.FromUnits(units)
	if err != nil {
		log.Fatal(err)
	}

	// events, err := cli.EventsFromUnits(tw, units)

	eventsub := event.NewClient(tw)

	eventsub.OnNotification = func(resp event.ResponseBody) {
		if resp.Payload.Subscription != nil {
			condition := resp.Payload.Subscription.Condition

			if userID, ok := condition["broadcaster_user_id"].(string); ok {
				user, _ := tw.UserByID(userID)
				unit := downloader.NewUnit(user.Login, downloader.Quality1080p60.String())

				if unit.Error == nil {
					go func() {
						fmt.Println("Starting to record the stream for: ", unit.ID)
						// unit.Writer, unit.Error = cli.NewFile(unit.GetTitle(), unit.Quality, option.Output)

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

	if err := eventsub.DialWS(ctx, events); err != nil {
		return err
	}

	return nil
}

func initChat(client *twitch.Client, conf *config.Config) {
	if err := conf.AuthorizeAndSaveUserData(client); err != nil {
		log.Fatal(err)
	}
	chat.Open(client, conf)
}

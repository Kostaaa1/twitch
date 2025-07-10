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
	"sync"
	"syscall"
	"time"

	"github.com/Kostaaa1/twitch/internal/cli"
	"github.com/Kostaaa1/twitch/internal/cli/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
	"github.com/Kostaaa1/twitch/pkg/twitch/event"
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

	flag.StringVar(&option.Input, "input", "", "Twitch URL, VOD ID, clip slug, or channel name. Comma-separated values enable batch downloads. If output is set, download starts automatically.")
	flag.StringVar(&option.Output, "output", conf.Downloader.Output, "Destination path for downloaded files.")
	flag.StringVar(&option.Quality, "quality", "", "Video quality: best, 1080, 720, 480, 360, 160, worst, or audio.")
	flag.DurationVar(&option.Start, "start", time.Duration(0), "Start time for VOD segment (e.g., 1h30m0s). Only for VODs.")
	flag.DurationVar(&option.End, "end", time.Duration(0), "End time for VOD segment (e.g., 1h45m0s). Only for VODs.")
	flag.IntVar(&option.Threads, "threads", 0, "Number of parallel downloads (batch mode only).")

	flag.StringVar(&option.Channel, "channel", "", "Twitch channel name.")

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

	client := twitch.NewClient(httpClient, &conf.Creds)

	if option.Authorize {
		client.Authorize()
	}

	if option.Channel != "" {
		videos, err := client.GetVideosByChannelName(option.Channel, 100)
		if err != nil {
			log.Fatal(err)
		}
		b, _ := json.MarshalIndent(videos, "", "  ")
		fmt.Println(string(b))
		return
	}

	if len(os.Args) == 1 {
		initChat(client)
		return
	}

	initDownloader(client)
}

func initDownloader(client *twitch.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dl := downloader.New(ctx, client, conf.Downloader)
	dl.SetThreads(option.Threads)

	var wg sync.WaitGroup

	// eventsub downloader
	if option.Subscribe {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			if err := initEventSub(ctx, dl); err != nil {
				log.Fatal(err)
			}
		}()
		wg.Wait()
	} else {
		units := option.GetUnitsFromInputWithWriter(dl)

		if conf.Downloader.ShowSpinner {
			spin := spinner.New(units, conf.Downloader.SpinnerModel, cancel)
			defer close(spin.ProgressChan())
			dl.SetProgressChannel(spin.ProgressChan())

			wg.Add(1)
			go func() {
				defer wg.Done()
				spin.Run()
			}()
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			dl.BatchDownload(units)
		}()

		wg.Wait()
	}
}

func initEventSub(ctx context.Context, dl *downloader.Downloader) error {
	units := option.GetUnitsFromInput(dl)

	events, err := cli.EventsFromUnits(dl, units)
	if err != nil {
		log.Fatal(err)
	}

	eventsub := event.NewClient(dl.TWApi)

	eventsub.OnNotification = func(resp event.ResponseBody) {
		if resp.Payload.Subscription != nil {
			condition := resp.Payload.Subscription.Condition

			if userID, ok := condition["broadcaster_user_id"].(string); ok {
				user, _ := dl.TWApi.UserByID(userID)
				unit := downloader.NewUnit(user.Login, downloader.Quality1080p60.String())

				if unit.Error == nil {
					unit.Writer, unit.Error = cli.NewFile(dl, unit, option.Output)
					go func() {
						fmt.Println("Starting to record the stream for: ", unit.ID)

						unit.Writer, unit.Error = cli.NewFile(dl, unit, option.Output)
						if err := dl.Download(*unit); err != nil {
							fmt.Println("error occured: ", err)

							isLive, _ := dl.TWApi.IsChannelLive(user.Login)
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

func initChat(client *twitch.Client) {
	if err := conf.AuthorizeAndSaveUserData(client); err != nil {
		log.Fatal(err)
	}
	chat.Open(client, conf)
}

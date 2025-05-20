package main

import (
	"context"
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

	flag.StringVar(&option.Category, "category", "", "Twitch category name.")
	flag.StringVar(&option.Channel, "channel", "", "Twitch channel name.")

	flag.BoolVar(&option.Authorize, "auth", false, "Authorize with Twitch. It is mostly needed for CLI chat feature and Helix API. Downloader is not using authorization tokens")
	flag.BoolVar(&option.Subscribe, "subscribe", false, "Enable live stream monitoring: starts a websocket server and uses channel names from --input to automatically download streams when they go live. Useful for automation (e.g., with systemd).")

	flag.Parse()
}

func main() {
	defer func() {
		conf.Save()
	}()

	client := twitch.NewClient(http.DefaultClient, &conf.Creds)

	if option.Authorize {
		client.Authorize()
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

	units := option.ProcessFlags(dl)
	spin := spinner.New(units, conf.Downloader.SpinnerModel, cancel)

	var wg sync.WaitGroup

	if option.Subscribe {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			if err := initEventSub(ctx, dl, units); err != nil {
				log.Fatal(err)
			}
		}()
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spin.Run()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			dl.SetProgressChannel(spin.ProgChan())
			dl.SetThreads(option.Threads)
			dl.BatchDownload(units)
		}()
	}

	wg.Wait()
	close(spin.ProgChan())
}

func initEventSub(ctx context.Context, dl *downloader.Downloader, units []downloader.Unit) error {
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
						if err := dl.Download(*unit); err != nil {
							fmt.Println("error while recording the stream: ", err)
							return
						}
						fmt.Println("Stream recording ended for: ", unit.ID)
					}()
				}
			}
		}
	}

	events, err := cli.EventsFromUnits(dl, units)
	if err != nil {
		return err
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

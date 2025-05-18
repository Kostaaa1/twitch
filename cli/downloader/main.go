package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Kostaaa1/twitch/internal/cli"
	"github.com/Kostaaa1/twitch/internal/cli/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/downloader"
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
	flag.StringVar(&option.Subscribe, "subscribe", "", "Comma-separated list of channel names to monitor and automatically download live streams when they go online. Useful for automation with tools like systemd.")

	flag.Parse()
}

func initDownloader() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := twitch.NewClient(http.DefaultClient, &conf.Creds)
	dl := downloader.New(ctx, client, conf.Downloader)

	units := option.ProcessFlags(dl)

	spin := spinner.New(units, conf.Downloader.SpinnerModel, cancel)
	dl.SetProgressChannel(spin.ProgChan())
	dl.SetThreads(option.Threads)

	var wg sync.WaitGroup

	if option.Subscribe != "" {
		wg.Add(1)
		go func() {

		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		spin.Run()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		dl.BatchDownload(units)
	}()

	wg.Wait()
	close(spin.ProgChan())
}

func initChat() {
	tw := twitch.NewClient(nil, &conf.Creds)

	if err := tw.Authorize(); err != nil {
		log.Fatal(err)
	}

	user, err := tw.UserByChannelName("")
	if err != nil {
		log.Fatalf("failed to get the user info: %v\n", err)
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
	conf.Save()

	chat.Open(tw, conf)
}

func main() {
	if len(os.Args) == 0 {
		initChat()
		return
	}

	if option.Output != "" {
		initDownloader()
	}

	// client := twitch.NewClient(http.DefaultClient, &conf.Creds)
	// user1, err := client.UserByChannelName("39deph")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// user2, err := client.UserByChannelName("kosta")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// user3, err := client.UserByChannelName("ksota")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// user4, err := client.UserByChannelName("39daph")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(user1.ID)
	// fmt.Println(user2.ID)
	// fmt.Println(user3.ID)
	// fmt.Println(user4.ID)

	// client.Stream(user1.ID)
}

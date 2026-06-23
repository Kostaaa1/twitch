package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Kostaaa1/twitch/internal/cli"
	"github.com/Kostaaa1/twitch/internal/cli/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/internal/downloader"
	"github.com/Kostaaa1/twitch/pkg/spinner"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix/eventsub"
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
	defer func() {
		if err := conf.Save(); err != nil {
			log.Fatal(err)
		}
	}()

	ctx := context.Background()
	flag := cli.ParseFlags(*conf)

	httpc := http.DefaultClient
	tw := &twitch.Client{
		Gql:   gql.New(httpc),
		Helix: helix.New(httpc, helix.WithOAuthCreds(&conf.OAuthCreds)),
	}

	switch {
	case flag.Authorize:
		if err := runLogin(ctx, tw, conf); err != nil {
			log.Fatal(err)
		}
	case flag.Print:
		if err := runPrint(ctx, tw); err != nil {
			log.Fatal(err)
		}
	case len(os.Args) == 1:
		if err := runChat(ctx, tw, conf); err != nil {
			log.Fatal(err)
		}
	default:
		if err := runDownloader(ctx, tw, conf, flag); err != nil {
			log.Fatal(err)
		}
	}
}

func runDownloader(
	ctx context.Context,
	tw *twitch.Client,
	conf *config.Config,
	flag cli.Flag,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	units, err := flag.UnitsFromInput(ctx, tw)
	if err != nil {
		return err
	}

	twitchUnits, kickUnits := cli.FilterUnits(units)
	_ = kickUnits

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
			startTwitchDownloader(ctx, tw, spin, flag, twitchUnits, downloadGroup)
		}
		// if len(kickUnits) > 0 {
		// 	startKickDownloader(ctx, spin, flag.Threads, kickUnits, downloadGroup)
		// }
		if err := downloadGroup.Wait(); err == nil {
			cancel()
		}
		return nil
	})

	return g.Wait()
}

func startTwitchDownloader(
	ctx context.Context,
	tw *twitch.Client,
	spin *spinner.Model,
	flag cli.Flag,
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
		if flag.Watch {
			return runTwitchEventSub(ctx, tw, dl, twitchUnits)
		} else {
			return batchDownloadTwitchUnits(ctx, flag.Threads, twitchUnits, dl)
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

	errCh := make(chan error)

	for _, unit := range units {
		g.Go(func() error {
			if err := dl.Download(ctx, &unit); err != nil {
				errCh <- err
			}
			return nil
		})
	}

	g.Wait()

	var dlErr error
	for err := range errCh {
		dlErr = errors.Join(dlErr, err)
	}
	close(errCh)

	return dlErr
}

// func startKickDownloader(
// 	ctx context.Context,
// 	spin *spinner.Model,
// 	threads int,
// 	kickUnits []kick.Unit,
// 	g *errgroup.Group,
// ) {
// 	c := kick.New()

// 	if spin != nil {
// 		c.SetProgressNotifier(func(pm kick.Progress) {
// 			if ctx.Err() != nil {
// 				return
// 			}
// 			spin.C <- spinner.Message{
// 				ID:    pm.ID,
// 				Bytes: pm.Bytes,
// 				Err:   pm.Error,
// 				Done:  pm.Done,
// 			}
// 		})
// 	}

// 	g.Go(func() error {
// 		g, ctx := errgroup.WithContext(ctx)
// 		if threads > 0 {
// 			g.SetLimit(threads)
// 		}

// 		var err error

// 		for _, unit := range kickUnits {
// 			g.Go(func() error {
// 				e := c.Download(ctx, unit)
// 				if e != nil {
// 					err = errors.Join(err, e)
// 				}
// 				return nil
// 			})
// 		}

// 		g.Wait()

// 		return err
// 	})
// }

func runTwitchEventSub(
	ctx context.Context,
	tw *twitch.Client,
	dl *downloader.Downloader,
	units []downloader.Unit,
) error {
	e, err := eventsub.WithWebsocket(
		ctx,
		tw.Helix,
		eventsub.WebsocketConnArgs{
			KeepaliveSeconds: 30,
			OnNotification: func(msg eventsub.EventSubMessage) {
				for _, unit := range units {
					if msg.Payload.Event.BroadcasterUserLogin == unit.ID {
						switch msg.Metadata.SubscriptionType {
						case eventsub.StreamOnline:
							// start downloading
							go dl.Download(ctx, &unit)
						case eventsub.StreamOffline:
							// cancel downloading
						}
					}
				}
			},
		},
	)
	if err != nil {
		return err
	}

	// ----- Delete previous subscriptions since we are using wss its session bound ------
	// currentSubs, err := e.Subscriptions().Get().Run(ctx)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// for _, sub := range currentSubs.Data {
	// 	if err := e.Subscriptions().Delete(sub.ID).Run(ctx); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	for _, unit := range units {
		user, err := tw.Helix.Users().UserLogin(unit.ID).Run(ctx)
		if err != nil {
			return err
		}

		id := user.Data[0].ID

		ev1 := e.StreamOnlineEvent(id)
		events := []eventsub.Event{ev1}

		for _, event := range events {
			resp, err := e.Subscriptions().Create(event).Run(ctx)
			if err != nil {
				return err
			}
			_ = resp
			// print(resp)
		}
	}

	// verify
	subs, err := e.Subscriptions().Get().Run(ctx)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(subs, "", " ")
	fmt.Println(string(b))

	return e.Wait()
}

func runChat(ctx context.Context, tw *twitch.Client, conf *config.Config) error {
	if err := tw.Helix.Authorize(ctx, helix.AuthOpts{}); err != nil {
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

	return chat.Open(ctx, tw, conf)
}

func runPrint(ctx context.Context, tw *twitch.Client) error {
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("invalid usage: --print <channel_name>")
	}

	channel := args[0]

	about, err := tw.Gql.ChannelRoot_AboutPanel(ctx, channel)
	if err != nil {
		return err
	}

	limit := 20

	videos, err := tw.Gql.FilterableVideoTower_Videos(ctx, channel, limit)
	if err != nil {
		return err
	}

	clips, err := tw.Gql.ClipsCardsUser(ctx, channel, limit, gql.AllTime)
	if err != nil {
		return err
	}

	cli.PrintChannel(about, videos, clips)

	return nil
}

// Users should have options to either use their app credentials / or just authorize with
// twitch --login --client_id= --redirect_url= -client_secret=
func runLogin(
	ctx context.Context,
	tw *twitch.Client,
	conf *config.Config,
) error {
	scanner := bufio.NewScanner(os.Stdin)

	if conf.OAuthCreds.ClientID == "" {
		fmt.Print("please provide client ID: ")
		if !scanner.Scan() {
			if scanner.Err() != nil {
				return scanner.Err()
			}
		}
		clientID := strings.TrimSpace(scanner.Text())
		if clientID == "" {
			return errors.New("client ID must be provided")
		}
		conf.OAuthCreds.ClientID = clientID
	}

	if conf.OAuthCreds.ClientSecret == "" {
		fmt.Print("please provide client secret: ")
		if !scanner.Scan() {
			if scanner.Err() != nil {
				return scanner.Err()
			}
		}
		clientSecret := strings.TrimSpace(scanner.Text())
		if clientSecret == "" {
			return errors.New("client ID must be provided")
		}
		conf.OAuthCreds.ClientSecret = clientSecret
	}

	if conf.OAuthCreds.RedirectURL == "" {
		fmt.Print("please provide redirect URL: ")
		if !scanner.Scan() {
			if scanner.Err() != nil {
				return scanner.Err()
			}
		}
		redirectURL := strings.TrimSpace(scanner.Text())
		if redirectURL == "" {
			return errors.New("redirect URL must be provided")
		}
		conf.OAuthCreds.RedirectURL = redirectURL
	}

	return tw.Helix.Authorize(ctx, helix.AuthOpts{
		Scopes: []helix.Scope{
			helix.ChannelManageRedemptions,
			helix.ChannelReadHypeTrain,
			helix.ChannelReadRedemptions,
			helix.ChannelReadSubscriptions,
			helix.ModeratorReadChatters,
			helix.UserManageBlockedUsers,
			helix.UserReadBlockedUsers,
			helix.ChatEdit,
			helix.ChatRead,
			helix.UserReadFollows,
			helix.UserReadSubscriptions,
		},
	})
}

/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Kostaaa1/twitch/internal/cli/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"

	// "github.com/Kostaaa1/twitch/pkg/twitch/chat"
	"github.com/Kostaaa1/twitch/pkg/twitch/helix"

	"github.com/spf13/cobra"
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.Get()
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			fmt.Println("saving")
			if err := config.Save(); err != nil {
				log.Fatal(err)
			}
		}()

		client := helix.New(
			http.DefaultClient,
			helix.WithOAuthCreds(&conf.OAuthCreds),
		)

		ctx := context.Background()

		users, err := client.Users().Run(ctx)
		if err != nil {
			fmt.Println("this failed", err)
			log.Fatal(err)
		}

		user := users.Data[0].Login
		conf.User.Login = user

		if err := chat.Open(ctx, client, conf); err != nil {
			log.Fatal(err)
		}

		// c, err := chat.DialWS(
		// 	conf.User.Login,
		// 	conf.OAuthCreds.UserToken.AccessToken,
		// 	conf.CommandLineChat.OpenedChats,
		// )
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// msgCh := make(chan interface{})

		// c.SetMessageChan(msgCh)

		// var wg sync.WaitGroup

		// wg.Add(1)
		// go func() {
		// 	defer wg.Done()
		// 	for msg := range msgCh {
		// 		fmt.Println("MESSAGE:", msg)
		// 	}
		// }()

		// wg.Add(1)
		// go func() {
		// 	defer wg.Done()
		// 	c.Connect()
		// }()

		// wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// chatCmd.PersistentFlags().String("foo", "", "A help for foo")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// chatCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

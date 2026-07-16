package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/spf13/cobra"
)

type optsType struct {
	set  string
	get  bool
	path bool
}

var opts optsType

// flag > env vars > config > default

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		command := args[0]

		switch strings.ToLower(command) {
		case "get":
			cfg, err := config.Get()
			if err != nil {
				log.Fatal(err)
			}

			b, err := json.MarshalIndent(cfg, "", " ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(b))

		case "init":
			// accept paths
			config.Get()
		case "set":

		case "path":
			dir, err := config.Dir()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(dir)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.PersistentFlags().StringVar(&opts.set, "set", "", "")
	configCmd.PersistentFlags().BoolVar(&opts.get, "get", false, "")
	configCmd.PersistentFlags().BoolVar(&opts.path, "path", false, "")
}

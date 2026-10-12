package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/spf13/cobra"
)

type optsType struct {
	set  string
	init bool
	get  bool
	path bool
	rm   bool
}

var opts optsType

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			conf, err := config.Get()
			if err != nil {
				log.Fatal(err)
			}
			b, err := json.MarshalIndent(conf, "", " ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(b))
			return
		}

		dir, err := config.DefaultDirPath()
		if err != nil {
			log.Fatal(err)
		}

		switch strings.ToLower(args[0]) {
		case "init":
			if err := config.Save(config.Default()); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("config initialized at path: %s\n", dir)
		case "set":
			fmt.Println("config field set successfully")
		case "path":
			fmt.Printf("config path: %s\n", dir)
		case "rm":
			if err := os.RemoveAll(dir); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("config dir path removed: %s\n", dir)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.PersistentFlags().StringVar(&opts.set, "set", "", "")
	configCmd.PersistentFlags().BoolVar(&opts.init, "init", false, "")
	configCmd.PersistentFlags().BoolVar(&opts.path, "path", false, "")
	configCmd.PersistentFlags().BoolVar(&opts.rm, "rm", false, "")
}

package main

import (
	"fmt"
	"github.com/denisschmidt/uploader/constants"
	"github.com/denisschmidt/uploader/internal/server"
	"github.com/urfave/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "uploader"
	app.Version = fmt.Sprintf("%s [git:%s:%s]\ncompiled using %s at %s", constants.Version,
		constants.Branch, constants.Revision, constants.Compiler, constants.BuildTime)
	app.Author = "denzel"
	app.Usage = "Upload files"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Usage:  "Config file path",
			EnvVar: "UPLOADER_CONFIG_PATH",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:      "version",
			ShortName: "v",
			Usage:     "Retrieve the version number",
			Action: func(c *cli.Context) {
				fmt.Printf("picfit %s\n", constants.Version)
			},
		},
	}
	app.Action = func(c *cli.Context) {
		config := c.String("config")

		if config != "" {
			if _, err := os.Stat(config); err != nil {
				fmt.Fprintf(os.Stderr, "Can't find config file `%s`\n", config)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Can't find config file\n")
			os.Exit(1)
		}

		// context.Background()
		err := server.Run(config)

		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

package main

import (
	"github.com/harryrose/godm/cli/commands"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	app := &cli.App{
		Name:  "godm",
		Usage: "manage a GoDM queue",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     commands.ArgQueueHost,
				Aliases:  []string{"q"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     commands.ArgKey,
				Aliases:  []string{"k"},
				Required: true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "add",
				Usage:  "Queue an item for download",
				Action: commands.Add,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        commands.ArgQueue,
						DefaultText: commands.DefQueue,
					},
					&cli.StringFlag{
						Name:        commands.ArgCategory,
						Aliases:     []string{"cat"},
						DefaultText: commands.DefCategory,
					},
				},
				ArgsUsage: "<source_url> <destination_path>",
			},
			{
				Name:  "clear",
				Usage: "Remove all items from an object",
				Subcommands: []*cli.Command{
					{
						Name:   "history",
						Usage:  "Clear a queue's finished items",
						Action: commands.ClearHistory,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        commands.ArgQueue,
								DefaultText: commands.DefQueue,
							},
						},
					},
				},
			},
			{
				Name: "show",
				Subcommands: []*cli.Command{
					{
						Name:   "queue",
						Usage:  "Show a queue's items and their status",
						Action: commands.ShowQueue,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        commands.ArgQueue,
								DefaultText: commands.DefQueue,
							},
						},
					},
					{
						Name:   "history",
						Usage:  "Show a queue's finished items and their status",
						Action: commands.ShowHistory,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        commands.ArgQueue,
								DefaultText: commands.DefQueue,
							},
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

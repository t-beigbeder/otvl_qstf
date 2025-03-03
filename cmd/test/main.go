package main

import (
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	app := &cli.App{
		Name:  "bssms",
		Usage: "use one subcommand",
		Commands: []*cli.Command{
			getClientCmd(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func getClientCmd() *cli.Command {
	return &cli.Command{
		Name:        "client",
		Description: "test client application",
		Before: func(cc *cli.Context) error {
			cc.App.Metadata["hns"] = []string{}
			cc.App.Metadata["cd"] = ""
			return nil
		},
		Flags: getClientFlags(),
		Action: func(cc *cli.Context) error {
			err := provisioner.RunPhase0(cc.App.Metadata["cd"].(string), cc.App.Metadata["hns"].([]string))
			return err
		},
	}
}

func getClientFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "hosts",
			Usage: "names of hosts to be installed",
			Action: func(cc *cli.Context, hns []string) error {
				cc.App.Metadata["hns"] = hns
				return nil
			},
		},
		&cli.StringFlag{
			Name:  "cd",
			Usage: "configuration directory, defaults to .conf/.bssms",
			Action: func(cc *cli.Context, cd string) error {
				cc.App.Metadata["cd"] = cd
				return nil
			},
		},
	}
}

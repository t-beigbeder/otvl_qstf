package main

import (
	"fmt"
	"github.com/t-beigbeder/otvl_qstf/internal/common"
	"github.com/t-beigbeder/otvl_qstf/internal/netutils"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	app := &cli.App{
		Name:  "test",
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
			var err error
			fmt.Fprintf(os.Stdout, "client %s\n", cc.String("address"))
			return err
		},
	}
}

func getClientFlags() []cli.Flag {
	return append([]cli.Flag{
		&cli.StringFlag{
			Name: "address", Aliases: []string{"a"},
			Required: true,
			Usage:    "host:port of the QUIC server",
			Action: func(cc *cli.Context, addr string) error {
				_, _, err := netutils.GetIPPort(addr)
				return err
			},
		},
	}, getTlsFlags()...)
}

func getTlsFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "skip tls verification, insecure!",
		},
		&cli.BoolFlag{
			Name:  "self",
			Usage: "generate on the fly a self-signed certificate",
		},
		&cli.StringFlag{
			Name:  "self-host",
			Usage: "host for the self-signed certificate",
		},
		&cli.StringFlag{
			Name: "cert",
			Action: func(cc *cli.Context, p string) error {
				return checkPath("cert", p)
			},
		},
		&cli.StringFlag{
			Name: "key",
			Action: func(cc *cli.Context, p string) error {
				return checkPath("key", p)
			},
		},
	}
}

func checkPath(kind, path string) error {
	if !common.FileExists(path) {
		return fmt.Errorf("%s file %s does not exist", kind, path)
	}
	return nil
}

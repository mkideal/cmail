package main

import (
	"fmt"

	"github.com/mkideal/cli"

	"github.com/mkideal/cmail/repository"
	"github.com/mkideal/cmail/smtpd/etc"
	"github.com/mkideal/cmail/smtpd/server"
)

type argT struct {
	cli.Helper
	ConfigFilename string `cli:"config" usage:"config file name" dft:"$SMTPD_CONFIG_FILE"`
	etc.Config
}

func run(ctx *cli.Context, argv *argT) error {
	if err := etc.LoadConfig(argv.ConfigFilename, ctx.Args()); err != nil {
		return err
	}

	repo, err := repository.Mysql(argv.DBSource)
	if err != nil {
		return err
	}

	// new smtp server
	svr := server.New(repo)
	onErr := func(e error) { err = e }
	addr := fmt.Sprintf("%s:%d", etc.Conf().Host, etc.Conf().Port)
	svr.Start(addr, func(e error) {
		err = e
		if err == nil {
			ctx.String("listening on %s\n", addr)
		}
	}, onErr)
	return err
}

func main() {
	//cli.SetUsageStyle(cli.ManualStyle)
	cli.Run(new(argT), func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)
		if argv.Help {
			ctx.WriteUsage()
			return nil
		}
		return run(ctx, argv)
	})
}

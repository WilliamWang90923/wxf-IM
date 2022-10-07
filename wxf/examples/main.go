package main

import (
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wangxuefeng90923/wxf/examples/mock"
)

const version = "v1"

func main() {
	flag.Parse()
	root := &cobra.Command{
		Use:     "wxf",
		Version: version,
		Short:   "tools",
	}
	ctx := context.Background()

	// mock
	root.AddCommand(mock.NewClientCmd(ctx))
	root.AddCommand(mock.NewServerCmd(ctx))

	if err := root.Execute(); err != nil {
		logrus.WithError(err).Fatal("Could not run command")
	}
}

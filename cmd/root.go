package cmd

import (
	"fmt"
	"log"
	"os"

	cc "github.com/ivanpirog/coloredcobra"
	"github.com/mikelorant/muting2/internal/app"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var (
		bind      string
		debug     bool
		host      string
		name      string
		namespace string
		service   string
	)

	cmd := &cobra.Command{
		Use:   "muting2",
		Short: "A brief description of your application",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := app.Options{
				Bind:      bind,
				Debug:     debug,
				Host:      host,
				Name:      name,
				Namespace: namespace,
				Service:   service,
			}

			if err := app.New(opts); err != nil {
				log.Fatalf("unable to start app: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Debug")
	cmd.Flags().StringVarP(&bind, "bind", "", ":8443", "Address to bind")
	cmd.Flags().StringVarP(&host, "host", "", "", "Host endpoint name")
	cmd.Flags().StringVarP(&name, "name", "", "muting", "Resource name")
	cmd.Flags().StringVarP(&namespace, "namespace", "", "default", "Resource namespace")
	cmd.Flags().StringVarP(&service, "service", "", "muting", "Resource service")

	cc.Init(&cc.Config{
		RootCmd:         cmd,
		Headings:        cc.HiGreen + cc.Bold,
		Commands:        cc.HiYellow + cc.Bold,
		Example:         cc.Italic,
		ExecName:        cc.Bold,
		Flags:           cc.Bold,
		NoExtraNewlines: true,
		NoBottomNewline: true,
	})

	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

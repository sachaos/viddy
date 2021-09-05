package main

import (
	"fmt"
	"os"

	"github.com/rivo/tview"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
)

var version string

var DefaultTheme = tview.Theme{}

func main() {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigName("viddy")
	v.AddConfigPath(xdg.ConfigHome)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(1)
		}
	}

	conf, err := newConfig(v, os.Args[1:])
	if conf.runtime.help {
		help()
		os.Exit(0)
	}

	if conf.runtime.version {
		fmt.Printf("viddy version: %s\n", version)
		os.Exit(0)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	tview.Styles = conf.theme.Theme

	app := NewViddy(conf)

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func help() {
	fmt.Println(`
Viddy well, gopher. Viddy well.

Usage:
 viddy [options] command

Options:
  -d, --differences          highlight changes between updates
  -n, --interval <interval>  seconds to wait between updates (default "2s")
  -p, --precise              attempt run command in precise intervals
  -c, --clockwork            run command in precise intervals forcibly
  -t, --no-title             turn off header
  --shell                    shell (default "sh")
  --shell-options            additional shell options

 -h, --help     display this help and exit
 -v, --version  output version information and exit`)
}

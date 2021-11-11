package main

import (
	"fmt"
	"os"

	"github.com/adrg/xdg"
	"github.com/fatih/color"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
	"github.com/tcnksm/go-latest"
)

var version string

var githubTag = &latest.GithubTag{
	Owner:             "sachaos",
	Repository:        "viddy",
	FixVersionStrFunc: latest.DeleteFrontV(),
}

func printVersion() {
	fmt.Printf("viddy version: %s\n", version)

	res, err := latest.Check(githubTag, version)
	if err == nil && res.Outdated {
		text := color.YellowString(fmt.Sprintf("%s is not latest, you should upgrade to v%s", version, res.Current))
		fmt.Fprintln(os.Stderr, text)
	}

	os.Exit(0)
}

func main() {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigName("viddy")
	v.AddConfigPath(xdg.ConfigHome)

	_ = v.ReadInConfig()

	conf, err := newConfig(v, os.Args[1:])
	if conf.runtime.help {
		help()
		os.Exit(0)
	}

	if conf.runtime.version {
		printVersion()
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
  -b, --bell                 ring terminal bell changes between updates
  -d, --differences          highlight changes between updates
  -n, --interval <interval>  seconds to wait between updates (default "2s")
  -p, --precise              attempt run command in precise intervals
  -c, --clockwork            run command in precise intervals forcibly
  -t, --no-title             turn off header
  --shell                    shell (default "sh")
  --shell-options            additional shell options
  --unfold                   unfold command result
  --pty                      run on pty (experimental, not for Windows)

 -h, --help     display this help and exit
 -v, --version  output version information and exit`)
}

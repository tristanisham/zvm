// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/tristanisham/zvm/cli"
	"github.com/tristanisham/zvm/cli/meta"

	"github.com/charmbracelet/log"

	_ "embed"
)

//go:embed help.txt
var helpTxt string

func main() {
	zvm := cli.Initialize()
	args := os.Args[1:]
	if _, ok := os.LookupEnv("ZVM_DEBUG"); ok {
		log.SetLevel(log.DebugLevel)
	}

	if len(args) == 0 {
		helpMsg()
		// zvm.AlertIfUpgradable()
		os.Exit(0)
	}

	// zvm.AlertIfUpgradable()
	versionFlag := flag.Bool("version", false, "Print ZVM version information")
	// Install flags
	installFlagSet := flag.NewFlagSet("install", flag.ExitOnError)
	installDeps := installFlagSet.String("D", "", "Specify additional dependencies to install with Zig")

	// LS flags
	lsFlagSet := flag.NewFlagSet("ls", flag.ExitOnError)
	lsRemote := lsFlagSet.Bool("all", false, "List all available versions of Zig to install")

	// Global config
	sVersionMapUrl := flag.String("vmu", "", "Set ZVM's version map URL for custom Zig distribution servers")
	sColorToggle := flag.Bool("color", true, "Turn on or off ZVM's color output")
	flag.Parse()

	if *versionFlag {
		fmt.Println(meta.VerCopy)
		os.Exit(0)
	}

	if sVersionMapUrl != nil && len(*sVersionMapUrl) != 0 {
		log.Debug("user passed vmu", "url", *sVersionMapUrl)
		switch *sVersionMapUrl {
		case "default":
			if err := zvm.Settings.ResetVersionMap(); err != nil {
				meta.CtaFatal(err)
			}
		case "mach":
			if err := zvm.Settings.SetVersionMapUrl("https://machengine.org/zig/index.json"); err != nil {
				log.Info("Run `-vmu default` to reset your version map.")
				meta.CtaFatal(err)
			}

		default:

			if err := zvm.Settings.SetVersionMapUrl(*sVersionMapUrl); err != nil {
				log.Info("Run `-vmu default` to reset your version map.")
				meta.CtaFatal(err)
			}
		}

	}

	if sColorToggle != nil {
		if *sColorToggle != zvm.Settings.UseColor {
			if *sColorToggle {
				zvm.Settings.YesColor()
			} else {
				zvm.Settings.NoColor()
			}
		}

	}

	args = flag.Args()

	for i, arg := range args {

		switch arg {

		case "install", "i":
			installFlagSet.Parse(args[i+1:])
			// signal to install zls after zig

			req := cli.ExtractInstall(args[len(args)-1])
			req.Version = strings.TrimPrefix(req.Version, "v")
			// log.Debug(req, "deps", *installDeps)

			if err := zvm.Install(req.Package); err != nil {
				meta.CtaFatal(err)
			}

			if *installDeps != "" {
				switch *installDeps {
				case "zls":
					
					zvm.InstallZls(req.Package)
				}
			}

			return
		case "use":
			if len(args) > i+1 {
				version := strings.TrimPrefix(args[i+1], "v")
				if err := zvm.Use(version); err != nil {
					meta.CtaFatal(err)
				}
			}
			return

		case "ls":
			lsFlagSet.Parse(args[i+1:])
			log.Debug("Version Map", "url", zvm.Settings.VersionMapUrl)
			if *lsRemote {
				if err := zvm.ListRemoteAvailable(); err != nil {
					meta.CtaFatal(err)
				}
			} else {
				if err := zvm.ListVersions(); err != nil {
					meta.CtaFatal(err)
				}
			}

			return
		case "uninstall", "rm":
			if len(args) > i+1 {
				version := strings.TrimPrefix(args[i+1], "v")
				if err := zvm.Uninstall(version); err != nil {
					meta.CtaFatal(err)
				}
			}
			return

		case "sync":
			if err := zvm.Sync(); err != nil {
				meta.CtaFatal(err)
			}

		case "clean":
			// msg := "Clean is a beta command, and may not be included in the next release."
			// if zvm.Settings.UseColor {
			// 	fmt.Println(clr.Blue(msg))
			// } else {
			// 	fmt.Println(msg)
			// }

			if err := zvm.Clean(); err != nil {
				if zvm.Settings.UseColor {
					meta.CtaFatal(err)
				} else {
					meta.CtaFatal(err)
				}
			}
			return

		case "upgrade":
			if err := zvm.Upgrade(); err != nil {
				log.Error("this is a new command, and may have some issues.\nConsider reporting your problem on Github :)", "github", "https://github.com/tristanisham/zvm/issues")
				meta.CtaFatal(err)
			}

		case "version":
			fmt.Println(meta.VerCopy)
			return
		case "help":
			//zvm.Settings.UseColor
			helpMsg()

			return
			// Settings
		default:
			log.Fatalf("invalid argument %q. Please run `zvm help`.\n", arg)
		}

	}

}

func helpMsg() {
	helpTmpl, err := template.New("help").Parse(helpTxt)
	if err != nil {
		fmt.Printf("Sorry! There was a rendering error (%q). The version is %s\n", err, meta.VERSION)
		fmt.Println(helpTxt)
		return
	}

	if err := helpTmpl.Execute(os.Stdout, map[string]string{"Version": meta.VERSION}); err != nil {
		fmt.Printf("Sorry! There was a rendering error (%q). The version is %s\n", err, meta.VERSION)
		fmt.Println(helpTxt)
		return
	}
}

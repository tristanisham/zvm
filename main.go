// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	// "errors"
	// "flag"
	// "fmt"
	// "html/template"
	"errors"
	"os"
	"strings"

	"github.com/tristanisham/zvm/cli"
	"github.com/tristanisham/zvm/cli/meta"
	opts "github.com/urfave/cli/v2"

	"github.com/charmbracelet/log"

	_ "embed"
)


var zvm cli.ZVM

var zvmApp = &opts.App{
	Name:      "ZVM",
	Usage: "Zig Version Manager",
	Description: "zvm lets you easily install, upgrade, and switch between different versions of Zig.",
	HelpName:  "zvm",
	Version:   meta.VERSION,
	Copyright: "Copyright © 2022 Tristan Isham",
	Suggest:   true,
	Before: func(ctx *opts.Context) error {
		zvm = *cli.Initialize()
		return nil
	},
	// app-global flags
	Flags: []opts.Flag{
		&opts.StringFlag{
			Name:  "color",
			Usage: "enable (on, yes/y, enabled, true) or disable (off, no/n, disabled, false) colored ZVM output",
			Value: "toggle",
			Action: func(ctx *opts.Context, val string) error {
				switch val {
				case "on", "yes", "enabled", "y", "true":
					zvm.Settings.YesColor()

				case "off", "no", "disabled", "n", "false":
					zvm.Settings.NoColor()

				default:
					zvm.Settings.ToggleColor()
				}

				return nil
			},
		},
	},
	Commands: []*opts.Command{
		{
			Name:    "install",
			Usage:   "download and install a version of Zig",
			Aliases: []string{"i"},
			Flags: []opts.Flag{
				&opts.BoolFlag{
					Name:    "zls",
					// Aliases: []string{"z"},
					Usage:   "install ZLS",
				},
			},
			Description: "To install the latest version, use `master`",
			Args:        true,
			ArgsUsage:   " <ZIG VERSION>",
			Action: func(ctx *opts.Context) error {
				versionArg := strings.TrimPrefix(ctx.Args().First(), "v")

				if versionArg == "" {
					return errors.New("no version provided")
				}

				req := cli.ExtractInstall(versionArg)
				req.Version = strings.TrimPrefix(req.Version, "v")

				// // Verify the requeste Zig version is good
				// if err := zvm.ZigVersionIsValid(req.Package); err != nil {
				// 	return err
				// }

				// // If ZLS install requested, verify that the versions match
				// if ctx.Bool("zls") {
				// 	if err := zvm.ZlsVersionIsValid(req.Package); err != nil {
				// 		return err
				// 	}
				// }

				// Install Zig
				if err := zvm.Install(req.Package); err != nil {
					return err
				}

				// Install ZLS (if requested)
				if ctx.Bool("zls") {
					if err := zvm.InstallZls(req.Package); err != nil {
						return err
					}
				}

				return nil
			},
		},
		{
			Name:  "use",
			Usage: "switch between versions of Zig",
			Args:  true,
			Action: func(ctx *opts.Context) error {
				versionArg := strings.TrimPrefix(ctx.Args().First(), "v")
				return zvm.Use(versionArg)
			},
		},
		{
			Name:    "list",
			Usage:   "list installed Zig versions",
			Aliases: []string{"ls"},
			Args:    true,
			Flags: []opts.Flag{
				&opts.BoolFlag{
					Name:    "all",
					Aliases: []string{"a"},
					Usage:   "list remote Zig versions available for download",
				},
			},
			Action: func(ctx *opts.Context) error {
				log.Debug("Version Map", "url", zvm.Settings.VersionMapUrl)
				if ctx.Bool("all") {
					return zvm.ListRemoteAvailable()
				} else {
					return zvm.ListVersions()
				}
			},
		},
		{
			Name:    "uninstall",
			Usage:   "remove an installed version of Zig",
			Aliases: []string{"rm"},
			Args:    true,
			Action: func(ctx *opts.Context) error {
				versionArg := strings.TrimPrefix(ctx.Args().First(), "v")
				return zvm.Uninstall(versionArg)
			},
		},
		{
			Name:  "clean",
			Usage: "remove build artifacts (good if you're a scrub)",
			Action: func(ctx *opts.Context) error {
				return zvm.Clean()
			},
		},
		{
			Name:  "upgrade",
			Usage: "self-upgrade ZVM",
			Action: func(ctx *opts.Context) error {
				return zvm.Upgrade()
			},
		},
		{
			Name:  "vmu",
			Usage: "set ZVM's version map URL for custom Zig distribution servers",
			Args:  true,
			Action: func(ctx *opts.Context) error {
				url := ctx.Args().First()
				log.Debug("user passed vmu", "url", url)

				switch url {
				case "default":
					return zvm.Settings.ResetVersionMap()

				case "mach":
					if err := zvm.Settings.SetVersionMapUrl("https://machengine.org/zig/index.json"); err != nil {
						log.Info("Run `zvm vmu default` to reset your version map.")
						return err
					}

				default:
					if err := zvm.Settings.SetVersionMapUrl(url); err != nil {
						log.Info("Run `zvm vmu default` to reset your verison map.")
						return err
					}
				}

				return nil
			},
		},
	},
}

func main() {
	if _, ok := os.LookupEnv("ZVM_DEBUG"); ok {
		log.SetLevel(log.DebugLevel)
	}

	// run and report errors
	if err := zvmApp.Run(os.Args); err != nil {
		meta.CtaFatal(err)
	}
}

// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tristanisham/clr"
	"github.com/tristanisham/zvm/cli"
	"github.com/tristanisham/zvm/cli/meta"
	opts "github.com/urfave/cli/v3"

	"github.com/charmbracelet/log"
)

var (
	zvm                      = *cli.Initialize()
	printUpgradeNotice  bool = true
	BuildUpgradeMessage      = "You should probably use your system package manager to update ZVM."
)

func init() {
	opts.HelpPrinter = printZVMHelp
}

var zvmApp = &opts.Command{
	Name:                  "zvm",
	Usage:                 "Zig Version Manager",
	Description:           "zvm lets you easily install, upgrade, and switch between different versions of Zig.",
	Version:               meta.VerCopy,
	Copyright:             fmt.Sprintf("Copyright © %d Tristan Isham", time.Now().Year()),
	Suggest:               true,
	EnableShellCompletion: true,
	ConfigureShellCompletionCommand: func(cmd *opts.Command) {
		cmd.Hidden = false
		cmd.Usage = "Generate a shell completion script"
		cmd.Description = "Emit a completion script for bash, zsh, fish, or pwsh (PowerShell).\n" +
			"Example: source <(zvm completion bash)   # bash\n" +
			"         zvm completion zsh > _zvm       # zsh, place on $fpath\n" +
			"         zvm completion pwsh > zvm.ps1   # PowerShell, dot-source in profile"

		for _, shellCmd := range cmd.Commands {
			if shellCmd.Name != "fish" {
				continue
			}

			shellCmd.Action = func(_ context.Context, command *opts.Command) error {
				completion, err := command.Root().ToFishCompletion()
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(command.Root().Writer, completion)
				return err
			}
		}
	},
	// Route errors back through main() so they are reported via meta.CtaFatal
	// instead of urfave/cli calling os.Exit itself. Without this, ExitCoder
	// errors (e.g. an unknown completion shell) exit the process mid-Run,
	// bypassing our error reporting and making Run untestable.
	ExitErrHandler: func(ctx context.Context, cmd *opts.Command, err error) {},
	// app-global flags
	Flags: []opts.Flag{
		&opts.StringFlag{
			Name:  "color",
			Usage: "enable (on, yes/y, enabled, true) or disable (off, no/n, disabled, false) colored ZVM output",
			Value: "toggle",
			Action: func(ctx context.Context, cmd *opts.Command, val string) error {
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
			Usage:   "Download and install a version of Zig",
			Aliases: []string{"i"},
			Flags: []opts.Flag{
				&opts.BoolFlag{
					Name: "zls",
					// Aliases: []string{"z"},
					Usage: "install ZLS",
				},
				&opts.BoolFlag{
					Name:    "force",
					Aliases: []string{"f"},
					Usage:   "force installation even if the version is already installed",
				},
				&opts.BoolFlag{
					Name:  "full",
					Usage: "use the 'full' zls compatibility mode",
				},
				&opts.BoolFlag{
					Name:  "nomirror",
					Usage: "download Zig from ziglang.org instead of a community mirror",
				},
				&opts.StringFlag{
					Name:    "target-os",
					Usage:   "override the target operating system (e.g., linux, macos, windows, freebsd)",
					Sources: opts.EnvVars("ZVM_TARGET_OS"),
				},
				&opts.StringFlag{
					Name:    "target-arch",
					Usage:   "override the target architecture (e.g., x86_64, aarch64, arm, riscv64)",
					Sources: opts.EnvVars("ZVM_TARGET_ARCH"),
				},
				&opts.IntFlag{
					Name:    "http.timeout",
					Usage:   "set a custom timeout for http requests",
					Sources: opts.EnvVars("ZVM_HTTP_TIMEOUT"),
				},
			},
			Description: "To install the latest version, use `master`",
			// Args:        true,
			ArgsUsage: " <ZIG VERSION>",
			Action: func(ctx context.Context, cmd *opts.Command) error {
				versionArg := strings.TrimPrefix(cmd.Args().First(), "v")

				if versionArg == "" {
					return errors.New("no version provided")
				}

				req := cli.ExtractInstall(versionArg)
				req.Version = strings.TrimPrefix(req.Version, "v")

				force := zvm.Settings.AlwaysForceInstall

				if cmd.Bool("force") {
					force = cmd.Bool("force")
				}

				zlsCompat := "only-runtime"
				if cmd.Bool("full") {
					zlsCompat = "full"
				}

				if v := cmd.String("target-os"); v != "" {
					os.Setenv("ZVM_TARGET_OS", v)
				}

				if v := cmd.String("target-arch"); v != "" {
					os.Setenv("ZVM_TARGET_ARCH", v)
				}

				// HTTP Settings
				if v := cmd.Int64("http.timeout"); v != 0 {
					os.Setenv("ZVM_HTTP_TIMEOUT", strconv.FormatInt(v, 10))
				}

				// Install Zig
				resolvedVersion, err := zvm.Install(req.Package, force, !cmd.Bool("nomirror"))
				if err != nil {
					return err
				}

				// Install ZLS (if requested)
				if cmd.Bool("zls") {
					if err := zvm.InstallZls(resolvedVersion, zlsCompat, force); err != nil {
						return err
					}
				}

				return nil
			},
		},
		{
			Name:  "use",
			Usage: "Switch between versions of Zig",
			// Args:  true,
			Flags: []opts.Flag{
				&opts.BoolFlag{
					Name:  "sync",
					Usage: "sync your current version of Zig with the repository",
				},
			},
			Action: func(ctx context.Context, cmd *opts.Command) error {
				if cmd.Bool("sync") {
					return zvm.Sync()
				} else {
					versionArg := strings.TrimPrefix(cmd.Args().First(), "v")

					if versionArg == "" {
						emptyArgErrs := fmt.Errorf("command 'use' requires 1 valid Zig version as an argument")
						minZig, err := cli.ExtractMinimumZigVersion()
						if err != nil {
							// so if there is no arg, it checks to see if it can get a value from build.zig.zon
							// if that value isn't found, it returns both errors.
							// if it is found, it pushes up the value and runs use like normal.
							emptyArgErrs = errors.Join(emptyArgErrs, err)
							return emptyArgErrs
						}

						versionArg = minZig

					}

					resolvedVer, err := zvm.Use(versionArg)
					if err != nil {
						return err
					}

					fmt.Printf("Now using Zig %s\n", resolvedVer)
					return nil
				}
			},
		},
		{
			Name:  "run",
			Usage: "Run a command with the given Zig version",
			// Args:  true,
			SkipFlagParsing: true,
			Action: func(ctx context.Context, cmd *opts.Command) error {
				versionArg := strings.TrimPrefix(cmd.Args().First(), "v")
				cmds := cmd.Args().Tail()

				log.Debug("run cmd", "version", versionArg, "args...", cmds)

				if err := zvm.Run(versionArg, cmds); err != nil {
					if errors.Is(err, cli.ErrUnsupportedVersion) || errors.Is(err, cli.ErrMissingArgument) {
						minZig, err := cli.ExtractMinimumZigVersion()
						if err != nil {
							return fmt.Errorf("version %q is not a known Zig version and no minimum_zig_version found: %w", versionArg, err)
						}
						log.Debug("falling back to minimum_zig_version", "version", versionArg, "minZig", minZig)
						redoneArgs := []string{cmd.Args().First()}
						redoneArgs = append(redoneArgs, cmds...)
						log.Debug("running with minZig", "minZig", minZig, "args", redoneArgs)
						return zvm.Run(minZig, redoneArgs)
					}
					return err
				}

				return nil

			},
		},
		{
			Name:    "list-remote",
			Usage:   "List all remote Zig versions",
			Aliases: []string{"ls-remote"},
			Flags: []opts.Flag{
				&opts.BoolFlag{
					Name:  "json",
					Usage: "print remote Zig versions as JSON",
				},
			},
			Action: func(ctx context.Context, cmd *opts.Command) error {
				if cmd.Bool("json") {
					return zvm.ListRemoteAvailableJSON()
				}

				return zvm.ListRemoteAvailable()
			},
		},
		{
			Name:    "list",
			Usage:   "List installed Zig versions.",
			Aliases: []string{"ls"},
			// Args:    true,
			Flags: []opts.Flag{
				&opts.BoolFlag{
					Name:    "all",
					Aliases: []string{"a"},
					Usage:   "list remote Zig versions available for download, based on your version map",
				},
				&opts.BoolFlag{
					Name:  "vmu",
					Usage: "list set version maps",
				},
			},
			Action: func(ctx context.Context, cmd *opts.Command) error {
				log.Debug("Version Map", "url", zvm.Settings.VersionMapUrl, "cmd", "list/ls")
				if cmd.Bool("all") {
					log.Warnf("this flag is deprecated. Please use the %s command", clr.Yellow("ls-remote"))
					return zvm.ListRemoteAvailable()
				} else if cmd.Bool("vmu") {
					if len(zvm.Settings.VersionMapUrl) == 0 {
						if err := zvm.Settings.ResetVersionMap(); err != nil {
							return err
						}
					}

					if len(zvm.Settings.ZlsVMU) == 0 {
						if err := zvm.Settings.ResetZlsVMU(); err != nil {
							return err
						}
					}

					vmu := zvm.Settings.VersionMapUrl
					zrw := zvm.Settings.ZlsVMU

					fmt.Printf("Zig VMU: %s\nZLS VMU: %s\n", vmu, zrw)
					return nil
				} else {
					return zvm.ListVersions()
				}
			},
		},
		{
			Name:    "uninstall",
			Usage:   "Remove an installed version of Zig",
			Aliases: []string{"rm"},
			// Args:    true,
			Action: func(ctx context.Context, cmd *opts.Command) error {
				versionArg := strings.TrimPrefix(cmd.Args().First(), "v")
				return zvm.Uninstall(versionArg)
			},
		},
		{
			Name:  "clean",
			Usage: "Remove build artifacts (good if you're a scrub)",
			Action: func(ctx context.Context, cmd *opts.Command) error {
				return zvm.Clean()
			},
		},
		{
			Name:  "upgrade",
			Usage: "Self-upgrade ZVM",
			Action: func(ctx context.Context, cmd *opts.Command) error {
				if meta.NoAutoUpgrades {
					// This is where you as a distributor or builder can specify how to upgrade
					// zvm on your system.
					meta.CtaBuilderMsg(BuildUpgradeMessage)
				}
				printUpgradeNotice = false
				return zvm.Upgrade()

			},
		},
		{
			Name:  "mirrorlist",
			Usage: "Set ZVM's mirror list URL for custom Zig distribution servers, or set to \"disabled\" to download directly from ziglang.org",
			Action: func(ctx context.Context, cmd *opts.Command) error {
				url := cmd.Args().First()
				log.Debug("user passed mirrorlist", "url", url)

				switch url {
				case "default":
					return zvm.Settings.ResetMirrorList()

				default:
					if err := zvm.Settings.SetMirrorListUrl(url); err != nil {
						if url == "" {
							err = fmt.Errorf("%wURL cannot be an empty string", err)
						}
						fmt.Println("Run `zvm mirrorlist default` to reset your mirror list.")
						return err
					}
				}

				return nil
			},
		},
		{
			Name:  "version",
			Usage: "Print the current version of ZVM",
			Action: func(ctx context.Context, cmd *opts.Command) error {
				fmt.Println("zvm version " + meta.VerCopy)
				return nil
			},
		},
		{
			Name:  "vmu",
			Usage: "Set ZVM's version map URL for custom Zig distribution servers",
			// Args:  true,
			Commands: []*opts.Command{
				{
					Name:  "zig",
					Usage: "Set ZVM's version map URL for custom Zig distribution servers",
					// Args:      true,
					ArgsUsage: "",

					Action: func(ctx context.Context, cmd *opts.Command) error {
						url := cmd.Args().First()
						log.Debug("user passed VMU", "url", url)

						switch url {
						case "default":
							return zvm.Settings.ResetVersionMap()

						case "mach":
							if err := zvm.Settings.SetVersionMapUrl("https://machengine.org/zig/index.json"); err != nil {
								fmt.Println("Run `zvm vmu zig default` to reset your version map.")
								return err
							}

						default:
							if err := zvm.Settings.SetVersionMapUrl(url); err != nil {
								fmt.Println("Run `zvm vmu zig default` to reset your verison map.")
								return err
							}
						}

						return nil
					},
				},
				{
					Name:  "zls",
					Usage: "Set ZVM's version map URL for custom ZLS Release Workers",
					// Args:  true,
					Action: func(ctx context.Context, cmd *opts.Command) error {
						url := cmd.Args().First()
						log.Debug("user passed zrw", "url", url)

						switch url {
						case "default":
							return zvm.Settings.ResetZlsVMU()

						default:
							if err := zvm.Settings.SetZlsVMU(url); err != nil {
								fmt.Println("Run `zvm vmu zls default` to reset your release worker.")
								return err
							}
						}

						return nil
					},
				},
			},
		},
	},
}

func main() {
	if meta.Debug {
		log.SetLevel(log.DebugLevel)
	}

	_, checkUpgradeDisabled := os.LookupEnv("ZVM_SET_CU")

	if meta.NoAutoUpgrades {
		checkUpgradeDisabled = true
	}

	log.Debug("Automatic Upgrade Checker", "disabled", checkUpgradeDisabled, "noAutoUpgrades", meta.NoAutoUpgrades)

	// Upgrade
	upSig := make(chan string, 1)

	if !checkUpgradeDisabled {
		go func(out chan<- string) {
			if tag, ok, _ := cli.CanIUpgrade(); ok {
				out <- tag
			} else {
				out <- ""
			}
		}(upSig)
	} else {
		upSig <- ""
	}

	// run and report errors
	if err := zvmApp.Run(context.Background(), os.Args); err != nil {
		// 		if meta.VERSION == "v0.7.9" && errors.Is(err, cli.ErrInvalidVersionMap) {
		// 			meta.CtaGeneric("Help", `Encountered an issue while trying to install ZLS for Zig 'master'.

		// Problem: ZVM v0.7.7 and v0.7.8 may have saved an invalid 'zlsVersionMapUrl' to your settings,
		// which causes this error. The latest version, v0.7.9, can fix this issue by using the correct URL.

		// To resolve this:
		// 1. Open your ZVM settings file: '~/.zvm/settings.json'
		// 2. Remove the 'zlsVersionMapUrl' key & value from the file (if present).
		// What happens next: ZVM will automatically use the correct version map the next time you run it
		// If the issue persists, please double-check your settings and try again, or create a GitHub Issue.`)
		// 		}

		// Handler-directive errors control how we shut down without showing
		// the user the wrapped message (it is only debug-logged).
		switch {
		case errors.Is(err, cli.ErrFailQuietly):
			log.Debug("failing quietly", "error", err)
			os.Exit(1)
		case errors.Is(err, cli.ErrFailClean):
			log.Debug("failing clean", "error", err)
			if cerr := zvm.Clean(); cerr != nil {
				log.Debug("clean failed while handling ErrFailClean", "error", cerr)
			}
			os.Exit(1)
		default:
			meta.CtaFatal(err)
		}
	}

	if tag := <-upSig; tag != "" {

		if !meta.NoAutoUpgrades {
			if printUpgradeNotice {
				meta.CtaUpgradeAvailable(tag)
			} else {
				fmt.Printf("You are now using ZVM %s\n", tag)
			}
		}

	}
}

func formatHelpSection(section string) string {
	if !zvm.Settings.UseColor {
		return section
	}
	return clr.Blue(section)
}

func printZVMHelp(w io.Writer, templ string, data any) {
	styledTemplate := strings.NewReplacer(
		"GLOBAL OPTIONS:", `{{section "GLOBAL OPTIONS:"}}`,
		"NAME:", `{{section "NAME:"}}`,
		"USAGE:", `{{section "USAGE:"}}`,
		"VERSION:", `{{section "VERSION:"}}`,
		"DESCRIPTION:", `{{section "DESCRIPTION:"}}`,
		"COMMANDS:", `{{section "COMMANDS:"}}`,
		"OPTIONS:", `{{section "OPTIONS:"}}`,
		"CATEGORY:", `{{section "CATEGORY:"}}`,
		"COPYRIGHT:", `{{section "COPYRIGHT:"}}`,
	).Replace(templ)

	opts.HelpPrinterCustom(w, styledTemplate, data, map[string]any{
		"section": formatHelpSection,
	})
}

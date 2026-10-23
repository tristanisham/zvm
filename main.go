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
	"golang.org/x/mod/semver"

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
	ShellComplete:         completeZVMCommands,
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
					Name:    "skip-shasum",
					Aliases: []string{"s"},
					Usage:   "skip SHA-256 verification and the unverified-download confirmation",
				},
				&opts.BoolFlag{
					Name:    "skip-use",
					Aliases: []string{"k"},
					Usage:   "install without setting the version as active",
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
				versionArg, err := resolveVersionArg(ctx, cmd.Args().First())
				if err != nil {
					return err
				}

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

				var WellsFargo error

				// Install Zig
				resolvedVersion, err := zvm.Install(req.Package, force, cmd.Bool("skip-shasum"), !cmd.Bool("nomirror"), cmd.Bool("skip-use"))
				if err != nil {
					if !zvm.Settings.AlwaysInstallZLS {
						return err
					}
					WellsFargo = errors.Join(WellsFargo, err)
				}

				// Install ZLS (if requested)
				if cmd.Bool("zls") {
					if err := zvm.InstallZls(resolvedVersion, zlsCompat, force, cmd.Bool("skip-shasum"), cmd.Bool("skip-use")); err != nil {
						WellsFargo = errors.Join(WellsFargo, err)
					}
				}

				if WellsFargo != nil {
					return WellsFargo
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
				&opts.BoolFlag{
					Name:    "global",
					Aliases: []string{"g"},
					Usage:   "change the global default even in a shell set up with `zvm env`",
				},
			},
			Action: func(ctx context.Context, cmd *opts.Command) error {
				if cmd.Bool("sync") {
					return zvm.Sync()
				} else {
					versionArg, err := resolveVersionArg(ctx, cmd.Args().First())
					if err != nil {
						return err
					}

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

					global := cmd.Bool("global")
					resolvedVer, perShell, err := zvm.UseVersion(versionArg, global)
					if err != nil {
						return err
					}

					switch {
					case perShell:
						fmt.Printf("Now using Zig %s in this shell only (run `zvm use --global %s` to change the default)\n", resolvedVer, resolvedVer)
					case global && os.Getenv(cli.MultishellEnv) != "":
						fmt.Printf("Now using Zig %s as the global default (this shell keeps its own version; run `zvm use %s` to switch it)\n", resolvedVer, resolvedVer)
					default:
						fmt.Printf("Now using Zig %s\n", resolvedVer)
					}
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
				versionArg, err := resolveVersionArg(ctx, cmd.Args().First())
				if err != nil {
					return err
				}
				cmds := cmd.Args().Tail()

				log.Debug("run cmd", "version", versionArg, "args...", cmds)

				if err := zvm.Run(versionArg, cmds); err != nil {
					if errors.Is(err, cli.ErrUnsupportedVersion) || errors.Is(err, cli.ErrMissingArgument) {
						minZig, err := cli.ExtractMinimumZigVersion()
						if err != nil {
							return fmt.Errorf("version %q is not a known Zig version and no minimum_zig_version found: %w", versionArg, err)
						}
						log.Debug("falling back to minimum_zig_version", "version", versionArg, "minZig", minZig)
						// Without an explicit version, the first argument is
						// part of the Zig command. Preserve its original spelling.
						normalizedVersion := versionArg
						if strings.HasPrefix(normalizedVersion, ".") {
							normalizedVersion = "0" + normalizedVersion
						}
						if versionArg != "" && !semver.IsValid("v"+normalizedVersion) {
							cmds = cmd.Args().Slice()
						}
						return zvm.Run(minZig, cmds)
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
			Name:    "alias",
			Usage:   "Give an installed Zig version a name",
			Aliases: []string{"kv"},
			Flags: []opts.Flag{
				&opts.BoolFlag{
					Name:  "clear",
					Usage: "clear all aliases",
				},
				&opts.BoolFlag{
					Name:    "delete",
					Aliases: []string{"d"},
					Usage:   "delete the alias for the given name",
				},
			},
			Action: func(ctx context.Context, cmd *opts.Command) error {
				name := cmd.Args().First()

				if cmd.Bool("clear") {
					return zvm.ClearAliases(ctx)
				}
				if cmd.Bool("delete") {
					return zvm.DeleteAlias(ctx, name)
				}

				switch cmd.Args().Len() {
				case 0:
					aliases, err := zvm.ListAliases(ctx)
					if err != nil {
						return err
					}
					zvm.PrintAliases(aliases)
					return nil
				case 1:
					return zvm.Alias(ctx, name, nil)
				case 2:
					value := cmd.Args().Get(1)
					return zvm.Alias(ctx, name, &value)
				default:
					return fmt.Errorf("%w: alias accepts at most a name and version", cli.ErrInvalidInput)
				}
			},
		},
		{
			Name:    "uninstall",
			Usage:   "Remove an installed version of Zig",
			Aliases: []string{"rm"},
			// Args:    true,
			Action: func(ctx context.Context, cmd *opts.Command) error {
				versionArg, err := resolveVersionArg(ctx, cmd.Args().First())
				if err != nil {
					return err
				}
				return zvm.Uninstall(versionArg)
			},
		},
		{
			Name:  "clean",
			Usage: "Remove build artifacts (good if you're a scrub)",
			Description: "Removes downloaded archives from the ZVM base directory, dedupe-store\n" +
				"entries no installed version links to any more, and per-shell links\n" +
				"(from `zvm env`) that are over a week old and whose shell has exited.",
			Action: func(ctx context.Context, cmd *opts.Command) error {
				if err := zvm.Clean(); err != nil {
					return err
				}
				pruned, err := zvm.PruneStore()
				if err != nil {
					return err
				}
				if pruned.Removed > 0 {
					fmt.Printf("Removed %d unreferenced store files (%s)\n", pruned.Removed, cli.FormatBytes(pruned.BytesFreed))
				}
				n, err := cli.PruneMultishells(cli.MultishellMaxAge)
				if n > 0 {
					fmt.Printf("Removed %d stale per-shell link(s)\n", n)
				}
				return err
			},
		},
		{
			Name:  "env",
			Usage: "Print shell code that gives this shell its own Zig version",
			Description: "Creates a per-shell link to the current global Zig version, exports it as\n" +
				"ZVM_MULTISHELL_PATH and prepends it to PATH. Afterwards `zvm use` switches\n" +
				"only that shell; `zvm use --global` changes the default.\n" +
				"With --use-on-cd, entering a directory switches to the version declared by\n" +
				"the nearest build.zig (`//! zvm-lock: <version>`) or build.zig.zon\n" +
				"(`.minimum_zig_version`) in it or any parent.\n" +
				"Example: eval \"$(zvm env --shell bash --use-on-cd)\"          # bash, zsh\n" +
				"         zvm env --shell fish --use-on-cd | source            # fish\n" +
				"         zvm env --shell powershell --use-on-cd | Out-String | Invoke-Expression",
			Flags: []opts.Flag{
				&opts.StringFlag{
					Name:  "shell",
					Usage: "shell to emit code for: bash, zsh, fish, or powershell (default: from $SHELL)",
				},
				&opts.BoolFlag{
					Name:  "use-on-cd",
					Usage: "switch Zig automatically when entering a directory with a version file",
				},
			},
			Action: func(ctx context.Context, cmd *opts.Command) error {
				// This output is eval'd by the shell, so errors must go to
				// stderr: anything on stdout would be run as shell code.
				shell, err := cli.DetectShell()
				if name := cmd.String("shell"); name != "" {
					shell, err = cli.NormalizeShell(name)
				}
				if err == nil {
					err = zvm.Env(os.Stdout, shell, cmd.Bool("use-on-cd"), os.Getppid())
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "zvm env: %v\n", err)
					return errors.Join(cli.ErrFailQuietly, err)
				}
				return nil
			},
		},
		{
			Name:      "dedupe",
			Usage:     "Hard-link identical files shared between installed Zig versions",
			ArgsUsage: " [ZIG VERSION...]",
			Description: "Hashes every regular file in the given installed versions (default: all)\n" +
				"and replaces duplicates with hard links into ~/.zvm/store, so files that\n" +
				"are identical across versions are stored on disk once. Files are only\n" +
				"merged when content, permissions and owner all match. Safe to re-run\n" +
				"and to interrupt. Files that cannot be hard-linked are left alone.\n\n" +
				"Note: a deduped file is shared, so editing one in place edits it in\n" +
				"every version that shares it.",
			Flags: []opts.Flag{
				&opts.StringFlag{
					Name:  "auto",
					Usage: "set whether `zvm install` dedupes new versions automatically (on/off), then exit",
				},
			},
			Action: func(ctx context.Context, cmd *opts.Command) error {
				if cmd.IsSet("auto") {
					var enabled bool
					switch strings.ToLower(cmd.String("auto")) {
					case "on", "yes", "enabled", "y", "true":
						enabled = true
					case "off", "no", "disabled", "n", "false":
						enabled = false
					default:
						return fmt.Errorf("%w: --auto expects on or off", cli.ErrInvalidInput)
					}
					if err := zvm.Settings.SetAutoDedupe(enabled); err != nil {
						return err
					}
					if enabled {
						fmt.Println("Auto-dedupe after install: ON. Run `zvm dedupe` once to fold in versions installed earlier.")
					} else {
						fmt.Println("Auto-dedupe after install: OFF")
					}
					return nil
				}

				versions := make([]string, 0, cmd.Args().Len())
				for _, arg := range cmd.Args().Slice() {
					v, err := resolveVersionArg(ctx, arg)
					if err != nil {
						return err
					}
					versions = append(versions, v)
				}

				report, err := zvm.Dedupe(versions)
				if err != nil {
					return err
				}
				pruned, err := zvm.PruneStore()
				if err != nil {
					return err
				}

				fmt.Printf("Deduped %d version(s): %s\n", len(report.Versions), strings.Join(report.Versions, ", "))
				fmt.Printf("  files scanned:         %d\n", report.Scanned)
				fmt.Printf("  added to store:        %d\n", report.Stored)
				fmt.Printf("  replaced by hard link: %d\n", report.Linked)
				fmt.Printf("  already deduped:       %d\n", report.AlreadyLinked)
				fmt.Printf("  ignored (empty/special): %d\n", report.Ignored)
				fmt.Printf("  left untouched:        %d\n", report.Skipped)
				if pruned.Removed > 0 {
					fmt.Printf("  unreferenced store files removed: %d\n", pruned.Removed)
				}
				fmt.Printf("Disk used by versions + store: %s -> %s (saved %s)\n",
					cli.FormatBytes(report.BytesBefore), cli.FormatBytes(report.BytesAfter-pruned.BytesFreed),
					cli.FormatBytes(report.BytesSaved()+pruned.BytesFreed))
				fmt.Printf("  exact bytes: %d -> %d\n", report.BytesBefore, report.BytesAfter-pruned.BytesFreed)

				const maxProblems = 20
				for i, p := range report.Problems {
					if i == maxProblems {
						fmt.Printf("  ... and %d more (run with ZVM_DEBUG=1 to see all)\n", len(report.Problems)-maxProblems)
						break
					}
					fmt.Println("  note:", p)
				}
				return nil
			},
		},
		{
			Name:   cli.OnCdCommand,
			Usage:  "Internal: called by the `zvm env --use-on-cd` hook",
			Hidden: true,
			Action: func(ctx context.Context, cmd *opts.Command) error {
				zvm.UseOnCd(os.Stdout, os.Stderr)
				return nil
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
			Name:      "settings",
			Usage:     "View or edit settings.json (opens an editor when no path is given)",
			ArgsUsage: "[.key [value]]",
			Description: "With no arguments in a terminal, open an interactive editor (vim keys or arrows).\n" +
				"Select a key with jq dot notation to print it, or pass a value to replace it:\n" +
				"  zvm settings .useColor           # print a value\n" +
				"  zvm settings .useColor false     # validate and save a new value\n" +
				"  zvm settings --check             # report invalid entries in settings.json",
			Flags: []opts.Flag{
				&opts.BoolFlag{
					Name:  "check",
					Usage: "report invalid entries in settings.json and exit non-zero if there are any",
				},
				&opts.BoolFlag{
					Name:    "raw-output",
					Aliases: []string{"r"},
					Usage:   "print string values without JSON quotes, like jq -r",
				},
			},
			Action: func(ctx context.Context, cmd *opts.Command) error {
				return zvm.SettingsCommand(cmd.Args().Slice(), cmd.Bool("check"), cmd.Bool("raw-output"), os.Stdout)
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
			Name:  "bug",
			Usage: "Start a bug report",
			Description: "Open a prefilled ZVM issue in your browser.\n" +
				"The report includes your ZVM version, settings, and host details.\n" +
				"Paths under your home directory are shortened to ~, and custom mirror\n" +
				"or version-map URLs are replaced by a digest. Review it before submitting.",
			Flags: []opts.Flag{
				&opts.BoolFlag{
					Name:  "print",
					Usage: "print the report URL instead of opening a browser",
				},
			},
			Action: func(ctx context.Context, cmd *opts.Command) error {
				return zvm.Bug(cmd.Bool("print"))
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
							if err := zvm.Settings.SetVersionMapUrl(cli.MachVersionMapUrl); err != nil {
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

	// `zvm env` output is eval'd by the shell and the cd hook runs on every
	// directory change: neither may wait on the network or print a notice.
	// Only the subcommand counts: `zvm run 0.15.1 env` runs `zig env` and
	// should still check for upgrades.
	if sub := subcommand(os.Args[1:]); sub == "env" || sub == cli.OnCdCommand {
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

	// `zvm settings` reports problems itself.
	if subcommand(os.Args[1:]) != "settings" {
		zvm.Settings.WarnProblems()
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

func resolveVersionArg(ctx context.Context, versionArg string) (string, error) {
	aliasValue, ok, err := zvm.ResolveAlias(ctx, versionArg)
	if err != nil {
		return "", err
	}
	if ok {
		versionArg = aliasValue
	}
	return strings.TrimPrefix(versionArg, "v"), nil
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

// completeZVMCommands extends urfave/cli's live command completion with the
// aliases accepted by ZVM. The default completer only emits each command's
// canonical name, even though aliases are part of the public CLI surface.
func completeZVMCommands(ctx context.Context, cmd *opts.Command) {
	opts.DefaultCompleteWithFlags(ctx, cmd)

	for _, command := range cmd.VisibleCommands() {
		for _, alias := range command.Aliases {
			if command.Usage == "" {
				fmt.Fprintln(cmd.Root().Writer, alias)
				continue
			}
			fmt.Fprintf(cmd.Root().Writer, "%s:%s\n", alias, command.Usage)
		}
	}
}

// subcommand returns the name of the command in args (os.Args[1:]), skipping
// global flags and the value of --color.
func subcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--color" || arg == "-color":
			i++ // skip its value
		case strings.HasPrefix(arg, "-"):
		default:
			return arg
		}
	}
	return ""
}

package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"strings"
	"zvm/cli"
	"zvm/cli/meta"

	"github.com/charmbracelet/log"

	_ "embed"

	"github.com/tristanisham/clr"
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

	// Install flags
	installFlagSet := flag.NewFlagSet("install", flag.ExitOnError)
    installDeps := installFlagSet.String("D", "", "Specify additional dependencies to install with Zig")

	// LS flags
	lsFlagSet := flag.NewFlagSet("ls", flag.ExitOnError)
	lsRemote := lsFlagSet.Bool("all", false, "List all available versions of Zig to install")

	for i, arg := range args {
		switch arg {
		case "install", "i":
			installFlagSet.Parse(args[i+1:])
			// signal to install zls after zig

			req := cli.ExtractInstall(args[len(args)-1])
			req.Version = strings.TrimPrefix(req.Version, "v")
			log.Debug(req, "deps", *installDeps)

			if err := zvm.Install(req.Package); err != nil {
				log.Fatal(err)
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
					log.Fatal(err)
				}
			}
			return

		case "ls":
			lsFlagSet.Parse(args[i+1:])
			
			if *lsRemote {
				if err := zvm.ListRemoteAvailable(); err != nil {
					log.Fatal(err)
				}
			} else {
				if err := zvm.ListVersions(); err != nil {
					log.Fatal(err)
				}
			}
			
			return
		case "uninstall", "rm":
			if len(args) > i+1 {
				version := strings.TrimPrefix(args[i+1], "v")
				if err := zvm.Uninstall(version); err != nil {
					log.Fatal(err)
				}
			}
			return

		case "sync":
			if err := zvm.Sync(); err != nil {
				log.Fatal(err)
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
					log.Fatal(clr.Red(err))
				} else {
					log.Fatal(err)
				}
			}
			return

		case "upgrade":
			if err := zvm.Upgrade(); err != nil {
				log.Error("this is a new command, and may have some issues. Consider reporting your problem on Github :)", "github", "https://github.com/tristanisham/zvm/issues")
				log.Fatal(err)
			}

		case "version", "--version", "-v":
			fmt.Println(meta.VERSION)
			return
		case "help", "--help", "-h":
			//zvm.Settings.UseColor
			helpMsg()

			return
			// Settings
		case "--nocolor", "--nocolour":
			zvm.Settings.NoColor()
		case "--color", "--colour":
			zvm.Settings.ToggleColor()
		case "--yescolor", "--yescolour":
			zvm.Settings.YesColor()
		default:
			fmt.Printf("ERROR: Invalid argument %s. Please check out --help.\n", arg)
			os.Exit(1)
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

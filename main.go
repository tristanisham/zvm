package main

import (
	"fmt"
	"github.com/charmbracelet/log"
	"html/template"
	"os"
	"strings"
	"zvm/cli"
	"zvm/cli/meta"

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
		alertIfUpgradable(zvm.Settings.UseColor)
		os.Exit(0)
	}

	alertIfUpgradable(zvm.Settings.UseColor)

	for i, arg := range args {
		switch arg {
		case "install", "i":
			if len(args) > i+1 {
				version := strings.TrimPrefix(args[i+1], "v")
				if err := zvm.Install(version); err != nil {
					log.Fatal(err)
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
			if err := zvm.ListVersions(); err != nil {
				log.Fatal(err)
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

// alertIfUpgradable checks to see if a new version of ZVM is out, and alerts the user via stdout.
func alertIfUpgradable(colors bool) {

	if _, noCheckUpgrade := os.LookupEnv("ZVM_NO_CHECK_UPDATE"); !noCheckUpgrade {
		upgradable, tagName, err := cli.CanIUpgrade()
		if err != nil {
			log.Info("failed new zvm version check")
		}

		if upgradable {
			coloredText := "zvm upgrade"
			if colors {
				coloredText = clr.Blue("zvm upgrade")
			}

			fmt.Printf("There's a new version of ZVM (%s).\n Run '%s' to install it!\n", tagName, coloredText)
		}
	}

}

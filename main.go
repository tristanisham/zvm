package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"zvm/cli"

	"github.com/tristanisham/clr"
)

func main() {
	zvm := cli.Initialize()
	args := os.Args[1:]
	for i, arg := range args {
		switch arg {
		case "install", "i":
			if len(args) > i+1 {
				version := strings.TrimPrefix(args[i+1], "v")
				if err := zvm.Install(version); err != nil {
					log.Fatal(err)
				}
			}
		case "use":
			if len(args) > i+1 {
				version := strings.TrimPrefix(args[i+1], "v")
				if err := zvm.Use(version); err != nil {
					log.Fatal(err)
				}
			}
		case "ls":
			if err := zvm.ListVersions(); err != nil {
				log.Fatal(err)
			}
		case "uninstall", "rm":
			if len(args) > i+1 {
				version := strings.TrimPrefix(args[i+1], "v")
				if err := zvm.Uninstall(version); err != nil {
					log.Fatal(err)
				}
			}
		case "clean":
			msg := "Clean is a beta command, and may not be included in the next release."
			if zvm.Settings.UseColor {
				fmt.Println(clr.Blue(msg))
			} else {
				fmt.Println(msg)
			}

			if err := zvm.Clean(); err != nil {
				if zvm.Settings.UseColor {
					log.Fatal(clr.Red(err))
				} else {
					log.Fatal(err)
				}
			}
		case "version", "--version", "-v":
			fmt.Println("zvm v0.1.2")
			return
		case "help", "--help", "-h":
			var help string
			if zvm.Settings.UseColor {
				help += clr.Blue("Install\n\t")
				help += clr.White("zvm i/install ") + clr.Green("<zig version>\n")
				help += clr.Blue("Use\n\t")
				help += clr.White("zmv use ") + clr.Green("<zig version>\n")
				help += clr.Blue("Version\n\t")
				help += clr.White("version\n")
				help += clr.Blue("Help\n\t")
				help += clr.White("help\n")
				fmt.Println(help)
				return
			} else {
				help += "Install\n\t"
				help += "zvm i/install <zig version>\n"
				help += "Use\n\t"
				help += "zmv use <zig version>\n"
				help += "Version\n\t"
				help += "version\n"
				help += "Help\n\t"
				help += "help\n"
				fmt.Println(help)
				return
			}
			// Settings
		case "--nocolor", "--nocolour":
			zvm.Settings.NoColor()
		case "--color", "--colour":
			zvm.Settings.ToggleColor()
		case "--yescolor", "--yescolour":
			zvm.Settings.YesColor()
		}

	}

}

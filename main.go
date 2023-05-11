package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"zvm/cli"

	"github.com/tristanisham/clr"
	_ "embed"
)

//go:embed help.txt
var helpTxt string
var VERSION = "v0.1.7"

func main() {
	zvm := cli.Initialize()
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println(helpTxt)
		os.Exit(0)
	}
	
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
			fmt.Println(VERSION)
			return
		case "help", "--help", "-h":
			//zvm.Settings.UseColor 
			fmt.Println(helpTxt)
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

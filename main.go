package main

import (
	"fmt"
	"log"
	"os"
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
				if err := zvm.Install(args[i+1]); err != nil {
					log.Fatal(err)
				}
			}
		case "use":
			if len(args) > i+1 {
				if err := zvm.Use(args[i+1]); err != nil {
					log.Fatal(err)
				}
			}
		case "version", "--version", "-v":
			fmt.Println("zvm v0.0.1-beta.1")
			return
		case "help", "--help", "-h":
			var help string
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
		}
	}

}

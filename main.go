package main

import (
	"os"
	"zvm/cli"
)

func main() {
	args := os.Args[1:]
	for i, arg := range args {
		switch arg {
		case "install", "i":
			if len(args) > i+1 {
				cli.Install(args[i+1])
			}
		}
	}

}

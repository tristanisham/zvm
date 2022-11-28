package main

import (
	"log"
	"os"
	"zvm/cli"
)

func main() {
	args := os.Args[1:]
	for i, arg := range args {
		switch arg {
		case "install", "i":
			if len(args) > i+1 {
				if err := cli.Install(args[i+1]); err != nil {
					log.Fatal(err)
				}
			}
		}
	}

}

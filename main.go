package main

import (
	"log"
	"os"
	"zvm/cli"
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
		}
	}

}

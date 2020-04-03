package main

import (
	"log"
	"os"

	"github.com/pilosa/tools/loader"
)

func main() {
	if err := loader.Run(os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"flag"
	"fmt"
)

func main() {
	flags, err := NewFlags()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		flag.Usage()
		return
	}

	switch flags.Mode() {
	case "record":
		record(flags.Path(), flags.Interrupt())
	case "playback":
		playback(flags.Path(), flags.Wait(), flags.Trim())
	}
}

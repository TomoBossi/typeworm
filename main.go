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
		err = Record(flags.Path(), flags.Interrupt(), flags.Overwrite())
	case "playback":
		err = Playback(flags.Path(), flags.Wait(), flags.Trim())
	}
	if err != nil {
		panic(err)
	}
}

package main

import (
	"flag"
	"fmt"
	"path/filepath"
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
		if !flags.Session() {
			config := recordConfiguration{
				path:      flags.Path(),
				stop:      flags.StopKey(),
				overwrite: flags.Overwrite(),
			}
			if err = Record(config); err != nil {
				panic(err)
			}
		}
	case "playback":
		if !flags.Session() {
			config := playbackConfiguration{
				path:      flags.Path(),
				wait:      flags.Wait(),
				trim:      flags.Trim(),
				blacklist: []string{},
			}
			if err = Playback(config); err != nil {
				panic(err)
			}
		} else {
			pathQueue := []string{}
			startIndex := 0
			dirPath := ""
			isDirectory, err := isDir(flags.Path())
			if err != nil {
				panic(err)
			}
			if isDirectory {
				dirPath = flags.Path()
			} else {
				dirPath = filepath.Dir(flags.Path())
			}
			pathQueue, err = listDir(dirPath, ".tw")
			if err != nil {
				panic(err)
			}
			if !isDirectory {
				fmt.Println(pathQueue)
				fmt.Println(flags.Path())
				startIndex = indexOf(filepath.Join(dirPath, filepath.Base(flags.Path())), pathQueue)
				fmt.Println(startIndex)
			}
			config := playbackSessionConfiguration{
				pathQueue:  pathQueue,
				startIndex: uint(startIndex),
				wait:       flags.Wait(),
				trim:       flags.Trim(),
				loop:       flags.Loop(),
				blacklist:  []string{flags.StopKey(), flags.NextKey(), flags.RedoKey()},
				stop:       flags.StopKey(),
				next:       flags.NextKey(),
				redo:       flags.RedoKey(),
			}
			if err = PlaybackSession(config); err != nil {
				panic(err)
			}
		}
	}
}

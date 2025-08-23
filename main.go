package main

import (
	"flag"
	"fmt"
	"path/filepath"
)

func main() {
	flags, err := NewFlags()
	if err != nil {
		flag.Usage()
		fmt.Println()
		panic(err)
	}

	switch flags.Mode() {
	case "record":
		if !flags.Session() {
			config := recordConfiguration{
				keyboard:  nil,
				path:      flags.Path(),
				stop:      flags.StopKey(),
				overwrite: flags.Overwrite(),
			}
			if err = Record(config); err != nil {
				panic(err)
			}
		} else {
			config := recordSessionConfiguration{
				keyboard:   nil,
				pathFormat: flags.Path(),
				offset:     flags.Offset(),
				overwrite:  flags.Overwrite(),
				stop:       flags.StopKey(),
				next:       flags.NextKey(),
				redo:       flags.RedoKey(),
			}
			if err = RecordSession(config); err != nil {
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
				startIndex = indexOf(filepath.Join(dirPath, filepath.Base(flags.Path())), pathQueue)
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

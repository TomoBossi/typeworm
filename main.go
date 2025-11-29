package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/tomobossi/kyev"
)

func main() {
	flags, err := newFlags()
	if err != nil {
		flag.Usage()
		fmt.Println()
		panic(err)
	}

	keyboard, err := kyev.GetKeyboard(flags.keyboardNameMatch, flags.keyboardPhysMatch)
	if err != nil {
		panic(err)
	}

	switch flags.mode {
	case "record":
		if !flags.session {
			config := recordConfiguration{
				keyboard:  keyboard,
				path:      flags.path,
				stop:      flags.stopKey,
				overwrite: flags.overwrite,
			}
			if err = Record(config); err != nil {
				panic(err)
			}
		} else {
			config := recordSessionConfiguration{
				keyboard:   keyboard,
				pathFormat: flags.path,
				offset:     flags.offset,
				overwrite:  flags.overwrite,
				stop:       flags.stopKey,
				next:       flags.nextKey,
				redo:       flags.redoKey,
			}
			if err = RecordSession(config); err != nil {
				panic(err)
			}
		}
	case "playback":
		if !flags.session {
			config := playbackConfiguration{
				path:      flags.path,
				wait:      flags.wait,
				trim:      flags.trim,
				blacklist: []string{},
			}
			if err = Playback(config); err != nil {
				panic(err)
			}
		} else {
			var pathQueue []string
			startIndex := 0
			dirPath := ""
			isDirectory, err := isDir(flags.path)
			if err != nil {
				panic(err)
			}
			if isDirectory {
				dirPath = flags.path
			} else {
				dirPath = filepath.Dir(flags.path)
			}
			pathQueue, err = listDir(dirPath, ".tw")
			if err != nil {
				panic(err)
			}
			if !isDirectory {
				startIndex = indexOf(filepath.Join(dirPath, filepath.Base(flags.path)), pathQueue)
			}
			config := playbackSessionConfiguration{
				keyboard:   keyboard,
				pathQueue:  pathQueue,
				startIndex: uint(startIndex),
				wait:       flags.wait,
				trim:       flags.trim,
				loop:       flags.loop,
				blacklist:  []string{flags.stopKey, flags.nextKey, flags.redoKey},
				stop:       flags.stopKey,
				next:       flags.nextKey,
				redo:       flags.redoKey,
			}
			if err = PlaybackSession(config); err != nil {
				panic(err)
			}
		}
	}
}

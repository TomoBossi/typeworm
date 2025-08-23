package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"slices"
	"time"
)

type flags struct {
	loop      bool
	mode      string
	nextKey   string
	offset    uint
	overwrite bool
	path      string
	redoKey   string
	session   bool
	stopKey   string
	trim      bool
	wait      time.Duration
}

func NewFlags() (*flags, error) {
	loop := flag.Bool("loop", true, "DEFAULT true - Loop through the files in the directory pointed to by PATH during a playback session. If the SESSION flag is not set or MODE is not playback, this flag will be ignored.")
	mode := flag.String("mode", "", "REQUIRED - Start in record/rec/r or playback/play/p mode.")
	offset := flag.Uint("offset", 0, "DEFAULT 0 - Starting value for the autoincremental numeric identifier used to render the file names throughout a session of recording. If MODE is not record or the SESSION flag is not set, this flag will be ignored.")
	overwrite := flag.Bool("overwrite", false, "DEFAULT false - Overwrite the contents of the file pointed to by PATH if it already exists. If the file doesn't exist or MODE is playback, this flag will be ignored.")
	path := flag.String("path", "", "REQUIRED - Path to the .tw file to record to or play back from. When MODE is record, the file will be created if it doesn't exist. To overwrite the file if it already exists, set the OVERWRITE flag. If the SESSION flag is set, the file name must include a single %d as a placeholder for an autoincremental numerical identifier which can be set to start at a custom value using the OFFSET flag. When mode is PLAYBACK, the file must already exist. If the SESSION flag is set, PATH can point to an existing directory instead of a file, in which case all .tw files in the directory will be played back in ascending alphanumerical order. If PATH points to a file, it will be used as the entry point for playback of .tw files in its immediate parent directory")
	nextKey := flag.String("next-key", "LEFTCTRL", "DEFAULT LEFTCTRL - Label of the key used to continue to the next file in a session of recording or playback. The chosen label must be mapped to a known keycode. If the SESSION flag is not set, this flag will be ignored.")
	redoKey := flag.String("redo-key", "LEFTSHIFT", "DEFAULT LEFTSHIFT - Label of the key used to redo a recording throughout a recording session. The chosen label must be mapped to a known keycode. If the SESSION flag is not set, this flag will be ignored.")
	session := flag.Bool("session", false, "DEFAULT false - Prevent typeworm from stopping after recording to or playing back from a single file. If the file name contains %d and MODE is record, typeworm will first try to record to the file with %d equal to the OFFSET flag. After recording the offset value will be autoincremented to record to the next file in the sequence.")
	stopKey := flag.String("stop-key", "ESC", "DEFAULT ESC - Label of the key used to stop recording or playing back and exit typeworm. The chosen label must be mapped to a known keycode, and won't be recorded.")
	trim := flag.Bool("trim", true, "DEFAULT true - Skip the leading deadtime between the start of the recording and the first input during playback. If MODE is record, this flag will be ignored.")
	wait := flag.Uint("wait", 0, "DEFAULT 0 - Time between inputs during playback (milliseconds). If not specified or 0, inputs will be played back according to their exact timings as they were recorded. If MODE is record, this flag will be ignored.")
	flag.Parse()

	// invalid mode
	if !slices.Contains([]string{"record", "rec", "r", "playback", "play", "p"}, *mode) {
		return nil, fmt.Errorf("mode must be record/rec/r or playback/play/p")
	} else if slices.Contains([]string{"record", "rec", "r"}, *mode) {
		*mode = "record"
	} else {
		*mode = "playback"
	}

	// missing path
	if *path == "" {
		return nil, fmt.Errorf("file path not provided")
	}

	// check if path contains a single integer format specifier %d at the base
	hasSpecifier, err := hasIntegerSpecifier(*path)
	if err != nil {
		return nil, err
	}

	// check if path is directory
	paths := []string{}
	isDirectory := false
	filePath := *path

	if *mode == "record" && *session {
		if !hasSpecifier {
			return nil, fmt.Errorf("path base must contain an integer format specifier for a recording session")
		}
		filePath = fmt.Sprintf(filePath, *offset)
	}

	if *mode == "playback" && *session {
		isDirectory, err := isDir(filePath)
		if err != nil {
			return nil, err
		}
		if isDirectory {
			paths, err = listDir(filePath, ".tw")
			if err != nil {
				return nil, err
			}
		}
	}

	// path contains an integer format specifier (outside record session mode)
	if hasSpecifier && !(*mode == "record" && *session) {
		return nil, fmt.Errorf("path cannot contain an integer format specifier unless in a recording session")
	}

	// path doesn't point to a file (outside playback session mode)
	if isDirectory && !(*mode == "playback" && *session) {
		return nil, fmt.Errorf("path must point to a file unless in a playback session")
	}

	// path is a directory in playback session mode but contains no .tw files
	if isDirectory && *mode == "playback" && *session && len(paths) == 0 {
		return nil, fmt.Errorf("directory must contain at least one .tw file")
	}

	// invalid file extension
	if !isDirectory && filepath.Ext(*path) != ".tw" {
		return nil, fmt.Errorf("file must have a .tw extension")
	}

	// file doesn't exist in playback mode
	if err := checkExistsPlayback(*path); err != nil && *mode == "playback" && !isDirectory {
		return nil, err
	}

	// file exists with no overwrite directive in record mode
	if err := checkExistsRecord(filePath, *overwrite); err != nil && *mode == "record" {
		return nil, err
	}

	// invalid key label for stop key
	if _, ok := labelKeycode[*stopKey]; !ok {
		return nil, fmt.Errorf("unknown key label for stop key")
	}

	// invalid key label for next key
	if _, ok := labelKeycode[*nextKey]; !ok && *session {
		return nil, fmt.Errorf("unknown key label for next key")
	}

	// invalid key label for redo key
	if _, ok := labelKeycode[*redoKey]; !ok && *session {
		return nil, fmt.Errorf("unknown key label for redo key")
	}

	// overlapping keys
	if *session && (*stopKey == *nextKey || *stopKey == *redoKey || *nextKey == *redoKey) {
		return nil, fmt.Errorf("cannot set overlapping keys for stop, next and redo actions")
	}

	return &flags{
		loop:      *loop,
		mode:      *mode,
		nextKey:   *nextKey,
		offset:    *offset,
		overwrite: *overwrite,
		path:      *path,
		redoKey:   *redoKey,
		session:   *session,
		stopKey:   *stopKey,
		trim:      *trim,
		wait:      time.Duration(*wait) * time.Millisecond,
	}, nil
}

func (f flags) Loop() bool {
	return f.loop
}

func (f flags) Mode() string {
	return f.mode
}

func (f flags) NextKey() string {
	return f.nextKey
}

func (f flags) Offset() uint {
	return f.offset
}

func (f flags) Overwrite() bool {
	return f.overwrite
}

func (f flags) Path() string {
	return f.path
}

func (f flags) RedoKey() string {
	return f.redoKey
}

func (f flags) Session() bool {
	return f.session
}

func (f flags) StopKey() string {
	return f.stopKey
}

func (f flags) Trim() bool {
	return f.trim
}

func (f flags) Wait() time.Duration {
	return f.wait
}

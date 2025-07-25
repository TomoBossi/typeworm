package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type flags struct {
	interrupt string
	mode      string
	path      string
	wait      time.Duration
	trim      bool
	overwrite bool
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	return err == nil, err
}

func checkExistsPlayback(path string) error {
	if pathExists, _ := exists(path); !pathExists {
		return fmt.Errorf("file must exist for playback mode")
	}
	return nil
}

func checkExistsRecord(path string, overwrite bool) error {
	if pathExists, _ := exists(path); pathExists && !overwrite {
		return fmt.Errorf("file already exists, and the overwrite flag was not set")
	}
	return nil
}

func NewFlags() (*flags, error) {
	mode := flag.String("mode", "", "REQUIRED - Start in record/rec/r or playback/play/p mode.")
	path := flag.String("path", "", "REQUIRED - Path to the .tw file to record to or play back from. When recording, the file will be created if it doesn't exist. To overwrite the file if it already exists, set the overwrite flag. When playing back, the file must already exist.")
	wait := flag.Uint("wait", 0, "Time between inputs during playback (milliseconds). If not provided or 0, inputs will be played back according to their exact timings as they were recorded. If mode is record, this flag will be ignored.")
	trim := flag.Bool("trim", true, "Skip the leading deadtime between the start of the recording and the first input during playback. This flag is true by default. If mode is record, this flag will be ignored.")
	overwrite := flag.Bool("overwrite", false, "Overwrite the contents of the file pointed to by path if it already exists. If the file doesn't exist or mode is playback, this flag will be ignored.")
	interrupt := flag.String("interrupt", "ESC", "Label of the key used to exit typeworm. The chosen label must be mapped to a known keycode, and won't be recorded. This flag is ESC by default.")
	flag.Parse()

	if !slices.Contains([]string{"record", "rec", "r", "playback", "play", "p"}, *mode) {
		return nil, fmt.Errorf("mode must be record/rec/r or playback/play/p")
	} else if slices.Contains([]string{"record", "rec", "r"}, *mode) {
		*mode = "record"
	} else {
		*mode = "playback"
	}

	if *path == "" {
		return nil, fmt.Errorf("file path not provided")
	}

	if filepath.Ext(*path) != ".tw" {
		return nil, fmt.Errorf("file must have a .tw extension")
	}

	if err := checkExistsPlayback(*path); err != nil && *mode == "playback" {
		return nil, err
	}

	if err := checkExistsRecord(*path, *overwrite); err != nil && *mode == "record" {
		return nil, err
	}

	if _, ok := labelKeycode[*interrupt]; !ok {
		return nil, fmt.Errorf("unknown key label for interrupt key")
	}

	waitDuration := time.Duration(*wait) * time.Millisecond

	return &flags{
		interrupt: *interrupt,
		mode:      *mode,
		path:      *path,
		wait:      waitDuration,
		trim:      *trim,
		overwrite: *overwrite,
	}, nil
}

func (f flags) Interrupt() string {
	return f.interrupt
}

func (f flags) Mode() string {
	return f.mode
}

func (f flags) Path() string {
	return f.path
}

func (f flags) Wait() time.Duration {
	return f.wait
}

func (f flags) Trim() bool {
	return f.trim
}

func (f flags) Overwrite() bool {
	return f.overwrite
}

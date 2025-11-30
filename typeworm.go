package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tomobossi/keynput"
	"github.com/tomobossi/kyev"
)

type input struct {
	timestamp time.Duration
	key       string
}

type recordConfiguration struct {
	keyboard  *kyev.Keyboard
	path      string
	stop      string
	overwrite bool
}

type recordSessionConfiguration struct {
	keyboard   *kyev.Keyboard
	pathFormat string
	offset     uint
	overwrite  bool
	stop       string
	next       string
	redo       string
}

type playbackConfiguration struct {
	path      string
	wait      time.Duration
	trim      bool
	blacklist []string
}

type playbackSessionConfiguration struct {
	keyboard   *kyev.Keyboard
	pathQueue  []string
	startIndex uint
	wait       time.Duration
	trim       bool
	loop       bool
	blacklist  []string
	stop       string
	next       string
	redo       string
}

func fmtDuration(duration time.Duration) string {
	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60
	milliseconds := int(duration.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, milliseconds)
}

func parseDuration(s string) (time.Duration, error) {
	duration, err := time.ParseDuration(strings.NewReplacer(":", "m", ".", "s").Replace(s) + "ms")
	if err != nil {
		return 0, err
	}

	return duration, nil
}

func sleep(start time.Time, timestamp, deadtime time.Duration, wait time.Duration, trim bool, first bool) {
	if first && !trim {
		time.Sleep(deadtime)
	} else if !first {
		if wait == 0 {
			if trim {
				time.Sleep(timestamp - time.Since(start) - deadtime)
			} else {
				time.Sleep(timestamp - time.Since(start))
			}
		} else {
			time.Sleep(wait)
		}
	}
}

func Record(config recordConfiguration) error {
	if err := checkExistsRecord(config.path, config.overwrite); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(config.path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(config.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	var inputs []input
	start := time.Now()
	recording := true

	fmt.Printf("recording keys to %s - press %s to stop\n", config.path, config.stop)
	for recording {
		keypresses, err := config.keyboard.GetKeyPresses()
		if err != nil {
			continue
		}

		for _, keypress := range keypresses {
			if keypress.Code == kyev.LabelKeycodeMap[config.stop] {
				recording = false
				break
			} else if label, ok := kyev.KeycodeLabelMap[keypress.Code]; ok {
				inputs = append(inputs, input{time.Since(start), label})
			}
		}
	}

	writer := bufio.NewWriter(file)
	for _, i := range inputs {
		writer.WriteString(fmtDuration(i.timestamp) + " " + i.key + "\n")
	}
	writer.Flush()
	fmt.Printf("%d inputs were recorded\n", len(inputs))
	return nil
}

func RecordSession(config recordSessionConfiguration) error {
	if config.keyboard == nil {
		var err error
		config.keyboard, err = kyev.GetKeyboard("keyboard", "usb")
		if err != nil {
			return err
		}
	}

	recordConfig := recordConfiguration{
		keyboard:  config.keyboard,
		path:      "",
		stop:      config.stop,
		overwrite: config.overwrite,
	}

	i := config.offset
	last := i
	continueRecording := true

	for {
		if continueRecording {
			recordConfig.path = fmt.Sprintf(config.pathFormat, i)
			err := Record(recordConfig)
			if err != nil {
				return err
			}

			last = i
			i++

			continueRecording = false
			fmt.Printf("press %s to stop, %s to start recording to %s, or %s to record over the last file again\n", config.stop, config.next, fmt.Sprintf(config.pathFormat, i), config.redo)
		}

		start := time.Now()
		keypresses, err := config.keyboard.GetKeyPresses()
		if err != nil {
			return err
		}

		stopRecording := false
		for _, keypress := range keypresses {
			if timevalToTime(keypress.Time).After(start) {
				switch keypress.Code {
				case kyev.LabelKeycodeMap[config.stop]:
					stopRecording = true
				case kyev.LabelKeycodeMap[config.next]:
					recordConfig.overwrite = config.overwrite
					continueRecording = true
				case kyev.LabelKeycodeMap[config.redo]:
					i = last
					recordConfig.overwrite = true
					continueRecording = true
				}
			}
		}
		if stopRecording {
			break
		}
	}
	return nil
}

func Playback(config playbackConfiguration) error {
	if err := checkExistsPlayback(config.path); err != nil {
		return err
	}

	file, err := os.Open(config.path)
	if err != nil {
		return err
	}
	defer file.Close()

	var inputs []input
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 {
			return fmt.Errorf("file contains a malformed line")
		}
		timestamp, err := parseDuration(parts[0])
		if err != nil {
			return err
		}
		if slices.Contains(config.blacklist, parts[1]) {
			return fmt.Errorf("file contains a blacklisted key")
		}
		inputs = append(inputs, input{timestamp, parts[1]})
	}

	if len(inputs) == 0 {
		return fmt.Errorf("file does not contain recorded inputs")
	}

	virtualKeyboard, err := keynput.NewKeyboard("typeworm")
	if err != nil {
		return err
	}
	defer virtualKeyboard.Close()

	fmt.Printf("playing back from %s\n", config.path)
	start := time.Now()
	deadtime := inputs[0].timestamp
	for j, i := range inputs {
		sleep(start, i.timestamp, deadtime, config.wait, config.trim, j == 0)
		if code, ok := kyev.LabelKeycodeMap[i.key]; ok {
			err := virtualKeyboard.KeyPress(code)
			if err != nil {
				return err
			}
		}
	}
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("%d inputs were played back\n", len(inputs))
	return nil
}

func timevalToTime(tv kyev.Timeval) time.Time {
	return time.Unix(int64(tv.Sec), int64(tv.Usec)*1000)
}

func PlaybackSession(config playbackSessionConfiguration) error {
	playbackConfig := playbackConfiguration{
		path:      "",
		wait:      config.wait,
		trim:      config.trim,
		blacklist: config.blacklist,
	}

	i := config.startIndex
	last := i
	numPaths := uint(len(config.pathQueue))
	continuePlayback := true

	for i < numPaths {
		if continuePlayback {
			playbackConfig.path = config.pathQueue[i]
			err := Playback(playbackConfig)
			if err != nil {
				return err
			}

			last = i
			i++
			if i == numPaths {
				if config.loop {
					i = 0
				} else {
					break
				}
			}

			continuePlayback = false
			fmt.Printf("press %s to stop, %s to start playing back from %s, or %s to play back from last file again\n", config.stop, config.next, config.pathQueue[i], config.redo)
		}

		start := time.Now()
		keypresses, err := config.keyboard.GetKeyPresses()
		if err != nil {
			return err
		}

		stopPlayback := false
		for _, keypress := range keypresses {
			if timevalToTime(keypress.Time).After(start) {
				switch keypress.Code {
				case kyev.LabelKeycodeMap[config.stop]:
					stopPlayback = true
				case kyev.LabelKeycodeMap[config.next]:
					continuePlayback = true
				case kyev.LabelKeycodeMap[config.redo]:
					i = last
					continuePlayback = true
				}
			}
		}
		if stopPlayback {
			break
		}
	}
	return nil
}

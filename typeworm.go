package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/bendahl/uinput"
	evdev "github.com/gvalkov/golang-evdev"
)

var keycodeLabel = map[uint16]string{ // typeworm can be extended by mapping new keycodes to unique labels
	evdev.KEY_0:         "0",
	evdev.KEY_1:         "1",
	evdev.KEY_2:         "2",
	evdev.KEY_3:         "3",
	evdev.KEY_4:         "4",
	evdev.KEY_5:         "5",
	evdev.KEY_6:         "6",
	evdev.KEY_7:         "7",
	evdev.KEY_8:         "8",
	evdev.KEY_9:         "9",
	evdev.KEY_A:         "A",
	evdev.KEY_B:         "B",
	evdev.KEY_C:         "C",
	evdev.KEY_D:         "D",
	evdev.KEY_E:         "E",
	evdev.KEY_F:         "F",
	evdev.KEY_G:         "G",
	evdev.KEY_H:         "H",
	evdev.KEY_I:         "I",
	evdev.KEY_J:         "J",
	evdev.KEY_K:         "K",
	evdev.KEY_L:         "L",
	evdev.KEY_M:         "M",
	evdev.KEY_N:         "N",
	evdev.KEY_O:         "O",
	evdev.KEY_P:         "P",
	evdev.KEY_Q:         "Q",
	evdev.KEY_R:         "R",
	evdev.KEY_S:         "S",
	evdev.KEY_T:         "T",
	evdev.KEY_U:         "U",
	evdev.KEY_V:         "V",
	evdev.KEY_W:         "W",
	evdev.KEY_X:         "X",
	evdev.KEY_Y:         "Y",
	evdev.KEY_Z:         "Z",
	evdev.KEY_UP:        "UP",
	evdev.KEY_DOWN:      "DOWN",
	evdev.KEY_LEFT:      "LEFT",
	evdev.KEY_RIGHT:     "RIGHT",
	evdev.KEY_ESC:       "ESC",
	evdev.KEY_LEFTCTRL:  "LEFTCTRL",
	evdev.KEY_LEFTSHIFT: "LEFTSHIFT",
}

var labelKeycode = func() map[string]uint16 {
	m := make(map[string]uint16)
	for keycode, label := range keycodeLabel {
		m[label] = keycode
	}
	return m
}()

type input struct {
	timestamp time.Duration
	key       string
}

type recordConfiguration struct {
	keyboard  *evdev.InputDevice
	path      string
	stop      string
	overwrite bool
}

type recordSessionConfiguration struct {
	keyboard   *evdev.InputDevice
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

func findKeyboard() (*evdev.InputDevice, error) {
	var keyboard *evdev.InputDevice
	devices, _ := evdev.ListInputDevices()
	for _, dev := range devices {
		if strings.Contains(strings.ToLower(dev.Name), "keyboard") {
			if strings.Contains(strings.ToLower(dev.Phys), "usb") {
				return dev, nil
			} else if keyboard == nil {
				keyboard = dev
			}
		}
	}
	if keyboard != nil {
		return keyboard, nil
	}
	return nil, fmt.Errorf("no keyboard device found")
}

func Record(config recordConfiguration) error {
	if config.keyboard == nil {
		var err error
		config.keyboard, err = findKeyboard()
		if err != nil {
			return err
		}
	}

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
		events, err := config.keyboard.Read()
		if err != nil {
			continue
		}

		for _, ev := range events {
			if ev.Type == evdev.EV_KEY && ev.Value == 1 {
				if ev.Code == labelKeycode[config.stop] {
					recording = false
					break
				} else if label, ok := keycodeLabel[ev.Code]; ok {
					inputs = append(inputs, input{time.Since(start), label})
				}
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
		config.keyboard, err = findKeyboard()
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
		events, err := config.keyboard.Read()
		if err != nil {
			return err
		}

		stopRecording := false
		for _, ev := range events {
			if ev.Type == evdev.EV_KEY && ev.Value == 1 && timevalToTime(ev.Time).After(start) {
				switch ev.Code {
				case labelKeycode[config.stop]:
					stopRecording = true
				case labelKeycode[config.next]:
					recordConfig.overwrite = config.overwrite
					continueRecording = true
				case labelKeycode[config.redo]:
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

	virtualKeyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("typeworm"))
	if err != nil {
		return err
	}
	defer virtualKeyboard.Close()

	fmt.Printf("playing back from %s\n", config.path)
	start := time.Now()
	deadtime := inputs[0].timestamp
	for j, i := range inputs {
		sleep(start, i.timestamp, deadtime, config.wait, config.trim, j == 0)
		if code, ok := labelKeycode[i.key]; ok {
			_ = virtualKeyboard.KeyPress(int(code))
		}
	}
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("%d inputs were played back\n", len(inputs))
	return nil
}

func timevalToTime(tv syscall.Timeval) time.Time {
	return time.Unix(int64(tv.Sec), int64(tv.Usec)*1000)
}

func PlaybackSession(config playbackSessionConfiguration) error {
	playbackConfig := playbackConfiguration{
		path:      "",
		wait:      config.wait,
		trim:      config.trim,
		blacklist: config.blacklist,
	}

	keyboard, err := findKeyboard()
	if err != nil {
		return err
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
		events, err := keyboard.Read()
		if err != nil {
			return err
		}

		stopPlayback := false
		for _, ev := range events {
			if ev.Type == evdev.EV_KEY && ev.Value == 1 && timevalToTime(ev.Time).After(start) {
				switch ev.Code {
				case labelKeycode[config.stop]:
					stopPlayback = true
				case labelKeycode[config.next]:
					continuePlayback = true
				case labelKeycode[config.redo]:
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

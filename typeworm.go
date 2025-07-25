package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bendahl/uinput"
	evdev "github.com/gvalkov/golang-evdev"
)

type input struct {
	timestamp time.Duration
	key       string
}

var keycodeLabel = map[uint16]string{ // typeworm can be extended by mapping new keycodes to unique labels
	evdev.KEY_0:     "0",
	evdev.KEY_1:     "1",
	evdev.KEY_2:     "2",
	evdev.KEY_3:     "3",
	evdev.KEY_4:     "4",
	evdev.KEY_5:     "5",
	evdev.KEY_6:     "6",
	evdev.KEY_7:     "7",
	evdev.KEY_8:     "8",
	evdev.KEY_9:     "9",
	evdev.KEY_A:     "A",
	evdev.KEY_B:     "B",
	evdev.KEY_C:     "C",
	evdev.KEY_D:     "D",
	evdev.KEY_E:     "E",
	evdev.KEY_F:     "F",
	evdev.KEY_G:     "G",
	evdev.KEY_H:     "H",
	evdev.KEY_I:     "I",
	evdev.KEY_J:     "J",
	evdev.KEY_K:     "K",
	evdev.KEY_L:     "L",
	evdev.KEY_M:     "M",
	evdev.KEY_N:     "N",
	evdev.KEY_O:     "O",
	evdev.KEY_P:     "P",
	evdev.KEY_Q:     "Q",
	evdev.KEY_R:     "R",
	evdev.KEY_S:     "S",
	evdev.KEY_T:     "T",
	evdev.KEY_U:     "U",
	evdev.KEY_V:     "V",
	evdev.KEY_W:     "W",
	evdev.KEY_X:     "X",
	evdev.KEY_Y:     "Y",
	evdev.KEY_Z:     "Z",
	evdev.KEY_UP:    "UP",
	evdev.KEY_DOWN:  "DOWN",
	evdev.KEY_LEFT:  "LEFT",
	evdev.KEY_RIGHT: "RIGHT",
	evdev.KEY_ESC:   "ESC",
}

var labelKeycode = func() map[string]uint16 {
	m := make(map[string]uint16)
	for keycode, label := range keycodeLabel {
		m[label] = keycode
	}
	return m
}()

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

func Record(path, interrupt string, overwrite bool) error {
	err := checkExistsRecord(path, overwrite)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	keyboard, err := findKeyboard()
	if err != nil {
		return err
	}

	var inputs []input
	start := time.Now()
	recording := true

	fmt.Printf("recording keys to %s - press %s to stop\n", path, interrupt)
	for recording {
		events, err := keyboard.Read()
		if err != nil {
			continue
		}

		for _, ev := range events {
			if ev.Type == evdev.EV_KEY && ev.Value == 1 {
				timestamp := time.Since(start)
				if ev.Code == labelKeycode[interrupt] {
					recording = false
					break
				} else if label, ok := keycodeLabel[ev.Code]; ok {
					inputs = append(inputs, input{timestamp, label})
				}
			}
		}
	}

	writer := bufio.NewWriter(file)
	for _, i := range inputs {
		writer.WriteString(fmtDuration(i.timestamp) + " " + i.key + "\n")
	}
	writer.Flush()
	return nil
}

func Playback(path string, wait time.Duration, trim bool) error {
	err := checkExistsPlayback(path)
	if err != nil {
		return err
	}

	file, err := os.Open(path)
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

	start := time.Now()
	deadtime := inputs[0].timestamp
	for j, i := range inputs {
		sleep(start, i.timestamp, deadtime, wait, trim, j == 0)
		if code, ok := labelKeycode[i.key]; ok {
			_ = virtualKeyboard.KeyPress(int(code))
		}
	}
	return nil
}

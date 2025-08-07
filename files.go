package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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

func hasIntegerSpecifier(path string) (bool, error) {
	if isDirectory, err := isDir(path); err != nil {
		return false, err
	} else if isDirectory && strings.Contains(path, "%d") {
		return false, fmt.Errorf("file parent directories cannot include format specifiers")
	}
	if dir := filepath.Dir(path); strings.Contains(dir, "%d") {
		return false, fmt.Errorf("file parent directories cannot include format specifiers")
	}
	filename := filepath.Base(path)
	count := strings.Count(filename, "%d")
	if count > 1 {
		return false, fmt.Errorf("file name cannot include more than one integer format specifier")
	}
	return count == 1, nil
}

func isDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Println("Error:", err)
		return false, err
	}
	return info.IsDir(), nil
}

func listDir(path, ext string) ([]string, error) {
	if isDirectory, err := isDir(path); err != nil {
		return nil, err
	} else if !isDirectory {
		return nil, fmt.Errorf("path must point to a directory")
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ext || ext == "") {
			paths = append(paths, filepath.Join(path, entry.Name()))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func indexOf[T comparable](element T, slice []T) int {
	for i, value := range slice {
		if value == element {
			return i
		}
	}
	return -1
}

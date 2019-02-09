// +build !windows

package main

import (
	"os"
	"os/exec"
	"path"
)

func getFallbackCacheDirname() string {
	return path.Join(os.Getenv("HOME"), ".cache")
}

func openURL(url string) error {
	return exec.Command("open", url).Start()
}

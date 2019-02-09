package main

import (
	"os"
	"os/exec"
)

func getFallbackCacheDirname() string {
	return os.Getenv("LOCALAPPDATA")
}

func openURL(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

package main

import (
	"os"
)

func getFallbackCacheDirname() string {
	return os.Getenv("LOCALAPPDATA")
}

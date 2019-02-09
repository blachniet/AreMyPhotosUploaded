// +build !windows

package main

import (
	"os"
	"path"
)

func getFallbackCacheDirname() string {
	return path.Join(os.Getenv("HOME"), ".cache")
}

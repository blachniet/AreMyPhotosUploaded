// +build darwin linux

package main

import (
	"os"
	"path"
)

func init() {
	cacheDir := os.Getenv("XDG_CACHE_HOME")

	if cacheDir == "" {
		cacheDir = path.Join(os.Getenv("HOME"), ".cache")
	}

	cacheDirname = path.Join(cacheDir, "blachniet.com", "AreMyPhotosUploaded")
	oauthConfigFilename = path.Join(cacheDirname, "oauth.json")
	tokenFilename = path.Join(cacheDirname, "token.json")
}

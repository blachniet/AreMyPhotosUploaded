package main

import (
	"os"
	"path"
)

var (
	cacheDirname        string
	oauthConfigFilename string
	tokenFilename       string
)

func init() {
	cacheDir := os.Getenv("XDG_CACHE_HOME")

	if cacheDir == "" {
		cacheDir = os.Getenv("LOCALAPPDATA")
	}

	cacheDirname = path.Join(cacheDir, "blachniet.com", "AreMyPhotosUploaded")
	oauthConfigFilename = path.Join(cacheDirname, "oauth.json")
	tokenFilename = path.Join(cacheDirname, "token.json")
}

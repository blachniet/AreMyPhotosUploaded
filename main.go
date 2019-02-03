package main

// gcloud projects create my-sample-photo-helper
// gcloud config set project my-sample-photo-helper
// gcloud services enable photoslibrary.googleapis.com
//
// # Create OAuth Client ID
// # https://console.developers.google.com/apis/credentials/oauthclient
//
// # Configure consent screen - just fill out application name
//
// # Create "Other" OAuth client
// # Copy Client ID and client secret
//
// Path to cache token file:
//

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/photoslibrary/v1"
)

var (
	cacheDirname        string
	oauthConfigFilename string
	tokenFilename       string
)

// Problems:
// photoslibrary doesn't expose the file name. I would need to compare
// purely based on timestamps, and camera metadata. Maybe I could go
// next-level and do some image comparisons, but that seems like overkill
func main() {
	dirname := os.Args[1]
	ctx := context.Background()

	// Ensure the cache directory exists
	if err := os.MkdirAll(cacheDirname, 0700); err != nil {
		log.Fatalf("Error creating cache directory: %+v", err)
	}

	// Create a temporary output file
	outputFile, err := ioutil.TempFile("", "armpup-*.html")
	if err != nil {
		log.Fatalf("Error creating output file: %+v", err)
	}
	defer outputFile.Close()

	// Get an HTTP client for talking to the Google Photos service
	client, err := getClient(ctx)
	if err != nil {
		log.Fatalf("Error creating HTTP client: %+v", err)
	}

	librarian, err := newLibrarian(client)
	if err != nil {
		log.Fatalf("Error creating librarian: %+v", err)
	}

	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		log.Fatalf("Error reading directory: %+v", err)
	}

	fmt.Fprintln(outputFile, htmlHeader)

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		filename := path.Join(dirname, f.Name())
		exifDate, err := getExifDate(filename)
		if err != nil {
			// Missing EXIF data
			fmt.Fprintf(outputFile, "<tr class=\"warning\">")
			fmt.Fprintf(outputFile, "<td title=\"%v\">%v</td>", filename, f.Name())
			fmt.Fprintf(outputFile, "<td><img src=\"%v\"></td>", filename)
			fmt.Fprintf(outputFile, "<td>No EXIF date found.</td>")
			fmt.Fprintln(outputFile, "</tr>")
		} else {
			mediaItem, err := librarian.getPhotoByDate(ctx, exifDate)
			if err != nil {
				log.Fatalf("Error retrieving media item from librarian: %+v", err)
			}

			if mediaItem == nil {
				// No match found
				fmt.Fprintf(outputFile, "<tr class=\"danger\">")
				fmt.Fprintf(outputFile, "<td title=\"%v\">%v</td>", filename, f.Name())
				fmt.Fprintf(outputFile, "<td><img src=\"%v\"></td>", filename)
				fmt.Fprintf(outputFile, "<td>Not Found!</td>")
				fmt.Fprintln(outputFile, "</tr>")
			} else {
				// Match found
				fmt.Fprintf(outputFile, "<tr>")
				fmt.Fprintf(outputFile, "<td title=\"%v\">%v</td>", filename, f.Name())
				fmt.Fprintf(outputFile, "<td><img src=\"%v\">", filename)
				fmt.Fprintf(outputFile, "<td><a href=\"%v\"><img src=\"%v\"></a>", mediaItem.ProductUrl, mediaItem.BaseUrl)
				fmt.Fprintln(outputFile, "</tr>")
			}
		}
	}

	fmt.Fprintf(outputFile, htmlFooter)
	fmt.Printf("Output written to %v\n", outputFile.Name())
}

func getExifDate(path string) (time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return time.Unix(0, 0), err
	}

	x, err := exif.Decode(f)
	if err != nil {
		return time.Unix(0, 0), err
	}

	return x.DateTime()
}

func getOAuthConfig() (*oauth2.Config, error) {
	var config oauth2.Config

	// Try to parse the configuration from a cache file
	configFile, err := os.Open(oauthConfigFilename)
	if err == nil {
		err = json.NewDecoder(configFile).Decode(&config)
		configFile.Close()
		if err == nil {
			return &config, nil
		}
	}

	// If we didn't find a config in the cache, prompt for one.
	fmt.Printf("OAuth Client ID: ")
	fmt.Scan(&config.ClientID)
	fmt.Printf("OAuth Client Secret:")
	fmt.Scan(&config.ClientSecret)

	config.Endpoint = google.Endpoint
	config.Scopes = []string{photoslibrary.PhotoslibraryReadonlyScope}
	config.RedirectURL = "urn:ietf:wg:oauth:2.0:oob"

	// Save the config in the cache
	configFile, err = os.Create(oauthConfigFilename)
	if err != nil {
		return nil, err
	}

	return &config, json.NewEncoder(configFile).Encode(&config)
}

func getClient(ctx context.Context) (*http.Client, error) {
	// Get the OAuth configuration
	config, err := getOAuthConfig()
	if err != nil {
		return nil, err
	}

	// Use an existing token if we can
	// -------------------------------------------
	var token *oauth2.Token
	tokenFile, err := os.Open(tokenFilename)
	if err == nil {
		err = json.NewDecoder(tokenFile).Decode(&token)
		if err == nil {
			return config.Client(ctx, token), nil
		}
	}

	// We didn't find a token, acquire a new one
	// -------------------------------------------
	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	url := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Printf("1. Authorize this application: %v\n", url)
	fmt.Printf("2. Paste the authorization code here: ")

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, err
	}

	token, err = config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	// Serialize the token to a file for later use
	tokenFile, err = os.Create(tokenFilename)
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(tokenFile).Encode(token)
	if err != nil {
		return nil, err
	}

	client := config.Client(ctx, token)
	return client, err
}

const (
	htmlHeader = `
<html>
<head>
	<style>
		img { width: 200px; }

		.warning {
			background-color: #ffc107;
			color: #343a40;
		}

		.danger {
			background-color: #dc3545;
			color: #ffffff;
		}
	</style>
</head>
<body>
	<table>`
	htmlFooter = `
	</table>
</body>
</html>`
)

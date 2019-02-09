package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
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

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("You must provide 1 argument, the path to the directory containing pictures.")
	}

	dirname := os.Args[1]
	ctx := context.Background()
	cacheDirname = getCacheDirname()
	oauthConfigFilename = path.Join(cacheDirname, "oauth.json")
	tokenFilename = path.Join(cacheDirname, "token.json")

	// Ensure the cache directory exists
	if err := os.MkdirAll(cacheDirname, 0700); err != nil {
		log.Fatalf("Error creating cache directory: %+v", err)
	}

	// Create a HTML report file
	htmlFile, err := ioutil.TempFile("", "armpup-*.html")
	if err != nil {
		log.Fatalf("Error creating HTML report file: %+v", err)
	}
	defer htmlFile.Close()

	// Create a text file containing paths to all files that we get a match for
	matchesFile, err := ioutil.TempFile("", "armpup-*.matches.txt")
	if err != nil {
		log.Fatalf("Error creating matches text file: %+v", err)
	}
	defer matchesFile.Close()

	fmt.Printf("Report : %v\n", htmlFile.Name())
	fmt.Printf("Matches: %v\n", matchesFile.Name())

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

	fmt.Fprintln(htmlFile, htmlHeader)

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		filename := path.Join(dirname, f.Name())
		exifDate, err := getExifDate(filename)
		if err != nil {
			// Missing EXIF data
			fmt.Fprintf(htmlFile, "<tr class=\"warning\">")
			fmt.Fprintf(htmlFile, "<td title=\"%v\">%v</td>", filename, f.Name())
			fmt.Fprintf(htmlFile, "<td><img src=\"%v\"></td>", filename)
			fmt.Fprintf(htmlFile, "<td>No EXIF date found.</td>")
			fmt.Fprintln(htmlFile, "</tr>")
		} else {
			mediaItem, err := librarian.getPhotoByDate(ctx, exifDate)
			if err != nil {
				log.Fatalf("Error retrieving media item from librarian: %+v", err)
			}

			if mediaItem == nil {
				// No match found
				fmt.Fprintf(htmlFile, "<tr class=\"danger\">")
				fmt.Fprintf(htmlFile, "<td title=\"%v\">%v</td>", filename, f.Name())
				fmt.Fprintf(htmlFile, "<td><img src=\"%v\"></td>", filename)
				fmt.Fprintf(htmlFile, "<td>Not Found!</td>")
				fmt.Fprintln(htmlFile, "</tr>")
			} else {
				// Match found
				fmt.Fprintln(matchesFile, filename)

				fmt.Fprintf(htmlFile, "<tr>")
				fmt.Fprintf(htmlFile, "<td title=\"%v\">%v</td>", filename, f.Name())
				fmt.Fprintf(htmlFile, "<td><img src=\"%v\">", filename)
				fmt.Fprintf(htmlFile, "<td><a href=\"%v\"><img src=\"%v\"></a>", mediaItem.ProductUrl, mediaItem.BaseUrl)
				fmt.Fprintln(htmlFile, "</tr>")
			}
		}
	}

	fmt.Fprintf(htmlFile, htmlFooter)
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

	// Save the config in the cache
	configFile, err = os.Create(oauthConfigFilename)
	if err != nil {
		return nil, err
	}

	return &config, json.NewEncoder(configFile).Encode(&config)
}

// getAuthCode obtains an authorization code via the loopback IP address
// https://developers.google.com/identity/protocols/OAuth2InstalledApp#redirect-uri_loopback
func getAuthCode(config *oauth2.Config) string {
	// Listen on an available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	// Acquire the port we are listening on
	port := listener.Addr().(*net.TCPAddr).Port

	// Set the redirect URL with the port we are listening on
	config.RedirectURL = fmt.Sprintf("http://127.0.0.1:%v/callback", port)

	// Open the authorization page in a web browser
	openURL(config.AuthCodeURL("state", oauth2.AccessTypeOffline))

	codeChan := make(chan string)
	defer close(codeChan)

	// Launch a goroutine that listens on the loopback and acquires the
	// authorization code.
	go func() {
		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code == "" {
				fmt.Fprintln(w, authCodeErrorPage)
			} else {
				fmt.Fprintln(w, authCodeSuccessPage)
			}

			codeChan <- code
		})

		http.Serve(listener, nil)
	}()

	// Wait for an authorization code from the above goroutine
	code := <-codeChan

	if code == "" {
		log.Fatalf("Error obtaining auth code.")
	}

	return code
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

	code := getAuthCode(config)
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

func getCacheDirname() string {
	dirname := os.Getenv("XDG_CACHE_HOME")

	if dirname == "" {
		dirname = getFallbackCacheDirname()
	}

	return path.Join(dirname, "blachniet.com", "AreMyPhotosUploaded")
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
	authCodeSuccessPage = `
<html>
<body>
	<h1>Success!</h1>
	<p>You may close this window now.</p>
</body>
</html>`
	authCodeErrorPage = `
<html>
<body>
	<h1>Error!</h1>
	<p>There was an error obtaining the authorization code. You may close this window now.</p>
</body>
</html>`
)

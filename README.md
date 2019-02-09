# AreMyPhotosUploaded

Determine whether local photos exist in your Google Photos library.

## Usage

### 1. Create a Google Cloud application

This application requires read-only access to your Google Photos library. You must create your own Google Cloud application to provide that access:

```bash
# 1. Create a new Google Cloud project. You can also perform these
#    steps through https://console.cloud.google.com/
$ gcloud projects create my-sample-photo-helper
$ gcloud config set project my-sample-photo-helper

# 2. Enable the Photos API
$ gcloud services enable photoslibrary.googleapis.com

# 3. Create OAuth Client ID
#    https://console.developers.google.com/apis/credentials/oauthclient
#    When configuring the consent screen, at least fill the "application name".

# 4. Create "Other" OAuth client. You will need the Client ID and
#    Client Secret from this step the first time you run the application.
```

### 2. Get the app and run

This assumes that you have Go installed and that `$GOPATH/bin` is in your `PATH`.

```bash
go get github.com/blachniet/AreMyPhotosUploaded
AreMyPhotosUploaded ~/Pictures
```

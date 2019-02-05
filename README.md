# AreMyPhotosUploaded

Scans a given directory for photos and looks for the same photos in your Google Photos library.

## Setup

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
#    All you need to do when configuring the consent screen is fill
#    the "application name".

# 4. Create "Other" OAuth client. You will need the Client ID and
#    Client Secret from this step the first time you run the application.
```

package main

import (
	"context"
	"net/http"
	"time"

	"google.golang.org/api/photoslibrary/v1"
)

type librarian struct {
	service *photoslibrary.Service
	cache   map[time.Time][]*photoslibrary.MediaItem
}

func newLibrarian(client *http.Client) (*librarian, error) {
	service, err := photoslibrary.New(client)
	if err != nil {
		return nil, err
	}

	return &librarian{
		service: service,
		cache:   map[time.Time][]*photoslibrary.MediaItem{},
	}, nil
}

func (l *librarian) getPhotoByDate(ctx context.Context, t time.Time) (*photoslibrary.MediaItem, error) {
	t = t.UTC()
	day := getDateUTC(t)

	// Look for the item in the cache
	itemsForDay, ok := l.cache[day]
	if !ok {
		// No cache entry for this day, populate the cache
		err := l.populateCacheForDay(ctx, t)
		if err != nil {
			return nil, err
		}

		itemsForDay = l.cache[day]
	}

	for _, mediaItem := range itemsForDay {
		if mediaItem.MediaMetadata.CreationTime == t.Format(time.RFC3339) {
			return mediaItem, nil
		}
	}

	// Not found
	return nil, nil
}

func (l *librarian) populateCacheForDay(ctx context.Context, t time.Time) error {
	// Ensure we make an entry for this day in the cache
	l.cache[getDateUTC(t)] = []*photoslibrary.MediaItem{}

	// Build a search for pictures 1 day before thru 1 day after the target
	start := t.Add(-24 * time.Hour)
	end := t.Add(48 * time.Hour)
	dateRange := toPhotosLibraryDateRange(start, end)
	return l.service.MediaItems.Search(&photoslibrary.SearchMediaItemsRequest{
		Filters: &photoslibrary.Filters{
			DateFilter: &photoslibrary.DateFilter{
				Ranges: []*photoslibrary.DateRange{
					&dateRange,
				},
			},
		},
	}).Pages(ctx, l.handleSearchResponsePage)
}

func (l *librarian) handleSearchResponsePage(resp *photoslibrary.SearchMediaItemsResponse) error {
	for _, mediaItem := range resp.MediaItems {
		t, err := time.Parse(time.RFC3339, mediaItem.MediaMetadata.CreationTime)
		if err != nil {
			// Error parsing time
			return err
		}

		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		itemsForDay, ok := l.cache[day]
		if !ok {
			l.cache[day] = []*photoslibrary.MediaItem{mediaItem}
		} else {
			// TODO: Ensure that this media item is NOT already in the slice
			l.cache[day] = append(itemsForDay, mediaItem)
		}
	}

	return nil
}

func toPhotosLibraryDateRange(start, end time.Time) photoslibrary.DateRange {
	return photoslibrary.DateRange{
		StartDate: toPhotosLibraryDate(start),
		EndDate:   toPhotosLibraryDate(end),
	}
}

func toPhotosLibraryDate(t time.Time) *photoslibrary.Date {
	return &photoslibrary.Date{
		Year:  int64(t.Year()),
		Month: int64(t.Month()),
		Day:   int64(t.Day()),
	}
}

func getDateUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

package google

import (
	"context"
	"net/http"
	"net/url"
)

type ListCalendarEntriesOptions struct {
	MaxResults    int
	MinAccessRole string
	PageToken     string
	ShowDeleted   bool
	ShowHidden    bool
	SyncToken     string
}

func (c *Client) ListCalendarEntries(
	ctx context.Context,
	options ListCalendarEntriesOptions,
) (CalendarList, error) {
	query := url.Values{}
	setPositiveInt(query, "maxResults", options.MaxResults)
	setString(query, "minAccessRole", options.MinAccessRole)
	setString(query, "pageToken", options.PageToken)
	setBool(query, "showDeleted", options.ShowDeleted)
	setBool(query, "showHidden", options.ShowHidden)
	setString(query, "syncToken", options.SyncToken)
	var calendars CalendarList
	headers, err := c.doJSON(
		ctx,
		"list calendar entries",
		http.MethodGet,
		"/users/me/calendarList",
		query,
		nil,
		"",
		&calendars,
	)
	if err != nil {
		return CalendarList{}, err
	}
	if calendars.ETag == "" {
		calendars.ETag = headers.Get("ETag")
	}
	return calendars, nil
}

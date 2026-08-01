package google

import (
	"context"
	"net/http"
)

// WriteCalendarOptions carries the last observed ETag for If-Match.
type WriteCalendarOptions struct {
	ETag string
}

func (c *Client) CreateCalendar(ctx context.Context, input CalendarInput) (Calendar, error) {
	if err := required("create calendar", "Summary", input.Summary); err != nil {
		return Calendar{}, err
	}
	var calendar Calendar
	headers, err := c.doJSON(
		ctx,
		"create calendar",
		http.MethodPost,
		"/calendars",
		nil,
		input,
		"",
		&calendar,
	)
	if err != nil {
		return Calendar{}, err
	}
	setCalendarETag(&calendar, headers)
	return calendar, nil
}

func (c *Client) GetCalendar(ctx context.Context, calendarID string) (Calendar, error) {
	if err := required("get calendar", "CalendarID", calendarID); err != nil {
		return Calendar{}, err
	}
	var calendar Calendar
	headers, err := c.doJSON(
		ctx,
		"get calendar",
		http.MethodGet,
		"/calendars/"+escaped(calendarID),
		nil,
		nil,
		"",
		&calendar,
	)
	if err != nil {
		return Calendar{}, err
	}
	setCalendarETag(&calendar, headers)
	return calendar, nil
}

func (c *Client) UpdateCalendar(
	ctx context.Context,
	calendarID string,
	input CalendarInput,
	options WriteCalendarOptions,
) (Calendar, error) {
	if err := required("update calendar", "CalendarID", calendarID); err != nil {
		return Calendar{}, err
	}
	if err := required("update calendar", "Summary", input.Summary); err != nil {
		return Calendar{}, err
	}
	var calendar Calendar
	headers, err := c.doJSON(
		ctx,
		"update calendar",
		http.MethodPut,
		"/calendars/"+escaped(calendarID),
		nil,
		input,
		options.ETag,
		&calendar,
	)
	if err != nil {
		return Calendar{}, err
	}
	setCalendarETag(&calendar, headers)
	return calendar, nil
}

func (c *Client) DeleteCalendar(
	ctx context.Context,
	calendarID string,
	options WriteCalendarOptions,
) error {
	if err := required("delete calendar", "CalendarID", calendarID); err != nil {
		return err
	}
	_, err := c.doJSON(
		ctx,
		"delete calendar",
		http.MethodDelete,
		"/calendars/"+escaped(calendarID),
		nil,
		nil,
		options.ETag,
		nil,
	)
	return err
}

func setCalendarETag(calendar *Calendar, headers http.Header) {
	if calendar.ETag == "" {
		calendar.ETag = headers.Get("ETag")
	}
}

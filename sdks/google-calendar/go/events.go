package google

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// SendUpdates controls which attendees receive event notifications.
type SendUpdates string

const (
	SendUpdatesAll          SendUpdates = "all"
	SendUpdatesExternalOnly SendUpdates = "externalOnly"
	SendUpdatesNone         SendUpdates = "none"
)

// CreateEventOptions controls attendee notifications and response expansion.
type CreateEventOptions struct {
	SendUpdates  SendUpdates
	MaxAttendees int
}

// GetEventOptions controls the event representation returned by Google.
type GetEventOptions struct {
	MaxAttendees int
	TimeZone     string
}

// ListEventsOptions maps the optional query parameters of events.list.
type ListEventsOptions struct {
	AlwaysIncludeEmail      bool
	EventTypes              []string
	ICalUID                 string
	MaxAttendees            int
	MaxResults              int
	OrderBy                 string
	PageToken               string
	PrivateExtendedProperty []string
	Query                   string
	SharedExtendedProperty  []string
	ShowDeleted             bool
	ShowHiddenInvitations   bool
	SingleEvents            bool
	SyncToken               string
	TimeMax                 string
	TimeMin                 string
	TimeZone                string
	UpdatedMin              string
}

// UpdateEventOptions includes optimistic-concurrency and notification controls.
type UpdateEventOptions struct {
	ETag         string
	SendUpdates  SendUpdates
	MaxAttendees int
}

// DeleteEventOptions includes optimistic-concurrency and notification controls.
type DeleteEventOptions struct {
	ETag        string
	SendUpdates SendUpdates
}

func (c *Client) CreateEvent(
	ctx context.Context,
	calendarID string,
	input EventInput,
	options CreateEventOptions,
) (Event, error) {
	if err := required("create event", "CalendarID", calendarID); err != nil {
		return Event{}, err
	}
	query := eventWriteQuery(options.SendUpdates, options.MaxAttendees)
	var event Event
	headers, err := c.doJSON(
		ctx,
		"create event",
		http.MethodPost,
		"/calendars/"+escaped(calendarID)+"/events",
		query,
		input,
		"",
		&event,
	)
	if err != nil {
		return Event{}, err
	}
	setEventETag(&event, headers)
	return event, nil
}

func (c *Client) GetEvent(
	ctx context.Context,
	calendarID string,
	eventID string,
	options GetEventOptions,
) (Event, error) {
	if err := required("get event", "CalendarID", calendarID); err != nil {
		return Event{}, err
	}
	if err := required("get event", "EventID", eventID); err != nil {
		return Event{}, err
	}
	query := url.Values{}
	setPositiveInt(query, "maxAttendees", options.MaxAttendees)
	setString(query, "timeZone", options.TimeZone)
	var event Event
	headers, err := c.doJSON(
		ctx,
		"get event",
		http.MethodGet,
		"/calendars/"+escaped(calendarID)+"/events/"+escaped(eventID),
		query,
		nil,
		"",
		&event,
	)
	if err != nil {
		return Event{}, err
	}
	setEventETag(&event, headers)
	return event, nil
}

func (c *Client) ListEvents(
	ctx context.Context,
	calendarID string,
	options ListEventsOptions,
) (EventList, error) {
	if err := required("list events", "CalendarID", calendarID); err != nil {
		return EventList{}, err
	}
	query := url.Values{}
	setBool(query, "alwaysIncludeEmail", options.AlwaysIncludeEmail)
	for _, eventType := range options.EventTypes {
		setStringAdd(query, "eventTypes", eventType)
	}
	setString(query, "iCalUID", options.ICalUID)
	setPositiveInt(query, "maxAttendees", options.MaxAttendees)
	setPositiveInt(query, "maxResults", options.MaxResults)
	setString(query, "orderBy", options.OrderBy)
	setString(query, "pageToken", options.PageToken)
	for _, property := range options.PrivateExtendedProperty {
		setStringAdd(query, "privateExtendedProperty", property)
	}
	setString(query, "q", options.Query)
	for _, property := range options.SharedExtendedProperty {
		setStringAdd(query, "sharedExtendedProperty", property)
	}
	setBool(query, "showDeleted", options.ShowDeleted)
	setBool(query, "showHiddenInvitations", options.ShowHiddenInvitations)
	setBool(query, "singleEvents", options.SingleEvents)
	setString(query, "syncToken", options.SyncToken)
	setString(query, "timeMax", options.TimeMax)
	setString(query, "timeMin", options.TimeMin)
	setString(query, "timeZone", options.TimeZone)
	setString(query, "updatedMin", options.UpdatedMin)

	var events EventList
	headers, err := c.doJSON(
		ctx,
		"list events",
		http.MethodGet,
		"/calendars/"+escaped(calendarID)+"/events",
		query,
		nil,
		"",
		&events,
	)
	if err != nil {
		return EventList{}, err
	}
	if events.ETag == "" {
		events.ETag = headers.Get("ETag")
	}
	return events, nil
}

func (c *Client) UpdateEvent(
	ctx context.Context,
	calendarID string,
	eventID string,
	input EventInput,
	options UpdateEventOptions,
) (Event, error) {
	if err := required("update event", "CalendarID", calendarID); err != nil {
		return Event{}, err
	}
	if err := required("update event", "EventID", eventID); err != nil {
		return Event{}, err
	}
	query := eventWriteQuery(options.SendUpdates, options.MaxAttendees)
	var event Event
	headers, err := c.doJSON(
		ctx,
		"update event",
		http.MethodPut,
		"/calendars/"+escaped(calendarID)+"/events/"+escaped(eventID),
		query,
		input,
		options.ETag,
		&event,
	)
	if err != nil {
		return Event{}, err
	}
	setEventETag(&event, headers)
	return event, nil
}

func (c *Client) DeleteEvent(
	ctx context.Context,
	calendarID string,
	eventID string,
	options DeleteEventOptions,
) error {
	if err := required("delete event", "CalendarID", calendarID); err != nil {
		return err
	}
	if err := required("delete event", "EventID", eventID); err != nil {
		return err
	}
	query := url.Values{}
	setString(query, "sendUpdates", string(options.SendUpdates))
	_, err := c.doJSON(
		ctx,
		"delete event",
		http.MethodDelete,
		"/calendars/"+escaped(calendarID)+"/events/"+escaped(eventID),
		query,
		nil,
		options.ETag,
		nil,
	)
	return err
}

func eventWriteQuery(sendUpdates SendUpdates, maxAttendees int) url.Values {
	query := url.Values{}
	query.Set("conferenceDataVersion", "1")
	setString(query, "sendUpdates", string(sendUpdates))
	setPositiveInt(query, "maxAttendees", maxAttendees)
	return query
}

func setEventETag(event *Event, headers http.Header) {
	if event.ETag == "" {
		event.ETag = headers.Get("ETag")
	}
}

func setString(values url.Values, key, value string) {
	if value = strings.TrimSpace(value); value != "" {
		values.Set(key, value)
	}
}

func setStringAdd(values url.Values, key, value string) {
	if value = strings.TrimSpace(value); value != "" {
		values.Add(key, value)
	}
}

func setPositiveInt(values url.Values, key string, value int) {
	if value > 0 {
		values.Set(key, strconv.Itoa(value))
	}
}

func setBool(values url.Values, key string, value bool) {
	if value {
		values.Set(key, "true")
	}
}

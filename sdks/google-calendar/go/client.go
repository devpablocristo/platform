package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultCalendarBaseURL = "https://www.googleapis.com/calendar/v3"
	maxResponseBody        = 2 << 20
)

// ClientConfig configures a Calendar API client for one bearer token.
type ClientConfig struct {
	AccessToken string
	BaseURL     string
	HTTPClient  *http.Client
}

// Client is a low-level Google Calendar API client.
type Client struct {
	accessToken string
	baseURL     string
	httpClient  *http.Client
}

// EventsService is the low-level Google Calendar events surface.
type EventsService interface {
	CreateEvent(context.Context, string, EventInput, CreateEventOptions) (Event, error)
	GetEvent(context.Context, string, string, GetEventOptions) (Event, error)
	ListEvents(context.Context, string, ListEventsOptions) (EventList, error)
	UpdateEvent(context.Context, string, string, EventInput, UpdateEventOptions) (Event, error)
	DeleteEvent(context.Context, string, string, DeleteEventOptions) error
}

// CalendarsService manages secondary calendars. It intentionally does not
// model CalendarList membership, which is a separate Google API resource.
type CalendarsService interface {
	CreateCalendar(context.Context, CalendarInput) (Calendar, error)
	GetCalendar(context.Context, string) (Calendar, error)
	UpdateCalendar(context.Context, string, CalendarInput, WriteCalendarOptions) (Calendar, error)
	DeleteCalendar(context.Context, string, WriteCalendarOptions) error
}

// FreeBusyService queries availability across calendars and groups.
type FreeBusyService interface {
	QueryFreeBusy(context.Context, FreeBusyRequest) (FreeBusyResponse, error)
}

// CalendarAPI combines the event, secondary-calendar and FreeBusy surfaces.
type CalendarAPI interface {
	EventsService
	CalendarsService
	FreeBusyService
}

var _ CalendarAPI = (*Client)(nil)

// NewClient builds a client. Callers may inject an HTTP client for timeouts,
// custom transports and deterministic tests.
func NewClient(config ClientConfig) (*Client, error) {
	accessToken := strings.TrimSpace(config.AccessToken)
	if accessToken == "" {
		return nil, validationError("calendar", "AccessToken", "is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultCalendarBaseURL
	}
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil || parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return nil, validationError("calendar", "BaseURL", "must be a valid absolute URL")
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		accessToken: accessToken,
		baseURL:     baseURL,
		httpClient:  httpClient,
	}, nil
}

func (c *Client) doJSON(
	ctx context.Context,
	operation string,
	method string,
	resourcePath string,
	query url.Values,
	input any,
	ifMatch string,
	output any,
) (http.Header, error) {
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			return nil, &ResponseError{Operation: operation, Err: fmt.Errorf("encode request: %w", err)}
		}
		body = bytes.NewReader(raw)
	}

	endpoint := c.baseURL + resourcePath
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, &TransportError{Operation: operation + " build request", Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if etag := strings.TrimSpace(ifMatch); etag != "" {
		req.Header.Set("If-Match", etag)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &TransportError{Operation: operation, Err: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return resp.Header.Clone(), &ResponseError{Operation: operation, Err: fmt.Errorf("read body: %w", err)}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return resp.Header.Clone(), decodeAPIError(resp.StatusCode, resp.Header, raw)
	}
	if output == nil || len(bytes.TrimSpace(raw)) == 0 {
		return resp.Header.Clone(), nil
	}
	if err := json.Unmarshal(raw, output); err != nil {
		return resp.Header.Clone(), &ResponseError{Operation: operation, Err: fmt.Errorf("decode JSON: %w", err)}
	}
	return resp.Header.Clone(), nil
}

func required(operation, field, value string) error {
	if strings.TrimSpace(value) == "" {
		return validationError(operation, field, "is required")
	}
	return nil
}

func escaped(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}

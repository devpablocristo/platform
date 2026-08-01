package google

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(ClientConfig{
		AccessToken: "access-token",
		BaseURL:     server.URL,
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestNewClientRequiresAccessToken(t *testing.T) {
	t.Parallel()
	_, err := NewClient(ClientConfig{})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if validationErr.Field != "AccessToken" {
		t.Fatalf("field = %q, want AccessToken", validationErr.Field)
	}
}

func TestEventCRUDAndMeetPending(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Errorf("Authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/calendars/team@example.com/events":
			if got := r.URL.Query().Get("conferenceDataVersion"); got != "1" {
				t.Errorf("conferenceDataVersion = %q", got)
			}
			if got := r.URL.Query().Get("sendUpdates"); got != "all" {
				t.Errorf("sendUpdates = %q", got)
			}
			var input EventInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Errorf("decode create input: %v", err)
			}
			if input.ConferenceData == nil ||
				input.ConferenceData.CreateRequest == nil ||
				input.ConferenceData.CreateRequest.RequestID != "meet-request-1" {
				t.Errorf("unexpected conference request: %#v", input.ConferenceData)
			}
			if input.EventID != "0123456789abcdefghijklmnopqrstuv" {
				t.Errorf("deterministic event id = %q", input.EventID)
			}
			w.Header().Set("ETag", `"event-v1"`)
			_, _ = io.WriteString(w, `{
				"id":"event-1",
				"summary":"Planning",
				"start":{"dateTime":"2026-08-01T10:00:00Z"},
				"end":{"dateTime":"2026-08-01T11:00:00Z"},
				"conferenceData":{"createRequest":{
					"requestId":"meet-request-1",
					"conferenceSolutionKey":{"type":"hangoutsMeet"},
					"status":{"statusCode":"pending"}
				}}
			}`)
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/team@example.com/events/event-1":
			w.Header().Set("ETag", `"event-v1"`)
			_, _ = io.WriteString(w, `{
				"id":"event-1",
				"summary":"Planning",
				"start":{"dateTime":"2026-08-01T10:00:00Z"},
				"end":{"dateTime":"2026-08-01T11:00:00Z"}
			}`)
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/team@example.com/events":
			if r.URL.Query().Get("singleEvents") != "true" {
				t.Errorf("singleEvents was not encoded")
			}
			_, _ = io.WriteString(w, `{
				"etag":"\"list-v1\"",
				"nextPageToken":"next",
				"items":[{"id":"event-1","summary":"Planning","start":{},"end":{}}]
			}`)
		case r.Method == http.MethodPut && r.URL.Path == "/calendars/team@example.com/events/event-1":
			if got := r.Header.Get("If-Match"); got != `"event-v1"` {
				t.Errorf("If-Match = %q", got)
			}
			if got := r.URL.Query().Get("conferenceDataVersion"); got != "1" {
				t.Errorf("conferenceDataVersion = %q", got)
			}
			_, _ = io.WriteString(w, `{
				"id":"event-1",
				"etag":"\"event-v2\"",
				"summary":"Planning updated",
				"start":{},
				"end":{},
				"conferenceData":{
					"entryPoints":[{"entryPointType":"video","uri":"https://meet.google.com/abc-defg-hij"}],
					"createRequest":{"requestId":"meet-request-1","conferenceSolutionKey":{"type":"hangoutsMeet"},"status":{"statusCode":"success"}}
				}
			}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/calendars/team@example.com/events/event-1":
			if got := r.Header.Get("If-Match"); got != `"event-v2"` {
				t.Errorf("If-Match = %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	input := EventInput{
		EventID:        "0123456789abcdefghijklmnopqrstuv",
		Summary:        "Planning",
		Start:          EventDateTime{DateTime: "2026-08-01T10:00:00Z"},
		End:            EventDateTime{DateTime: "2026-08-01T11:00:00Z"},
		ConferenceData: NewMeetConferenceData("meet-request-1"),
	}
	created, err := client.CreateEvent(
		context.Background(),
		"team@example.com",
		input,
		CreateEventOptions{SendUpdates: SendUpdatesAll},
	)
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if created.ETag != `"event-v1"` {
		t.Errorf("created ETag = %q", created.ETag)
	}
	if !created.ConferenceData.Pending() {
		t.Fatalf("conference should be pending: %#v", created.ConferenceData)
	}

	got, err := client.GetEvent(
		context.Background(),
		"team@example.com",
		"event-1",
		GetEventOptions{},
	)
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if got.ID != "event-1" || got.ETag != `"event-v1"` {
		t.Fatalf("unexpected event: %#v", got)
	}

	list, err := client.ListEvents(
		context.Background(),
		"team@example.com",
		ListEventsOptions{SingleEvents: true, MaxResults: 10},
	)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(list.Items) != 1 || list.NextPageToken != "next" {
		t.Fatalf("unexpected list: %#v", list)
	}

	updated, err := client.UpdateEvent(
		context.Background(),
		"team@example.com",
		"event-1",
		input,
		UpdateEventOptions{ETag: created.ETag},
	)
	if err != nil {
		t.Fatalf("UpdateEvent: %v", err)
	}
	if updated.ETag != `"event-v2"` {
		t.Errorf("updated ETag = %q", updated.ETag)
	}
	if updated.ConferenceData.Pending() {
		t.Fatal("successful conference is still pending")
	}
	if got := updated.ConferenceData.MeetURI(); got != "https://meet.google.com/abc-defg-hij" {
		t.Errorf("MeetURI = %q", got)
	}

	if err := client.DeleteEvent(
		context.Background(),
		"team@example.com",
		"event-1",
		DeleteEventOptions{ETag: updated.ETag},
	); err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}
}

func TestCreateEventRejectsInvalidDeterministicID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("invalid event id reached the transport")
	})
	tests := []struct {
		name    string
		input   EventInput
		message string
	}{
		{
			name:    "too short",
			input:   EventInput{EventID: "abcd"},
			message: "between 5 and 1024",
		},
		{
			name:    "outside base32hex alphabet",
			input:   EventInput{EventID: "event-w"},
			message: "lowercase base32hex",
		},
		{
			name:    "conflicts with ical uid",
			input:   EventInput{EventID: "abcde", ICalUID: "external@example.com"},
			message: "cannot be combined",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := client.CreateEvent(
				context.Background(),
				"primary",
				test.input,
				CreateEventOptions{},
			)
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
			if !strings.Contains(validationErr.Message, test.message) {
				t.Fatalf("message = %q, want substring %q", validationErr.Message, test.message)
			}
		})
	}
}

func TestCreateEventReturnsTypedConflict(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":{
			"code":409,
			"status":"ALREADY_EXISTS",
			"message":"The requested identifier already exists.",
			"errors":[{"domain":"global","reason":"duplicate","message":"duplicate request id"}]
		}}`)
	})

	_, err := client.CreateEvent(
		context.Background(),
		"primary",
		EventInput{},
		CreateEventOptions{},
	)
	if !IsConflict(err) {
		t.Fatalf("expected conflict, got %T: %v", err, err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Status != "ALREADY_EXISTS" ||
		len(apiErr.Details) != 1 ||
		apiErr.Details[0].Reason != "duplicate" {
		t.Fatalf("unexpected APIError: %#v", apiErr)
	}
}

func TestUpdateEventReturnsTypedPreconditionFailure(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-Match"); got != `"stale"` {
			t.Errorf("If-Match = %q", got)
		}
		w.WriteHeader(http.StatusPreconditionFailed)
		_, _ = io.WriteString(w, `{"error":{
			"code":412,
			"status":"FAILED_PRECONDITION",
			"message":"Precondition Failed"
		}}`)
	})

	_, err := client.UpdateEvent(
		context.Background(),
		"primary",
		"event-1",
		EventInput{},
		UpdateEventOptions{ETag: `"stale"`},
	)
	if !IsPreconditionFailed(err) {
		t.Fatalf("expected precondition failure, got %T: %v", err, err)
	}
	if StatusCode(err) != http.StatusPreconditionFailed {
		t.Fatalf("StatusCode = %d", StatusCode(err))
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestClientReturnsTypedTimeout(t *testing.T) {
	t.Parallel()
	client, err := NewClient(ClientConfig{
		AccessToken: "access-token",
		BaseURL:     "https://calendar.test",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.GetEvent(context.Background(), "primary", "event-1", GetEventOptions{})
	if !IsTimeout(err) {
		t.Fatalf("expected timeout, got %T: %v", err, err)
	}
	var transportErr *TransportError
	if !errors.As(err, &transportErr) {
		t.Fatalf("expected TransportError, got %T", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline was not preserved: %v", err)
	}
}

func TestSecondaryCalendarLifecycle(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/calendars":
			var input CalendarInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Errorf("decode calendar: %v", err)
			}
			if input.Summary != "Resource room" {
				t.Errorf("summary = %q", input.Summary)
			}
			w.Header().Set("ETag", `"calendar-v1"`)
			_, _ = io.WriteString(w, `{"id":"secondary-1","summary":"Resource room"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/secondary-1":
			_, _ = io.WriteString(w, `{"id":"secondary-1","etag":"\"calendar-v1\"","summary":"Resource room"}`)
		case r.Method == http.MethodPut && r.URL.Path == "/calendars/secondary-1":
			if got := r.Header.Get("If-Match"); got != `"calendar-v1"` {
				t.Errorf("update If-Match = %q", got)
			}
			_, _ = io.WriteString(w, `{"id":"secondary-1","etag":"\"calendar-v2\"","summary":"Room 2"}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/calendars/secondary-1":
			if got := r.Header.Get("If-Match"); got != `"calendar-v2"` {
				t.Errorf("delete If-Match = %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	created, err := client.CreateCalendar(context.Background(), CalendarInput{Summary: "Resource room"})
	if err != nil {
		t.Fatalf("CreateCalendar: %v", err)
	}
	if created.ID != "secondary-1" || created.ETag != `"calendar-v1"` {
		t.Fatalf("created = %#v", created)
	}
	if _, err := client.GetCalendar(context.Background(), created.ID); err != nil {
		t.Fatalf("GetCalendar: %v", err)
	}
	updated, err := client.UpdateCalendar(
		context.Background(),
		created.ID,
		CalendarInput{Summary: "Room 2"},
		WriteCalendarOptions{ETag: created.ETag},
	)
	if err != nil {
		t.Fatalf("UpdateCalendar: %v", err)
	}
	options := WriteCalendarOptions{ETag: updated.ETag}
	if err := client.DeleteCalendar(context.Background(), created.ID, options); err != nil {
		t.Fatalf("DeleteCalendar: %v", err)
	}
}

func TestQueryFreeBusy(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/freeBusy" {
			http.NotFound(w, r)
			return
		}
		var input FreeBusyRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Errorf("decode freebusy input: %v", err)
		}
		if len(input.Items) != 1 || input.Items[0].ID != "secondary-1" {
			t.Errorf("items = %#v", input.Items)
		}
		_, _ = io.WriteString(w, `{
			"timeMin":"2026-08-01T00:00:00Z",
			"timeMax":"2026-08-02T00:00:00Z",
			"calendars":{"secondary-1":{"busy":[{
				"start":"2026-08-01T10:00:00Z",
				"end":"2026-08-01T11:00:00Z"
			}]}}
		}`)
	})

	output, err := client.QueryFreeBusy(context.Background(), FreeBusyRequest{
		TimeMin: "2026-08-01T00:00:00Z",
		TimeMax: "2026-08-02T00:00:00Z",
		Items:   []FreeBusyItem{{ID: "secondary-1"}},
	})
	if err != nil {
		t.Fatalf("QueryFreeBusy: %v", err)
	}
	busy := output.Calendars["secondary-1"].Busy
	if len(busy) != 1 || !strings.HasSuffix(busy[0].Start, "10:00:00Z") {
		t.Fatalf("busy = %#v", busy)
	}
}

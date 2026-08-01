package google

// EventDateTime maps Google's start/end representation. Date is an ISO date
// for all-day events; DateTime is RFC3339 for timed events.
type EventDateTime struct {
	Date     string `json:"date,omitempty"`
	DateTime string `json:"dateTime,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
}

type EventAttendee struct {
	ID             string `json:"id,omitempty"`
	Email          string `json:"email,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	Organizer      bool   `json:"organizer,omitempty"`
	Self           bool   `json:"self,omitempty"`
	Resource       bool   `json:"resource,omitempty"`
	Optional       bool   `json:"optional,omitempty"`
	ResponseStatus string `json:"responseStatus,omitempty"`
	Comment        string `json:"comment,omitempty"`
}

type EventOrganizer struct {
	ID          string `json:"id,omitempty"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Self        bool   `json:"self,omitempty"`
}

type EventReminders struct {
	UseDefault bool               `json:"useDefault"`
	Overrides  []ReminderOverride `json:"overrides,omitempty"`
}

type ReminderOverride struct {
	Method  string `json:"method"`
	Minutes int    `json:"minutes"`
}

type ExtendedProperties struct {
	Private map[string]string `json:"private,omitempty"`
	Shared  map[string]string `json:"shared,omitempty"`
}

type ConferenceData struct {
	CreateRequest      *CreateConferenceRequest `json:"createRequest,omitempty"`
	EntryPoints        []ConferenceEntryPoint   `json:"entryPoints,omitempty"`
	ConferenceSolution *ConferenceSolution      `json:"conferenceSolution,omitempty"`
	ConferenceID       string                   `json:"conferenceId,omitempty"`
	Notes              string                   `json:"notes,omitempty"`
	Signature          string                   `json:"signature,omitempty"`
}

type CreateConferenceRequest struct {
	RequestID             string                `json:"requestId"`
	ConferenceSolutionKey ConferenceSolutionKey `json:"conferenceSolutionKey"`
	Status                ConferenceStatus      `json:"status,omitempty"`
}

type ConferenceSolutionKey struct {
	Type string `json:"type"`
}

type ConferenceStatus struct {
	StatusCode string `json:"statusCode,omitempty"`
}

type ConferenceEntryPoint struct {
	EntryPointType string `json:"entryPointType,omitempty"`
	URI            string `json:"uri,omitempty"`
	Label          string `json:"label,omitempty"`
	PIN            string `json:"pin,omitempty"`
	AccessCode     string `json:"accessCode,omitempty"`
	MeetingCode    string `json:"meetingCode,omitempty"`
	Passcode       string `json:"passcode,omitempty"`
	Password       string `json:"password,omitempty"`
}

type ConferenceSolution struct {
	Key  ConferenceSolutionKey `json:"key"`
	Name string                `json:"name,omitempty"`
	Icon string                `json:"iconUri,omitempty"`
}

const (
	ConferenceSolutionGoogleMeet = "hangoutsMeet"
	ConferenceStatusPending      = "pending"
	ConferenceStatusSuccess      = "success"
	ConferenceStatusFailure      = "failure"
)

// NewMeetConferenceData requests a Google Meet conference for an event.
// RequestID must be unique per logical request so retries remain idempotent.
func NewMeetConferenceData(requestID string) *ConferenceData {
	return &ConferenceData{
		CreateRequest: &CreateConferenceRequest{
			RequestID: requestID,
			ConferenceSolutionKey: ConferenceSolutionKey{
				Type: ConferenceSolutionGoogleMeet,
			},
		},
	}
}

// Pending reports whether Google is still creating the conference.
func (c *ConferenceData) Pending() bool {
	return c != nil &&
		c.CreateRequest != nil &&
		c.CreateRequest.Status.StatusCode == ConferenceStatusPending
}

// MeetURI returns the first video entry point, when one is available.
func (c *ConferenceData) MeetURI() string {
	if c == nil {
		return ""
	}
	for _, entry := range c.EntryPoints {
		if entry.EntryPointType == "video" {
			return entry.URI
		}
	}
	return ""
}

// EventInput contains the writable fields of a Google Calendar event.
type EventInput struct {
	// EventID is an optional caller-provided Google event identifier. Google
	// accepts lowercase base32hex identifiers (0-9, a-v) between 5 and 1024
	// characters. Supplying a deterministic value makes create retries
	// idempotent: a repeated insert can be reconciled through events.get.
	EventID            string              `json:"id,omitempty"`
	Summary            string              `json:"summary,omitempty"`
	Description        string              `json:"description,omitempty"`
	Location           string              `json:"location,omitempty"`
	Start              EventDateTime       `json:"start"`
	End                EventDateTime       `json:"end"`
	Attendees          []EventAttendee     `json:"attendees,omitempty"`
	Recurrence         []string            `json:"recurrence,omitempty"`
	Transparency       string              `json:"transparency,omitempty"`
	Visibility         string              `json:"visibility,omitempty"`
	ICalUID            string              `json:"iCalUID,omitempty"`
	Sequence           int                 `json:"sequence,omitempty"`
	ColorID            string              `json:"colorId,omitempty"`
	GuestsCanInvite    *bool               `json:"guestsCanInviteOthers,omitempty"`
	GuestsCanModify    *bool               `json:"guestsCanModify,omitempty"`
	GuestsCanSeeOthers *bool               `json:"guestsCanSeeOtherGuests,omitempty"`
	AnyoneCanAddSelf   *bool               `json:"anyoneCanAddSelf,omitempty"`
	Reminders          *EventReminders     `json:"reminders,omitempty"`
	ExtendedProperties *ExtendedProperties `json:"extendedProperties,omitempty"`
	ConferenceData     *ConferenceData     `json:"conferenceData,omitempty"`
}

// Event is the Google Calendar event resource returned by the API.
type Event struct {
	EventInput
	Kind        string          `json:"kind,omitempty"`
	ETag        string          `json:"etag,omitempty"`
	ID          string          `json:"id,omitempty"`
	Status      string          `json:"status,omitempty"`
	HTMLLink    string          `json:"htmlLink,omitempty"`
	Created     string          `json:"created,omitempty"`
	Updated     string          `json:"updated,omitempty"`
	HangoutLink string          `json:"hangoutLink,omitempty"`
	Organizer   *EventOrganizer `json:"organizer,omitempty"`
}

type EventList struct {
	Kind          string  `json:"kind,omitempty"`
	ETag          string  `json:"etag,omitempty"`
	Summary       string  `json:"summary,omitempty"`
	Description   string  `json:"description,omitempty"`
	Updated       string  `json:"updated,omitempty"`
	TimeZone      string  `json:"timeZone,omitempty"`
	AccessRole    string  `json:"accessRole,omitempty"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
	NextSyncToken string  `json:"nextSyncToken,omitempty"`
	Items         []Event `json:"items,omitempty"`
}

// CalendarInput contains writable fields for a secondary calendar.
type CalendarInput struct {
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	TimeZone    string `json:"timeZone,omitempty"`
}

type Calendar struct {
	CalendarInput
	Kind string `json:"kind,omitempty"`
	ETag string `json:"etag,omitempty"`
	ID   string `json:"id,omitempty"`
}

type FreeBusyItem struct {
	ID string `json:"id"`
}

// FreeBusyRequest uses RFC3339 strings for TimeMin and TimeMax.
type FreeBusyRequest struct {
	TimeMin              string         `json:"timeMin"`
	TimeMax              string         `json:"timeMax"`
	TimeZone             string         `json:"timeZone,omitempty"`
	GroupExpansionMax    int            `json:"groupExpansionMax,omitempty"`
	CalendarExpansionMax int            `json:"calendarExpansionMax,omitempty"`
	Items                []FreeBusyItem `json:"items"`
}

type BusyPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type FreeBusyCalendar struct {
	Busy   []BusyPeriod  `json:"busy,omitempty"`
	Errors []ErrorDetail `json:"errors,omitempty"`
}

type FreeBusyGroup struct {
	Calendars []string      `json:"calendars,omitempty"`
	Errors    []ErrorDetail `json:"errors,omitempty"`
}

type FreeBusyResponse struct {
	Kind      string                      `json:"kind,omitempty"`
	TimeMin   string                      `json:"timeMin,omitempty"`
	TimeMax   string                      `json:"timeMax,omitempty"`
	Calendars map[string]FreeBusyCalendar `json:"calendars,omitempty"`
	Groups    map[string]FreeBusyGroup    `json:"groups,omitempty"`
}

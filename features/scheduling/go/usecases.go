package scheduling

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	corescheduling "github.com/devpablocristo/platform/jobs/go"
	"github.com/google/uuid"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	schedulingdomain "github.com/devpablocristo/platform/features/scheduling/go/domain"
)

var (
	// Repository adapters wrap these errors so the application core never
	// imports a database driver or ORM-specific error.
	ErrNotFound                 = errors.New("scheduling: not found")
	ErrAlreadyExists            = errors.New("scheduling: already exists")
	ErrCapacityExceeded         = errors.New("scheduling: resource capacity exceeded")
	ErrQueueInactive            = errors.New("scheduling: queue inactive")
	ErrRemoteJoinDisabled       = errors.New("scheduling: remote join disabled")
	errNoTicketWaiting          = errors.New("scheduling: no ticket waiting")
	errTransitionNotAllowed     = errors.New("scheduling: transition not allowed")
	errNoDiscreteSchedulingSlot = errors.New("scheduling: no discrete slot match")
)

// RepositoryPort is implemented by the consumer. ReserveBookings and
// RescheduleBooking must reserve all allocations atomically and return
// ErrCapacityExceeded when any resource would exceed its capacity.
type RepositoryPort interface {
	ListBranches(ctx context.Context) ([]schedulingdomain.Branch, error)
	GetBranch(ctx context.Context, branchID uuid.UUID) (schedulingdomain.Branch, error)
	CreateBranch(ctx context.Context, in schedulingdomain.Branch) (schedulingdomain.Branch, error)
	ListServices(ctx context.Context) ([]schedulingdomain.Service, error)
	GetService(ctx context.Context, serviceID uuid.UUID) (schedulingdomain.Service, error)
	CreateService(ctx context.Context, in schedulingdomain.Service) (schedulingdomain.Service, error)
	ListResources(ctx context.Context, branchID *uuid.UUID) ([]schedulingdomain.Resource, error)
	GetResource(ctx context.Context, resourceID uuid.UUID) (schedulingdomain.Resource, error)
	ListServiceResources(ctx context.Context, branchID, serviceID uuid.UUID, selected *uuid.UUID) ([]schedulingdomain.Resource, error)
	CreateResource(ctx context.Context, in schedulingdomain.Resource) (schedulingdomain.Resource, error)
	ListAvailabilityRules(ctx context.Context, branchID, resourceID *uuid.UUID) ([]schedulingdomain.AvailabilityRule, error)
	ListApplicableAvailabilityRules(ctx context.Context, branchID uuid.UUID, resourceID *uuid.UUID, day time.Time) ([]schedulingdomain.AvailabilityRule, error)
	CreateAvailabilityRule(ctx context.Context, in schedulingdomain.AvailabilityRule) (schedulingdomain.AvailabilityRule, error)
	ListBlockedRanges(ctx context.Context, branchID, resourceID *uuid.UUID, day *time.Time) ([]schedulingdomain.BlockedRange, error)
	ListBlockedRangesBetween(ctx context.Context, branchID uuid.UUID, resourceID *uuid.UUID, startAt, endAt time.Time) ([]schedulingdomain.BlockedRange, error)
	CreateBlockedRange(ctx context.Context, in schedulingdomain.BlockedRange) (schedulingdomain.BlockedRange, error)
	UpdateBlockedRange(ctx context.Context, in schedulingdomain.BlockedRange) (schedulingdomain.BlockedRange, error)
	DeleteBlockedRange(ctx context.Context, id uuid.UUID) error
	CreateCalendarEvent(ctx context.Context, in schedulingdomain.CalendarEvent) (schedulingdomain.CalendarEvent, error)
	GetCalendarEvent(ctx context.Context, id uuid.UUID) (schedulingdomain.CalendarEvent, error)
	ListCalendarEvents(ctx context.Context, filter schedulingdomain.ListCalendarEventsFilter) ([]schedulingdomain.CalendarEvent, error)
	ListCalendarEventsOccupyingResource(ctx context.Context, branchID, resourceID uuid.UUID, from, to time.Time) ([]schedulingdomain.CalendarEvent, error)
	UpdateCalendarEvent(ctx context.Context, in schedulingdomain.CalendarEvent) (schedulingdomain.CalendarEvent, error)
	DeleteCalendarEvent(ctx context.Context, id uuid.UUID) error
	AllocatedResourceUnits(ctx context.Context, resourceID uuid.UUID, occupiesFrom, occupiesUntil time.Time, excludeBookingID *uuid.UUID) (int, error)
	ConcurrentServiceBookings(ctx context.Context, serviceID uuid.UUID, occupiesFrom, occupiesUntil time.Time, excludeBookingID *uuid.UUID) (int, error)
	ReserveBookings(ctx context.Context, in []schedulingdomain.Booking) ([]schedulingdomain.Booking, error)
	GetBookingByID(ctx context.Context, bookingID uuid.UUID) (schedulingdomain.Booking, error)
	ListBookings(ctx context.Context, filter schedulingdomain.ListBookingsFilter) ([]schedulingdomain.Booking, error)
	UpdateBookingStatus(ctx context.Context, bookingID uuid.UUID, status schedulingdomain.BookingStatus, confirmedAt, cancelledAt *time.Time, notes string) (schedulingdomain.Booking, error)
	MarkBookingReminderSent(ctx context.Context, bookingID uuid.UUID, sentAt time.Time) (schedulingdomain.Booking, error)
	ExpireOverdueHolds(ctx context.Context, limit int) ([]schedulingdomain.Booking, error)
	RescheduleBooking(ctx context.Context, in schedulingdomain.Booking) (schedulingdomain.Booking, error)
	CreateBookingActionToken(ctx context.Context, in schedulingdomain.BookingActionToken) (schedulingdomain.BookingActionToken, error)
	GetBookingActionTokenByHash(ctx context.Context, tokenHash string) (schedulingdomain.BookingActionToken, error)
	MarkBookingActionTokenUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error
	ListQueues(ctx context.Context, branchID *uuid.UUID) ([]schedulingdomain.Queue, error)
	GetQueueByID(ctx context.Context, queueID uuid.UUID) (schedulingdomain.Queue, error)
	UpdateQueueStatus(ctx context.Context, queueID uuid.UUID, status schedulingdomain.QueueStatus) (schedulingdomain.Queue, error)
	CreateQueue(ctx context.Context, in schedulingdomain.Queue) (schedulingdomain.Queue, error)
	CreateQueueTicket(ctx context.Context, in schedulingdomain.QueueTicket) (schedulingdomain.QueueTicket, error)
	GetQueueTicket(ctx context.Context, queueID, ticketID uuid.UUID) (schedulingdomain.QueueTicket, error)
	GetQueueTicketPosition(ctx context.Context, queueID, ticketID uuid.UUID) (schedulingdomain.QueuePosition, error)
	CallNextTicket(ctx context.Context, queueID uuid.UUID, servingResourceID *uuid.UUID, operatorUserID *string) (schedulingdomain.QueueTicket, error)
	MarkQueueTicketServing(ctx context.Context, queueID, ticketID uuid.UUID, servingResourceID *uuid.UUID, operatorUserID *string) (schedulingdomain.QueueTicket, error)
	UpdateQueueTicketStatus(ctx context.Context, queueID, ticketID uuid.UUID, status schedulingdomain.QueueTicketStatus, servingResourceID *uuid.UUID, operatorUserID *string) (schedulingdomain.QueueTicket, error)
	ReassignQueueTicket(ctx context.Context, queueID, ticketID uuid.UUID, servingResourceID *uuid.UUID, operatorUserID *string) (schedulingdomain.QueueTicket, error)
	ReturnQueueTicketToWaiting(ctx context.Context, queueID, ticketID uuid.UUID) (schedulingdomain.QueueTicket, error)
	CreateWaitlistEntry(ctx context.Context, in schedulingdomain.WaitlistEntry) (schedulingdomain.WaitlistEntry, error)
	ListWaitlistEntries(ctx context.Context, filter schedulingdomain.ListWaitlistFilter) ([]schedulingdomain.WaitlistEntry, error)
	ListPendingWaitlistEntries(ctx context.Context, limit int) ([]schedulingdomain.WaitlistEntry, error)
	UpdateWaitlistEntryStatus(ctx context.Context, entryID uuid.UUID, status schedulingdomain.WaitlistStatus, expiresAt, notifiedAt *time.Time, bookingID *uuid.UUID, notes string) (schedulingdomain.WaitlistEntry, error)
	DashboardStats(ctx context.Context, branchID *uuid.UUID, day time.Time, timezone string) (schedulingdomain.DashboardStats, error)
	ListDayAgenda(ctx context.Context, branchID *uuid.UUID, day time.Time) ([]schedulingdomain.DayAgendaItem, error)
}

type AuditPort interface {
	Log(ctx context.Context, actor, action, resourceType, resourceID string, payload map[string]any)
}

type NotificationPort interface {
	Enqueue(ctx context.Context, eventType string, payload map[string]any) error
}

type Usecases struct {
	repo          RepositoryPort
	audit         AuditPort
	notifications NotificationPort
}

type Option func(*Usecases)

func WithNotifications(n NotificationPort) Option { return func(u *Usecases) { u.notifications = n } }

func NewUsecases(repo RepositoryPort, audit AuditPort, opts ...Option) *Usecases {
	uc := &Usecases{repo: repo, audit: audit}
	for _, opt := range opts {
		opt(uc)
	}
	return uc
}

func (u *Usecases) ListBranches(ctx context.Context) ([]schedulingdomain.Branch, error) {
	return u.repo.ListBranches(ctx)
}

func (u *Usecases) CreateBranch(ctx context.Context, actor string, in schedulingdomain.Branch) (schedulingdomain.Branch, error) {
	if strings.TrimSpace(in.Name) == "" {
		return schedulingdomain.Branch{}, domainerr.Validation("name is required")
	}
	if strings.TrimSpace(in.Code) == "" {
		return schedulingdomain.Branch{}, domainerr.Validation("code is required")
	}
	if _, err := time.LoadLocation(strings.TrimSpace(in.Timezone)); err != nil {
		return schedulingdomain.Branch{}, domainerr.Validation("timezone must be a valid IANA timezone")
	}
	in.ID = ensureUUID(in.ID)
	in.Code = normalizeCode(in.Code)
	out, err := u.repo.CreateBranch(ctx, in)
	if err != nil {
		return schedulingdomain.Branch{}, mapRepoError(err, "branch", in.ID)
	}
	u.logAudit(ctx, actor, "scheduling.branch.created", "scheduling_branch", out.ID.String(), map[string]any{"code": out.Code})
	return out, nil
}

func (u *Usecases) ListServices(ctx context.Context) ([]schedulingdomain.Service, error) {
	return u.repo.ListServices(ctx)
}

func (u *Usecases) CreateService(ctx context.Context, actor string, in schedulingdomain.Service) (schedulingdomain.Service, error) {
	if strings.TrimSpace(in.Name) == "" {
		return schedulingdomain.Service{}, domainerr.Validation("name is required")
	}
	if strings.TrimSpace(in.Code) == "" {
		return schedulingdomain.Service{}, domainerr.Validation("code is required")
	}
	mode := normalizeFulfillmentMode(in.FulfillmentMode)
	if mode == "" {
		return schedulingdomain.Service{}, domainerr.Validation("invalid fulfillment_mode")
	}
	if in.DefaultDurationMinutes <= 0 {
		in.DefaultDurationMinutes = 30
	}
	if in.DefaultDurationMinutes > 720 {
		return schedulingdomain.Service{}, domainerr.Validation("default_duration_minutes must be <= 720")
	}
	if in.BufferBeforeMinutes < 0 || in.BufferAfterMinutes < 0 {
		return schedulingdomain.Service{}, domainerr.Validation("buffers must be >= 0")
	}
	if in.SlotGranularityMinutes <= 0 {
		in.SlotGranularityMinutes = 15
	}
	if in.MaxConcurrentBookings <= 0 {
		in.MaxConcurrentBookings = 1
	}
	if in.MaxConcurrentBookings > 100_000 {
		return schedulingdomain.Service{}, domainerr.Validation("max_concurrent_bookings must be <= 100000")
	}
	if in.MinCancelNoticeMinutes < 0 {
		return schedulingdomain.Service{}, domainerr.Validation("min_cancel_notice_minutes must be >= 0")
	}
	for _, resourceID := range in.ResourceIDs {
		if _, err := u.repo.GetResource(ctx, resourceID); err != nil {
			return schedulingdomain.Service{}, mapRepoError(err, "resource", resourceID)
		}
	}
	in.ID = ensureUUID(in.ID)
	in.Code = normalizeCode(in.Code)
	in.FulfillmentMode = mode
	out, err := u.repo.CreateService(ctx, in)
	if err != nil {
		return schedulingdomain.Service{}, mapRepoError(err, "service", in.ID)
	}
	u.logAudit(ctx, actor, "scheduling.service.created", "scheduling_service", out.ID.String(), map[string]any{"code": out.Code, "mode": out.FulfillmentMode})
	return out, nil
}

func (u *Usecases) ListResources(ctx context.Context, branchID *uuid.UUID) ([]schedulingdomain.Resource, error) {
	return u.repo.ListResources(ctx, branchID)
}

func (u *Usecases) CreateResource(ctx context.Context, actor string, in schedulingdomain.Resource) (schedulingdomain.Resource, error) {
	if in.BranchID == uuid.Nil {
		return schedulingdomain.Resource{}, domainerr.Validation("branch_id is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return schedulingdomain.Resource{}, domainerr.Validation("name is required")
	}
	if strings.TrimSpace(in.Code) == "" {
		return schedulingdomain.Resource{}, domainerr.Validation("code is required")
	}
	kind := normalizeResourceKind(in.Kind)
	if kind == "" {
		return schedulingdomain.Resource{}, domainerr.Validation("invalid kind")
	}
	if in.Capacity <= 0 {
		in.Capacity = 1
	}
	if in.Capacity > 100_000 {
		return schedulingdomain.Resource{}, domainerr.Validation("capacity must be <= 100000")
	}
	branch, err := u.repo.GetBranch(ctx, in.BranchID)
	if err != nil {
		return schedulingdomain.Resource{}, mapRepoError(err, "branch", in.BranchID)
	}
	in.ID = ensureUUID(in.ID)
	in.Kind = kind
	in.Code = normalizeCode(in.Code)
	if strings.TrimSpace(in.Timezone) == "" {
		in.Timezone = branch.Timezone
	}
	if _, err := time.LoadLocation(strings.TrimSpace(in.Timezone)); err != nil {
		return schedulingdomain.Resource{}, domainerr.Validation("resource timezone must be a valid IANA timezone")
	}
	out, err := u.repo.CreateResource(ctx, in)
	if err != nil {
		return schedulingdomain.Resource{}, mapRepoError(err, "resource", in.ID)
	}
	u.logAudit(ctx, actor, "scheduling.resource.created", "scheduling_resource", out.ID.String(), map[string]any{"code": out.Code, "branch_id": out.BranchID.String()})
	return out, nil
}

func (u *Usecases) ListAvailabilityRules(ctx context.Context, branchID, resourceID *uuid.UUID) ([]schedulingdomain.AvailabilityRule, error) {
	return u.repo.ListAvailabilityRules(ctx, branchID, resourceID)
}

func (u *Usecases) CreateAvailabilityRule(ctx context.Context, actor string, in schedulingdomain.AvailabilityRule) (schedulingdomain.AvailabilityRule, error) {
	if in.BranchID == uuid.Nil {
		return schedulingdomain.AvailabilityRule{}, domainerr.Validation("branch_id is required")
	}
	if in.Weekday < 0 || in.Weekday > 6 {
		return schedulingdomain.AvailabilityRule{}, domainerr.Validation("weekday must be between 0 and 6")
	}
	kind := normalizeAvailabilityRuleKind(in.Kind)
	if kind == "" {
		return schedulingdomain.AvailabilityRule{}, domainerr.Validation("invalid kind")
	}
	startClock, err := corescheduling.ParseClock(in.StartTime)
	if err != nil {
		return schedulingdomain.AvailabilityRule{}, domainerr.Validation("start_time must be HH:MM")
	}
	endClock, err := corescheduling.ParseClock(in.EndTime)
	if err != nil {
		return schedulingdomain.AvailabilityRule{}, domainerr.Validation("end_time must be HH:MM")
	}
	if !endClock.After(startClock) {
		return schedulingdomain.AvailabilityRule{}, domainerr.Validation("start_time must be before end_time")
	}
	if kind == schedulingdomain.AvailabilityRuleKindResource && (in.ResourceID == nil || *in.ResourceID == uuid.Nil) {
		return schedulingdomain.AvailabilityRule{}, domainerr.Validation("resource_id is required for resource rules")
	}
	in.ID = ensureUUID(in.ID)
	in.Kind = kind
	out, err := u.repo.CreateAvailabilityRule(ctx, in)
	if err != nil {
		return schedulingdomain.AvailabilityRule{}, mapRepoError(err, "availability_rule", in.ID)
	}
	u.logAudit(ctx, actor, "scheduling.availability_rule.created", "scheduling_availability_rule", out.ID.String(), map[string]any{"branch_id": out.BranchID.String(), "kind": out.Kind})
	return out, nil
}

func (u *Usecases) ListBlockedRanges(ctx context.Context, branchID, resourceID *uuid.UUID, day *time.Time) ([]schedulingdomain.BlockedRange, error) {
	return u.repo.ListBlockedRanges(ctx, branchID, resourceID, day)
}

func (u *Usecases) CreateBlockedRange(ctx context.Context, actor string, in schedulingdomain.BlockedRange) (schedulingdomain.BlockedRange, error) {
	kind, err := validateBlockedRangeFields(in)
	if err != nil {
		return schedulingdomain.BlockedRange{}, err
	}
	in.ID = ensureUUID(in.ID)
	in.Kind = kind
	in.CreatedBy = actor
	out, err := u.repo.CreateBlockedRange(ctx, in)
	if err != nil {
		return schedulingdomain.BlockedRange{}, mapRepoError(err, "blocked_range", in.ID)
	}
	u.logAudit(ctx, actor, "scheduling.blocked_range.created", "scheduling_blocked_range", out.ID.String(), map[string]any{"branch_id": out.BranchID.String(), "kind": out.Kind})
	return out, nil
}

func (u *Usecases) UpdateBlockedRange(ctx context.Context, actor string, id uuid.UUID, in schedulingdomain.BlockedRange) (schedulingdomain.BlockedRange, error) {
	if id == uuid.Nil {
		return schedulingdomain.BlockedRange{}, domainerr.Validation("id is required")
	}
	kind, err := validateBlockedRangeFields(in)
	if err != nil {
		return schedulingdomain.BlockedRange{}, err
	}
	in.ID = id
	in.Kind = kind
	out, err := u.repo.UpdateBlockedRange(ctx, in)
	if err != nil {
		return schedulingdomain.BlockedRange{}, mapRepoError(err, "blocked_range", id)
	}
	u.logAudit(ctx, actor, "scheduling.blocked_range.updated", "scheduling_blocked_range", out.ID.String(), map[string]any{"branch_id": out.BranchID.String(), "kind": out.Kind})
	return out, nil
}

func (u *Usecases) DeleteBlockedRange(ctx context.Context, actor string, id uuid.UUID) error {
	if id == uuid.Nil {
		return domainerr.Validation("id is required")
	}
	if err := u.repo.DeleteBlockedRange(ctx, id); err != nil {
		return mapRepoError(err, "blocked_range", id)
	}
	u.logAudit(ctx, actor, "scheduling.blocked_range.deleted", "scheduling_blocked_range", id.String(), nil)
	return nil
}

// ── calendar events (agenda interna) ────────────────────────────────────────

func (u *Usecases) ListCalendarEvents(ctx context.Context, filter schedulingdomain.ListCalendarEventsFilter) ([]schedulingdomain.CalendarEvent, error) {
	return u.repo.ListCalendarEvents(ctx, filter)
}

func (u *Usecases) GetCalendarEvent(ctx context.Context, id uuid.UUID) (schedulingdomain.CalendarEvent, error) {
	if id == uuid.Nil {
		return schedulingdomain.CalendarEvent{}, domainerr.Validation("id is required")
	}
	out, err := u.repo.GetCalendarEvent(ctx, id)
	if err != nil {
		return schedulingdomain.CalendarEvent{}, mapRepoError(err, "calendar_event", id)
	}
	return out, nil
}

func (u *Usecases) CreateCalendarEvent(ctx context.Context, actor string, in schedulingdomain.CalendarEvent) (schedulingdomain.CalendarEvent, error) {
	status, visibility, err := validateCalendarEventFields(in)
	if err != nil {
		return schedulingdomain.CalendarEvent{}, err
	}
	in.ID = ensureUUID(in.ID)
	in.Status = status
	in.Visibility = visibility
	in.CreatedBy = actor
	out, err := u.repo.CreateCalendarEvent(ctx, in)
	if err != nil {
		return schedulingdomain.CalendarEvent{}, mapRepoError(err, "calendar_event", in.ID)
	}
	u.logAudit(ctx, actor, "scheduling.calendar_event.created", "scheduling_calendar_event", out.ID.String(), map[string]any{"title": out.Title, "resource_id": uuidPtrString(out.ResourceID)})
	return out, nil
}

func (u *Usecases) UpdateCalendarEvent(ctx context.Context, actor string, id uuid.UUID, in schedulingdomain.CalendarEvent) (schedulingdomain.CalendarEvent, error) {
	if id == uuid.Nil {
		return schedulingdomain.CalendarEvent{}, domainerr.Validation("id is required")
	}
	status, visibility, err := validateCalendarEventFields(in)
	if err != nil {
		return schedulingdomain.CalendarEvent{}, err
	}
	in.ID = id
	in.Status = status
	in.Visibility = visibility
	out, err := u.repo.UpdateCalendarEvent(ctx, in)
	if err != nil {
		return schedulingdomain.CalendarEvent{}, mapRepoError(err, "calendar_event", id)
	}
	u.logAudit(ctx, actor, "scheduling.calendar_event.updated", "scheduling_calendar_event", out.ID.String(), map[string]any{"title": out.Title, "status": string(out.Status)})
	return out, nil
}

func (u *Usecases) DeleteCalendarEvent(ctx context.Context, actor string, id uuid.UUID) error {
	if id == uuid.Nil {
		return domainerr.Validation("id is required")
	}
	if err := u.repo.DeleteCalendarEvent(ctx, id); err != nil {
		return mapRepoError(err, "calendar_event", id)
	}
	u.logAudit(ctx, actor, "scheduling.calendar_event.deleted", "scheduling_calendar_event", id.String(), nil)
	return nil
}

func validateCalendarEventFields(in schedulingdomain.CalendarEvent) (schedulingdomain.CalendarEventStatus, schedulingdomain.CalendarEventVisibility, error) {
	if strings.TrimSpace(in.Title) == "" {
		return "", "", domainerr.Validation("title is required")
	}
	if in.StartAt.IsZero() || in.EndAt.IsZero() {
		return "", "", domainerr.Validation("start_at and end_at are required")
	}
	if !in.EndAt.After(in.StartAt) {
		return "", "", domainerr.Validation("end_at must be after start_at")
	}
	status := schedulingdomain.CalendarEventStatus(strings.TrimSpace(strings.ToLower(string(in.Status))))
	if status == "" {
		status = schedulingdomain.CalendarEventStatusScheduled
	}
	switch status {
	case schedulingdomain.CalendarEventStatusScheduled,
		schedulingdomain.CalendarEventStatusDone,
		schedulingdomain.CalendarEventStatusCancelled:
	default:
		return "", "", domainerr.Validation("invalid status")
	}
	visibility := schedulingdomain.CalendarEventVisibility(strings.TrimSpace(strings.ToLower(string(in.Visibility))))
	if visibility == "" {
		visibility = schedulingdomain.CalendarEventVisibilityTeam
	}
	switch visibility {
	case schedulingdomain.CalendarEventVisibilityTeam,
		schedulingdomain.CalendarEventVisibilityPrivate:
	default:
		return "", "", domainerr.Validation("invalid visibility")
	}
	return status, visibility, nil
}

func uuidPtrString(p *uuid.UUID) string {
	if p == nil {
		return ""
	}
	return p.String()
}

func validateBlockedRangeFields(in schedulingdomain.BlockedRange) (schedulingdomain.BlockedRangeKind, error) {
	if in.BranchID == uuid.Nil {
		return "", domainerr.Validation("branch_id is required")
	}
	kind := normalizeBlockedRangeKind(in.Kind)
	if kind == "" {
		return "", domainerr.Validation("invalid kind")
	}
	if in.StartAt.IsZero() || in.EndAt.IsZero() {
		return "", domainerr.Validation("start_at and end_at are required")
	}
	if !in.EndAt.After(in.StartAt) {
		return "", domainerr.Validation("end_at must be after start_at")
	}
	return kind, nil
}

func (u *Usecases) ListAvailableSlots(ctx context.Context, query schedulingdomain.SlotQuery) ([]schedulingdomain.TimeSlot, error) {
	if query.BranchID == uuid.Nil || query.ServiceID == uuid.Nil {
		return nil, domainerr.Validation("branch_id and service_id are required")
	}
	if query.Date.IsZero() {
		return nil, domainerr.Validation("date is required")
	}
	participants, err := normalizeParticipants(query.Participants)
	if err != nil {
		return nil, err
	}
	query.Participants = participants
	return u.listAvailableSlots(ctx, query)
}

func (u *Usecases) ListBookings(ctx context.Context, filter schedulingdomain.ListBookingsFilter) ([]schedulingdomain.Booking, error) {
	return u.repo.ListBookings(ctx, filter)
}

func (u *Usecases) GetBookingByID(ctx context.Context, bookingID uuid.UUID) (schedulingdomain.Booking, error) {
	out, err := u.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", bookingID)
	}
	return out, nil
}

func (u *Usecases) CreateBooking(ctx context.Context, actor string, in schedulingdomain.CreateBookingInput) (schedulingdomain.Booking, error) {
	if in.BranchID == uuid.Nil || in.ServiceID == uuid.Nil {
		return schedulingdomain.Booking{}, domainerr.Validation("branch_id and service_id are required")
	}
	if strings.TrimSpace(in.CustomerName) == "" {
		return schedulingdomain.Booking{}, domainerr.Validation("customer_name is required")
	}
	if in.StartAt.IsZero() {
		return schedulingdomain.Booking{}, domainerr.Validation("start_at is required")
	}
	participants, err := normalizeParticipants(in.Participants)
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	in.Participants = participants
	branch, service, err := u.loadBookingScope(ctx, in.BranchID, in.ServiceID)
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	if service.FulfillmentMode == schedulingdomain.FulfillmentModeQueue {
		return schedulingdomain.Booking{}, domainerr.Validation("service is queue-only")
	}
	status := in.Status
	if status == "" {
		status = defaultBookingStatus(in)
	}
	status = normalizeBookingStatus(status)
	if status == "" {
		return schedulingdomain.Booking{}, domainerr.Validation("invalid booking status")
	}

	if in.EndAt != nil && !in.EndAt.IsZero() {
		if in.Recurrence != nil {
			return schedulingdomain.Booking{}, domainerr.Validation("end_at cannot be combined with recurrence")
		}
		if !in.EndAt.After(in.StartAt) {
			return schedulingdomain.Booking{}, domainerr.Validation("end_at must be after start_at")
		}
		return u.createSingleBookingFromExplicitRange(ctx, actor, branch, service, in, status, in.StartAt.UTC(), in.EndAt.UTC(), nil)
	}

	recurrence, err := normalizeBookingRecurrence(in.Recurrence, in.StartAt, branch.Timezone)
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	starts, err := expandRecurringBookingStarts(in.StartAt, recurrence, branch.Timezone)
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	bookingsToCreate, err := u.prepareBookingsForStarts(ctx, actor, branch, service, in, status, starts, recurrence)
	if err != nil {
		if errors.Is(err, errNoDiscreteSchedulingSlot) && recurrence == nil &&
			hasRequestedResources(in.ResourceID, in.Resources) {
			endUTC := inferBookingEndFromServiceDefaults(service, in.StartAt.UTC())
			if !endUTC.After(in.StartAt.UTC()) {
				return schedulingdomain.Booking{}, domainerr.Conflict("slot not available")
			}
			return u.createSingleBookingFromExplicitRange(ctx, actor, branch, service, in, status, in.StartAt.UTC(), endUTC, nil)
		}
		return schedulingdomain.Booking{}, err
	}
	if len(bookingsToCreate) == 0 {
		return schedulingdomain.Booking{}, domainerr.Conflict("slot not available")
	}
	created, err := u.repo.ReserveBookings(ctx, bookingsToCreate)
	if err != nil {
		if isBookingOverlapErr(err) {
			return schedulingdomain.Booking{}, domainerr.Conflict("slot not available")
		}
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", bookingsToCreate[0].ID)
	}
	for _, item := range created {
		u.logAudit(ctx, actor, "scheduling.booking.created", "scheduling_booking", item.ID.String(), map[string]any{"service_id": item.ServiceID.String(), "allocations": item.Allocations, "status": item.Status})
		u.emitEvent(ctx, "scheduling.booking.created", map[string]any{"booking_id": item.ID.String(), "branch_id": item.BranchID.String(), "service_id": item.ServiceID.String()})
	}
	return created[0], nil
}

// createSingleBookingFromExplicitRange persiste un turno con [startUTC, endUTC] ya resueltos (calendario interno,
// cliente que envía end_at, o fallback por duración por defecto del servicio).
func (u *Usecases) createSingleBookingFromExplicitRange(
	ctx context.Context,
	actor string,
	branch schedulingdomain.Branch,
	service schedulingdomain.Service,
	in schedulingdomain.CreateBookingInput,
	status schedulingdomain.BookingStatus,
	startUTC, endUTC time.Time,
	excludeBookingID *uuid.UUID,
) (schedulingdomain.Booking, error) {
	if !endUTC.After(startUTC) {
		return schedulingdomain.Booking{}, domainerr.Validation("end_at must be after start_at")
	}
	allocations, err := NormalizeResourceAllocationsForParticipants(in.ResourceID, in.Resources, in.Participants)
	if err != nil {
		return schedulingdomain.Booking{}, domainerr.Validation(err.Error())
	}
	if len(allocations) == 0 {
		return schedulingdomain.Booking{}, domainerr.Validation("at least one resource allocation is required")
	}
	occFrom := startUTC.Add(-time.Duration(service.BufferBeforeMinutes) * time.Minute)
	occUntil := endUTC.Add(time.Duration(service.BufferAfterMinutes) * time.Minute)
	allocations, err = u.validateBookingAllocationsRangeFits(
		ctx,
		branch,
		service,
		allocations,
		startUTC,
		endUTC,
		occFrom,
		occUntil,
		excludeBookingID,
	)
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	metadata := cloneMetadata(in.Metadata)
	booking := schedulingdomain.Booking{
		ID:             uuid.New(),
		BranchID:       branch.ID,
		ServiceID:      service.ID,
		ResourceID:     allocations[0].ResourceID,
		Allocations:    allocations,
		Participants:   in.Participants,
		PartyID:        in.PartyID,
		Reference:      buildBookingReference(startUTC, service.Code),
		CustomerName:   strings.TrimSpace(in.CustomerName),
		CustomerPhone:  strings.TrimSpace(in.CustomerPhone),
		CustomerEmail:  strings.TrimSpace(in.CustomerEmail),
		Status:         status,
		Source:         normalizeBookingSource(in.Source),
		IdempotencyKey: strings.TrimSpace(in.IdempotencyKey),
		StartAt:        startUTC,
		EndAt:          endUTC,
		OccupiesFrom:   occFrom,
		OccupiesUntil:  occUntil,
		HoldExpiresAt:  in.HoldUntil,
		Notes:          strings.TrimSpace(in.Notes),
		Metadata:       metadata,
		CreatedBy:      actor,
	}
	if booking.Source == "" {
		booking.Source = schedulingdomain.BookingSourceAdmin
	}
	if booking.Status == schedulingdomain.BookingStatusConfirmed {
		now := time.Now().UTC()
		booking.ConfirmedAt = &now
	}
	created, err := u.repo.ReserveBookings(ctx, []schedulingdomain.Booking{booking})
	if err != nil {
		if isBookingOverlapErr(err) {
			return schedulingdomain.Booking{}, domainerr.Conflict("slot not available")
		}
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", booking.ID)
	}
	item := created[0]
	u.logAudit(ctx, actor, "scheduling.booking.created", "scheduling_booking", item.ID.String(), map[string]any{"service_id": item.ServiceID.String(), "allocations": item.Allocations, "status": item.Status})
	u.emitEvent(ctx, "scheduling.booking.created", map[string]any{"booking_id": item.ID.String(), "branch_id": item.BranchID.String(), "service_id": item.ServiceID.String()})
	return item, nil
}

func inferBookingEndFromServiceDefaults(service schedulingdomain.Service, startUTC time.Time) time.Time {
	minutes := service.DefaultDurationMinutes
	if minutes <= 0 {
		minutes = service.SlotGranularityMinutes
	}
	if minutes <= 0 {
		minutes = 30
	}
	return startUTC.Add(time.Duration(minutes) * time.Minute)
}

const maxRecurringBookingOccurrences = 60

func (u *Usecases) prepareBookingsForStarts(
	ctx context.Context,
	actor string,
	branch schedulingdomain.Branch,
	service schedulingdomain.Service,
	in schedulingdomain.CreateBookingInput,
	status schedulingdomain.BookingStatus,
	starts []time.Time,
	recurrence *schedulingdomain.BookingRecurrence,
) ([]schedulingdomain.Booking, error) {
	seriesID := uuid.New()
	out := make([]schedulingdomain.Booking, 0, len(starts))
	for index, startAt := range starts {
		candidateSlots, err := u.listAvailableSlots(ctx, schedulingdomain.SlotQuery{
			BranchID:     in.BranchID,
			ServiceID:    in.ServiceID,
			Date:         startAt,
			Participants: in.Participants,
			ResourceID:   in.ResourceID,
			Resources:    in.Resources,
		})
		if err != nil {
			return nil, err
		}
		matchingSlots := filterSlotsByStart(candidateSlots, startAt.UTC(), in.ResourceID, in.Resources, in.Participants)
		if len(matchingSlots) == 0 {
			if recurrence == nil {
				return nil, errNoDiscreteSchedulingSlot
			}
			return nil, domainerr.Conflict("slot not available")
		}
		metadata := cloneMetadata(in.Metadata)
		idempotencyKey := strings.TrimSpace(in.IdempotencyKey)
		if recurrence != nil {
			metadata = appendRecurrenceMetadata(metadata, seriesID, *recurrence, index, len(starts))
			if idempotencyKey != "" {
				idempotencyKey = fmt.Sprintf("%s#%02d", idempotencyKey, index+1)
			}
		}
		out = append(out, buildBookingFromSlot(actor, branch, service, in, matchingSlots[0], status, idempotencyKey, metadata))
	}
	return out, nil
}

func buildBookingFromSlot(
	actor string,
	branch schedulingdomain.Branch,
	service schedulingdomain.Service,
	in schedulingdomain.CreateBookingInput,
	slot schedulingdomain.TimeSlot,
	status schedulingdomain.BookingStatus,
	idempotencyKey string,
	metadata map[string]any,
) schedulingdomain.Booking {
	booking := schedulingdomain.Booking{
		ID:             uuid.New(),
		BranchID:       branch.ID,
		ServiceID:      service.ID,
		ResourceID:     slot.ResourceID,
		Allocations:    append([]schedulingdomain.ResourceAllocation(nil), slot.Allocations...),
		Participants:   in.Participants,
		PartyID:        in.PartyID,
		Reference:      buildBookingReference(slot.StartAt, service.Code),
		CustomerName:   strings.TrimSpace(in.CustomerName),
		CustomerPhone:  strings.TrimSpace(in.CustomerPhone),
		CustomerEmail:  strings.TrimSpace(in.CustomerEmail),
		Status:         status,
		Source:         normalizeBookingSource(in.Source),
		IdempotencyKey: idempotencyKey,
		StartAt:        slot.StartAt.UTC(),
		EndAt:          slot.EndAt.UTC(),
		OccupiesFrom:   slot.OccupiesFrom.UTC(),
		OccupiesUntil:  slot.OccupiesUntil.UTC(),
		HoldExpiresAt:  in.HoldUntil,
		Notes:          strings.TrimSpace(in.Notes),
		Metadata:       metadata,
		CreatedBy:      actor,
	}
	if booking.Source == "" {
		booking.Source = schedulingdomain.BookingSourceAdmin
	}
	if booking.Status == schedulingdomain.BookingStatusConfirmed {
		now := time.Now().UTC()
		booking.ConfirmedAt = &now
	}
	return booking
}

func normalizeBookingRecurrence(recurrence *schedulingdomain.BookingRecurrence, startAt time.Time, timezone string) (*schedulingdomain.BookingRecurrence, error) {
	if recurrence == nil {
		return nil, nil
	}
	freq := strings.ToLower(strings.TrimSpace(recurrence.Freq))
	if freq != "daily" && freq != "weekly" && freq != "monthly" {
		return nil, domainerr.Validation("invalid recurrence freq")
	}
	interval := recurrence.Interval
	if interval <= 0 {
		interval = 1
	}
	if interval > 365 {
		return nil, domainerr.Validation("recurrence interval is too large")
	}
	count := recurrence.Count
	if count <= 0 && recurrence.Until == nil {
		return nil, domainerr.Validation("recurrence count or until is required")
	}
	if count > maxRecurringBookingOccurrences {
		return nil, domainerr.Validation("recurrence count exceeds limit")
	}
	normalized := &schedulingdomain.BookingRecurrence{
		Freq:     freq,
		Interval: interval,
		Count:    count,
	}
	if recurrence.Until != nil {
		until := recurrence.Until.UTC()
		if until.Before(startAt.UTC()) {
			return nil, domainerr.Validation("recurrence until must be after start_at")
		}
		normalized.Until = &until
	}
	if freq == "weekly" {
		weekdays := normalizeWeekdays(recurrence.ByWeekday)
		if len(weekdays) == 0 {
			loc := loadSchedulingLocation(timezone)
			weekdays = []int{int(startAt.In(loc).Weekday())}
		}
		normalized.ByWeekday = weekdays
	}
	return normalized, nil
}

func expandRecurringBookingStarts(startAt time.Time, recurrence *schedulingdomain.BookingRecurrence, timezone string) ([]time.Time, error) {
	if recurrence == nil {
		return []time.Time{startAt.UTC()}, nil
	}
	loc := loadSchedulingLocation(timezone)
	localStart := startAt.In(loc)
	until := recurrence.Until
	appendIfValid := func(items []time.Time, candidate time.Time) []time.Time {
		if until != nil && candidate.UTC().After(until.UTC()) {
			return items
		}
		return append(items, candidate.UTC())
	}
	starts := make([]time.Time, 0, maxRecurringBookingOccurrences)
	switch recurrence.Freq {
	case "daily":
		for cursor := localStart; len(starts) < maxRecurringBookingOccurrences; cursor = cursor.AddDate(0, 0, recurrence.Interval) {
			if until != nil && cursor.UTC().After(until.UTC()) {
				break
			}
			starts = appendIfValid(starts, cursor)
			if recurrence.Count > 0 && len(starts) >= recurrence.Count {
				break
			}
		}
	case "weekly":
		weekdays := recurrence.ByWeekday
		baseWeekStart := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -int(localStart.Weekday()))
		for week := 0; len(starts) < maxRecurringBookingOccurrences; week += recurrence.Interval {
			weekBase := baseWeekStart.AddDate(0, 0, week*7)
			produced := false
			for _, weekday := range weekdays {
				day := weekBase.AddDate(0, 0, weekday)
				candidate := time.Date(day.Year(), day.Month(), day.Day(), localStart.Hour(), localStart.Minute(), localStart.Second(), localStart.Nanosecond(), loc)
				if candidate.Before(localStart) {
					continue
				}
				if until != nil && candidate.UTC().After(until.UTC()) {
					continue
				}
				starts = append(starts, candidate.UTC())
				produced = true
				if recurrence.Count > 0 && len(starts) >= recurrence.Count {
					break
				}
				if len(starts) >= maxRecurringBookingOccurrences {
					break
				}
			}
			if recurrence.Count > 0 && len(starts) >= recurrence.Count {
				break
			}
			if until != nil && !produced && weekBase.After(until.In(loc)) {
				break
			}
		}
		sort.Slice(starts, func(i, j int) bool { return starts[i].Before(starts[j]) })
	case "monthly":
		for cursor := localStart; len(starts) < maxRecurringBookingOccurrences; cursor = addMonthsPreservingDay(cursor, recurrence.Interval) {
			if until != nil && cursor.UTC().After(until.UTC()) {
				break
			}
			starts = appendIfValid(starts, cursor)
			if recurrence.Count > 0 && len(starts) >= recurrence.Count {
				break
			}
		}
	default:
		return nil, domainerr.Validation("invalid recurrence freq")
	}
	if len(starts) == 0 {
		return nil, domainerr.Validation("recurrence did not generate occurrences")
	}
	return starts, nil
}

func appendRecurrenceMetadata(
	metadata map[string]any,
	seriesID uuid.UUID,
	recurrence schedulingdomain.BookingRecurrence,
	index int,
	total int,
) map[string]any {
	next := cloneMetadata(metadata)
	recurrencePayload := map[string]any{
		"series_id":        seriesID.String(),
		"freq":             recurrence.Freq,
		"interval":         recurrence.Interval,
		"occurrence_index": index + 1,
		"occurrence_count": total,
	}
	if recurrence.Count > 0 {
		recurrencePayload["count"] = recurrence.Count
	}
	if recurrence.Until != nil {
		recurrencePayload["until"] = recurrence.Until.UTC().Format(time.RFC3339)
	}
	if len(recurrence.ByWeekday) > 0 {
		recurrencePayload["by_weekday"] = recurrence.ByWeekday
	}
	next["recurrence"] = recurrencePayload
	return next
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}
	next := make(map[string]any, len(metadata))
	for key, value := range metadata {
		next[key] = value
	}
	return next
}

func normalizeWeekdays(days []int) []int {
	seen := make(map[int]struct{}, len(days))
	out := make([]int, 0, len(days))
	for _, day := range days {
		if day < 0 || day > 6 {
			continue
		}
		if _, ok := seen[day]; ok {
			continue
		}
		seen[day] = struct{}{}
		out = append(out, day)
	}
	sort.Ints(out)
	return out
}

func loadSchedulingLocation(timezone string) *time.Location {
	if loc, err := time.LoadLocation(strings.TrimSpace(timezone)); err == nil {
		return loc
	}
	return time.UTC
}

func addMonthsPreservingDay(base time.Time, months int) time.Time {
	targetMonth := time.Date(base.Year(), base.Month()+time.Month(months), 1, base.Hour(), base.Minute(), base.Second(), base.Nanosecond(), base.Location())
	lastDay := time.Date(targetMonth.Year(), targetMonth.Month()+1, 0, base.Hour(), base.Minute(), base.Second(), base.Nanosecond(), base.Location()).Day()
	day := base.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(targetMonth.Year(), targetMonth.Month(), day, base.Hour(), base.Minute(), base.Second(), base.Nanosecond(), base.Location())
}

func (u *Usecases) CheckInBooking(ctx context.Context, bookingID uuid.UUID, actor string) (schedulingdomain.Booking, error) {
	return u.transitionBooking(ctx, bookingID, actor, schedulingdomain.BookingStatusCheckedIn, "")
}

func (u *Usecases) StartBookingService(ctx context.Context, bookingID uuid.UUID, actor string) (schedulingdomain.Booking, error) {
	return u.transitionBooking(ctx, bookingID, actor, schedulingdomain.BookingStatusInService, "")
}

func (u *Usecases) CompleteBooking(ctx context.Context, bookingID uuid.UUID, actor string) (schedulingdomain.Booking, error) {
	return u.transitionBooking(ctx, bookingID, actor, schedulingdomain.BookingStatusCompleted, "")
}

func (u *Usecases) MarkBookingNoShow(ctx context.Context, bookingID uuid.UUID, actor string, reason string) (schedulingdomain.Booking, error) {
	return u.transitionBooking(ctx, bookingID, actor, schedulingdomain.BookingStatusNoShow, reason)
}

func (u *Usecases) ConfirmBooking(ctx context.Context, bookingID uuid.UUID, actor string) (schedulingdomain.Booking, error) {
	current, err := u.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", bookingID)
	}
	if !canTransitionBooking(current.Status, schedulingdomain.BookingStatusConfirmed) {
		return schedulingdomain.Booking{}, domainerr.Conflict("booking cannot transition to confirmed")
	}
	now := time.Now().UTC()
	out, err := u.repo.UpdateBookingStatus(ctx, bookingID, schedulingdomain.BookingStatusConfirmed, &now, nil, current.Notes)
	if err != nil {
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", bookingID)
	}
	u.logAudit(ctx, actor, "scheduling.booking.confirmed", "scheduling_booking", out.ID.String(), nil)
	return out, nil
}

func (u *Usecases) CancelBooking(ctx context.Context, bookingID uuid.UUID, actor, reason string) (schedulingdomain.Booking, error) {
	return u.cancelBooking(ctx, bookingID, actor, reason, false)
}

func (u *Usecases) CancelBookingByToken(ctx context.Context, tokenRaw, reason string) (schedulingdomain.Booking, error) {
	token, err := u.lookupBookingActionToken(ctx, tokenRaw)
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	if token.Action != schedulingdomain.BookingActionCancel {
		return schedulingdomain.Booking{}, domainerr.Validation("token action mismatch")
	}
	out, err := u.cancelBooking(ctx, token.BookingID, "public-token", reason, true)
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	_ = u.repo.MarkBookingActionTokenUsed(ctx, token.ID, time.Now().UTC())
	return out, nil
}

func (u *Usecases) ConfirmBookingByToken(ctx context.Context, tokenRaw string) (schedulingdomain.Booking, error) {
	token, err := u.lookupBookingActionToken(ctx, tokenRaw)
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	if token.Action != schedulingdomain.BookingActionConfirm {
		return schedulingdomain.Booking{}, domainerr.Validation("token action mismatch")
	}
	out, err := u.ConfirmBooking(ctx, token.BookingID, "public-token")
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	_ = u.repo.MarkBookingActionTokenUsed(ctx, token.ID, time.Now().UTC())
	return out, nil
}

func (u *Usecases) CreateBookingActionTokens(ctx context.Context, bookingID uuid.UUID, ttl time.Duration) (map[schedulingdomain.BookingActionType]schedulingdomain.BookingActionToken, error) {
	if ttl <= 0 {
		ttl = 48 * time.Hour
	}
	booking, err := u.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, mapRepoError(err, "booking", bookingID)
	}
	out := make(map[schedulingdomain.BookingActionType]schedulingdomain.BookingActionToken, 2)
	for _, action := range []schedulingdomain.BookingActionType{schedulingdomain.BookingActionConfirm, schedulingdomain.BookingActionCancel} {
		if !u.bookingSupportsAction(booking.Status, action) {
			continue
		}
		rawToken, tokenHash, err := newActionToken()
		if err != nil {
			return nil, err
		}
		token, err := u.repo.CreateBookingActionToken(ctx, schedulingdomain.BookingActionToken{
			ID:        uuid.New(),
			BookingID: booking.ID,
			Action:    action,
			Token:     rawToken,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().UTC().Add(ttl),
			Metadata: map[string]any{
				"booking_reference": booking.Reference,
				"booking_status":    booking.Status,
			},
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return nil, err
		}
		token.Token = rawToken
		out[action] = token
	}
	return out, nil
}

func (u *Usecases) ExpireOverdueHolds(ctx context.Context, limit int) ([]schedulingdomain.Booking, error) {
	items, err := u.repo.ExpireOverdueHolds(ctx, limit)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		u.emitEvent(ctx, "scheduling.booking.expired", map[string]any{"booking_id": item.ID.String()})
	}
	return items, nil
}

func (u *Usecases) MarkBookingReminderSent(ctx context.Context, bookingID uuid.UUID, sentAt time.Time) (schedulingdomain.Booking, error) {
	return u.repo.MarkBookingReminderSent(ctx, bookingID, sentAt)
}

func (u *Usecases) RescheduleBooking(ctx context.Context, actor string, in schedulingdomain.RescheduleBookingInput) (schedulingdomain.Booking, error) {
	current, err := u.repo.GetBookingByID(ctx, in.BookingID)
	if err != nil {
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", in.BookingID)
	}
	if !canRescheduleBooking(current.Status) {
		return schedulingdomain.Booking{}, domainerr.Conflict("booking cannot be rescheduled")
	}
	branchID := current.BranchID
	if in.BranchID != uuid.Nil {
		branchID = in.BranchID
	}
	resourceID := in.ResourceID
	requestedAllocations := in.Resources
	participants, err := normalizeParticipants(current.Participants)
	if err != nil {
		return schedulingdomain.Booking{}, err
	}
	current.Participants = participants
	if len(requestedAllocations) == 0 && resourceID == nil {
		if len(current.Allocations) > 0 {
			requestedAllocations = append([]schedulingdomain.ResourceAllocation(nil), current.Allocations...)
		} else {
			resourceID = &current.ResourceID
		}
	}
	allocations, err := NormalizeResourceAllocationsForParticipants(resourceID, requestedAllocations, participants)
	if err != nil {
		return schedulingdomain.Booking{}, domainerr.Validation(err.Error())
	}
	if len(allocations) == 0 {
		return schedulingdomain.Booking{}, domainerr.Validation("at least one resource allocation is required")
	}

	if in.EndAt != nil && !in.EndAt.IsZero() {
		// Custom-duration mode: the caller is moving and/or resizing the booking outside
		// of the canonical service slots (used by the owner's internal calendar). We still
		// validate against availability rules, blocked ranges and existing bookings so the
		// rescheduled booking remains coherent with what public clients see.
		if !in.EndAt.After(in.StartAt) {
			return schedulingdomain.Booking{}, domainerr.Validation("end_at must be after start_at")
		}
		branch, err := u.repo.GetBranch(ctx, branchID)
		if err != nil {
			return schedulingdomain.Booking{}, mapRepoError(err, "branch", branchID)
		}
		service, err := u.repo.GetService(ctx, current.ServiceID)
		if err != nil {
			return schedulingdomain.Booking{}, mapRepoError(err, "service", current.ServiceID)
		}
		startUTC := in.StartAt.UTC()
		endUTC := in.EndAt.UTC()
		occFrom := startUTC.Add(-time.Duration(service.BufferBeforeMinutes) * time.Minute)
		occUntil := endUTC.Add(time.Duration(service.BufferAfterMinutes) * time.Minute)
		excludeID := current.ID
		allocations, err = u.validateBookingAllocationsRangeFits(
			ctx,
			branch,
			service,
			allocations,
			startUTC,
			endUTC,
			occFrom,
			occUntil,
			&excludeID,
		)
		if err != nil {
			return schedulingdomain.Booking{}, err
		}
		current.BranchID = branchID
		current.ResourceID = allocations[0].ResourceID
		current.Allocations = allocations
		current.StartAt = startUTC
		current.EndAt = endUTC
		current.OccupiesFrom = occFrom
		current.OccupiesUntil = occUntil
	} else {
		slots, err := u.listAvailableSlots(ctx, schedulingdomain.SlotQuery{
			BranchID:     branchID,
			ServiceID:    current.ServiceID,
			Date:         in.StartAt,
			Participants: current.Participants,
			ResourceID:   resourceID,
			Resources:    allocations,
		})
		if err != nil {
			return schedulingdomain.Booking{}, err
		}
		matching := filterSlotsByStart(slots, in.StartAt.UTC(), resourceID, allocations, participants)
		if len(matching) == 0 {
			return schedulingdomain.Booking{}, domainerr.Conflict("slot not available")
		}
		slot := matching[0]
		current.BranchID = branchID
		current.ResourceID = slot.ResourceID
		current.Allocations = append([]schedulingdomain.ResourceAllocation(nil), slot.Allocations...)
		current.StartAt = slot.StartAt
		current.EndAt = slot.EndAt
		current.OccupiesFrom = slot.OccupiesFrom
		current.OccupiesUntil = slot.OccupiesUntil
	}

	out, err := u.repo.RescheduleBooking(ctx, current)
	if err != nil {
		if isBookingOverlapErr(err) {
			return schedulingdomain.Booking{}, domainerr.Conflict("slot not available")
		}
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", current.ID)
	}
	u.logAudit(ctx, actor, "scheduling.booking.rescheduled", "scheduling_booking", out.ID.String(), map[string]any{"start_at": out.StartAt, "end_at": out.EndAt})
	u.emitEvent(ctx, "scheduling.booking.rescheduled", map[string]any{"booking_id": out.ID.String(), "start_at": out.StartAt, "end_at": out.EndAt})
	return out, nil
}

// validateBookingRangeFits checks that an arbitrary [startAt, endAt] booking range
// (with the corresponding occupation window built from service buffers) is acceptable
// for the given resource: inside availability rules, not overlapping any blocked range,
// and not overlapping any existing booking. Used by reschedule's custom-duration path.
func (u *Usecases) validateBookingAllocationsRangeFits(
	ctx context.Context,
	branch schedulingdomain.Branch,
	service schedulingdomain.Service,
	allocations []schedulingdomain.ResourceAllocation,
	startAt, endAt, occFrom, occUntil time.Time,
	excludeBookingID *uuid.UUID,
) ([]schedulingdomain.ResourceAllocation, error) {
	hydrated := make([]schedulingdomain.ResourceAllocation, 0, len(allocations))
	for _, allocation := range allocations {
		resource, err := u.loadServiceResource(ctx, branch.ID, service.ID, allocation.ResourceID)
		if err != nil {
			return nil, err
		}
		allocation, err = ResolveAllocationUnits(allocation, resource.Capacity)
		if err != nil {
			return nil, domainerr.Conflict(err.Error())
		}
		if err := u.validateBookingRangeFits(
			ctx,
			branch,
			service,
			resource,
			allocation.Units,
			startAt,
			endAt,
			occFrom,
			occUntil,
			excludeBookingID,
		); err != nil {
			return nil, err
		}
		allocation.ResourceName = resource.Name
		hydrated = append(hydrated, allocation)
	}
	return hydrated, nil
}

func (u *Usecases) validateBookingRangeFits(
	ctx context.Context,
	branch schedulingdomain.Branch,
	service schedulingdomain.Service,
	resource schedulingdomain.Resource,
	units int,
	startAt, endAt, occFrom, occUntil time.Time,
	excludeBookingID *uuid.UUID,
) error {
	loc := time.UTC
	if strings.TrimSpace(branch.Timezone) != "" {
		if l, err := time.LoadLocation(branch.Timezone); err == nil {
			loc = l
		}
	}
	if strings.TrimSpace(resource.Timezone) != "" {
		if l, err := time.LoadLocation(resource.Timezone); err == nil {
			loc = l
		}
	}
	dayLocal := startAt.In(loc)
	rules, err := u.repo.ListApplicableAvailabilityRules(ctx, branch.ID, &resource.ID, dayLocal)
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return domainerr.Conflict("outside business hours")
	}
	branchWindows := make([]corescheduling.Window, 0)
	resourceWindows := make([]corescheduling.Window, 0)
	for _, rule := range rules {
		startClock, err := corescheduling.ParseClock(rule.StartTime)
		if err != nil {
			continue
		}
		endClock, err := corescheduling.ParseClock(rule.EndTime)
		if err != nil {
			continue
		}
		s := time.Date(dayLocal.Year(), dayLocal.Month(), dayLocal.Day(), startClock.Hour(), startClock.Minute(), 0, 0, loc)
		e := time.Date(dayLocal.Year(), dayLocal.Month(), dayLocal.Day(), endClock.Hour(), endClock.Minute(), 0, 0, loc)
		if !e.After(s) {
			continue
		}
		w := corescheduling.Window{Start: s, End: e}
		if rule.Kind == schedulingdomain.AvailabilityRuleKindBranch {
			branchWindows = append(branchWindows, w)
		} else {
			resourceWindows = append(resourceWindows, w)
		}
	}
	activeWindows := corescheduling.IntersectWindows(branchWindows, resourceWindows)
	if !rangeFitsAnyWindow(startAt, endAt, activeWindows) {
		return domainerr.Conflict("outside business hours")
	}
	blocked, err := u.repo.ListBlockedRangesBetween(ctx, branch.ID, &resource.ID, occFrom, occUntil)
	if err != nil {
		return err
	}
	if len(blocked) > 0 {
		return domainerr.Conflict("overlaps with a blocked range")
	}
	allocated, err := u.repo.AllocatedResourceUnits(ctx, resource.ID, occFrom, occUntil, excludeBookingID)
	if err != nil {
		return err
	}
	if allocated+units > resource.Capacity {
		return domainerr.Conflict("slot not available")
	}
	concurrent, err := u.repo.ConcurrentServiceBookings(ctx, service.ID, occFrom, occUntil, excludeBookingID)
	if err != nil {
		return err
	}
	if concurrent+1 > service.MaxConcurrentBookings {
		return domainerr.Conflict("slot not available")
	}
	return nil
}

func rangeFitsAnyWindow(start, end time.Time, windows []corescheduling.Window) bool {
	startUTC := start.UTC()
	endUTC := end.UTC()
	for _, w := range windows {
		if !startUTC.Before(w.Start.UTC()) && !endUTC.After(w.End.UTC()) {
			return true
		}
	}
	return false
}

func (u *Usecases) ListQueues(ctx context.Context, branchID *uuid.UUID) ([]schedulingdomain.Queue, error) {
	return u.repo.ListQueues(ctx, branchID)
}

func (u *Usecases) CreateQueue(ctx context.Context, actor string, in schedulingdomain.Queue) (schedulingdomain.Queue, error) {
	if in.BranchID == uuid.Nil {
		return schedulingdomain.Queue{}, domainerr.Validation("branch_id is required")
	}
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Code) == "" {
		return schedulingdomain.Queue{}, domainerr.Validation("name and code are required")
	}
	if _, err := u.repo.GetBranch(ctx, in.BranchID); err != nil {
		return schedulingdomain.Queue{}, mapRepoError(err, "branch", in.BranchID)
	}
	if in.ServiceID != nil && *in.ServiceID != uuid.Nil {
		service, err := u.repo.GetService(ctx, *in.ServiceID)
		if err != nil {
			return schedulingdomain.Queue{}, mapRepoError(err, "service", *in.ServiceID)
		}
		if service.FulfillmentMode == schedulingdomain.FulfillmentModeSchedule {
			return schedulingdomain.Queue{}, domainerr.Validation("service is schedule-only")
		}
	}
	strategy := normalizeQueueStrategy(in.Strategy)
	if strategy == "" {
		strategy = schedulingdomain.QueueStrategyFIFO
	}
	status := normalizeQueueStatus(in.Status)
	if status == "" {
		status = schedulingdomain.QueueStatusActive
	}
	if in.AvgServiceSecond <= 0 {
		in.AvgServiceSecond = 600
	}
	in.ID = ensureUUID(in.ID)
	in.Code = normalizeCode(in.Code)
	in.Strategy = strategy
	in.Status = status
	if strings.TrimSpace(in.TicketPrefix) == "" {
		in.TicketPrefix = strings.ToUpper(in.Code[:min(3, len(in.Code))])
	}
	out, err := u.repo.CreateQueue(ctx, in)
	if err != nil {
		return schedulingdomain.Queue{}, mapRepoError(err, "queue", in.ID)
	}
	u.logAudit(ctx, actor, "scheduling.queue.created", "scheduling_queue", out.ID.String(), map[string]any{"code": out.Code})
	return out, nil
}

func (u *Usecases) PauseQueue(ctx context.Context, queueID uuid.UUID, actor string) (schedulingdomain.Queue, error) {
	return u.transitionQueue(ctx, queueID, actor, schedulingdomain.QueueStatusPaused)
}

func (u *Usecases) ReopenQueue(ctx context.Context, queueID uuid.UUID, actor string) (schedulingdomain.Queue, error) {
	return u.transitionQueue(ctx, queueID, actor, schedulingdomain.QueueStatusActive)
}

func (u *Usecases) CloseQueue(ctx context.Context, queueID uuid.UUID, actor string) (schedulingdomain.Queue, error) {
	return u.transitionQueue(ctx, queueID, actor, schedulingdomain.QueueStatusClosed)
}

func (u *Usecases) IssueQueueTicket(ctx context.Context, actor string, in schedulingdomain.CreateQueueTicketInput) (schedulingdomain.QueueTicket, error) {
	if in.QueueID == uuid.Nil {
		return schedulingdomain.QueueTicket{}, domainerr.Validation("queue_id is required")
	}
	if strings.TrimSpace(in.CustomerName) == "" {
		return schedulingdomain.QueueTicket{}, domainerr.Validation("customer_name is required")
	}
	queue, err := u.repo.GetQueueByID(ctx, in.QueueID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapRepoError(err, "queue", in.QueueID)
	}
	if queue.Status != schedulingdomain.QueueStatusActive {
		return schedulingdomain.QueueTicket{}, domainerr.Conflict("queue is not active")
	}
	if !queue.AllowRemoteJoin && in.Source != schedulingdomain.QueueTicketSourceReception {
		return schedulingdomain.QueueTicket{}, domainerr.Conflict("queue does not allow remote join")
	}
	if in.Priority <= 0 {
		in.Priority = 100
	}
	source := normalizeQueueTicketSource(in.Source)
	if source == "" {
		source = schedulingdomain.QueueTicketSourceReception
	}
	out, err := u.repo.CreateQueueTicket(ctx, schedulingdomain.QueueTicket{
		ID:             uuid.New(),
		QueueID:        queue.ID,
		BranchID:       queue.BranchID,
		ServiceID:      queue.ServiceID,
		PartyID:        in.PartyID,
		CustomerName:   strings.TrimSpace(in.CustomerName),
		CustomerPhone:  strings.TrimSpace(in.CustomerPhone),
		CustomerEmail:  strings.TrimSpace(in.CustomerEmail),
		Status:         schedulingdomain.QueueTicketStatusWaiting,
		Priority:       in.Priority,
		Source:         source,
		IdempotencyKey: strings.TrimSpace(in.IdempotencyKey),
		Notes:          strings.TrimSpace(in.Notes),
		Metadata:       in.Metadata,
	})
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapQueueError(err, queue.ID)
	}
	u.logAudit(ctx, actor, "scheduling.queue.ticket.created", "scheduling_queue_ticket", out.ID.String(), map[string]any{"queue_id": queue.ID.String(), "number": out.Number})
	return out, nil
}

func (u *Usecases) GetQueueTicketPosition(ctx context.Context, queueID, ticketID uuid.UUID) (schedulingdomain.QueuePosition, error) {
	out, err := u.repo.GetQueueTicketPosition(ctx, queueID, ticketID)
	if err != nil {
		return schedulingdomain.QueuePosition{}, mapRepoError(err, "queue_ticket", ticketID)
	}
	return out, nil
}

func (u *Usecases) CallNextTicket(ctx context.Context, queueID uuid.UUID, actor string, servingResourceID *uuid.UUID, operatorUserID *string) (schedulingdomain.QueueTicket, error) {
	out, err := u.repo.CallNextTicket(ctx, queueID, servingResourceID, operatorUserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, errNoTicketWaiting) {
			return schedulingdomain.QueueTicket{}, domainerr.NotFound("no waiting tickets")
		}
		return schedulingdomain.QueueTicket{}, mapQueueError(err, queueID)
	}
	u.logAudit(ctx, actor, "scheduling.queue.ticket.called", "scheduling_queue_ticket", out.ID.String(), map[string]any{"queue_id": queueID.String(), "number": out.Number})
	return out, nil
}

func (u *Usecases) MarkTicketServing(ctx context.Context, queueID, ticketID uuid.UUID, actor string, servingResourceID *uuid.UUID, operatorUserID *string) (schedulingdomain.QueueTicket, error) {
	current, err := u.repo.GetQueueTicket(ctx, queueID, ticketID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapRepoError(err, "queue_ticket", ticketID)
	}
	if !canTransitionQueueTicket(current.Status, schedulingdomain.QueueTicketStatusServing) {
		return schedulingdomain.QueueTicket{}, domainerr.Conflict("ticket cannot transition to serving")
	}
	out, err := u.repo.MarkQueueTicketServing(ctx, queueID, ticketID, servingResourceID, operatorUserID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapQueueError(err, queueID)
	}
	u.logAudit(ctx, actor, "scheduling.queue.ticket.serving", "scheduling_queue_ticket", out.ID.String(), nil)
	return out, nil
}

func (u *Usecases) CompleteTicket(ctx context.Context, queueID, ticketID uuid.UUID, actor string, servingResourceID *uuid.UUID, operatorUserID *string) (schedulingdomain.QueueTicket, error) {
	current, err := u.repo.GetQueueTicket(ctx, queueID, ticketID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapRepoError(err, "queue_ticket", ticketID)
	}
	if !canTransitionQueueTicket(current.Status, schedulingdomain.QueueTicketStatusCompleted) {
		return schedulingdomain.QueueTicket{}, domainerr.Conflict("ticket cannot transition to completed")
	}
	out, err := u.repo.UpdateQueueTicketStatus(ctx, queueID, ticketID, schedulingdomain.QueueTicketStatusCompleted, servingResourceID, operatorUserID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapQueueError(err, queueID)
	}
	u.logAudit(ctx, actor, "scheduling.queue.ticket.completed", "scheduling_queue_ticket", out.ID.String(), nil)
	return out, nil
}

func (u *Usecases) CancelTicket(ctx context.Context, queueID, ticketID uuid.UUID, actor string) (schedulingdomain.QueueTicket, error) {
	current, err := u.repo.GetQueueTicket(ctx, queueID, ticketID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapRepoError(err, "queue_ticket", ticketID)
	}
	if !canTransitionQueueTicket(current.Status, schedulingdomain.QueueTicketStatusCancelled) {
		return schedulingdomain.QueueTicket{}, domainerr.Conflict("ticket cannot transition to cancelled")
	}
	out, err := u.repo.UpdateQueueTicketStatus(ctx, queueID, ticketID, schedulingdomain.QueueTicketStatusCancelled, current.ServingResourceID, current.OperatorUserID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapQueueError(err, queueID)
	}
	u.logAudit(ctx, actor, "scheduling.queue.ticket.cancelled", "scheduling_queue_ticket", out.ID.String(), nil)
	return out, nil
}

func (u *Usecases) MarkTicketNoShow(ctx context.Context, queueID, ticketID uuid.UUID, actor string) (schedulingdomain.QueueTicket, error) {
	current, err := u.repo.GetQueueTicket(ctx, queueID, ticketID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapRepoError(err, "queue_ticket", ticketID)
	}
	if !canTransitionQueueTicket(current.Status, schedulingdomain.QueueTicketStatusNoShow) {
		return schedulingdomain.QueueTicket{}, domainerr.Conflict("ticket cannot transition to no_show")
	}
	out, err := u.repo.UpdateQueueTicketStatus(ctx, queueID, ticketID, schedulingdomain.QueueTicketStatusNoShow, current.ServingResourceID, current.OperatorUserID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapQueueError(err, queueID)
	}
	u.logAudit(ctx, actor, "scheduling.queue.ticket.no_show", "scheduling_queue_ticket", out.ID.String(), nil)
	return out, nil
}

func (u *Usecases) ReassignTicket(ctx context.Context, queueID, ticketID uuid.UUID, actor string, servingResourceID *uuid.UUID, operatorUserID *string) (schedulingdomain.QueueTicket, error) {
	if servingResourceID == nil || *servingResourceID == uuid.Nil {
		return schedulingdomain.QueueTicket{}, domainerr.Validation("serving_resource_id is required")
	}
	if _, err := u.repo.GetResource(ctx, *servingResourceID); err != nil {
		return schedulingdomain.QueueTicket{}, mapRepoError(err, "resource", *servingResourceID)
	}
	out, err := u.repo.ReassignQueueTicket(ctx, queueID, ticketID, servingResourceID, operatorUserID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapQueueError(err, queueID)
	}
	u.logAudit(ctx, actor, "scheduling.queue.ticket.reassigned", "scheduling_queue_ticket", out.ID.String(), map[string]any{"serving_resource_id": servingResourceID.String()})
	return out, nil
}

func (u *Usecases) ReturnTicketToWaiting(ctx context.Context, queueID, ticketID uuid.UUID, actor string) (schedulingdomain.QueueTicket, error) {
	current, err := u.repo.GetQueueTicket(ctx, queueID, ticketID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapRepoError(err, "queue_ticket", ticketID)
	}
	if current.Status != schedulingdomain.QueueTicketStatusCalled && current.Status != schedulingdomain.QueueTicketStatusServing {
		return schedulingdomain.QueueTicket{}, domainerr.Conflict("ticket cannot transition to waiting")
	}
	out, err := u.repo.ReturnQueueTicketToWaiting(ctx, queueID, ticketID)
	if err != nil {
		return schedulingdomain.QueueTicket{}, mapQueueError(err, queueID)
	}
	u.logAudit(ctx, actor, "scheduling.queue.ticket.returned_to_waiting", "scheduling_queue_ticket", out.ID.String(), nil)
	return out, nil
}

func (u *Usecases) JoinWaitlist(ctx context.Context, actor string, in schedulingdomain.CreateWaitlistInput) (schedulingdomain.WaitlistEntry, error) {
	if in.BranchID == uuid.Nil || in.ServiceID == uuid.Nil {
		return schedulingdomain.WaitlistEntry{}, domainerr.Validation("branch_id and service_id are required")
	}
	if strings.TrimSpace(in.CustomerName) == "" {
		return schedulingdomain.WaitlistEntry{}, domainerr.Validation("customer_name is required")
	}
	if in.RequestedStartAt.IsZero() {
		return schedulingdomain.WaitlistEntry{}, domainerr.Validation("requested_start_at is required")
	}
	_, service, err := u.loadBookingScope(ctx, in.BranchID, in.ServiceID)
	if err != nil {
		return schedulingdomain.WaitlistEntry{}, err
	}
	if service.FulfillmentMode == schedulingdomain.FulfillmentModeQueue {
		return schedulingdomain.WaitlistEntry{}, domainerr.Validation("service is queue-only")
	}
	if !service.AllowWaitlist {
		return schedulingdomain.WaitlistEntry{}, domainerr.Conflict("service does not allow waitlist")
	}
	if in.ResourceID != nil && *in.ResourceID != uuid.Nil {
		if _, err := u.repo.GetResource(ctx, *in.ResourceID); err != nil {
			return schedulingdomain.WaitlistEntry{}, mapRepoError(err, "resource", *in.ResourceID)
		}
	}
	source := normalizeWaitlistSource(in.Source)
	if source == "" {
		source = defaultWaitlistSource(actor)
	}
	now := time.Now().UTC()
	out, err := u.repo.CreateWaitlistEntry(ctx, schedulingdomain.WaitlistEntry{
		ID:               uuid.New(),
		BranchID:         in.BranchID,
		ServiceID:        in.ServiceID,
		ResourceID:       in.ResourceID,
		PartyID:          in.PartyID,
		CustomerName:     strings.TrimSpace(in.CustomerName),
		CustomerPhone:    strings.TrimSpace(in.CustomerPhone),
		CustomerEmail:    strings.TrimSpace(in.CustomerEmail),
		RequestedStartAt: in.RequestedStartAt.UTC(),
		Status:           schedulingdomain.WaitlistStatusPending,
		Source:           source,
		IdempotencyKey:   strings.TrimSpace(in.IdempotencyKey),
		Notes:            strings.TrimSpace(in.Notes),
		Metadata:         in.Metadata,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return schedulingdomain.WaitlistEntry{}, err
	}
	u.logAudit(ctx, actor, "scheduling.waitlist.joined", "scheduling_waitlist_entry", out.ID.String(), map[string]any{"service_id": out.ServiceID.String()})
	return out, nil
}

func (u *Usecases) ListWaitlistEntries(ctx context.Context, filter schedulingdomain.ListWaitlistFilter) ([]schedulingdomain.WaitlistEntry, error) {
	return u.repo.ListWaitlistEntries(ctx, filter)
}

func (u *Usecases) ProcessWaitlistAvailability(ctx context.Context, now time.Time, limit int) ([]schedulingdomain.WaitlistEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	entries, err := u.repo.ListPendingWaitlistEntries(ctx, limit)
	if err != nil {
		return nil, err
	}
	notified := make([]schedulingdomain.WaitlistEntry, 0)
	for _, entry := range entries {
		if entry.RequestedStartAt.Before(now.UTC()) {
			_, _ = u.repo.UpdateWaitlistEntryStatus(ctx, entry.ID, schedulingdomain.WaitlistStatusExpired, nil, nil, nil, "requested slot already passed")
			continue
		}
		slots, err := u.listAvailableSlots(ctx, schedulingdomain.SlotQuery{
			BranchID:     entry.BranchID,
			ServiceID:    entry.ServiceID,
			Date:         entry.RequestedStartAt,
			Participants: 1,
			ResourceID:   entry.ResourceID,
		})
		if err != nil {
			continue
		}
		matching := filterSlotsByStart(slots, entry.RequestedStartAt.UTC(), entry.ResourceID, nil, 1)
		if len(matching) == 0 {
			continue
		}
		expiresAt := minTimePtr(ptrTime(now.UTC().Add(2*time.Hour)), ptrTime(entry.RequestedStartAt.UTC()))
		updated, err := u.repo.UpdateWaitlistEntryStatus(ctx, entry.ID, schedulingdomain.WaitlistStatusNotified, expiresAt, ptrTime(now.UTC()), nil, entry.Notes)
		if err != nil {
			continue
		}
		notified = append(notified, updated)
	}
	return notified, nil
}

func (u *Usecases) Dashboard(ctx context.Context, branchID *uuid.UUID, day time.Time) (schedulingdomain.DashboardStats, error) {
	timezone := "UTC"
	if branchID != nil && *branchID != uuid.Nil {
		branch, err := u.repo.GetBranch(ctx, *branchID)
		if err != nil {
			return schedulingdomain.DashboardStats{}, mapRepoError(err, "branch", *branchID)
		}
		timezone = branch.Timezone
	}
	return u.repo.DashboardStats(ctx, branchID, day, timezone)
}

func (u *Usecases) DayAgenda(ctx context.Context, branchID *uuid.UUID, day time.Time) ([]schedulingdomain.DayAgendaItem, error) {
	return u.repo.ListDayAgenda(ctx, branchID, day)
}

func (u *Usecases) loadBookingScope(ctx context.Context, branchID, serviceID uuid.UUID) (schedulingdomain.Branch, schedulingdomain.Service, error) {
	branch, err := u.repo.GetBranch(ctx, branchID)
	if err != nil {
		return schedulingdomain.Branch{}, schedulingdomain.Service{}, mapRepoError(err, "branch", branchID)
	}
	if !branch.Active {
		return schedulingdomain.Branch{}, schedulingdomain.Service{}, domainerr.Conflict("branch is inactive")
	}
	service, err := u.repo.GetService(ctx, serviceID)
	if err != nil {
		return schedulingdomain.Branch{}, schedulingdomain.Service{}, mapRepoError(err, "service", serviceID)
	}
	if !service.Active {
		return schedulingdomain.Branch{}, schedulingdomain.Service{}, domainerr.Conflict("service is inactive")
	}
	return branch, service, nil
}

func (u *Usecases) listAvailableSlots(ctx context.Context, query schedulingdomain.SlotQuery) ([]schedulingdomain.TimeSlot, error) {
	branch, service, err := u.loadBookingScope(ctx, query.BranchID, query.ServiceID)
	if err != nil {
		return nil, err
	}
	branchLoc, err := time.LoadLocation(branch.Timezone)
	if err != nil {
		return nil, domainerr.Validation("invalid branch timezone")
	}
	dayLocal := time.Date(query.Date.Year(), query.Date.Month(), query.Date.Day(), 0, 0, 0, 0, branchLoc)

	allocations, err := NormalizeResourceAllocationsForParticipants(query.ResourceID, query.Resources, query.Participants)
	if err != nil {
		return nil, domainerr.Validation(err.Error())
	}
	if len(allocations) > 0 {
		slotSets := make(map[uuid.UUID][]schedulingdomain.TimeSlot, len(allocations))
		for index := range allocations {
			resource, err := u.loadServiceResource(ctx, branch.ID, service.ID, allocations[index].ResourceID)
			if err != nil {
				return nil, err
			}
			allocations[index], err = ResolveAllocationUnits(allocations[index], resource.Capacity)
			if err != nil {
				return []schedulingdomain.TimeSlot{}, nil
			}
			allocations[index].ResourceName = resource.Name
			slots, err := u.listAvailableSlotsForResource(ctx, branch, service, resource, dayLocal, branchLoc)
			if err != nil {
				return nil, err
			}
			slotSets[resource.ID] = slots
		}
		return IntersectResourceSlots(allocations, slotSets), nil
	}

	resources, err := u.repo.ListServiceResources(ctx, branch.ID, service.ID, nil)
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return []schedulingdomain.TimeSlot{}, nil
	}
	out := make([]schedulingdomain.TimeSlot, 0)
	for _, resource := range resources {
		slots, err := u.listAvailableSlotsForResource(ctx, branch, service, resource, dayLocal, branchLoc)
		if err != nil {
			return nil, err
		}
		for index := range slots {
			slots[index].Allocations[0].Units = query.Participants
			slots[index].Remaining = min(
				RemainingReservations(resource.Capacity, slots[index].AllocatedUnits, query.Participants),
				slots[index].ServiceRemaining,
			)
			if slots[index].Remaining > 0 {
				out = append(out, slots[index])
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartAt.Equal(out[j].StartAt) {
			return out[i].ResourceName < out[j].ResourceName
		}
		return out[i].StartAt.Before(out[j].StartAt)
	})
	return out, nil
}

func (u *Usecases) loadServiceResource(
	ctx context.Context,
	branchID, serviceID, resourceID uuid.UUID,
) (schedulingdomain.Resource, error) {
	resource, err := u.repo.GetResource(ctx, resourceID)
	if err != nil {
		return schedulingdomain.Resource{}, mapRepoError(err, "resource", resourceID)
	}
	if resource.BranchID != branchID || !resource.Active {
		return schedulingdomain.Resource{}, domainerr.Validation("resource does not belong to the active branch")
	}
	eligible, err := u.repo.ListServiceResources(ctx, branchID, serviceID, &resourceID)
	if err != nil {
		return schedulingdomain.Resource{}, err
	}
	if len(eligible) != 1 || eligible[0].ID != resourceID {
		return schedulingdomain.Resource{}, domainerr.Validation("resource is not enabled for service")
	}
	return resource, nil
}

func (u *Usecases) listAvailableSlotsForResource(
	ctx context.Context,
	branch schedulingdomain.Branch,
	service schedulingdomain.Service,
	resource schedulingdomain.Resource,
	dayLocal time.Time,
	branchLoc *time.Location,
) ([]schedulingdomain.TimeSlot, error) {
	resourceLoc := branchLoc
	if strings.TrimSpace(resource.Timezone) != "" {
		if loc, err := time.LoadLocation(resource.Timezone); err == nil {
			resourceLoc = loc
		}
	}
	rules, err := u.repo.ListApplicableAvailabilityRules(ctx, branch.ID, &resource.ID, dayLocal)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return []schedulingdomain.TimeSlot{}, nil
	}
	windowStart := time.Date(dayLocal.Year(), dayLocal.Month(), dayLocal.Day(), 0, 0, 0, 0, resourceLoc)
	windowEnd := windowStart.Add(24 * time.Hour)
	blocked, err := u.repo.ListBlockedRangesBetween(ctx, branch.ID, &resource.ID, windowStart.UTC(), windowEnd.UTC())
	if err != nil {
		return nil, err
	}
	events, err := u.repo.ListCalendarEventsOccupyingResource(ctx, branch.ID, resource.ID, windowStart.UTC(), windowEnd.UTC())
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		blocked = append(blocked, schedulingdomain.BlockedRange{StartAt: event.StartAt, EndAt: event.EndAt})
	}
	candidates := generateSlotsForResource(resourceLoc, branch, resource, service, dayLocal.In(resourceLoc), rules, blocked)
	out := make([]schedulingdomain.TimeSlot, 0, len(candidates))
	for _, slot := range candidates {
		allocated, err := u.repo.AllocatedResourceUnits(ctx, resource.ID, slot.OccupiesFrom, slot.OccupiesUntil, nil)
		if err != nil {
			return nil, err
		}
		concurrent, err := u.repo.ConcurrentServiceBookings(ctx, service.ID, slot.OccupiesFrom, slot.OccupiesUntil, nil)
		if err != nil {
			return nil, err
		}
		resourceRemaining := max(resource.Capacity-allocated, 0)
		serviceRemaining := max(service.MaxConcurrentBookings-concurrent, 0)
		if resourceRemaining == 0 || serviceRemaining == 0 {
			continue
		}
		slot.Allocations = []schedulingdomain.ResourceAllocation{{
			ResourceID:   resource.ID,
			ResourceName: resource.Name,
			Mode:         schedulingdomain.ResourceAllocationModeCapacity,
			Units:        1,
		}}
		slot.Remaining = resourceRemaining
		slot.ServiceRemaining = serviceRemaining
		slot.AllocatedUnits = allocated
		out = append(out, slot)
	}
	return out, nil
}

func generateSlotsForResource(loc *time.Location, branch schedulingdomain.Branch, resource schedulingdomain.Resource, service schedulingdomain.Service, day time.Time, rules []schedulingdomain.AvailabilityRule, blocked []schedulingdomain.BlockedRange) []schedulingdomain.TimeSlot {
	branchWindows := make([]corescheduling.Window, 0)
	resourceWindows := make([]corescheduling.Window, 0)
	for _, rule := range rules {
		startClock, err := corescheduling.ParseClock(rule.StartTime)
		if err != nil {
			continue
		}
		endClock, err := corescheduling.ParseClock(rule.EndTime)
		if err != nil {
			continue
		}
		start := time.Date(day.Year(), day.Month(), day.Day(), startClock.Hour(), startClock.Minute(), 0, 0, loc)
		end := time.Date(day.Year(), day.Month(), day.Day(), endClock.Hour(), endClock.Minute(), 0, 0, loc)
		if !end.After(start) {
			continue
		}
		granularity := service.SlotGranularityMinutes
		if rule.SlotGranularityMinutes != nil && *rule.SlotGranularityMinutes > 0 {
			granularity = *rule.SlotGranularityMinutes
		}
		window := corescheduling.Window{Start: start, End: end, GranularityMinutes: granularity}
		if rule.Kind == schedulingdomain.AvailabilityRuleKindBranch {
			branchWindows = append(branchWindows, window)
		} else {
			resourceWindows = append(resourceWindows, window)
		}
	}
	activeWindows := corescheduling.IntersectWindows(branchWindows, resourceWindows)
	blockedRanges := make([]corescheduling.BlockedRange, 0, len(blocked))
	for _, block := range blocked {
		blockedRanges = append(blockedRanges, corescheduling.BlockedRange{
			StartAt: block.StartAt.UTC(),
			EndAt:   block.EndAt.UTC(),
		})
	}
	timezone := branch.Timezone
	if strings.TrimSpace(resource.Timezone) != "" {
		timezone = resource.Timezone
	}

	coreSlots := corescheduling.GenerateSlots(activeWindows, blockedRanges, corescheduling.SlotSpec{
		DurationMinutes:           service.DefaultDurationMinutes,
		BufferBeforeMinutes:       service.BufferBeforeMinutes,
		BufferAfterMinutes:        service.BufferAfterMinutes,
		DefaultGranularityMinutes: service.SlotGranularityMinutes,
	})

	slots := make([]schedulingdomain.TimeSlot, 0, len(coreSlots))
	for _, slot := range coreSlots {
		slots = append(slots, schedulingdomain.TimeSlot{
			ResourceID:       resource.ID,
			ResourceName:     resource.Name,
			StartAt:          slot.StartAt.UTC(),
			EndAt:            slot.EndAt.UTC(),
			OccupiesFrom:     slot.OccupiesFrom.UTC(),
			OccupiesUntil:    slot.OccupiesUntil.UTC(),
			Timezone:         timezone,
			Remaining:        resource.Capacity,
			ServiceRemaining: service.MaxConcurrentBookings,
			AllocatedUnits:   0,
			GranularityMin:   slot.GranularityMinutes,
		})
	}
	return slots
}

// slotStartMatchesRequested tolera deriva sub-segundo y desalineación de milisegundos entre el cliente y los slots generados en el servidor.
func slotStartMatchesRequested(slotStart, requested time.Time) bool {
	a := slotStart.UTC()
	b := requested.UTC()
	if a.Equal(b) {
		return true
	}
	return a.Truncate(time.Minute).Equal(b.Truncate(time.Minute))
}

func hasRequestedResources(resourceID *uuid.UUID, resources []schedulingdomain.ResourceAllocation) bool {
	return len(resources) > 0 || (resourceID != nil && *resourceID != uuid.Nil)
}

func filterSlotsByStart(
	slots []schedulingdomain.TimeSlot,
	startAt time.Time,
	resourceID *uuid.UUID,
	resources []schedulingdomain.ResourceAllocation,
	participants int,
) []schedulingdomain.TimeSlot {
	out := make([]schedulingdomain.TimeSlot, 0)
	requested, _ := NormalizeResourceAllocationsForParticipants(resourceID, resources, participants)
	for _, slot := range slots {
		if !slotStartMatchesRequested(slot.StartAt, startAt) {
			continue
		}
		if len(requested) > 0 && !slotContainsAllocations(slot, requested) {
			continue
		}
		out = append(out, slot)
	}
	return out
}

func slotContainsAllocations(slot schedulingdomain.TimeSlot, requested []schedulingdomain.ResourceAllocation) bool {
	if len(slot.Allocations) != len(requested) {
		return false
	}
	for _, requirement := range requested {
		found := false
		for _, allocation := range slot.Allocations {
			if allocation.ResourceID == requirement.ResourceID &&
				allocation.Mode == requirement.Mode &&
				(requirement.Mode == schedulingdomain.ResourceAllocationModeExclusive ||
					allocation.Units == requirement.Units) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (u *Usecases) transitionBooking(ctx context.Context, bookingID uuid.UUID, actor string, target schedulingdomain.BookingStatus, reason string) (schedulingdomain.Booking, error) {
	current, err := u.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", bookingID)
	}
	if !canTransitionBooking(current.Status, target) {
		return schedulingdomain.Booking{}, domainerr.Conflict("booking cannot transition to " + string(target))
	}
	var confirmedAt *time.Time
	var cancelledAt *time.Time
	now := time.Now().UTC()
	switch target {
	case schedulingdomain.BookingStatusConfirmed:
		confirmedAt = &now
	case schedulingdomain.BookingStatusCancelled:
		cancelledAt = &now
	}
	out, err := u.repo.UpdateBookingStatus(ctx, bookingID, target, confirmedAt, cancelledAt, strings.TrimSpace(reason))
	if err != nil {
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", bookingID)
	}
	u.logAudit(ctx, actor, "scheduling.booking."+string(target), "scheduling_booking", out.ID.String(), map[string]any{"reason": strings.TrimSpace(reason)})
	u.emitEvent(ctx, "scheduling.booking."+string(target), map[string]any{"booking_id": out.ID.String()})
	return out, nil
}

func (u *Usecases) cancelBooking(ctx context.Context, bookingID uuid.UUID, actor, reason string, enforcePolicy bool) (schedulingdomain.Booking, error) {
	current, err := u.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", bookingID)
	}
	if !canTransitionBooking(current.Status, schedulingdomain.BookingStatusCancelled) {
		return schedulingdomain.Booking{}, domainerr.Conflict("booking cannot transition to cancelled")
	}
	if enforcePolicy {
		service, err := u.repo.GetService(ctx, current.ServiceID)
		if err != nil {
			return schedulingdomain.Booking{}, mapRepoError(err, "service", current.ServiceID)
		}
		if service.MinCancelNoticeMinutes > 0 && time.Until(current.StartAt) < time.Duration(service.MinCancelNoticeMinutes)*time.Minute {
			return schedulingdomain.Booking{}, domainerr.Conflict("cancellation window closed")
		}
	}
	now := time.Now().UTC()
	out, err := u.repo.UpdateBookingStatus(ctx, bookingID, schedulingdomain.BookingStatusCancelled, nil, &now, strings.TrimSpace(reason))
	if err != nil {
		return schedulingdomain.Booking{}, mapRepoError(err, "booking", bookingID)
	}
	u.logAudit(ctx, actor, "scheduling.booking.cancelled", "scheduling_booking", out.ID.String(), map[string]any{"reason": strings.TrimSpace(reason)})
	u.emitEvent(ctx, "scheduling.booking.cancelled", map[string]any{"booking_id": out.ID.String()})
	return out, nil
}

func (u *Usecases) transitionQueue(ctx context.Context, queueID uuid.UUID, actor string, target schedulingdomain.QueueStatus) (schedulingdomain.Queue, error) {
	current, err := u.repo.GetQueueByID(ctx, queueID)
	if err != nil {
		return schedulingdomain.Queue{}, mapRepoError(err, "queue", queueID)
	}
	if current.Status == target {
		return current, nil
	}
	switch current.Status {
	case schedulingdomain.QueueStatusActive:
		if target != schedulingdomain.QueueStatusPaused && target != schedulingdomain.QueueStatusClosed {
			return schedulingdomain.Queue{}, domainerr.Conflict("queue cannot transition to " + string(target))
		}
	case schedulingdomain.QueueStatusPaused:
		if target != schedulingdomain.QueueStatusActive && target != schedulingdomain.QueueStatusClosed {
			return schedulingdomain.Queue{}, domainerr.Conflict("queue cannot transition to " + string(target))
		}
	case schedulingdomain.QueueStatusClosed:
		return schedulingdomain.Queue{}, domainerr.Conflict("queue cannot transition from closed")
	}
	out, err := u.repo.UpdateQueueStatus(ctx, queueID, target)
	if err != nil {
		return schedulingdomain.Queue{}, mapRepoError(err, "queue", queueID)
	}
	u.logAudit(ctx, actor, "scheduling.queue."+string(target), "scheduling_queue", out.ID.String(), nil)
	return out, nil
}

func (u *Usecases) bookingSupportsAction(status schedulingdomain.BookingStatus, action schedulingdomain.BookingActionType) bool {
	switch action {
	case schedulingdomain.BookingActionConfirm:
		return canTransitionBooking(status, schedulingdomain.BookingStatusConfirmed)
	case schedulingdomain.BookingActionCancel:
		return canTransitionBooking(status, schedulingdomain.BookingStatusCancelled)
	default:
		return false
	}
}

func (u *Usecases) lookupBookingActionToken(ctx context.Context, raw string) (schedulingdomain.BookingActionToken, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return schedulingdomain.BookingActionToken{}, domainerr.Validation("token is required")
	}
	sum := sha256.Sum256([]byte(raw))
	token, err := u.repo.GetBookingActionTokenByHash(ctx, hex.EncodeToString(sum[:]))
	if err != nil {
		return schedulingdomain.BookingActionToken{}, mapRepoError(err, "booking_action_token", uuid.Nil)
	}
	if token.UsedAt != nil {
		return schedulingdomain.BookingActionToken{}, domainerr.Conflict("token already used")
	}
	if token.VoidedAt != nil {
		return schedulingdomain.BookingActionToken{}, domainerr.Conflict("token no longer valid")
	}
	if token.ExpiresAt.Before(time.Now().UTC()) {
		return schedulingdomain.BookingActionToken{}, domainerr.Conflict("token expired")
	}
	return token, nil
}

func newActionToken() (string, string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

func minTimePtr(a, b *time.Time) *time.Time {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if a.Before(*b) {
		return a
	}
	return b
}

func mapRepoError(err error, resource string, id uuid.UUID) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		if id != uuid.Nil {
			return domainerr.NotFoundf(resource, id.String())
		}
		return domainerr.NotFound(resource)
	}
	if errors.Is(err, ErrCapacityExceeded) {
		return domainerr.Conflict("slot not available")
	}
	if errors.Is(err, ErrAlreadyExists) {
		return domainerr.Conflict(resource + " already exists")
	}
	return err
}

func mapQueueError(err error, queueID uuid.UUID) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return domainerr.NotFoundf("queue", queueID.String())
	}
	if errors.Is(err, ErrQueueInactive) {
		return domainerr.Conflict("queue is not active")
	}
	if errors.Is(err, ErrRemoteJoinDisabled) {
		return domainerr.Conflict("queue does not allow remote join")
	}
	return err
}

func isBookingOverlapErr(err error) bool {
	return errors.Is(err, ErrCapacityExceeded)
}

func defaultBookingStatus(in schedulingdomain.CreateBookingInput) schedulingdomain.BookingStatus {
	if in.HoldUntil != nil && !in.HoldUntil.IsZero() {
		return schedulingdomain.BookingStatusHold
	}
	switch normalizeBookingSource(in.Source) {
	case schedulingdomain.BookingSourcePublicWeb, schedulingdomain.BookingSourceWhatsApp:
		return schedulingdomain.BookingStatusPendingConfirmation
	default:
		return schedulingdomain.BookingStatusConfirmed
	}
}

func normalizeParticipants(participants int) (int, error) {
	if participants == 0 {
		return 1, nil
	}
	if participants < 0 {
		return 0, domainerr.Validation("participants must be positive")
	}
	if participants > maxAllocationUnits {
		return 0, domainerr.Validation("participants must be <= 100000")
	}
	return participants, nil
}

func canTransitionBooking(from, to schedulingdomain.BookingStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case schedulingdomain.BookingStatusHold:
		return to == schedulingdomain.BookingStatusPendingConfirmation || to == schedulingdomain.BookingStatusConfirmed || to == schedulingdomain.BookingStatusExpired || to == schedulingdomain.BookingStatusCancelled
	case schedulingdomain.BookingStatusPendingConfirmation:
		return to == schedulingdomain.BookingStatusConfirmed || to == schedulingdomain.BookingStatusExpired || to == schedulingdomain.BookingStatusCancelled
	case schedulingdomain.BookingStatusConfirmed:
		return to == schedulingdomain.BookingStatusCheckedIn || to == schedulingdomain.BookingStatusInService || to == schedulingdomain.BookingStatusCompleted || to == schedulingdomain.BookingStatusCancelled || to == schedulingdomain.BookingStatusNoShow
	case schedulingdomain.BookingStatusCheckedIn:
		return to == schedulingdomain.BookingStatusInService || to == schedulingdomain.BookingStatusCompleted || to == schedulingdomain.BookingStatusCancelled || to == schedulingdomain.BookingStatusNoShow
	case schedulingdomain.BookingStatusInService:
		return to == schedulingdomain.BookingStatusCompleted || to == schedulingdomain.BookingStatusCancelled
	default:
		return false
	}
}

func canRescheduleBooking(status schedulingdomain.BookingStatus) bool {
	switch status {
	case schedulingdomain.BookingStatusHold, schedulingdomain.BookingStatusPendingConfirmation, schedulingdomain.BookingStatusConfirmed, schedulingdomain.BookingStatusCheckedIn:
		return true
	default:
		return false
	}
}

func canTransitionQueueTicket(from, to schedulingdomain.QueueTicketStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case schedulingdomain.QueueTicketStatusWaiting:
		return to == schedulingdomain.QueueTicketStatusCalled || to == schedulingdomain.QueueTicketStatusServing || to == schedulingdomain.QueueTicketStatusCancelled || to == schedulingdomain.QueueTicketStatusNoShow
	case schedulingdomain.QueueTicketStatusCalled:
		return to == schedulingdomain.QueueTicketStatusServing || to == schedulingdomain.QueueTicketStatusCancelled || to == schedulingdomain.QueueTicketStatusNoShow
	case schedulingdomain.QueueTicketStatusServing:
		return to == schedulingdomain.QueueTicketStatusCompleted || to == schedulingdomain.QueueTicketStatusCancelled || to == schedulingdomain.QueueTicketStatusNoShow
	default:
		return false
	}
}

func normalizeCode(v string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(v), " ", "_"))
}

func normalizeFulfillmentMode(v schedulingdomain.FulfillmentMode) schedulingdomain.FulfillmentMode {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.FulfillmentModeSchedule):
		return schedulingdomain.FulfillmentModeSchedule
	case string(schedulingdomain.FulfillmentModeQueue):
		return schedulingdomain.FulfillmentModeQueue
	case string(schedulingdomain.FulfillmentModeHybrid):
		return schedulingdomain.FulfillmentModeHybrid
	default:
		return ""
	}
}

func normalizeResourceKind(v schedulingdomain.ResourceKind) schedulingdomain.ResourceKind {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.ResourceKindProfessional):
		return schedulingdomain.ResourceKindProfessional
	case string(schedulingdomain.ResourceKindDesk):
		return schedulingdomain.ResourceKindDesk
	case string(schedulingdomain.ResourceKindCounter):
		return schedulingdomain.ResourceKindCounter
	case string(schedulingdomain.ResourceKindBox):
		return schedulingdomain.ResourceKindBox
	case string(schedulingdomain.ResourceKindRoom):
		return schedulingdomain.ResourceKindRoom
	case string(schedulingdomain.ResourceKindGeneric):
		return schedulingdomain.ResourceKindGeneric
	default:
		return ""
	}
}

func normalizeAvailabilityRuleKind(v schedulingdomain.AvailabilityRuleKind) schedulingdomain.AvailabilityRuleKind {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.AvailabilityRuleKindBranch):
		return schedulingdomain.AvailabilityRuleKindBranch
	case string(schedulingdomain.AvailabilityRuleKindResource):
		return schedulingdomain.AvailabilityRuleKindResource
	default:
		return ""
	}
}

func normalizeBlockedRangeKind(v schedulingdomain.BlockedRangeKind) schedulingdomain.BlockedRangeKind {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.BlockedRangeKindHoliday):
		return schedulingdomain.BlockedRangeKindHoliday
	case string(schedulingdomain.BlockedRangeKindManual):
		return schedulingdomain.BlockedRangeKindManual
	case string(schedulingdomain.BlockedRangeKindMaintenance):
		return schedulingdomain.BlockedRangeKindMaintenance
	case string(schedulingdomain.BlockedRangeKindLeave):
		return schedulingdomain.BlockedRangeKindLeave
	default:
		return ""
	}
}

func normalizeBookingStatus(v schedulingdomain.BookingStatus) schedulingdomain.BookingStatus {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.BookingStatusHold):
		return schedulingdomain.BookingStatusHold
	case string(schedulingdomain.BookingStatusPendingConfirmation):
		return schedulingdomain.BookingStatusPendingConfirmation
	case string(schedulingdomain.BookingStatusConfirmed):
		return schedulingdomain.BookingStatusConfirmed
	case string(schedulingdomain.BookingStatusCheckedIn):
		return schedulingdomain.BookingStatusCheckedIn
	case string(schedulingdomain.BookingStatusInService):
		return schedulingdomain.BookingStatusInService
	case string(schedulingdomain.BookingStatusCompleted):
		return schedulingdomain.BookingStatusCompleted
	case string(schedulingdomain.BookingStatusCancelled):
		return schedulingdomain.BookingStatusCancelled
	case string(schedulingdomain.BookingStatusNoShow):
		return schedulingdomain.BookingStatusNoShow
	case string(schedulingdomain.BookingStatusExpired):
		return schedulingdomain.BookingStatusExpired
	default:
		return ""
	}
}

func normalizeBookingSource(v schedulingdomain.BookingSource) schedulingdomain.BookingSource {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.BookingSourceAdmin):
		return schedulingdomain.BookingSourceAdmin
	case string(schedulingdomain.BookingSourcePublicWeb):
		return schedulingdomain.BookingSourcePublicWeb
	case string(schedulingdomain.BookingSourceWhatsApp):
		return schedulingdomain.BookingSourceWhatsApp
	case string(schedulingdomain.BookingSourceAPI):
		return schedulingdomain.BookingSourceAPI
	default:
		return ""
	}
}

func normalizeQueueStatus(v schedulingdomain.QueueStatus) schedulingdomain.QueueStatus {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.QueueStatusActive):
		return schedulingdomain.QueueStatusActive
	case string(schedulingdomain.QueueStatusPaused):
		return schedulingdomain.QueueStatusPaused
	case string(schedulingdomain.QueueStatusClosed):
		return schedulingdomain.QueueStatusClosed
	default:
		return ""
	}
}

func normalizeQueueStrategy(v schedulingdomain.QueueStrategy) schedulingdomain.QueueStrategy {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.QueueStrategyFIFO):
		return schedulingdomain.QueueStrategyFIFO
	case string(schedulingdomain.QueueStrategyPriority):
		return schedulingdomain.QueueStrategyPriority
	default:
		return ""
	}
}

func normalizeQueueTicketSource(v schedulingdomain.QueueTicketSource) schedulingdomain.QueueTicketSource {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.QueueTicketSourceReception):
		return schedulingdomain.QueueTicketSourceReception
	case string(schedulingdomain.QueueTicketSourceWeb):
		return schedulingdomain.QueueTicketSourceWeb
	case string(schedulingdomain.QueueTicketSourceWhatsApp):
		return schedulingdomain.QueueTicketSourceWhatsApp
	case string(schedulingdomain.QueueTicketSourceAPI):
		return schedulingdomain.QueueTicketSourceAPI
	default:
		return ""
	}
}

func normalizeWaitlistSource(v schedulingdomain.WaitlistSource) schedulingdomain.WaitlistSource {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(schedulingdomain.WaitlistSourceAdmin):
		return schedulingdomain.WaitlistSourceAdmin
	case string(schedulingdomain.WaitlistSourcePublicWeb):
		return schedulingdomain.WaitlistSourcePublicWeb
	case string(schedulingdomain.WaitlistSourceWhatsApp):
		return schedulingdomain.WaitlistSourceWhatsApp
	case string(schedulingdomain.WaitlistSourceAPI):
		return schedulingdomain.WaitlistSourceAPI
	default:
		return ""
	}
}

func defaultWaitlistSource(actor string) schedulingdomain.WaitlistSource {
	if strings.Contains(strings.ToLower(strings.TrimSpace(actor)), "public") {
		return schedulingdomain.WaitlistSourcePublicWeb
	}
	return schedulingdomain.WaitlistSourceAdmin
}

func ensureUUID(v uuid.UUID) uuid.UUID {
	if v == uuid.Nil {
		return uuid.New()
	}
	return v
}

func buildBookingReference(startAt time.Time, serviceCode string) string {
	serviceCode = strings.ToUpper(strings.TrimSpace(serviceCode))
	if serviceCode == "" {
		serviceCode = "BK"
	}
	if len(serviceCode) > 4 {
		serviceCode = serviceCode[:4]
	}
	return fmt.Sprintf("%s-%s-%s", serviceCode, startAt.UTC().Format("20060102"), strings.ToUpper(uuid.NewString()[:6]))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (u *Usecases) logAudit(ctx context.Context, actor, action, resourceType, resourceID string, payload map[string]any) {
	if u.audit == nil {
		return
	}
	u.audit.Log(ctx, actor, action, resourceType, resourceID, payload)
}

func (u *Usecases) emitEvent(ctx context.Context, eventType string, payload map[string]any) {
	if u.notifications == nil {
		return
	}
	_ = u.notifications.Enqueue(ctx, eventType, payload)
}

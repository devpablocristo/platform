package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/devpablocristo/core/errors/go/domainerr"
	"github.com/devpablocristo/core/saas/go/notifications"
	"github.com/stripe/stripe-go/v81"

	admindomain "github.com/devpablocristo/core/saas/go/admin/usecases/domain"
	billingdomain "github.com/devpablocristo/core/saas/go/billing/usecases/domain"
)

const (
	maxConcurrentBillingNotifications = 64
	defaultConsoleBaseURL             = "http://localhost:5173"
)

type RuntimeConfig struct {
	StripeSecretKey       string
	StripeWebhookSecret   string
	StripePriceStarter    string
	StripePriceGrowth     string
	StripePriceEnterprise string
	ConsoleBaseURL        string
}

type RuntimeMetricsPort interface {
	IncBillingCheckouts(plan string)
	IncWebhooksReceived(provider, eventType string)
}

type RuntimeTenantSettingsPort interface {
	UpsertTenantSettings(context.Context, admindomain.TenantSettings) (admindomain.TenantSettings, error)
}

type RuntimeRepository interface {
	GetTenantBilling(context.Context, string) (billingdomain.TenantBilling, bool, error)
	UpsertTenantBilling(context.Context, billingdomain.TenantBilling) (billingdomain.TenantBilling, error)
	GetUsageSummary(context.Context, string) (billingdomain.UsageSummary, error)
	GetTenantName(context.Context, string) (string, error)
	FindTenantIDByCustomerID(context.Context, string) (string, bool, error)
	FindTenantIDByContractID(context.Context, string) (string, bool, error)
	FindUserEmailByExternalID(context.Context, string) (string, bool, error)
	FindPastDueBefore(context.Context, time.Time) ([]billingdomain.TenantBilling, error)
}

type Runtime struct {
	repo           RuntimeRepository
	tenantSettings RuntimeTenantSettingsPort
	stripe         StripeClientPort
	notifications  notifications.NotificationPort
	metrics        RuntimeMetricsPort
	logger         *slog.Logger
	notifSem       chan struct{}

	stripeEnabled   bool
	webhookSecret   string
	priceStarter    string
	priceGrowth     string
	priceEnterprise string
	consoleBaseURL  string
}

func NewRuntime(cfg RuntimeConfig, repo RuntimeRepository, tenantSettings RuntimeTenantSettingsPort, notif notifications.NotificationPort, metrics RuntimeMetricsPort, logger *slog.Logger) *Runtime {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runtime{
		repo:            repo,
		tenantSettings:  tenantSettings,
		stripe:          NewStripeClient(cfg.StripeSecretKey),
		notifications:   notif,
		metrics:         metrics,
		logger:          logger,
		notifSem:        make(chan struct{}, maxConcurrentBillingNotifications),
		stripeEnabled:   strings.TrimSpace(cfg.StripeSecretKey) != "",
		webhookSecret:   strings.TrimSpace(cfg.StripeWebhookSecret),
		priceStarter:    strings.TrimSpace(cfg.StripePriceStarter),
		priceGrowth:     strings.TrimSpace(cfg.StripePriceGrowth),
		priceEnterprise: strings.TrimSpace(cfg.StripePriceEnterprise),
		consoleBaseURL:  sanitizeConsoleBaseURL(cfg.ConsoleBaseURL),
	}
}

func (r *Runtime) Enabled() bool {
	return r != nil && r.stripeEnabled
}

func (r *Runtime) WebhookSecret() string {
	if r == nil {
		return ""
	}
	return r.webhookSecret
}

func (r *Runtime) ConstructAndVerifyWebhook(payload []byte, sigHeader, secret string) (stripe.Event, error) {
	return r.stripe.ConstructWebhookEvent(payload, sigHeader, secret)
}

func (r *Runtime) RecordWebhookReceived(provider, eventType string) {
	if r.metrics != nil {
		r.metrics.IncWebhooksReceived(provider, eventType)
	}
}

func (r *Runtime) GetBillingStatus(ctx context.Context, tenantID string) (billingdomain.BillingStatusView, error) {
	if err := r.requireStripeEnabled(); err != nil {
		return billingdomain.BillingStatusView{}, err
	}
	settings, err := r.ensureTenantBilling(ctx, tenantID)
	if err != nil {
		return billingdomain.BillingStatusView{}, err
	}
	usage, err := r.repo.GetUsageSummary(ctx, strings.TrimSpace(tenantID))
	if err != nil {
		return billingdomain.BillingStatusView{}, err
	}

	var currentPeriodEnd *time.Time
	if settings.ProviderContractID != nil && *settings.ProviderContractID != "" {
		sub, err := r.stripe.GetSubscription(*settings.ProviderContractID)
		if err == nil && sub != nil && sub.CurrentPeriodEnd > 0 {
			value := time.Unix(sub.CurrentPeriodEnd, 0).UTC()
			currentPeriodEnd = &value
		}
	}

	return billingdomain.BillingStatusView{
		PlanCode:         settings.PlanCode,
		BillingStatus:    settings.BillingStatus,
		CurrentPeriodEnd: currentPeriodEnd,
		HardLimits:       settings.HardLimits,
		Usage:            usage,
	}, nil
}

func (r *Runtime) CreateCheckoutSession(ctx context.Context, input billingdomain.CheckoutInput) (string, error) {
	if err := r.requireStripeEnabled(); err != nil {
		return "", err
	}

	plan := normalizePlan(input.PlanCode)
	if plan == "" {
		return "", domainerr.Validation("plan_code must be starter|growth|enterprise")
	}
	priceID := r.priceIDByPlan(plan)
	if priceID == "" {
		return "", domainerr.Internal("stripe price not configured for plan")
	}

	successURL := strings.TrimSpace(input.SuccessURL)
	cancelURL := strings.TrimSpace(input.CancelURL)
	if successURL == "" {
		successURL = r.consoleBaseURL + "/billing/success?plan=" + string(plan) + "&session_id={CHECKOUT_SESSION_ID}"
	}
	if cancelURL == "" {
		cancelURL = r.consoleBaseURL + "/billing?canceled=1"
	}
	if _, err := url.ParseRequestURI(successURL); err != nil {
		return "", domainerr.Validation("invalid success_url")
	}
	if _, err := url.ParseRequestURI(cancelURL); err != nil {
		return "", domainerr.Validation("invalid cancel_url")
	}

	settings, err := r.ensureTenantBilling(ctx, input.TenantID)
	if err != nil {
		return "", err
	}
	customerID, err := r.ensureStripeCustomer(ctx, settings, input.Actor)
	if err != nil {
		return "", err
	}

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Customer:   stripe.String(customerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: map[string]string{
			"tenant_id": strings.TrimSpace(input.TenantID),
			"plan_code": string(plan),
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"tenant_id": strings.TrimSpace(input.TenantID),
				"plan_code": string(plan),
			},
		},
	}
	if input.CustomerEmail != nil && strings.TrimSpace(*input.CustomerEmail) != "" {
		params.CustomerEmail = stripe.String(strings.TrimSpace(*input.CustomerEmail))
	}

	session, err := r.stripe.CreateCheckoutSession(params)
	if err != nil {
		return "", err
	}
	if session == nil || strings.TrimSpace(session.URL) == "" {
		return "", domainerr.Internal("stripe checkout session missing url")
	}
	if r.metrics != nil {
		r.metrics.IncBillingCheckouts(string(plan))
	}
	return session.URL, nil
}

func (r *Runtime) CreatePortalSession(ctx context.Context, input billingdomain.PortalInput) (string, error) {
	if err := r.requireStripeEnabled(); err != nil {
		return "", err
	}

	returnURL := strings.TrimSpace(input.ReturnURL)
	if returnURL == "" {
		returnURL = r.consoleBaseURL + "/billing"
	}
	if _, err := url.ParseRequestURI(returnURL); err != nil {
		return "", domainerr.Validation("invalid return_url")
	}

	settings, err := r.ensureTenantBilling(ctx, input.TenantID)
	if err != nil {
		return "", err
	}
	customerID, err := r.ensureStripeCustomer(ctx, settings, input.Actor)
	if err != nil {
		return "", err
	}

	session, err := r.stripe.CreatePortalSession(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	})
	if err != nil {
		return "", err
	}
	if session == nil || strings.TrimSpace(session.URL) == "" {
		return "", domainerr.Internal("stripe portal session missing url")
	}
	return session.URL, nil
}

func (r *Runtime) HandleWebhookEvent(ctx context.Context, event stripe.Event) error {
	if !r.Enabled() {
		return domainerr.Internal("stripe billing is not configured")
	}
	if event.Data == nil {
		return domainerr.Validation("stripe webhook event missing data")
	}

	switch event.Type {
	case "checkout.session.completed":
		return r.handleCheckoutCompleted(ctx, event.Data.Raw)
	case "customer.subscription.updated":
		return r.handleSubscriptionUpdated(ctx, event.Data.Raw)
	case "customer.subscription.deleted":
		return r.handleSubscriptionDeleted(ctx, event.Data.Raw)
	case "invoice.payment_succeeded":
		return r.handleInvoicePayment(ctx, event.Data.Raw, billingdomain.BillingActive)
	case "invoice.payment_failed":
		return r.handleInvoicePayment(ctx, event.Data.Raw, billingdomain.BillingPastDue)
	default:
		return nil
	}
}

type stripeCheckoutPayload struct {
	Customer     string            `json:"customer"`
	Subscription string            `json:"subscription"`
	Metadata     map[string]string `json:"metadata"`
}

type stripeSubscriptionPayload struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
	Status   string `json:"status"`
	Items    struct {
		Data []struct {
			Price struct {
				ID string `json:"id"`
			} `json:"price"`
		} `json:"data"`
	} `json:"items"`
}

type stripeSubscriptionDeletedPayload struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
}

type stripeInvoicePayload struct {
	Subscription string `json:"subscription"`
	Customer     string `json:"customer"`
}

func (r *Runtime) handleCheckoutCompleted(ctx context.Context, raw json.RawMessage) error {
	var payload stripeCheckoutPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	tenantID := strings.TrimSpace(payload.Metadata["tenant_id"])
	if tenantID == "" {
		tenantID = strings.TrimSpace(payload.Metadata["org_id"])
	}
	if tenantID == "" {
		return domainerr.Validation("checkout metadata missing tenant_id")
	}
	plan := normalizePlan(billingdomain.PlanCode(payload.Metadata["plan_code"]))
	if plan == "" {
		plan = billingdomain.PlanStarter
	}
	if err := r.applySubscriptionState(ctx, tenantID, plan, billingdomain.BillingActive, payload.Customer, payload.Subscription); err != nil {
		return err
	}
	tenantName, _ := r.repo.GetTenantName(ctx, tenantID)
	r.notifyAsync(tenantID, "plan_upgraded", map[string]string{
		"tenant_name":     tenantName,
		"plan_code":       string(plan),
		"action_url":      r.consoleBaseURL + "/billing",
		"preferences_url": r.consoleBaseURL + "/settings/notifications",
		"reference_id":    payload.Subscription,
		"subscription_id": payload.Subscription,
	})
	return nil
}

func (r *Runtime) handleSubscriptionUpdated(ctx context.Context, raw json.RawMessage) error {
	var payload stripeSubscriptionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	tenantID, ok, err := r.repo.FindTenantIDByContractID(ctx, payload.ID)
	if err != nil {
		return err
	}
	if !ok {
		tenantID, ok, err = r.repo.FindTenantIDByCustomerID(ctx, payload.Customer)
		if err != nil {
			return err
		}
	}
	if !ok {
		return nil
	}

	plan := billingdomain.PlanStarter
	if len(payload.Items.Data) > 0 {
		plan = r.planByPriceID(payload.Items.Data[0].Price.ID)
	}
	status := billingStatusFromStripe(payload.Status)
	return r.applySubscriptionState(ctx, tenantID, plan, status, payload.Customer, payload.ID)
}

func (r *Runtime) handleSubscriptionDeleted(ctx context.Context, raw json.RawMessage) error {
	var payload stripeSubscriptionDeletedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	tenantID, ok, err := r.repo.FindTenantIDByContractID(ctx, payload.ID)
	if err != nil {
		return err
	}
	if !ok {
		tenantID, ok, err = r.repo.FindTenantIDByCustomerID(ctx, payload.Customer)
		if err != nil {
			return err
		}
	}
	if !ok {
		return nil
	}

	settings, err := r.ensureTenantBilling(ctx, tenantID)
	if err != nil {
		return err
	}
	settings.PlanCode = billingdomain.PlanStarter
	settings.HardLimits = DefaultHardLimits(billingdomain.PlanStarter)
	settings.BillingStatus = billingdomain.BillingCanceled
	settings.ProviderContractID = nil
	settings.UpdatedAt = time.Now().UTC()
	if _, err := r.repo.UpsertTenantBilling(ctx, settings); err != nil {
		return err
	}
	if err := r.applyPlanSettings(ctx, tenantID, billingdomain.PlanStarter); err != nil {
		return err
	}

	tenantName, _ := r.repo.GetTenantName(ctx, tenantID)
	r.notifyAsync(tenantID, "subscription_canceled", map[string]string{
		"tenant_name":     tenantName,
		"plan_code":       string(billingdomain.PlanStarter),
		"action_url":      r.consoleBaseURL + "/billing",
		"preferences_url": r.consoleBaseURL + "/settings/notifications",
		"reference_id":    payload.ID,
		"subscription_id": payload.ID,
	})
	return nil
}

func (r *Runtime) handleInvoicePayment(ctx context.Context, raw json.RawMessage, status billingdomain.BillingStatus) error {
	var payload stripeInvoicePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	tenantID, ok, err := r.repo.FindTenantIDByContractID(ctx, payload.Subscription)
	if err != nil {
		return err
	}
	if !ok {
		tenantID, ok, err = r.repo.FindTenantIDByCustomerID(ctx, payload.Customer)
		if err != nil {
			return err
		}
	}
	if !ok {
		return nil
	}

	settings, err := r.ensureTenantBilling(ctx, tenantID)
	if err != nil {
		return err
	}
	settings.BillingStatus = status
	if status == billingdomain.BillingPastDue {
		now := time.Now().UTC()
		settings.PastDueSince = &now
	} else {
		settings.PastDueSince = nil
	}
	settings.UpdatedAt = time.Now().UTC()
	if _, err := r.repo.UpsertTenantBilling(ctx, settings); err != nil {
		return err
	}

	if status == billingdomain.BillingPastDue {
		tenantName, _ := r.repo.GetTenantName(ctx, tenantID)
		r.notifyAsync(tenantID, "payment_failed", map[string]string{
			"tenant_name":     tenantName,
			"action_url":      r.consoleBaseURL + "/billing",
			"preferences_url": r.consoleBaseURL + "/settings/notifications",
			"reference_id":    payload.Subscription,
			"subscription_id": payload.Subscription,
		})
	}
	return nil
}

func (r *Runtime) ensureTenantBilling(ctx context.Context, tenantID string) (billingdomain.TenantBilling, error) {
	item, ok, err := r.repo.GetTenantBilling(ctx, strings.TrimSpace(tenantID))
	if err != nil {
		return billingdomain.TenantBilling{}, err
	}
	if ok {
		return item, nil
	}
	now := time.Now().UTC()
	item = billingdomain.TenantBilling{
		TenantID:      strings.TrimSpace(tenantID),
		PlanCode:      billingdomain.PlanStarter,
		HardLimits:    DefaultHardLimits(billingdomain.PlanStarter),
		BillingStatus: billingdomain.BillingTrialing,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := r.repo.UpsertTenantBilling(ctx, item); err != nil {
		return billingdomain.TenantBilling{}, err
	}
	if err := r.applyPlanSettings(ctx, tenantID, billingdomain.PlanStarter); err != nil {
		return billingdomain.TenantBilling{}, err
	}
	return item, nil
}

func (r *Runtime) ensureStripeCustomer(ctx context.Context, settings billingdomain.TenantBilling, actor *string) (string, error) {
	if settings.ProviderCustomerID != nil && strings.TrimSpace(*settings.ProviderCustomerID) != "" {
		return strings.TrimSpace(*settings.ProviderCustomerID), nil
	}
	tenantName, err := r.repo.GetTenantName(ctx, settings.TenantID)
	if err != nil {
		return "", err
	}

	var email *string
	if actor != nil && strings.TrimSpace(*actor) != "" {
		resolved, ok, err := r.repo.FindUserEmailByExternalID(ctx, strings.TrimSpace(*actor))
		if err != nil {
			return "", err
		}
		if ok {
			email = &resolved
		}
	}

	params := &stripe.CustomerParams{
		Name: stripe.String(tenantName),
		Metadata: map[string]string{
			"tenant_id": settings.TenantID,
		},
	}
	if email != nil {
		params.Email = stripe.String(*email)
	}

	customer, err := r.stripe.CreateCustomer(params)
	if err != nil {
		return "", err
	}
	if customer == nil || strings.TrimSpace(customer.ID) == "" {
		return "", domainerr.Internal("stripe customer missing id")
	}

	customerID := strings.TrimSpace(customer.ID)
	settings.ProviderCustomerID = &customerID
	settings.UpdatedAt = time.Now().UTC()
	if _, err := r.repo.UpsertTenantBilling(ctx, settings); err != nil {
		return "", err
	}
	return customerID, nil
}

func (r *Runtime) applySubscriptionState(ctx context.Context, tenantID string, plan billingdomain.PlanCode, status billingdomain.BillingStatus, customerID, contractID string) error {
	settings, err := r.ensureTenantBilling(ctx, tenantID)
	if err != nil {
		return err
	}
	if settings.PlanCode != plan {
		settings.PlanCode = plan
		settings.HardLimits = DefaultHardLimits(plan)
		if err := r.applyPlanSettings(ctx, tenantID, plan); err != nil {
			return err
		}
	}
	if strings.TrimSpace(customerID) != "" {
		value := strings.TrimSpace(customerID)
		settings.ProviderCustomerID = &value
	}
	if strings.TrimSpace(contractID) != "" {
		value := strings.TrimSpace(contractID)
		settings.ProviderContractID = &value
	}
	settings.BillingStatus = status
	if status == billingdomain.BillingPastDue {
		now := time.Now().UTC()
		settings.PastDueSince = &now
	} else {
		settings.PastDueSince = nil
	}
	settings.UpdatedAt = time.Now().UTC()
	_, err = r.repo.UpsertTenantBilling(ctx, settings)
	return err
}

func (r *Runtime) applyPlanSettings(ctx context.Context, tenantID string, plan billingdomain.PlanCode) error {
	if r.tenantSettings == nil {
		return nil
	}
	_, err := r.tenantSettings.UpsertTenantSettings(ctx, admindomain.TenantSettings{
		TenantID:   strings.TrimSpace(tenantID),
		PlanCode:   string(plan),
		HardLimits: hardLimitsToMap(DefaultHardLimits(plan)),
		UpdatedAt:  time.Now().UTC(),
	})
	return err
}

func (r *Runtime) priceIDByPlan(plan billingdomain.PlanCode) string {
	switch plan {
	case billingdomain.PlanStarter:
		return r.priceStarter
	case billingdomain.PlanGrowth:
		return r.priceGrowth
	case billingdomain.PlanEnterprise:
		return r.priceEnterprise
	default:
		return ""
	}
}

func (r *Runtime) planByPriceID(priceID string) billingdomain.PlanCode {
	priceID = strings.TrimSpace(priceID)
	switch priceID {
	case r.priceStarter:
		return billingdomain.PlanStarter
	case r.priceGrowth:
		return billingdomain.PlanGrowth
	case r.priceEnterprise:
		return billingdomain.PlanEnterprise
	default:
		return billingdomain.PlanStarter
	}
}

func (r *Runtime) requireStripeEnabled() error {
	if !r.Enabled() {
		return domainerr.Internal("stripe billing is not configured")
	}
	return nil
}

func billingStatusFromStripe(raw string) billingdomain.BillingStatus {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "active":
		return billingdomain.BillingActive
	case "past_due":
		return billingdomain.BillingPastDue
	case "canceled":
		return billingdomain.BillingCanceled
	case "unpaid":
		return billingdomain.BillingUnpaid
	default:
		return billingdomain.BillingTrialing
	}
}

func sanitizeConsoleBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = defaultConsoleBaseURL
	}
	return strings.TrimRight(raw, "/")
}

func hardLimitsToMap(limits billingdomain.HardLimits) map[string]any {
	return map[string]any{
		"tools_max":            limits.ToolsMax,
		"run_rpm":              limits.RunRPM,
		"audit_retention_days": limits.AuditRetentionDays,
	}
}

func (r *Runtime) notifyAsync(tenantID, notifType string, data map[string]string) {
	if r.notifications == nil {
		return
	}
	payload := make(map[string]string, len(data))
	for key, value := range data {
		payload[key] = strings.TrimSpace(value)
	}
	select {
	case r.notifSem <- struct{}{}:
		go func() {
			defer func() { <-r.notifSem }()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := r.notifications.Notify(ctx, tenantID, notifType, payload); err != nil {
				r.logger.Error("failed async billing notification", "tenant_id", tenantID, "notification_type", notifType, "error", err)
			}
		}()
	default:
		r.logger.Warn("billing notification dropped: concurrency limit reached", "tenant_id", tenantID, "notification_type", notifType)
	}
}

func (r *Runtime) String() string {
	return fmt.Sprintf("billing.runtime(enabled=%t)", r.Enabled())
}

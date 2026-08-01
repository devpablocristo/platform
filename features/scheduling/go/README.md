# Scheduling core for Go

This module is a reusable application and domain core for scheduling and
virtual queues. It intentionally contains no HTTP framework, ORM, database
driver, migrations, seed data, authentication middleware, or product-specific
runtime.

The core provides:

- branches, services, resources, availability rules, and blocked ranges;
- slot generation with time zones, buffers, capacity, and service concurrency;
- atomic multi-resource booking requirements through `ResourceAllocation`;
- booking holds, confirmation, cancellation, rescheduling, check-in, and
  completion;
- queue tickets, waitlists, action tokens, audit hooks, and notification hooks.

## Hexagonal boundary

Consumers provide implementations of `RepositoryPort`, `AuditPort`, and
`NotificationPort`. The public domain and ports intentionally contain no
tenant identifier or tenancy policy. A multi-tenant consumer must bind a
tenant-scoped adapter instance before constructing `Usecases`.

`ReserveBookings` and `RescheduleBooking` are the transactional integrity
boundary. An adapter must lock or otherwise serialize every requested resource,
verify all capacities, exclusive allocations, participant units, and service
concurrency in the same transaction, and return `ErrCapacityExceeded` without
partially reserving anything.

The module publishes only framework-independent errors. Adapters map their own
driver errors to `ErrNotFound`, `ErrAlreadyExists`, or
`ErrCapacityExceeded`.

## Security

The public projection in `publicapi` intentionally omits customer phone, email,
and name. The core supports opaque action tokens for individual bookings but
does not expose enumeration or lookup of booking history by phone number.

## Migration from 0.1

Version 0.2 removes the former Gin, GORM, PostgreSQL migration, repository, and
seed packages. Product repositories own those adapters and migrations. Replace
the removed concrete repository with a `RepositoryPort` implementation and
carry `Booking.Allocations` as the authoritative resource reservation set;
`Booking.ResourceID` remains only as a primary-resource compatibility field.
Capacity allocations default to one unit per participant when `Units` is zero;
exclusive allocations reserve the resource's complete capacity.

CREATE INDEX IF NOT EXISTS idx_scheduling_bookings_org_phone_digits_start_desc
ON scheduling_bookings (
    org_id,
    (regexp_replace(customer_phone, '[^0-9]', '', 'g')),
    start_at DESC
);

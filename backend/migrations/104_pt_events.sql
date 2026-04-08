-- Migration 104: pt_events table
-- Tracks user interactions: CTA clicks and notification views (for metrics)

CREATE TABLE IF NOT EXISTS portal.pt_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type            VARCHAR(50) NOT NULL,              -- 'cta_click' | 'notif_view'
    notification_id UUID REFERENCES portal.pt_notifications(id) ON DELETE SET NULL,  -- nullable
    session_id      VARCHAR(255),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for metrics queries: count events by type over time
CREATE INDEX IF NOT EXISTS idx_pt_events_type_created
    ON portal.pt_events(type, created_at);

-- Index to look up events by notification (for view counts)
CREATE INDEX IF NOT EXISTS idx_pt_events_notification
    ON portal.pt_events(notification_id)
    WHERE notification_id IS NOT NULL;

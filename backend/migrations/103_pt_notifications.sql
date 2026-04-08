-- Migration 103: pt_notifications table
-- Stores admin-authored announcements shown on the public portal

CREATE TABLE IF NOT EXISTS portal.pt_notifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type         VARCHAR(50) NOT NULL DEFAULT 'info',
    title        VARCHAR(500) NOT NULL,
    body         TEXT NOT NULL,
    published    BOOLEAN NOT NULL DEFAULT false,
    published_at TIMESTAMP WITH TIME ZONE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for public API: fetch only published notifications ordered by most recent
CREATE INDEX IF NOT EXISTS idx_pt_notifications_published
    ON portal.pt_notifications(published, published_at DESC);

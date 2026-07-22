-- Phase 8: Enable Supabase realtime on findings so RealtimeDashboard's
-- postgres_changes subscription (INSERT/UPDATE) actually receives events.
-- Without this, the dashboard silently never updates live — inserts/updates
-- still succeed, they just don't broadcast.

alter publication supabase_realtime add table findings;
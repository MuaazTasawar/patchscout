-- Phase 8 (security): Enable RLS on all tables. The app's write path is
-- entirely server-side (Next.js API routes + Go worker), both using the
-- Supabase service role key, which bypasses RLS by design. The ONLY
-- direct-from-browser access is RealtimeDashboard.tsx subscribing to
-- `findings` with the anon key for postgres_changes — so anon gets
-- read-only access to findings, and nothing else.

alter table repos enable row level security;
alter table scan_jobs enable row level security;
alter table findings enable row level security;

-- repos / scan_jobs: no anon policies at all. The browser never touches
-- these directly — only via server API routes using the service role key,
-- which bypasses RLS regardless of policies present.

-- findings: anon can SELECT (required for the realtime subscription to
-- receive postgres_changes events), but cannot INSERT/UPDATE/DELETE.
-- All writes to findings happen via the service-role worker-callback and
-- PATCH /api/findings/[id] routes.
create policy "Allow anon read access to findings"
  on findings
  for select
  to anon
  using (true);
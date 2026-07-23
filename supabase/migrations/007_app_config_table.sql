-- Phase 8 (deploy config): Supabase's hosted Postgres does not grant
-- permission for `ALTER DATABASE ... SET app.settings.x` to project
-- owners (superuser-only), which the original notify_scan_worker()
-- relied on via current_setting(). Replace that with a simple config
-- table the function reads from instead — same effect, no elevated
-- privileges required, and updating the worker URL going forward (e.g.
-- switching from a local ngrok tunnel to the real Render URL) is just a
-- plain UPDATE statement, no migration needed each time.

create table if not exists app_config (
  key text primary key,
  value text not null
);

-- No anon/authenticated access at all — service role only (service role
-- bypasses RLS regardless, but enabling RLS here is still correct since
-- this table has no legitimate anon/authenticated use case).
alter table app_config enable row level security;

insert into app_config (key, value) values
  ('worker_webhook_url', 'https://your-render-worker-url.onrender.com/webhook/scan'),
  ('worker_webhook_secret', 'change-me')
on conflict (key) do nothing;

-- Table references are schema-qualified (public.app_config) because
-- search_path is pinned to '' below — an empty search_path means
-- unqualified names won't resolve at all.
create or replace function notify_scan_worker()
returns trigger as $$
declare
  worker_url text;
  worker_secret text;
begin
  select value into worker_url from public.app_config where key = 'worker_webhook_url';
  select value into worker_secret from public.app_config where key = 'worker_webhook_secret';

  perform net.http_post(
    url := worker_url,
    headers := jsonb_build_object(
      'Content-Type', 'application/json',
      'X-Webhook-Secret', coalesce(worker_secret, 'change-me')
    ),
    body := jsonb_build_object(
      'job_id', new.id,
      'repo_id', new.repo_id
    )
  );

  return new;
end;
$$ language plpgsql security definer set search_path = '';

revoke execute on function notify_scan_worker() from public;
revoke execute on function notify_scan_worker() from anon;
revoke execute on function notify_scan_worker() from authenticated;

-- To point the trigger at a real worker deployment, no migration needed —
-- just:
--   update app_config set value = 'https://your-worker.onrender.com/webhook/scan' where key = 'worker_webhook_url';
--   update app_config set value = 'your-shared-secret' where key = 'worker_webhook_secret';
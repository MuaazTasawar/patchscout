-- Phase 3: scan_jobs table + push-based webhook to the Go worker.

create table if not exists scan_jobs (
  id uuid primary key default uuid_generate_v4(),
  repo_id uuid not null references repos (id) on delete cascade,
  status text not null default 'queued' check (status in ('queued', 'cloning', 'scanning', 'completed', 'failed')),
  requested_by text,
  error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_scan_jobs_status on scan_jobs (status);
create index if not exists idx_scan_jobs_repo_id on scan_jobs (repo_id);

drop trigger if exists trg_scan_jobs_updated_at on scan_jobs;
create trigger trg_scan_jobs_updated_at
  before update on scan_jobs
  for each row execute function set_updated_at();

-- Now that scan_jobs exists, tighten findings.scan_job_id into a real FK.
alter table findings
  drop constraint if exists findings_scan_job_id_fkey;
alter table findings
  add constraint findings_scan_job_id_fkey
  foreign key (scan_job_id) references scan_jobs (id) on delete cascade;

-- Push-based job handoff: on insert into scan_jobs, call the Go worker's
-- webhook endpoint directly via pg_net (async HTTP from Postgres).
-- Replace the URL below with your deployed Render worker URL once known;
-- until then this trigger will fail silently (pg_net logs errors, doesn't
-- block the insert), which is fine for local dev against a stub worker.

create or replace function notify_scan_worker()
returns trigger as $$
declare
  worker_url text := current_setting('app.settings.worker_webhook_url', true);
  worker_secret text := current_setting('app.settings.worker_webhook_secret', true);
begin
  if worker_url is null then
    worker_url := 'https://your-render-worker-url.onrender.com/webhook/scan';
  end if;

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
$$ language plpgsql security definer;

drop trigger if exists trg_notify_scan_worker on scan_jobs;
create trigger trg_notify_scan_worker
  after insert on scan_jobs
  for each row execute function notify_scan_worker();

-- To set the worker URL/secret without editing this migration again:
--   alter database postgres set app.settings.worker_webhook_url = 'https://your-worker.onrender.com/webhook/scan';
--   alter database postgres set app.settings.worker_webhook_secret = 'your-shared-secret';
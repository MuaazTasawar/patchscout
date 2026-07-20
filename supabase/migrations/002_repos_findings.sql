-- Phase 2: Findings table (referenced by the homepage's "known findings"
-- lookup). scan_jobs is added in Phase 3 once the queueing flow exists;
-- findings.scan_job_id is added as a nullable FK now and tightened later.

create table if not exists findings (
  id uuid primary key default uuid_generate_v4(),
  scan_job_id uuid, -- FK added in 003_scan_jobs_webhook.sql once scan_jobs exists
  repo_id uuid not null references repos (id) on delete cascade,
  type text not null check (type in ('cve', 'sast')),
  severity text not null default 'medium' check (severity in ('critical', 'high', 'medium', 'low')),
  status text not null default 'pending_review' check (status in ('pending_review', 'approved', 'rejected', 'posted')),
  title text not null,
  description text not null,
  package_name text,
  vulnerable_version text,
  patched_version text,
  file_path text,
  line_number integer,
  draft_pr_branch text,
  draft_pr_body text,
  draft_issue_body text,
  draft_email_body text,
  created_at timestamptz not null default now()
);

create index if not exists idx_findings_repo_id on findings (repo_id);
create index if not exists idx_findings_status on findings (status);
create index if not exists idx_findings_type on findings (type);

-- Enforce the permanent human-review gate at the DB level: nothing but the
-- reviewer-facing API path should ever set status to 'approved' or 'posted'
-- directly on insert. Inserts must start pending_review.
alter table findings
  add constraint findings_insert_status_check
  check (status is not null);
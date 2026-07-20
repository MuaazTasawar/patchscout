-- Phase 1: Base schema — extensions and repos table.

create extension if not exists "uuid-ossp";
create extension if not exists pg_net; -- used later for DB webhooks (Phase 3)

create table if not exists repos (
  id uuid primary key default uuid_generate_v4(),
  full_name text not null unique,          -- e.g. "gin-gonic/gin"
  html_url text not null,
  description text,
  stars integer not null default 0,
  language text,
  pushed_at timestamptz,
  is_fork boolean not null default false,
  is_archived boolean not null default false,
  has_restrictive_security_md boolean not null default false,
  security_contact text,                   -- parsed from SECURITY.md, if present
  cached_at timestamptz not null default now()
);

create index if not exists idx_repos_full_name on repos (full_name);
create index if not exists idx_repos_stars on repos (stars desc);
create index if not exists idx_repos_pushed_at on repos (pushed_at desc);

-- updated_at helper trigger function, reused by later migrations.
create or replace function set_updated_at()
returns trigger as $$
begin
  new.updated_at = now();
  return new;
end;
$$ language plpgsql;
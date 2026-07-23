# PatchScout

A search-driven OSS bug-scanning and disclosure platform. Search a topic or repo, queue a scan, and get review-ready PRs (dependency CVE fixes) and draft GitHub issues (static analysis findings) — nothing goes live without manual approval.

Built as a portfolio project extending a track record of merged security-relevant PRs (including a [merged PR to gin-gonic/gin](https://github.com/gin-gonic/gin/pull/4699) adding regression tests for `responseWriter.Flush()`), and as a distributed-systems story for Forward Deployed Engineer interviews: a stateless web layer split from a long-running concurrent compute service.

## How it works

1. **Homepage** shows trending GitHub repos (scraped, all languages, cached ~4h via Next.js revalidation) alongside any previously-found issues from the database, so it's never empty.
2. **Search** a topic/language/keyword — hits the GitHub Search API live, filtered to repos with >100 stars, pushed within 90 days, excluding forks, archived repos, and repos with a restrictive `SECURITY.md`.
3. **Select repos** (or select all) and queue them for scanning.
4. A **Postgres trigger** (`pg_net`) fires on `scan_jobs` insert and pushes the job directly to the Go worker — no polling.
5. The **Go worker** shallow-clones the repo, parses manifests (`go.mod`, `requirements.txt`/`pyproject.toml`, `pubspec.yaml`), and runs two independent detection passes:
   - **Dependency CVEs** via OSV.dev batch queries
   - **Code-pattern bugs** via the Semgrep CLI (public ruleset)
6. Findings are converted into draft text — a review-ready **PR body** (for mechanical CVE version bumps with a known patch) or a draft **GitHub issue** (for Semgrep hits) — and POSTed back to Next.js.
7. Everything lands in the **review dashboard** at `status = pending_review`. Nothing is posted to GitHub automatically. You approve, reject, or edit each finding's draft text before anything goes public — this gate is permanent, not a v1 placeholder.
8. The dashboard updates live via **Supabase realtime** as the worker finishes each repo.

## Architecture

| Layer | Tech | Hosting |
|---|---|---|
| Frontend + API routes | Next.js 14 (App Router) | Vercel |
| Auth + DB + Realtime | Supabase (Postgres) | Supabase |
| Scan worker | Go (goroutines, shells out to Semgrep) | Render (Background Worker) |
| Job handoff | Supabase DB webhook (`pg_net`) → worker `/webhook/scan` | push, not poll |

The worker is a separate long-running service specifically to escape Vercel's serverless timeout limits — cloning, dependency-graph parsing, and Semgrep runs can take longer than a typical serverless function budget allows, especially under concurrent load.

## Project structure

```
patchscout/
├── web/            Next.js app — search UI, trending homepage, review dashboard, API routes
├── worker/         Go scan worker — webhook receiver, clone, manifest parsing, detection, draft generation
└── supabase/       SQL migrations — schema, triggers, realtime publication
```

## Setup

### Prerequisites
- Node.js 18+
- Go 1.22+
- A Supabase project
- [Semgrep CLI](https://semgrep.dev/docs/getting-started/) installed and on `PATH` (for local worker testing)
- A GitHub personal access token (for Search API + SECURITY.md reads)

### 1. Database
Run the migrations in `supabase/migrations/` in order (001 → 007) via the Supabase SQL Editor or CLI.

Migrations 005–007 enable Row Level Security (the app's write path is entirely server-side via the service role key, so `anon` gets read-only access to `findings` and nothing else — see 005 for details) and set up an `app_config` table for the worker webhook URL/secret.

**Note:** an earlier version of this setup tried to store the worker webhook URL via `alter database postgres set app.settings.x`. That fails on hosted Supabase — `ALTER DATABASE ... SET` requires superuser privileges that even the project owner doesn't get. The `app_config` table (migration 007) is the actual working approach.

Set the worker webhook URL/secret with a plain `UPDATE` — no migration needed each time you change it:

```sql
update app_config set value = 'https://your-worker.onrender.com/webhook/scan' where key = 'worker_webhook_url';
update app_config set value = 'your-shared-secret' where key = 'worker_webhook_secret';
```

### 2. Web app

```powershell
cd web
Copy-Item .env.local.example .env.local
# fill in NEXT_PUBLIC_SUPABASE_URL, NEXT_PUBLIC_SUPABASE_ANON_KEY,
# SUPABASE_SERVICE_ROLE_KEY, GITHUB_TOKEN, WORKER_CALLBACK_SECRET
npm install
npm run dev
```

**Note:** `@supabase/supabase-js` is pinned to an exact version (`2.45.4`, no `^`) in `package.json`. Newer 2.x releases changed how `postgrest-js` resolves generic `Insert`/`Update` types and broke typed `.insert()`/`.update()` calls against a hand-written `Database` type — don't loosen this pin without re-verifying against a real compile.

### 3. Go worker

```powershell
cd worker
Copy-Item .env.example .env
# fill in SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY, GITHUB_TOKEN,
# WORKER_CALLBACK_SECRET, NEXT_CALLBACK_URL
go mod tidy
go run ./cmd/worker
```

The worker listens on `:8080` by default, with:
- `POST /webhook/scan` — receives job payloads from the Supabase trigger
- `GET /healthz` — health check for Render's probe

### 4. Local end-to-end testing (before deploying)
Since the DB webhook trigger needs a real URL to POST to, testing the full pipeline locally (Postgres trigger → worker → detection → dashboard) requires exposing your local worker to the internet temporarily. Three terminals:

```powershell
# Terminal 1 — worker
cd worker
go run ./cmd/worker
# should print "listening on :8080"

# Terminal 2 — tunnel
ngrok http 8080
# copy the forwarding URL, e.g. https://abc123.ngrok-free.dev

# Terminal 3 — web app
cd web
npm run dev
# should print "Local: http://localhost:3000"
```

Then point the trigger at your tunnel:
```sql
update app_config set value = 'https://abc123.ngrok-free.dev/webhook/scan' where key = 'worker_webhook_url';
```

Free-tier ngrok issues a new URL every time you restart it — re-run that `UPDATE` whenever you restart the tunnel. Once the worker's deployed to Render for real, point `app_config` at the permanent Render URL instead and you can stop running ngrok/the worker locally.

To trigger a test scan without going through the UI, insert directly:
```sql
insert into repos (full_name, html_url, description, stars, language, pushed_at, is_fork, is_archived, has_restrictive_security_md, cached_at)
values ('owner/repo', 'https://github.com/owner/repo', null, 0, 'Go', now(), false, false, false, now())
returning id;

insert into scan_jobs (repo_id, status) values ('<id from above>', 'queued');
```
Watch the worker's terminal for `starting scan` → `found N manifest(s)` → `scan complete — N finding(s)`, then check `select status from scan_jobs where id = '...'` — it should move `queued → cloning → scanning → completed`.

### 5. Deploy
- **Web**: connect the repo to Vercel, set the same env vars as `.env.local`, deploy `web/` as the root.
- **Worker**: create a Render Background Worker, point it at `worker/`, build command `go build -o app ./cmd/worker`, start command `./app`. Install Semgrep in the build step (`pip install semgrep`) since the worker shells out to it.

## Scope notes

**In scope for v1:** Go, Python, and Flutter/Dart manifest parsing; on-demand search-driven scanning; OSV.dev + Semgrep detection; auto-drafted PR/issue text; permanent human review gate.

**Explicitly out of scope for v1:** always-on background crawling (replaced by on-demand search), Flutter-specific Semgrep rules, Redis/queue infra, scanning of live/running systems, auto-sent outreach emails, and scraping personal maintainer emails from commit history. Draft outreach emails are generated only for repos with a published security contact (from `SECURITY.md`), and are never sent automatically.

**Verified via end-to-end testing:** the full pipeline (trigger → clone → manifest parse → OSV/Semgrep → draft generation → dashboard) was tested against a real CVE (`CVE-2020-26160`, `github.com/dgrijalva/jwt-go`). That testing caught and fixed a real correctness bug: OSV advisories can bundle multiple unrelated packages under one advisory ID (e.g. a module and an unrelated forked import path), each with its own fix status — `extractFixedVersion` in `osv.go` now scopes its lookup to the specific package being queried, rather than taking the first `Fixed` version found anywhere in the advisory. Without this, a finding could be stamped with a "patched version" that doesn't actually exist for the vulnerable package, and an auto-drafted PR could suggest a broken bump.

## Stack

Go (worker), Next.js + TypeScript (frontend/API), Supabase/Postgres (DB, auth, realtime), Semgrep CLI, OSV.dev API, GitHub REST + Search API, Tailwind CSS, deployed on Vercel + Render.

---

Built by [Muaaz Tasawar](https://github.com/MuaazTasawar).
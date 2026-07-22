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
Run the migrations in `supabase/migrations/` in order (001 → 003) via the Supabase SQL Editor or CLI. Then enable realtime on `findings`:

```sql
alter publication supabase_realtime add table findings;
```

Set the worker webhook URL/secret once you have a Render deployment:

```sql
alter database postgres set app.settings.worker_webhook_url = 'https://your-worker.onrender.com/webhook/scan';
alter database postgres set app.settings.worker_webhook_secret = 'your-shared-secret';
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

### 4. Deploy
- **Web**: connect the repo to Vercel, set the same env vars as `.env.local`, deploy `web/` as the root.
- **Worker**: create a Render Background Worker, point it at `worker/`, build command `go build -o app ./cmd/worker`, start command `./app`. Install Semgrep in the build step (`pip install semgrep`) since the worker shells out to it.

## Scope notes

**In scope for v1:** Go, Python, and Flutter/Dart manifest parsing; on-demand search-driven scanning; OSV.dev + Semgrep detection; auto-drafted PR/issue text; permanent human review gate.

**Explicitly out of scope for v1:** always-on background crawling (replaced by on-demand search), Flutter-specific Semgrep rules, Redis/queue infra, scanning of live/running systems, auto-sent outreach emails, and scraping personal maintainer emails from commit history. Draft outreach emails are generated only for repos with a published security contact (from `SECURITY.md`), and are never sent automatically.

## Stack

Go (worker), Next.js + TypeScript (frontend/API), Supabase/Postgres (DB, auth, realtime), Semgrep CLI, OSV.dev API, GitHub REST + Search API, Tailwind CSS, deployed on Vercel + Render.

---

Built by [Muaaz Tasawar](https://github.com/MuaazTasawar).
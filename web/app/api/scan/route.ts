import { NextRequest, NextResponse } from "next/server";
import { supabaseServer } from "@/lib/supabaseServer";
import { checkSecurityMd } from "@/lib/github";

interface IncomingRepo {
  full_name: string;
  html_url: string;
  description: string | null;
  stars: number;
  language: string | null;
  pushed_at: string;
  security_contact: string | null;
}

interface ScanRequestBody {
  repos: IncomingRepo[];
}

/**
 * POST /api/scan
 * Takes the repos selected on the search page ("scan all results" is just
 * the client sending every result), upserts them into `repos`, creates one
 * `scan_jobs` row per repo, and returns the created job ids.
 *
 * Job pickup itself is NOT triggered here — a Postgres trigger + webhook
 * (see 003_scan_jobs_webhook.sql) fires on insert into scan_jobs and pushes
 * the job straight to the Go worker's /webhook endpoint. This route's only
 * responsibility is to write rows; it never calls the worker directly.
 */
export async function POST(req: NextRequest) {
  let body: ScanRequestBody;

  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON body" }, { status: 400 });
  }

  if (!body.repos || !Array.isArray(body.repos) || body.repos.length === 0) {
    return NextResponse.json({ error: "repos must be a non-empty array" }, { status: 400 });
  }

  if (body.repos.length > 25) {
    return NextResponse.json(
      { error: "Cannot queue more than 25 repos at once" },
      { status: 400 }
    );
  }

  try {
    const jobIds: { full_name: string; job_id: string }[] = [];

    for (const repo of body.repos) {
      // Re-check SECURITY.md at queue time in case it changed since search,
      // and to backfill security_contact if it wasn't captured earlier.
      const security = await checkSecurityMd(repo.full_name);
      if (security.isRestrictive) {
        continue; // silently skip — this repo opted out of scanning
      }

      const { data: repoRow, error: repoError } = await supabaseServer
        .from("repos")
        .upsert(
          {
            full_name: repo.full_name,
            html_url: repo.html_url,
            description: repo.description,
            stars: repo.stars,
            language: repo.language,
            pushed_at: repo.pushed_at,
            is_fork: false,
            is_archived: false,
            has_restrictive_security_md: false,
            security_contact: repo.security_contact ?? security.contact,
            cached_at: new Date().toISOString(),
          },
          { onConflict: "full_name" }
        )
        .select("id, full_name")
        .single();

      if (repoError) throw repoError;

      const { data: jobRow, error: jobError } = await supabaseServer
        .from("scan_jobs")
        .insert({
          repo_id: repoRow.id,
          status: "queued",
        })
        .select("id")
        .single();

      if (jobError) throw jobError;

      jobIds.push({ full_name: repo.full_name, job_id: jobRow.id });
    }

    if (jobIds.length === 0) {
      return NextResponse.json(
        { error: "All selected repos opted out of scanning via SECURITY.md" },
        { status: 422 }
      );
    }

    return NextResponse.json({ queued: jobIds }, { status: 201 });
  } catch (err) {
    console.error("POST /api/scan failed:", err);
    return NextResponse.json({ error: "Failed to queue scan jobs" }, { status: 500 });
  }
}
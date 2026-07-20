import { NextRequest, NextResponse } from "next/server";
import { supabaseServer } from "@/lib/supabaseServer";
import type { FindingInsert } from "@/lib/types";

/**
 * POST /api/webhook/worker-callback
 * The Go worker POSTs here once a scan completes with its findings. This
 * is the ONLY place `findings` rows get written — the worker never talks
 * to Supabase directly for findings, only for scan_jobs status. Every row
 * written here starts at status = 'pending_review' regardless of what the
 * worker sends, enforcing the permanent human-review gate at this
 * boundary too (defense in depth alongside the DB check constraint).
 */

interface WorkerFinding {
  scan_job_id: string;
  repo_id: string;
  type: "cve" | "sast";
  severity: "critical" | "high" | "medium" | "low";
  title: string;
  description: string;
  package_name?: string | null;
  vulnerable_version?: string | null;
  patched_version?: string | null;
  file_path?: string | null;
  line_number?: number | null;
  draft_pr_branch?: string | null;
  draft_pr_body?: string | null;
  draft_issue_body?: string | null;
  draft_email_body?: string | null;
}

interface CallbackBody {
  job_id: string;
  findings: WorkerFinding[];
}

export async function POST(req: NextRequest) {
  const secret = req.headers.get("x-webhook-secret");
  if (secret !== process.env.WORKER_CALLBACK_SECRET) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }

  let body: CallbackBody;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON body" }, { status: 400 });
  }

  if (!body.findings || !Array.isArray(body.findings)) {
    return NextResponse.json({ error: "findings must be an array" }, { status: 400 });
  }

  if (body.findings.length === 0) {
    return NextResponse.json({ inserted: 0 });
  }

  const rows: FindingInsert[] = body.findings.map((f) => ({
    scan_job_id: f.scan_job_id,
    repo_id: f.repo_id,
    type: f.type,
    severity: f.severity,
    status: "pending_review", // always — never trust the worker's status
    title: f.title,
    description: f.description,
    package_name: f.package_name ?? null,
    vulnerable_version: f.vulnerable_version ?? null,
    patched_version: f.patched_version ?? null,
    file_path: f.file_path ?? null,
    line_number: f.line_number ?? null,
    draft_pr_branch: f.draft_pr_branch ?? null,
    draft_pr_body: f.draft_pr_body ?? null,
    draft_issue_body: f.draft_issue_body ?? null,
    draft_email_body: f.draft_email_body ?? null,
  }));

  const { data, error } = await supabaseServer.from("findings").insert(rows).select("id");

  if (error) {
    console.error("worker-callback insert failed:", error);
    return NextResponse.json({ error: "Failed to store findings" }, { status: 500 });
  }

  return NextResponse.json({ inserted: data?.length ?? 0 }, { status: 201 });
}
import { NextRequest, NextResponse } from "next/server";
import { supabaseServer } from "@/lib/supabaseServer";
import type { FindingUpdate } from "@/lib/types";

/**
 * PATCH /api/findings/[id]
 * The single write path for reviewer decisions: approve, reject, or edit
 * a finding's draft text before approving. This is the human-review gate
 * in practice — nothing else in the app can move a finding out of
 * pending_review.
 *
 * Body: { action: "approve" | "reject" | "edit", edits?: {...} }
 * "approve" does NOT post to GitHub itself in v1 — it flips status to
 * 'approved' so the reviewer can then post it manually from the dashboard
 * (or a future phase can wire real posting behind an explicit second
 * confirmation). This keeps "approved in our system" and "posted to
 * GitHub" as separate, auditable steps.
 */
export async function PATCH(req: NextRequest, { params }: { params: { id: string } }) {
  const { id } = params;

  let body: { action: "approve" | "reject" | "edit"; edits?: Partial<FindingUpdate> };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON body" }, { status: 400 });
  }

  if (!body.action || !["approve", "reject", "edit"].includes(body.action)) {
    return NextResponse.json(
      { error: "action must be 'approve', 'reject', or 'edit'" },
      { status: 400 }
    );
  }

  const patch: FindingUpdate = {};

  if (body.action === "approve") {
    patch.status = "approved";
  } else if (body.action === "reject") {
    patch.status = "rejected";
  } else if (body.action === "edit") {
    if (!body.edits) {
      return NextResponse.json({ error: "edits object required for action=edit" }, { status: 400 });
    }
    // Only allow editing draft text fields — never status, never IDs.
    const allowedFields: (keyof FindingUpdate)[] = [
      "draft_pr_body",
      "draft_issue_body",
      "draft_email_body",
      "title",
      "description",
    ];
    for (const key of allowedFields) {
      if (key in body.edits) {
        (patch as Record<string, unknown>)[key] = (body.edits as Record<string, unknown>)[key];
      }
    }
  }

  const { data, error } = await supabaseServer
    .from("findings")
    .update(patch)
    .eq("id", id)
    .select("*");

  if (error) {
    console.error(`PATCH /api/findings/${id} failed:`, error);
    return NextResponse.json({ error: "Failed to update finding" }, { status: 500 });
  }

  const updated = data?.[0];
  if (!updated) {
    return NextResponse.json({ error: "Finding not found" }, { status: 404 });
  }

  return NextResponse.json({ finding: updated });
}
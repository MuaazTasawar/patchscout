import { NextRequest, NextResponse } from "next/server";
import { supabaseServer } from "@/lib/supabaseServer";

/**
 * GET /api/findings
 * Lists findings for the review dashboard, optionally filtered by status.
 * Defaults to pending_review since that's the primary dashboard view.
 */
export async function GET(req: NextRequest) {
  const status = req.nextUrl.searchParams.get("status") ?? "pending_review";

  let query = supabaseServer
    .from("findings")
    .select("*, repos(full_name, html_url)")
    .order("created_at", { ascending: false });

  if (status !== "all") {
    query = query.eq("status", status);
  }

  const { data, error } = await query;

  if (error) {
    console.error("GET /api/findings failed:", error);
    return NextResponse.json({ error: "Failed to load findings" }, { status: 500 });
  }

  return NextResponse.json({ findings: data ?? [] });
}
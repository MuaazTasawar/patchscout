"use client";

import { useEffect, useState, useCallback } from "react";
import { supabaseBrowser } from "@/lib/supabaseClient";
import FindingCard from "./FindingCard";

interface FindingRow {
  id: string;
  type: "cve" | "sast";
  severity: "critical" | "high" | "medium" | "low";
  status: "pending_review" | "approved" | "rejected" | "posted";
  title: string;
  description: string;
  package_name: string | null;
  vulnerable_version: string | null;
  patched_version: string | null;
  file_path: string | null;
  line_number: number | null;
  draft_pr_body: string | null;
  draft_issue_body: string | null;
  repos: { full_name: string; html_url: string } | null;
  created_at: string;
}

/**
 * Client component that fetches the initial pending_review findings list,
 * then subscribes to Supabase realtime so newly-inserted findings (from
 * the worker completing a scan) appear live without a page refresh — per
 * the spec's "Dashboard updates live via Supabase realtime subscriptions".
 */
export default function RealtimeDashboard() {
  const [findings, setFindings] = useState<FindingRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadFindings = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/findings?status=pending_review");
      const json = await res.json();
      if (json.error) throw new Error(json.error);
      setFindings(json.findings ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load findings");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadFindings();

    // Realtime: new findings inserted by the worker-callback route show up
    // immediately. We refetch via the API route (rather than trusting the
    // raw payload) so the repos join stays consistent.
    const channel = supabaseBrowser
      .channel("findings-realtime")
      .on(
        "postgres_changes",
        { event: "INSERT", schema: "public", table: "findings" },
        () => {
          loadFindings();
        }
      )
      .on(
        "postgres_changes",
        { event: "UPDATE", schema: "public", table: "findings" },
        (payload) => {
          setFindings((prev) =>
            prev.filter((f) => f.id !== (payload.new as { id: string }).id)
          );
        }
      )
      .subscribe();

    return () => {
      supabaseBrowser.removeChannel(channel);
    };
  }, [loadFindings]);

  async function handleDecision(id: string, action: "approve" | "reject") {
    // Optimistic removal — this card belongs to pending_review only.
    setFindings((prev) => prev.filter((f) => f.id !== id));

    try {
      const res = await fetch(`/api/findings/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Failed to update finding");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update finding");
      loadFindings(); // resync on failure
    }
  }

  async function handleEdit(
    id: string,
    field: "draft_pr_body" | "draft_issue_body",
    value: string
  ) {
    try {
      const res = await fetch(`/api/findings/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "edit", edits: { [field]: value } }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? "Failed to save edit");
      }
      setFindings((prev) =>
        prev.map((f) => (f.id === id ? { ...f, [field]: value } : f))
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save edit");
    }
  }

  if (loading) {
    return <p className="text-text-muted text-sm">Loading review queue...</p>;
  }

  if (error) {
    return <p className="text-danger text-sm">{error}</p>;
  }

  if (findings.length === 0) {
    return (
      <p className="text-text-muted text-sm">
        Nothing pending review. Queue a scan from the search page to get started.
      </p>
    );
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      {findings.map((f) => (
        <FindingCard
          key={f.id}
          id={f.id}
          type={f.type}
          severity={f.severity}
          status={f.status}
          title={f.title}
          description={f.description}
          packageName={f.package_name}
          vulnerableVersion={f.vulnerable_version}
          patchedVersion={f.patched_version}
          filePath={f.file_path}
          lineNumber={f.line_number}
          draftPrBody={f.draft_pr_body}
          draftIssueBody={f.draft_issue_body}
          repo={f.repos}
          onDecision={handleDecision}
          onEdit={handleEdit}
        />
      ))}
    </div>
  );
}
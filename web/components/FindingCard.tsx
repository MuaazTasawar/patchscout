"use client";

import { useState } from "react";

interface FindingRepo {
  full_name: string;
  html_url: string;
}

interface FindingCardProps {
  id: string;
  type: "cve" | "sast";
  severity: "critical" | "high" | "medium" | "low";
  status: "pending_review" | "approved" | "rejected" | "posted";
  title: string;
  description: string;
  packageName: string | null;
  vulnerableVersion: string | null;
  patchedVersion: string | null;
  filePath: string | null;
  lineNumber: number | null;
  draftPrBody: string | null;
  draftIssueBody: string | null;
  repo: FindingRepo | null;
  onDecision: (id: string, action: "approve" | "reject") => void;
  onEdit: (id: string, field: "draft_pr_body" | "draft_issue_body", value: string) => void;
}

const severityClass: Record<string, string> = {
  critical: "badge-critical",
  high: "badge-high",
  medium: "badge-medium",
  low: "badge-low",
};

export default function FindingCard({
  id,
  type,
  severity,
  status,
  title,
  description,
  packageName,
  vulnerableVersion,
  patchedVersion,
  filePath,
  lineNumber,
  draftPrBody,
  draftIssueBody,
  repo,
  onDecision,
  onEdit,
}: FindingCardProps) {
  const [editing, setEditing] = useState(false);
  const draftField = type === "cve" ? "draft_pr_body" : "draft_issue_body";
  const draftText = type === "cve" ? draftPrBody ?? "" : draftIssueBody ?? "";
  const [localDraft, setLocalDraft] = useState(draftText);

  function saveEdit() {
    onEdit(id, draftField, localDraft);
    setEditing(false);
  }

  return (
    <div className="card p-4 flex flex-col gap-3">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <span className={`badge ${severityClass[severity]}`}>{severity}</span>
            <span className="badge badge-medium">{type === "cve" ? "CVE" : "SAST"}</span>
            {status !== "pending_review" && (
              <span className={`badge ${status === "approved" ? "badge-ok" : "badge-critical"}`}>
                {status}
              </span>
            )}
          </div>
          <h3 className="font-mono text-sm text-text-primary">{title}</h3>
          {repo && (
            <a
              href={repo.html_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-text-muted hover:text-accent transition-colors"
            >
              {repo.full_name}
            </a>
          )}
        </div>
      </div>

      <p className="text-sm text-text-secondary">{description}</p>

      {type === "cve" && (
        <div className="text-xs font-mono text-text-muted flex flex-wrap gap-x-4 gap-y-1">
          {packageName && <span>package: {packageName}</span>}
          {vulnerableVersion && <span>vulnerable: {vulnerableVersion}</span>}
          {patchedVersion && <span>patched: {patchedVersion}</span>}
        </div>
      )}

      {type === "sast" && filePath && (
        <div className="text-xs font-mono text-text-muted">
          {filePath}
          {lineNumber ? `:${lineNumber}` : ""}
        </div>
      )}

      <div className="border-t border-border pt-3">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs text-text-muted font-mono uppercase tracking-wide">
            {type === "cve" ? "Draft PR" : "Draft Issue"}
          </span>
          {!editing && status === "pending_review" && (
            <button
              onClick={() => setEditing(true)}
              className="text-xs text-accent hover:text-accent-hover"
            >
              edit
            </button>
          )}
        </div>

        {editing ? (
          <div className="flex flex-col gap-2">
            <textarea
              value={localDraft}
              onChange={(e) => setLocalDraft(e.target.value)}
              rows={8}
              className="w-full bg-bg border border-border rounded-md p-2 text-xs font-mono text-text-primary focus:outline-none focus:border-accent"
            />
            <div className="flex gap-2">
              <button
                onClick={saveEdit}
                className="text-xs bg-accent hover:bg-accent-hover text-white px-3 py-1 rounded-md"
              >
                Save
              </button>
              <button
                onClick={() => {
                  setLocalDraft(draftText);
                  setEditing(false);
                }}
                className="text-xs text-text-secondary hover:text-text-primary px-3 py-1"
              >
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <pre className="text-xs font-mono text-text-secondary whitespace-pre-wrap bg-bg border border-border rounded-md p-2 max-h-48 overflow-y-auto">
            {draftText || "(no draft text)"}
          </pre>
        )}
      </div>

      {status === "pending_review" && !editing && (
        <div className="flex gap-2 pt-1">
          <button
            onClick={() => onDecision(id, "approve")}
            className="flex-1 bg-ok/15 hover:bg-ok/25 text-ok text-xs font-medium px-3 py-2 rounded-md border border-ok/30 transition-colors"
          >
            Approve
          </button>
          <button
            onClick={() => onDecision(id, "reject")}
            className="flex-1 bg-danger/15 hover:bg-danger/25 text-danger text-xs font-medium px-3 py-2 rounded-md border border-danger/30 transition-colors"
          >
            Reject
          </button>
        </div>
      )}
    </div>
  );
}
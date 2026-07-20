"use client";

import { useState } from "react";

interface SearchResult {
  full_name: string;
  html_url: string;
  description: string | null;
  stars: number;
  language: string | null;
  pushed_at: string;
  security_contact: string | null;
}

interface ScanQueueButtonProps {
  selectedRepos: string[]; // full_name list
  allResults: SearchResult[];
}

export default function ScanQueueButton({ selectedRepos, allResults }: ScanQueueButtonProps) {
  const [status, setStatus] = useState<"idle" | "queuing" | "queued" | "error">("idle");
  const [message, setMessage] = useState<string | null>(null);

  const disabled = selectedRepos.length === 0 || status === "queuing";

  async function handleQueue() {
    setStatus("queuing");
    setMessage(null);

    const repos = allResults.filter((r) => selectedRepos.includes(r.full_name));

    try {
      // POST /api/scan is implemented in Phase 3 — this component compiles
      // and is wired up now, but the queue action will 404 until then.
      const res = await fetch("/api/scan", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ repos }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? `Request failed (${res.status})`);
      }

      setStatus("queued");
      setMessage(`Queued ${repos.length} repo${repos.length > 1 ? "s" : ""} for scanning.`);
    } catch (err) {
      setStatus("error");
      setMessage(err instanceof Error ? err.message : "Failed to queue scan.");
    }
  }

  return (
    <div className="flex items-center gap-2">
      <button
        onClick={handleQueue}
        disabled={disabled}
        className="bg-accent hover:bg-accent-hover disabled:bg-text-muted disabled:cursor-not-allowed text-white text-xs font-mono px-3 py-1.5 rounded-md transition-colors"
      >
        {status === "queuing"
          ? "Queuing..."
          : `Scan selected (${selectedRepos.length})`}
      </button>
      {message && (
        <span className={`text-xs ${status === "error" ? "text-danger" : "text-ok"}`}>
          {message}
        </span>
      )}
    </div>
  );
}
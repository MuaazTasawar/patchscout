// Shared domain types + a minimal Supabase Database type.
// Extend the Database type as new tables are added in later phases.

export type RepoLanguage = "go" | "python" | "flutter" | "other";

export type ScanJobStatus =
  | "queued"
  | "cloning"
  | "scanning"
  | "completed"
  | "failed";

export type FindingType = "cve" | "sast";

export type FindingSeverity = "critical" | "high" | "medium" | "low";

export type FindingStatus =
  | "pending_review"
  | "approved"
  | "rejected"
  | "posted";

export interface Repo {
  id: string;
  full_name: string;
  html_url: string;
  description: string | null;
  stars: number;
  language: string | null;
  pushed_at: string;
  is_fork: boolean;
  is_archived: boolean;
  has_restrictive_security_md: boolean;
  security_contact: string | null;
  cached_at: string;
}

export interface ScanJob {
  id: string;
  repo_id: string;
  status: ScanJobStatus;
  requested_by: string | null;
  created_at: string;
  updated_at: string;
  error: string | null;
}

export interface Finding {
  id: string;
  scan_job_id: string;
  repo_id: string;
  type: FindingType;
  severity: FindingSeverity;
  status: FindingStatus;
  title: string;
  description: string;
  package_name: string | null;
  vulnerable_version: string | null;
  patched_version: string | null;
  file_path: string | null;
  line_number: number | null;
  draft_pr_branch: string | null;
  draft_pr_body: string | null;
  draft_issue_body: string | null;
  draft_email_body: string | null;
  created_at: string;
}

// Minimal Database type for @supabase/supabase-js typed client.
// Rows mirror the SQL migrations under supabase/migrations/.
//
// IMPORTANT: supabase-js v2's GenericTable constraint requires Row, Insert,
// Update, AND Relationships on every table entry. Omitting Relationships
// causes the table to fail the GenericSchema structural check silently,
// which makes .from()/.select() infer `never` instead of raising an error.
// We have no foreign-key joins wired through the typed client yet, so this
// is an empty array on every table for now.
export interface Database {
  public: {
    Tables: {
      repos: {
        Row: Repo;
        Insert: Partial<Repo> & { full_name: string; html_url: string };
        Update: Partial<Repo>;
        Relationships: [];
      };
      scan_jobs: {
        Row: ScanJob;
        Insert: Partial<ScanJob> & { repo_id: string };
        Update: Partial<ScanJob>;
        Relationships: [];
      };
      findings: {
        Row: Finding;
        Insert: Partial<Finding> & {
          scan_job_id: string;
          repo_id: string;
          type: FindingType;
          title: string;
          description: string;
        };
        Update: Partial<Finding>;
        Relationships: [];
      };
    };
    Views: Record<string, never>;
    Functions: Record<string, never>;
    Enums: Record<string, never>;
    CompositeTypes: Record<string, never>;
  };
}
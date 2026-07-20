import * as cheerioTypes from "cheerio"; // type-only reference; runtime import is dynamic below
import type { Repo } from "./types";

const GITHUB_API = "https://api.github.com";

function githubHeaders() {
  const token = process.env.GITHUB_TOKEN;
  return {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

export interface GithubSearchResult {
  full_name: string;
  html_url: string;
  description: string | null;
  stargazers_count: number;
  language: string | null;
  pushed_at: string;
  fork: boolean;
  archived: boolean;
}

const NINETY_DAYS_MS = 90 * 24 * 60 * 60 * 1000;
const MIN_STARS = 100;

/**
 * Live search against the GitHub Search API, filtered to:
 * - >100 stars
 * - pushed within the last 90 days
 * - excludes forks and archived repos
 * SECURITY.md restrictiveness is checked separately (checkSecurityMd)
 * since it requires a second API call per repo.
 */
export async function searchGithubRepos(query: string): Promise<GithubSearchResult[]> {
  if (!query || query.trim().length === 0) return [];

  const cutoff = new Date(Date.now() - NINETY_DAYS_MS)
    .toISOString()
    .split("T")[0];

  const q = `${query} in:name,description,readme stars:>=${MIN_STARS} pushed:>=${cutoff} fork:false archived:false`;

  const url = `${GITHUB_API}/search/repositories?q=${encodeURIComponent(
    q
  )}&sort=stars&order=desc&per_page=25`;

  const res = await fetch(url, {
    headers: githubHeaders(),
    next: { revalidate: 0 },
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(`GitHub search failed (${res.status}): ${body}`);
  }

  const json = await res.json();
  return (json.items ?? []) as GithubSearchResult[];
}

/**
 * Fetches SECURITY.md (if present) for a repo and does a lightweight
 * heuristic check for "restrictive" disclosure policies — e.g. policies
 * that explicitly forbid public disclosure or third-party scanning.
 * Also extracts a security contact email if one is published.
 */
export async function checkSecurityMd(
  fullName: string
): Promise<{ isRestrictive: boolean; contact: string | null }> {
  const url = `${GITHUB_API}/repos/${fullName}/contents/SECURITY.md`;

  const res = await fetch(url, {
    headers: githubHeaders(),
    next: { revalidate: 3600 },
  });

  if (res.status === 404) {
    return { isRestrictive: false, contact: null };
  }

  if (!res.ok) {
    // Fail open: if we can't read it, don't block the repo from search results.
    return { isRestrictive: false, contact: null };
  }

  const json = await res.json();
  const content = Buffer.from(json.content ?? "", "base64").toString("utf-8");
  const lower = content.toLowerCase();

  const restrictivePhrases = [
    "do not scan",
    "do not perform automated",
    "no automated testing",
    "no third-party scanning",
    "not permitted to scan",
  ];

  const isRestrictive = restrictivePhrases.some((p) =>
    lower.includes(p)
  );

  const emailMatch = content.match(
    /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/
  );

  const contact = emailMatch ? emailMatch[0] : null;

  return { isRestrictive, contact };
}

/**
 * Scrapes GitHub's public trending page (all languages, unfiltered).
 * No official API exists for trending, so this parses the HTML directly.
 * Called from app/api/trending/route.ts, which wraps this with Next.js
 * fetch caching (revalidate every few hours) — this function itself is
 * a plain scrape with no caching logic of its own.
 */
export async function scrapeTrendingRepos(): Promise<
  { full_name: string; html_url: string; description: string | null }[]
> {
  const res = await fetch("https://github.com/trending", {
    headers: {
      "User-Agent":
        "PatchScout/0.1 (+https://github.com/MuaazTasawar/patchscout)",
    },
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch trending page (${res.status})`);
  }

  const html = await res.text();
  const cheerio = await import("cheerio");
  const $ = cheerio.load(html);

  const results: {
    full_name: string;
    html_url: string;
    description: string | null;
  }[] = [];

  $("article.Box-row").each((_, el) => {
    const anchor = $(el).find("h2 a");
    const href = anchor.attr("href"); // "/owner/repo"

    if (!href) return;

    const fullName = href.replace(/^\//, "").trim();
    const description = $(el).find("p.col-9").text().trim() || null;

    results.push({
      full_name: fullName,
      html_url: `https://github.com${href}`,
      description,
    });
  });

  return results;
}
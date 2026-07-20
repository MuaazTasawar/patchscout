import { NextRequest, NextResponse } from "next/server";
import { searchGithubRepos, checkSecurityMd } from "@/lib/github";

export async function GET(req: NextRequest) {
  const query = req.nextUrl.searchParams.get("q");

  if (!query || query.trim().length === 0) {
    return NextResponse.json({ error: "Missing query parameter 'q'" }, { status: 400 });
  }

  try {
    const rawResults = await searchGithubRepos(query);

    // Check SECURITY.md for each result and drop restrictive ones.
    // Runs in parallel but capped to avoid hammering GitHub's rate limit.
    const withSecurity = await Promise.all(
      rawResults.slice(0, 25).map(async (repo) => {
        const security = await checkSecurityMd(repo.full_name);
        return { ...repo, security };
      })
    );

    const filtered = withSecurity
      .filter((repo) => !repo.security.isRestrictive)
      .map((repo) => ({
        full_name: repo.full_name,
        html_url: repo.html_url,
        description: repo.description,
        stars: repo.stargazers_count,
        language: repo.language,
        pushed_at: repo.pushed_at,
        security_contact: repo.security.contact,
      }));

    return NextResponse.json({ query, results: filtered });
  } catch (err) {
    console.error("GET /api/search failed:", err);
    return NextResponse.json({ error: "GitHub search failed" }, { status: 502 });
  }
}
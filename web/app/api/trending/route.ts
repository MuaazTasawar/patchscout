import { NextResponse } from "next/server";
import { scrapeTrendingRepos } from "@/lib/github";
import { supabaseServer } from "@/lib/supabaseServer";

// Revalidate every 4 hours instead of a fixed cron — simplest to implement
// first per the project spec. Can move to scheduled scraping later.
export const revalidate = 14400;

export async function GET() {
  try {
    const trending = await scrapeTrendingRepos();

    // Pull any previously-found issues/CVEs for these repos so the
    // homepage is never empty even right after a cold cache.
    const fullNames = trending.map((t) => t.full_name);

    const { data: knownRepos, error } = await supabaseServer
      .from("repos")
      .select("id, full_name, stars, language")
      .in("full_name", fullNames.length > 0 ? fullNames : [""]);

    if (error) throw error;

    const findingCounts: Record<string, number> = {};
    if (knownRepos && knownRepos.length > 0) {
      const repoIds = knownRepos.map((r) => r.id);
      const { data: findings } = await supabaseServer
        .from("findings")
        .select("repo_id")
        .in("repo_id", repoIds);

      for (const f of findings ?? []) {
        findingCounts[f.repo_id] = (findingCounts[f.repo_id] ?? 0) + 1;
      }
    }

    const enriched = trending.map((t) => {
      const known = knownRepos?.find((r) => r.full_name === t.full_name);
      return {
        ...t,
        stars: known?.stars ?? null,
        language: known?.language ?? null,
        known_findings_count: known ? findingCounts[known.id] ?? 0 : 0,
      };
    });

    return NextResponse.json({ repos: enriched });
  } catch (err) {
    console.error("GET /api/trending failed:", err);
    return NextResponse.json(
      { error: "Failed to load trending repos", repos: [] },
      { status: 500 }
    );
  }
}
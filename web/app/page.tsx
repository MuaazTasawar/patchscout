import SearchBar from "@/components/SearchBar";
import RepoCard from "@/components/RepoCard";

interface TrendingRepo {
  full_name: string;
  html_url: string;
  description: string | null;
  stars: number | null;
  language: string | null;
  known_findings_count: number;
}

async function getTrending(): Promise<TrendingRepo[]> {
  const base = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";
  const res = await fetch(`${base}/api/trending`, { next: { revalidate: 14400 } });
  if (!res.ok) return [];
  const json = await res.json();
  return json.repos ?? [];
}

export default async function HomePage() {
  const trending = await getTrending();

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-4 items-center text-center py-8">
        <h1 className="text-3xl font-semibold font-mono">
          patch<span className="text-accent">scout</span>
        </h1>
        <p className="text-text-secondary max-w-lg">
          Search public repos, scan them for known CVEs and code-pattern bugs,
          and turn confirmed findings into review-ready PRs and issues.
        </p>
        <div className="w-full max-w-xl">
          <SearchBar />
        </div>
      </section>

      <section>
        <h2 className="text-sm font-mono text-text-secondary mb-3 uppercase tracking-wide">
          Trending on GitHub
        </h2>
        {trending.length === 0 ? (
          <p className="text-text-muted text-sm">
            Trending repos unavailable right now — try a search instead.
          </p>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {trending.map((repo) => (
              <RepoCard
                key={repo.full_name}
                fullName={repo.full_name}
                htmlUrl={repo.html_url}
                description={repo.description}
                stars={repo.stars}
                language={repo.language}
                knownFindingsCount={repo.known_findings_count}
              />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
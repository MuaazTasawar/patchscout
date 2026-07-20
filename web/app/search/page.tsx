"use client";

import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import SearchBar from "@/components/SearchBar";
import RepoCard from "@/components/RepoCard";
import ScanQueueButton from "@/components/ScanQueueButton";

interface SearchResult {
  full_name: string;
  html_url: string;
  description: string | null;
  stars: number;
  language: string | null;
  pushed_at: string;
  security_contact: string | null;
}

export default function SearchPage() {
  const params = useSearchParams();
  const query = params.get("q") ?? "";

  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!query) return;
    setLoading(true);
    setError(null);
    setSelected(new Set());

    fetch(`/api/search?q=${encodeURIComponent(query)}`)
      .then((res) => res.json())
      .then((json) => {
        if (json.error) {
          setError(json.error);
          setResults([]);
        } else {
          setResults(json.results ?? []);
        }
      })
      .catch(() => setError("Search failed. Try again."))
      .finally(() => setLoading(false));
  }, [query]);

  function toggle(fullName: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(fullName)) next.delete(fullName);
      else next.add(fullName);
      return next;
    });
  }

  function selectAll() {
    setSelected(new Set(results.map((r) => r.full_name)));
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="max-w-xl">
        <SearchBar initialQuery={query} />
      </div>

      {loading && <p className="text-text-muted text-sm">Searching GitHub...</p>}
      {error && <p className="text-danger text-sm">{error}</p>}

      {!loading && !error && results.length > 0 && (
        <>
          <div className="flex items-center justify-between">
            <p className="text-text-secondary text-sm">
              {results.length} result{results.length > 1 ? "s" : ""} for "{query}"
            </p>
            <div className="flex items-center gap-3">
              <button
                onClick={selectAll}
                className="text-xs text-accent hover:text-accent-hover font-mono"
              >
                select all
              </button>
              <ScanQueueButton selectedRepos={Array.from(selected)} allResults={results} />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {results.map((repo) => (
              <RepoCard
                key={repo.full_name}
                fullName={repo.full_name}
                htmlUrl={repo.html_url}
                description={repo.description}
                stars={repo.stars}
                language={repo.language}
                selected={selected.has(repo.full_name)}
                onToggleSelect={() => toggle(repo.full_name)}
              />
            ))}
          </div>
        </>
      )}

      {!loading && !error && query && results.length === 0 && (
        <p className="text-text-muted text-sm">No repos matched your filters.</p>
      )}
    </div>
  );
}
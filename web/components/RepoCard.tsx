"use client";

interface RepoCardProps {
  fullName: string;
  htmlUrl: string;
  description: string | null;
  stars?: number | null;
  language?: string | null;
  knownFindingsCount?: number;
  selected?: boolean;
  onToggleSelect?: () => void;
}

export default function RepoCard({
  fullName,
  htmlUrl,
  description,
  stars,
  language,
  knownFindingsCount = 0,
  selected = false,
  onToggleSelect,
}: RepoCardProps) {
  return (
    <div
      className={`card p-4 flex flex-col gap-2 transition-colors ${
        selected ? "border-accent" : "hover:border-text-muted"
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <a
          href={htmlUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="font-mono text-sm text-text-primary hover:text-accent transition-colors truncate"
        >
          {fullName}
        </a>

        {onToggleSelect && (
          <input
            type="checkbox"
            checked={selected}
            onChange={onToggleSelect}
            className="accent-accent shrink-0"
          />
        )}
      </div>

      {description && (
        <p className="text-sm text-text-secondary line-clamp-2">
          {description}
        </p>
      )}

      <div className="flex items-center gap-3 text-xs text-text-muted font-mono mt-1">
        {typeof stars === "number" && (
          <span>★ {stars.toLocaleString()}</span>
        )}

        {language && <span>{language}</span>}

        {knownFindingsCount > 0 && (
          <span className="badge badge-medium">
            {knownFindingsCount} known finding
            {knownFindingsCount > 1 ? "s" : ""}
          </span>
        )}
      </div>
    </div>
  );
}
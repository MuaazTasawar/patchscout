"use client";

import { useState, FormEvent } from "react";
import { useRouter } from "next/navigation";

export default function SearchBar({ initialQuery = "" }: { initialQuery?: string }) {
  const [value, setValue] = useState(initialQuery);
  const router = useRouter();

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (value.trim().length === 0) return;
    router.push(`/search?q=${encodeURIComponent(value.trim())}`);
  }

  return (
    <form onSubmit={handleSubmit} className="flex gap-2 w-full">
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="Search a topic, language, or repo name..."
        className="flex-1 bg-surface border border-border rounded-md px-4 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:border-accent"
      />
      <button
        type="submit"
        className="bg-accent hover:bg-accent-hover text-white text-sm font-medium px-5 py-2 rounded-md transition-colors"
      >
        Search
      </button>
    </form>
  );
}
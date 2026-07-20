import Link from "next/link";

export default function Navbar() {
  return (
    <header className="border-b border-border bg-surface/60 backdrop-blur sticky top-0 z-50">
      <div className="max-w-6xl mx-auto px-4 h-14 flex items-center justify-between">
        <Link href="/" className="font-mono font-semibold text-text-primary tracking-tight">
          patch<span className="text-accent">scout</span>
        </Link>
        <nav className="flex items-center gap-6 text-sm text-text-secondary">
          <Link href="/" className="hover:text-text-primary transition-colors">
            Home
          </Link>
          <Link href="/search" className="hover:text-text-primary transition-colors">
            Search
          </Link>
          <Link href="/dashboard" className="hover:text-text-primary transition-colors">
            Review Queue
          </Link>
        </nav>
      </div>
    </header>
  );
}
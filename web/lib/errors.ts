// Small shared error-handling helpers used across API routes, so every
// route returns errors in a consistent { error: string } shape instead of
// each route inventing its own.

import { NextResponse } from "next/server";

export class ApiError extends Error {
  status: number;

  constructor(message: string, status = 500) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export function errorResponse(err: unknown, fallbackMessage = "Something went wrong") {
  if (err instanceof ApiError) {
    console.error(`[${err.status}] ${err.message}`);
    return NextResponse.json({ error: err.message }, { status: err.status });
  }

  if (err instanceof Error) {
    console.error(err.message, err.stack);
    return NextResponse.json({ error: fallbackMessage }, { status: 500 });
  }

  console.error("Unknown error shape:", err);
  return NextResponse.json({ error: fallbackMessage }, { status: 500 });
}

// Supabase's PostgrestError shape (duck-typed rather than imported, since
// we only need to check for the `message`/`code` fields it carries).
export function isSupabaseError(err: unknown): err is { message: string; code?: string } {
  return typeof err === "object" && err !== null && "message" in err;
}
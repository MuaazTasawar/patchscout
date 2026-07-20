"use client";

import { createClient } from "@supabase/supabase-js";
import type { Database } from "./types";

const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL as string;
const supabaseAnonKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY as string;

if (!supabaseUrl || !supabaseAnonKey) {
  throw new Error(
    "Missing NEXT_PUBLIC_SUPABASE_URL or NEXT_PUBLIC_SUPABASE_ANON_KEY. Check web/.env.local"
  );
}

// Browser client — used in client components (e.g. realtime subscriptions).
export const supabaseBrowser = createClient<Database>(supabaseUrl, supabaseAnonKey, {
  auth: {
    persistSession: true,
  },
});
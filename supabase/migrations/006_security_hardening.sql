-- Phase 8 (security, part 2): tighten function-level exposure per the
-- Supabase security advisor.
--
-- notify_scan_worker should only ever fire as a trigger on scan_jobs
-- insert — it should never be directly callable via the PostgREST RPC
-- endpoint (/rest/v1/rpc/notify_scan_worker) by anon/authenticated roles.
-- PostgreSQL grants EXECUTE on new functions to PUBLIC by default, and
-- anon/authenticated inherit through that pseudo-role, so PUBLIC must be
-- revoked explicitly, not just the named roles.
revoke execute on function notify_scan_worker() from public;
revoke execute on function notify_scan_worker() from anon;
revoke execute on function notify_scan_worker() from authenticated;

-- Pin search_path on SECURITY DEFINER-adjacent functions so they can't be
-- tricked by a malicious search_path into resolving an attacker-controlled
-- object instead of the intended one.
alter function set_updated_at() set search_path = '';
alter function notify_scan_worker() set search_path = '';
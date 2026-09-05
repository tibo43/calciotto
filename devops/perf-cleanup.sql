-- One-off cleanup for the perf load test (cmd/perfsetup + frontend/loadtest/run.js).
-- Run with: psql "$DATABASE_URL" -v group_id="'<the group id printed by perfsetup>'" -f devops/perf-cleanup.sql
--
-- Deletes, in FK-safe order (MatchPlayer/MatchRegistration/MatchVote have no
-- ON DELETE CASCADE to matches — see CLAUDE.md's MatchService.DeleteMatch
-- notes — so children go first):
--   match_votes, match_registrations, match_players  (scoped to the group's matches)
--   matches, group_memberships, teams                (scoped to the group)
--   groups                                            (the group itself)
--   password_reset_tokens, players                    (the perf admin + all
--                                                       "perfload+uN@..." test
--                                                       accounts, by email pattern)
--
-- Nothing here touches a row outside this group id or this email pattern —
-- real players/groups/matches are never in scope.

\echo 'Target group id:' :group_id
\echo 'Test account email pattern: perfload%@perfload.test'

-- ---------------------------------------------------------------------------
-- Step 1: verify before deleting anything. Re-run this block on its own (it
-- is read-only) until the counts look like what you expect, before running
-- the DELETE transaction below.
-- ---------------------------------------------------------------------------

select 'group'            as what, count(*) from groups             where id = :group_id::uuid
union all
select 'teams'            as what, count(*) from teams              where group_id = :group_id::uuid
union all
select 'matches'          as what, count(*) from matches             where group_id = :group_id::uuid
union all
select 'group_memberships' as what, count(*) from group_memberships where group_id = :group_id::uuid
union all
select 'match_players'    as what, count(*) from match_players
  where match_id in (select id from matches where group_id = :group_id::uuid)
union all
select 'match_registrations' as what, count(*) from match_registrations
  where match_id in (select id from matches where group_id = :group_id::uuid)
union all
select 'match_votes'      as what, count(*) from match_votes
  where match_id in (select id from matches where group_id = :group_id::uuid)
union all
select 'test players'     as what, count(*) from players
  where email like 'perfload%@perfload.test';

-- ---------------------------------------------------------------------------
-- Step 2: the actual cleanup. Uncomment (remove the leading "-- ") and run
-- once the counts above look right. Wrapped in a transaction so a mistake
-- (wrong group id, typo'd pattern) rolls back cleanly instead of half-applying.
-- ---------------------------------------------------------------------------

-- begin;

-- delete from match_votes
--   where match_id in (select id from matches where group_id = :group_id::uuid);

-- delete from match_registrations
--   where match_id in (select id from matches where group_id = :group_id::uuid);

-- delete from match_players
--   where match_id in (select id from matches where group_id = :group_id::uuid);

-- delete from matches where group_id = :group_id::uuid;

-- delete from group_memberships where group_id = :group_id::uuid;

-- delete from teams where group_id = :group_id::uuid;

-- delete from groups where id = :group_id::uuid;

-- delete from password_reset_tokens
--   where player_id in (select id from players where email like 'perfload%@perfload.test');

-- delete from players where email like 'perfload%@perfload.test';

-- commit;

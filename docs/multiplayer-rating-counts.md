# Multiplayer completion rating counts

The October 3, 2026 MGN emulator test exercised two three-player groups and four
two-player groups, including concurrent rooms and helper re-summons. All ten
summon handoffs reached their intended players and all RPCN joins succeeded.

One completed three-player group sent three `finalizeMultiPlay.spd` requests,
each with `gradeS=2` and the other grade fields zero. The previous implementation
acknowledged these requests but selected `GradeUnknown`, because it only accepted
an individual grade field equal to one. No ratings were recorded. The operator
confirmed that completing three-player multiplayer allows each player to rate
two other players; a two-player completion permits one rating.

The same single-rating assumption appears in the earlier
[DeSSE implementation](https://github.com/ymgve/desse/blob/master/emu/PlayerManager.py).
The new behavior follows the captured reports and operator confirmation.

## Behavior

- Completion fields `gradeS` through `gradeD` are independent count increments.
  Two S ratings add two to S; one S and one A add one to each bucket.
- All five increments are applied with one parameterized SQLite statement.
  Concurrent reports add to existing values without overwriting them.
- Zero ratings remain a successful no-op. Negative counts or counts beyond the
  signed 32-bit statistics format receive HTTP 400 before a write.
- Updates that would overflow an existing grade total fail without changing any
  grade bucket. A nonzero report for a missing character also fails instead of
  being silently acknowledged.
- The single-rating `updateOtherPlayerGrade.spd` path uses the same update logic
  with an increment of one. Its grade mapping remains unchanged.
- The completion response remains command `0x21`, total length six, success byte
  one, Base64 encoded with the required trailing newline. A failed database write
  is not acknowledged as a successful completion.
- Completion logs record each applied bucket count explicitly.

The patch does not change session counters, room tokens, summon discovery,
polling timing, regional separation, or the database schema. It does not backfill
historical ratings. The request does not supply a reliable completion identifier,
so duplicate-report deduplication remains a separate protocol investigation.

## Regression coverage and next acceptance test

Tests cover the anonymized captured three-player form, ordinary two-player
ratings, mixed ratings, zero ratings, invalid counts, failed writes, database
atomicity, concurrent updates, character isolation, and overflow boundaries.

For the next gameplay test, record grade totals before starting. Complete one
two-player run and one three-player run, deliberately select mixed ratings in
the latter, then read the totals again. Confirm that the delta for each character
matches the number and distribution of ratings actually received. Native PS3
acceptance remains useful independently of emulator acceptance.

Later group departures in the capture did not send completion reports; two were
preceded by host bloodstain uploads. There is no evidence that this post-completion
statistics bug caused a summon failure or prevented a boss clear.

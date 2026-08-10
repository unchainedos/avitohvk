---
name: deal-chain
description: Verify a change to avitohvk's deal/proposal/chown code preserves its five chain-deal invariants (right-to-left approval order, when an item locks to its deal, deal-beats-chain, cascading cancellation, deadlock retries, same-deal serialization) before considering it done. Use whenever writing or reviewing a diff touching internal/{repository,service,transport/handler}/{deal,proposal,chown}.
---

# Deal / proposal / chown invariant checklist

Five invariants, each backed by one specific piece of code. For any diff touching
`deal`, `proposal`, or `chown`, check it against every invariant below that the
diff's code path can reach — don't just eyeball it for "looks reasonable." Function
names are the stable anchor; line numbers are current as of this skill's writing
and will drift.

## 1. Right-to-left approval order

`isApprovalAllowed` in `internal/repository/proposal/postgres.go` (called from
`SetStatus` right before it accepts a proposal) allows approval only if one of
three is true: the caller is `cd.creator_id`, the caller offered the root item
(`t.item_id = cd.root_item_id`), or the caller's own recipient already has an
`ACCEPTED` proposal in the same deal **and** the root item is already locked
(`ri.is_locked`).

If you touch this query or its caller: creator and root-item-proposer must still
be able to approve at any time before the chain locks; every other participant
must still require both "chain is locked" and "my recipient already accepted."
`ErrOutOfOrder` (mapped to a 409 in `service.Approve`) is the signal this guard
fired — a test that expects early approval to succeed for a non-exempt
participant should fail against this function.

## 2. An item locks to its deal the moment its own proposal is accepted

`lockOfferedItem` in the same file runs inside `SetStatus`, gated on
`status == ProposalStatusAccepted`, and runs *before* the whole-chain lock
(`TryLockChain`) ever fires. If you refactor `SetStatus` or move locking later in
the flow, you reopen the bug this exists to prevent: the creator/root-holder (the
two roles allowed to approve early, per invariant 1) could `chown` their promised
item away before the chain locks, and completion (`Approve`'s `AllAccepted` branch
in `internal/service/proposal/service.go`) would silently overwrite the real
holder with a stale cached recipient.

## 3. Deal beats chain, deal beats deal

`checkItemHolder` in `internal/repository/proposal/postgres.go` is the actual
enforcement point (reached from both `props.CreateProposal` and `chown.Chown`,
since chown's `Chown` in `internal/service/chown/service.go` calls straight into
the same `proposals.CreateDeal`/`CreateProposal`):

- `is_locked = true` **and** `locked_by_deal_id IS NOT NULL` → `ErrItemLocked`,
  the offer is rejected. A real deal lock always wins.
- `is_locked = true` **and** `locked_by_deal_id IS NULL` (a bare chown claim) →
  silently released and overwritten by the new offer.

`lockOfferedItem`'s own CTE (invariant 2) and `TryLockChain`'s `qCancelCompeting`
CTE are the other half of "deal beats deal": both cancel any other still-`PENDING`
deal that references the same item, before or as they lock it for the winning
deal. If you add a new code path that locks an item, it must go through one of
these two, not a bare `UPDATE items SET is_locked = ...`.

## 4. Cascading cancellation releases only what this deal holds

Three call sites all follow the same three-step shape — `UpdateStatus(...,
CANCELLED)`, decline the deal's own proposals, then `UnlockAllForDeal`:

- `WithdrawProposal` (`service.go`): `DeclineAllExcept(dealID, actorID)` — every
  other proposal in the deal, not the withdrawer's own (already set separately).
- `ensureDealOpen`'s deadline-expiry branch (`service.go`): `DeclineAllForDeal` —
  no actor to exclude, the deadline hit for everyone.
- The competing-deal cancellation CTEs in `lockOfferedItem` and `TryLockChain`'s
  `qCancelCompeting` (`internal/repository/proposal/postgres.go`): decline that
  deal's transactions, then unlock `WHERE locked_by_deal_id IN (SELECT id FROM
  canceled)`.

That `locked_by_deal_id = <this deal>` scoping on the unlock step is load-bearing
— `UnlockAllForDeal` and both CTEs key off `locked_by_deal_id`, never off "any
item this deal's transactions mention." An item merely mentioned in a cancelled
deal's transaction history can belong to an unrelated chown claim or a different,
still-open deal; unlocking by transaction membership instead of
`locked_by_deal_id` would release something this deal never actually held.

## 5. Cross-deal work goes through `retryOnDeadlock`

`retryOnDeadlock` + `isDeadlock` (checks Postgres error code `40P01`) live in
`internal/service/proposal/service.go`. `Approve` wraps both the
`SetStatus(..., ProposalStatusAccepted)` call and the `TryLockChain` call in it —
both reach into another deal's rows via the CTEs from invariant 3/4, so both can
lose a deadlock race against a concurrent close of that other deal. If you add a
new call that can touch another deal's rows (another CTE, another cross-deal
`UPDATE`), it needs `retryOnDeadlock` too — the existing 20-attempt/jittered-sleep
budget silently does nothing for a call that isn't wrapped.

## 6. Same-deal mutations serialize via `LockDeal`

`DealRepository.LockDeal` (`internal/repository/deal/postgres.go`) takes a
Postgres advisory lock keyed on `hashtextextended(dealID, 0)` on a dedicated
connection acquired from the pool, released via a `defer release(context.
WithoutCancel(ctx))` so the unlock still runs even if the request context was
already cancelled. `CreateProposal`, `WithdrawProposal`, and `Approve` in
`service.go` all take this lock for the full duration of the request. Any new
service method that mutates a deal's proposals must take this lock too, or it can
race the three that already do (e.g. an unguarded new method racing `Approve`
could leave an `ACCEPTED` proposal attached to a deal that `WithdrawProposal`
just cancelled).

## Verifying

There's no standing regression script for this in the repo — verify with the
normal Go test suite:

```bash
go test ./internal/repository/proposal/... ./internal/repository/deal/... ./internal/service/proposal/... ./internal/service/chown/... -race
```

See the `go-testing` skill for this project's testing conventions (fakes, table-
driven tests, `dbtest`) if you're adding a new case for one of these invariants.

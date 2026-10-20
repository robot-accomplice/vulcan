# Decision 3: contract failure and external effects

Status: accepted, 2026-10-20 (option C; `on_contract_failure` defaults to `panic`). Applied in
spec §5.5.0, §5.5.1, §5.5.3, §6, §7.4, §10, Appendices B, C, and E, and the primer. Milestone 0, item 3 of the
[development plan](../development-plan.md). Answers [audit 2](../../experiments/audit-2.md), H1
and H2, and the plan's rows "Audit and ledger inconsistency" and "Refund overflow".

## Question

What does a contract failure do to effects that already happened: a database write, an HTTP
call, a published message? Which checks prevent an effect, and which only detect it? What must
the application do for transactions and idempotency, and what does the language give it?

## Current rule

- A contract that fails at runtime panics with a `*keel.ContractFailure` (§7). Decision 1 (§7.3)
  defines where the checks run.
- `requires` runs at entry. `ensures` and the invariant run at each `return`, before the deferred
  calls (§7.3).
- The runtime recovers a panic in a topic handler. The message is not acknowledged, and the
  transport applies its policy: by default, `direct` retries 5 times with backoff, then sends the
  message to its dead-letter route (§5.5.0, §5.5.1). A received message that fails its invariant
  or a `limit` goes to the dead-letter route at once.
- A panic in a served remote call reaches the caller as `keel.RemotePanic` (§5.5.3).
- "A state change and a publish are not atomic" (§5.5). The spec names the outbox pattern as the
  developer's choice.

## Problems

1. **The spec does not say that nothing is undone.** A panic runs the deferred calls of each frame,
   as in Go. Nothing else is undone. The spec does not state this, so a reader can believe that a
   failed contract cancels the operation.
2. **A commit in a `return` runs before the check.** In `return n, tx.Commit()`, Go evaluates
   `tx.Commit()` before the return check. If `ensures` then fails, the data is committed, and the
   deferred `tx.Rollback()` does nothing. If the check runs before the commit, the deferred
   rollback undoes the work. A test program (Go 1.26.1) confirmed both orders.
3. **A handler that fails a contract repeats its effects.** A contract failure is a bug in the
   code or the data, so it usually fails again on each retry. With the default `direct` policy, a
   handler that charges a card and then fails its `ensures` charges the card 6 times, then
   dead-letters the message.
4. **A remote contract failure leaves the outcome unknown.** The callee checks `ensures` after
   its body ran, and the caller checks it again after the reply. The caller receives
   `keel.RemotePanic` or panics itself. In both cases the callee's effects can have happened. The
   spec says this for a timeout, but not for a panic.
5. **A port's `ensures` fails after the provider's effects.** The wrapper of a port checks
   `ensures` in the caller after the provider returns (Appendix F). A failure there reports a bug
   in the provider after the provider committed.
6. **A recovered failure keeps the process running with bad state.** `net/http` recovers each
   handler panic, including a contract failure, and serves the next request with the same values.
   Decision 1 states this limit for invariants.
7. **Handlers have no delivery identity.** `idempotent` is required on handlers of a transport that
   redelivers, but the compiler cannot check it (§5.5). The runtime gives a handler no stable ID
   of the message, so the handler must dedupe on its own business key, or not at all.

## Constraints

- **Bodies stay Go** (design rule 4). A contract failure can change control flow, because it
  happens only when a contract is false. Nothing can change the effects of a correct body.
- **Mechanism, not policy** (design rule 1). Retry, transaction, and process policy belong to the
  developer, with defaults.
- **The language does not own the external systems.** A database, a payment provider, or a
  broker decides what is atomic. Keel cannot make two systems commit together.
- **Distributed features are deferred** (development plan, milestone 5). Rules for topics and
  remote calls can be decided now and built later.

## Options

**A. Document only.** State that nothing is undone, and list the responsibilities of the
application. Cost: low. Problems 2, 3, 6, and 7 stay.

**B. Language transactions.** Add a construct that groups effects and undoes them on a contract
failure, for example a `transaction` block with compensation functions. This is a policy, it
needs the cooperation of each external system, and it changes what Go code means. It contradicts
design rules 1 and 4.

**C. Document, and add four small mechanisms (recommended).** State the no-rollback rule and the
responsibilities, and add: a warning for effects in a `return`, no retry of contract failures by
default, a fail-stop process option, and a delivery ID for handlers.

## Recommendation: option C

### Rules

1. **Nothing is undone.** A contract failure is a Go panic. Each frame runs its deferred calls,
   as in Go, so a deferred `tx.Rollback()` runs. Keel undoes nothing else. An effect that happened
   before the failed check stays: a committed write, a sent request, a published message.
2. **`requires` prevents; `ensures` and the invariant detect.** Only `requires` runs before the
   body's effects. A rule that must stop an effect goes in the `requires` of the function that
   makes the effect, or in a check in the body before it. `keel audit` lists each `ensures` and
   each invariant of a function that reaches an effect before its return check, marked "detects
   after effects". An effect is a call that reaches a network or acquiring function (Appendix D),
   a port call, or a publish.
3. **Warning: an effect in a `return` (new K0525).** In a function with `ensures`, or a method of a
   struct with an invariant, a `return` whose expressions reach an effect gets a warning: the
   effect runs before the check. The hint shows how to check first, then make the effect:

   ```go
   n := amt
   if err := validate(n, amt); err != nil { return 0, err }   // or a function with ensures
   return n, tx.Commit()                                      // K0525 stays, for review
   ```

   The warning stays when the developer accepts the order, so that a reviewer sees it.
4. **Contract failures in handlers are not retried by default.** When a handler panics with a
   `*keel.ContractFailure`, the runtime sends the message to its dead-letter route at once, as for
   a message that fails its invariant. `DeadLetter.Error` holds the failure report. A transport
   declares this with a new setting, `retryContractFailures` (for `direct`: default `false`).
   Other panics and errors keep the retry policy. A dead-letter handler that fails a contract is
   counted and reported, with no retry.
5. **A remote contract failure has an unknown outcome.** When the callee fails a contract, the
   caller receives `keel.RemotePanic` with the failure report. When the caller's check of the
   reply fails, the caller panics. In both cases the callee's effects can have happened, as after
   `keel.Timeout`. §5.5.3 states this.
6. **Fail-stop on request.** The process setting `on_contract_failure` takes `panic` (default) or
   `exit`. With `exit`, the runtime writes the failure report to standard error and ends the
   process with exit code 3. It does not run deferred calls, and no caller can recover the
   failure. The `config` key sets the default for the system. The flag
   `--on-contract-failure` sets it for one process. `keel test` always uses `panic`, so that tests
   can check failures. With `exit`, the limit of decision 1, rule 5 does not apply: no later
   request can use the broken value.
7. **A delivery ID for handlers.** `keel.DeliveryID(ctx)` returns an ID of the message for one
   subscriber wire. It stays the same across redeliveries of that message, and it is different for
   each message. A transport with `Redelivers: true` must supply it. A handler uses it as an
   idempotency key, in the same transaction as its effects. Built in milestone 5, with topics.

### Responsibilities of the application

Keel states these in §7 and in the primer. It does not enforce them.

- **One transaction for one change.** Write a state change, its audit line, and each event to
  publish (an outbox row) in one database transaction. Publish from the outbox after the commit.
- **Check before the commit.** Run each check that guards a write before `Commit`. A deferred
  `Rollback` then undoes the work on a failure.
- **Idempotency keys at each retryable entry.** An HTTP `POST`, a remote method that a caller can
  retry, and each handler take a key: a client key, an operation ID in the arguments, or
  `keel.DeliveryID(ctx)`. Store the key and the outcome with the change, in the same transaction.
- **Recover the outcome on a retry.** A retry with a known key returns the stored outcome. It does
  not run the change again. This covers a lost reply after a commit.
- **Overflow-safe arithmetic before a write.** Body arithmetic is Go arithmetic and can wrap.
  Exact arithmetic exists only in contract expressions (§7.1). Write a guard so that it cannot
  overflow (`amount > charged - refunded`, not `refunded + amount > charged`). Put the guard in
  `requires`, or before the write.
- **Choose a process policy.** A container that holds money or other state that must stay
  consistent should consider `on_contract_failure: exit`.

## Examples

| Code | Result |
|---|---|
| `return n, tx.Commit()` in a function with `ensures` | K0525 warning. A failing `ensures` leaves the data committed. |
| Check, then `return n, tx.Commit()`, with `defer tx.Rollback()` | A failing check panics before the commit, and the deferred rollback undoes the work. K0525 still marks the `return`. |
| A handler that calls `Cards.Charge` and then fails its `ensures` | Not retried: one charge, then the dead-letter route, with the report in `Error`. |
| The same handler with `retryContractFailures: true` | Retried by policy. The card can be charged on each attempt unless the handler dedupes on `keel.DeliveryID(ctx)`. |
| A remote callee fails `ensures` after a write | The caller receives `keel.RemotePanic`. The write can be committed. |
| `on_contract_failure: exit` in a container with an HTTP entry leaf | A failure ends the process with code 3. `net/http` cannot recover it. |
| A refund guard `refunded + amount > charged` | Not a Keel error: body arithmetic is Go. The primer and §7.1 show the safe form, and an `ensures` with exact arithmetic detects the wrap, after the effects (rule 2). |

## Spec changes on acceptance

- §7: new §7.4 "Contract failures and effects", with rules 1 to 3 and 6, and the
  responsibilities of the application.
- §5.5.0 and §5.5.1: rule 4, and the `direct` setting `retryContractFailures`.
- §5.5.3: rule 5.
- §6 and Appendix C: `keel.DeliveryID`; the process setting and the exit code of rule 6.
- Appendix B: the config key `on_contract_failure`.
- Appendix E: K0525 (warning).
- §10 `keel audit`: the "detects after effects" list.
- `docs/guide.md`: one sentence on "nothing is undone; check before the commit".

## Open points

- **The default of `on_contract_failure`.** This draft keeps `panic`, which is Go's behavior and
  lets a server keep serving other requests. `exit` is safer for consistent state, and less
  available. Either default is a policy; the setting exists for both.
- **What counts as an effect for rule 2 and K0525.** This draft uses the network and acquiring
  functions, port calls, and publishes. A port call to a pure provider is not an effect, but the
  compiler cannot see the provider from the caller. Milestone 2 should measure the noise.
- **The form of the delivery ID.** A string, unique per subscriber wire. Whether it stays stable
  across a transport migration is a milestone-5 question.

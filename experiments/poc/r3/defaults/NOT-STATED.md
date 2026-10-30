# What `defaults v1` cannot state

Draft, not frozen. `gen_defaults.py` writes this file from the wishlist and `coverage.json`. The wanted text is the wishlist's.

This list is a result of plan-r3, section 4: it says what the four rule kinds cannot state.

| Status | Wanted rules |
|---|---:|
| Stated by the kinds | 43 |
| Stated in part | 13 |
| Not stated | 154 |
| Total | 210 |

`defaults v1` also holds the 4 dependency rules of plan-r3, section 3 (`DEP-LIBRARY`, `DEP-GATEWAY`, `DEP-ENTRY`, and `DEP-LOCK`, which is off). The wishlist does not ask for them.

A rule that the model already enforces and that a kind can state is in the file too: `ARCH-07`, `ARCH-10`, `ARCH-11`, `ARCH-19`, `HTTP-07` in part, and `SEC-30`. If a team deletes one of these, the rule stops firing, but its model check (K0101, K0205, K0125, K0211, K0124) stays. Plan-r3's "no hidden layer" holds for the rules file. It does not switch off the model. See `QN-base-apps.md`.

Two stated rules are wider than their wanted text, because a kind cannot see a purpose. `SEC-10` bans the broken algorithms in all code, not only in security code; its note in the wishlist expects a written exception for a checksum, which here is an edit of `from` or `except`. `DATA-15` bans `time.Now`, `time.Since`, and `time.Until` in all code of a plain leaf or a library, not only for a business decision.

Four stated rules hold a finite list where the wanted text is open or says "such as": `ARCH-19` (Appendix D's global mutators), `CTX-06` (the forms without a context in the standard library), and `ARCH-10` and `ARCH-11` (Appendix D's network functions, with holes H3 and H7).

Stated forms that hold a finite list, or are wider than the words, for the same reason: `ERR-13` reads `errors.New` and `fmt.Errorf` only. `HTTP-13` reads the patterns of `http.ServeMux`, where a method in the pattern gives 405 for the others. `HTTP-17` reads each cookie of an entry leaf, not only a session cookie. `ERR-01` allows a bare `defer` of a listed `Close` or `Rollback`, and it cannot tell a read handle from a write handle. `CON-03` lists its effects and adds each port call. `CODE-08` holds for each exported name, also a struct field and an interface method. `CODE-08` and `CON-01` name no topic, because release 1 has none.

`HTTP-01` and `HTTP-02` were stated in part, with "shape" as the reason for the rest. The kind `literal` states the rest, so both are now stated. They were not among the 39.

## Not stated, by reason

| Reason | Wanted rules | Meaning |
|---|---:|---|
| model | 5 | The model check enforces it, and no rule kind states it, so it is not in the rules file and a team cannot delete it there. |
| flow | 33 | It needs the flow of a value or the paths of a function. `transitions` follows one status field only, and no other kind follows flow. |
| meaning | 54 | Only a person or a reviewing model can judge it. It depends on what the code means. |
| run-time | 16 | Only a test, a probe, or the deployment shows a breach, when the code runs. |
| project-facts | 21 | It needs the facts of one project. A ruleset written before a project exists cannot name them. |
| outside-source | 14 | It is about CI, the repository, or a process, which a check of the source does not read. |
| declaration | 3 | It is about a declaration of the system (a config key or an external). No kind reads declarations. |
| deferred | 8 | It is about topics, remote ports, or transports, which release 1 defers. |

## Stated in part, by the reason for the rest

| Reason | Wanted rules |
|---|---:|
| flow | 1 |
| meaning | 7 |
| project-facts | 1 |
| outside-source | 1 |
| whole-program | 3 |

## The 39 wanted rules of amendment 6

| Rule | Outcome | Entries |
|---|---|---|
| ARCH-03 | stated | `ARCH-03` |
| ARCH-06 | stated | `ARCH-06` |
| ARCH-21 | stated | `ARCH-21` |
| ERR-01 | stated | `ERR-01` |
| ERR-03 | stated | `ERR-03` |
| ERR-04 | stated | `ERR-04` |
| ERR-09 | stated | `ERR-09` |
| ERR-13 | stated | `ERR-13` |
| CTX-01 | stated | `CTX-01` |
| CTX-02 | stated | `CTX-02` |
| CTX-03 | stated | `CTX-03` |
| CTX-07 | in part (meaning) | `CTX-07` |
| CTX-08 | in part (whole-program) | `CTX-08` |
| HTTP-09 | in part (meaning) | `HTTP-09` |
| HTTP-11 | stated | `HTTP-11` |
| HTTP-13 | stated | `HTTP-13` |
| HTTP-17 | stated | `HTTP-17` |
| SEC-01 | in part (meaning) | `SEC-01` |
| SEC-02 | in part (meaning) | `SEC-02` |
| SEC-07 | stated | `SEC-07` |
| DATA-03 | stated | `DATA-03` |
| DATA-14 | in part (meaning) | `DATA-14` |
| PAY-01 | in part (project-facts) | `PAY-01` |
| RES-02 | in part (flow) | `RES-02` |
| CONC-03 | stated | `CONC-03` |
| OBS-02 | stated | `OBS-02` |
| CON-01 | in part (meaning) | `CON-01` |
| CON-02 | not stated (meaning) |  |
| CON-03 | stated | `CON-03` |
| CODE-02 | stated | `CODE-02` |
| CODE-03 | stated | `CODE-03` |
| CODE-04 | stated | `CODE-04` |
| CODE-05 | stated | `CODE-05.json`, `CODE-05.xml` |
| CODE-06 | stated | `CODE-06` |
| CODE-08 | stated | `CODE-08` |
| CODE-09 | in part (whole-program) | `CODE-09` |
| CODE-11 | in part (whole-program) | `CODE-11.deprecated`, `CODE-11.unused` |
| TEST-04 | stated | `TEST-04` |
| BLD-07 | in part (meaning) | `BLD-07` |

Of the 39: 26 stated, 12 in part, 1 not stated.

## Stated in part

**CTX-07.** Context values use an unexported key type and hold only request-scoped data, not optional parameters.

- Reason: meaning. The unexported key type is stated. Whether a value holds only request-scoped data is for a person to judge.

**CTX-08.** A port method whose provider reaches a network or acquiring function takes `context.Context` as its first parameter.

- Reason: whole-program. A port that a gateway provides is stated. A plain leaf that reaches a network or an acquiring function itself, and a provider that reaches one through its own ports, need what a provider reaches, and L's analyzers read one package at a time.

**HTTP-09.** A gateway does not follow a redirect automatically, unless it checks the target against an allow-list.

- Reason: meaning. That a client sets `CheckRedirect` is stated. Whether that function checks the target against an allow-list is for a person to judge.

**SEC-01.** Source files, declaration files, and test fixtures hold no real secret, key, or password.

- Reason: meaning. A string in the code that has the form of a key, or that a name marks as a secret, is stated. Declaration files and fixtures are not read, and only a person can say that a value is real.

**SEC-02.** A settings field that holds a credential has the tag `vulcan:"secret"`.

- Reason: meaning. A settings field whose name shows a credential is stated. A credential under another name is not found, as the wishlist's note says.

**DATA-14.** Code uses `time.Time` for instants and `time.Duration` for periods, not integers or strings.

- Reason: meaning. A value whose name shows a time or a period is stated. A value under another name is not found, as the wishlist's note says.

**PAY-01.** A money amount is an integer in the currency's smallest unit, or an exact decimal, and is never `float32` or `float64`.

- Reason: project-facts. A float whose name shows money is stated. The name of the project's money type would give a stronger check, as the wishlist's note says.

**RES-02.** A retry loop has a limit on attempts, exponential backoff, and jitter.

- Reason: flow. A loop that calls `time.Sleep` is stated: it needs a condition, a growing wait, and a random value in its function. A retry in another form is not found, as the wishlist's note says, and that the wait grows or is random is read from the form only.

**API-04.** A production build exposes no debug or profiling route, and each old API version has an owner and an end date.

- Reason: outside-source. The ban on the debug and profiling packages is stated, in every build. An owner and an end date for each old API version are not in the source.

**CON-01.** A port method that changes state declares a `requires` for the limits of its arguments.

- Reason: meaning. That a clause exists is stated, on each port method that takes arguments and whose name does not show a read. Whether a method changes state, and whether a clause is enough, are for a person to judge.

**CODE-09.** Near-duplicate functions become one function, in the leaf or in a library.

- Reason: whole-program. Near-duplicates in one package are stated. A duplicate in another package needs a view of the whole module, and L's analyzers read one package at a time.

**CODE-11.** Code does not use a deprecated function, and has no unused name.

- Reason: whole-program. A use of a deprecated function, and an unexported name at package level that nothing uses, are stated. An unused exported name needs a view of the whole module, and an unused method or field is not read.

**BLD-07.** Each deployment-specific value comes from a flag or from the environment, and leaf code holds no address or hostname literal.

- Reason: meaning. An address or a host name in a string of a leaf is stated. Whether any other value is specific to a deployment is for a person to judge.

## model

- **ARCH-04.** A port signature carries no function, channel, or interface type other than `error` and `context.Context`. *K0123 and VL0123, in strict mode.*
- **ARCH-08.** The wires of a container form no cycle. *K0111. In L, constructor injection cannot build a cycle.*
- **ARCH-09.** A type that crosses a boundary comes from a library or the standard library. *K0112 and VL0112.*
- **ARCH-12.** Each container has one gateway leaf for each external system that it talks to. *K0207 and VL0207.*
- **ARCH-18.** A package-level `var` in a library holds no pointer, slice, map, channel, function, or interface, except an `error`. *K0210 and VL0210, in strict mode.*

## flow

- **ARCH-22.** A leaf declares no port method that its code does not call.
- **ERR-02.** A branch that receives a non-nil error reads the error value.
- **ERR-06.** Code handles an error once: it logs the error or returns it, not both.
- **ERR-11.** A function does not return a nil value together with a nil error, unless its doc comment says so.
- **ERR-12.** Code calls `recover` only at the top of a request or a goroutine, and that code logs the panic and returns a generic failure.
- **ERR-14.** An entry leaf does not send an error's text, a stack trace, a SQL message, or a file path to a client.
- **CTX-04.** Code calls the cancel function of each derived context on all paths.
- **CTX-05.** A function that has a `ctx` passes it on, and does not call `context.Background` or `context.TODO`.
- **HTTP-03.** Each call from a gateway to a remote system runs under a deadline, including database calls.
- **HTTP-04.** Code closes each `http.Response.Body` on each path.
- **HTTP-05.** An entry leaf limits a request body with `http.MaxBytesReader` before it reads or decodes the body.
- **HTTP-06.** A gateway bounds each response body and each decompressed stream that it reads, and does not call `io.ReadAll` or `io.Copy` on it without a limit.
- **HTTP-08.** An entry leaf calls `Server.Shutdown` when its startup context ends, and treats `http.ErrServerClosed` as a normal end.
- **HTTP-10.** A URL or a host that comes from request data does not reach a dial without an allow-list check.
- **SEC-05.** Logs and error messages hold no password, token, key, session ID, connection string, or card data.
- **SEC-08.** `exec.Command` runs no shell, and untrusted data reaches it only as a checked argument.
- **SEC-09.** Code that makes a key, token, session ID, or nonce uses `crypto/rand`.
- **SEC-13.** HTML output uses `html/template`, and code does not call `template.HTML` on untrusted data.
- **SEC-14.** A filename from outside the process opens through `os.Root`, `os.OpenInRoot`, or a `filepath.IsLocal` check.
- **SEC-28.** An integer conversion that can narrow a value has a range check first.
- **DATA-01.** Code calls `defer tx.Rollback()` immediately after `BeginTx` succeeds.
- **DATA-02.** Inside a transaction, code uses `tx` methods and not `db` methods.
- **DATA-04.** Code closes each `sql.Rows` and `sql.Stmt`, and checks `Rows.Err` after the loop.
- **DATA-05.** A database handle sets `SetMaxOpenConns`.
- **DATA-06.** A process opens each database once, in a constructor, and shares the handle.
- **DATA-07.** A transaction makes no network call and no port call that can reach a network function.
- **DATA-10.** Each check that guards a write runs before `Commit`, and `Commit` is the last step of the function.
- **RES-05.** A downstream call gets the remaining deadline of its caller, not a new and longer one.
- **CONC-01.** Each goroutine has a known stop condition, and something waits for it.
- **CONC-02.** A library initializer starts no goroutine.
- **OBS-03.** A process writes logs to standard output, and does not manage log files.
- **OBS-05.** Code uses the `Context` forms of `slog` calls when a `ctx` is in scope, so each line can carry a correlation ID.
- **CODE-07.** A library `init` is deterministic, and reads no file, environment, or clock.

## meaning

- **ARCH-13.** A gateway leaf holds translation only, and holds no business rule or orchestration.
- **ARCH-14.** An entry leaf only decodes, validates, calls a port, and encodes, and holds no business rule.
- **ARCH-15.** A port of a gateway uses the words of the consumer, and exposes no name, status code, or error type of the external system.
- **ARCH-20.** A leaf does not keep state in process memory that must survive a restart.
- **ERR-05.** A gateway returns its own error values at the port, and does not wrap a driver error such as `sql.ErrNoRows` with `%w`.
- **ERR-07.** Code does not call `panic` for an expected failure.
- **ERR-10.** A function reports failure with an `error` or a `bool` result, not with a special value such as `-1` or `""`.
- **ERR-15.** An entry leaf maps a client fault to 4xx and an unexpected failure to 5xx.
- **CTX-09.** A long loop or a multi-stage operation checks `ctx` between stages.
- **HTTP-18.** An entry route that uses cookie authentication and changes state has CSRF protection, and no GET route changes state.
- **HTTP-20.** The service rejects new requests with 503 when in-flight work exceeds a set limit.
- **SEC-03.** In production, a secret comes from a secret store or a mounted file, and an environment variable is the fallback only.
- **SEC-04.** The project has a written way to rotate each secret, and can rotate one without a code change.
- **SEC-06.** A type that holds a secret redacts itself with `slog.LogValuer` and `String`.
- **SEC-11.** Code uses reviewed crypto packages and authenticated modes such as AES-GCM, and defines no algorithm of its own.
- **SEC-12.** Password storage uses Argon2id, or scrypt or bcrypt where Argon2id is not available.
- **SEC-15.** A file upload has an allow-list of types, a generated name, a size limit, and storage outside the served tree.
- **SEC-19.** An entry leaf decodes a request into a type that holds only the fields the client can set, not into a stored or core type.
- **SEC-20.** An entry leaf returns only the fields that the caller may read, through a response type.
- **SEC-21.** An entry leaf validates each input on the server, by allow-list, for type and format and for meaning.
- **SEC-22.** A gateway validates data from an external system before it returns the data.
- **SEC-23.** Token verification rejects `alg: none` and takes the algorithm from configuration, not from the token.
- **SEC-24.** A login gives one generic failure message, takes about equal time for known and unknown users, and limits failed attempts per account.
- **SEC-26.** No password, token, or API key appears in a URL.
- **SEC-27.** An ID that a client sees is random and hard to guess.
- **DATA-08.** A change that depends on a read uses a conditional update, a row lock, a version column, or a unique constraint.
- **DATA-09.** One state change, its audit line, and each event to publish go in one database transaction.
- **DATA-12.** A schema change that breaks old code goes in steps: add the new form, move readers and writers, then remove the old form.
- **DATA-13.** Each list query has a limit.
- **PAY-02.** An amount travels with its currency, and code does not add, subtract, or compare two currencies.
- **PAY-03.** Rounding of money happens in named functions in one library, with a stated rule.
- **PAY-04.** A guard on money before a write cannot overflow, for example `amount > charged - refunded` and not `refunded + amount > charged`.
- **PAY-06.** The key and the outcome are stored with the change, in one transaction.
- **PAY-17.** A business operation across more than one system has a designed compensating step for each step that can succeed before a later step fails.
- **PAY-20.** Each change to money writes an audit record with the actor, the amount, and the reason.
- **RES-01.** Code retries only an operation that is idempotent.
- **RES-03.** One layer of a call path retries, and each other layer does not.
- **RES-04.** Code does not retry a permanent error or a malformed request.
- **RES-06.** A gateway to a dependency that can stay down for long has a defined way to fail fast, such as a circuit breaker.
- **RES-07.** A constructor returns an error for an invalid setting, and does not fall back to a silent default.
- **CONC-05.** A function is synchronous, unless asynchrony is its purpose.
- **CONC-06.** Code copies a slice or a map that it stores from a caller or returns from its own state.
- **CONC-07.** Work per item runs under a set concurrency limit, not in one goroutine for each item.
- **OBS-04.** Code logs authentication results, authorization failures, input validation failures, and use of high-risk functions.
- **API-01.** Each route that returns a collection pages from its first release, with a finite default and a maximum page size.
- **API-05.** Non-production environments use no production data, unless they have the same controls.
- **API-06.** A route's method matches its semantics: GET and HEAD are safe, and PUT and DELETE are idempotent.
- **API-07.** Error responses use one documented format, by default RFC 9457 problem details.
- **CON-02.** A rule that must stop an effect is a `requires` or a check before the effect, not an `ensures` or an invariant. *Moved from shape. Which rule must stop an effect is what the code means. A kind can read a clause, not its purpose.*
- **TEST-01.** A change of behavior comes with a test in the same change, and the test fails when the code is broken.
- **TEST-02.** A test names one behavior, and calls the leaf the way a real caller does.
- **TEST-03.** A test checks the resulting state, and checks calls only where the call is the effect.
- **TEST-07.** A leaf test uses a fake at its ports, and the suite also has tests that use the real provider where the real one is fast and deterministic.
- **TEST-10.** Code that parses untrusted input has a fuzz test.

## run-time

- **ARCH-16.** A gateway connects only to the hosts of its declared external system.
- **HTTP-12.** Each connection to an external system and each public endpoint uses TLS.
- **HTTP-14.** An entry route accepts only the content types that it declares.
- **HTTP-15.** An API response sets an accurate `Content-Type` and `X-Content-Type-Options: nosniff`.
- **HTTP-19.** An entry leaf exposes a readiness check and a liveness check, and the liveness check does not depend on any external system.
- **SEC-29.** The database account of a service has only the rights that it needs, and is not an administrator.
- **PAY-07.** A retry with a known key returns the stored outcome and does not run the change again.
- **PAY-08.** A known key with different parameters returns an error.
- **CONC-04.** CI runs each test with the race detector on a host that supports it.
- **OBS-06.** Each entry route and each gateway call reports latency as a histogram, traffic, errors, and saturation.
- **OBS-07.** A gateway passes W3C `traceparent` on outbound calls, and an entry leaf accepts it.
- **CON-04.** A fake bound in a test meets the contract of its port.
- **TEST-06.** Each test builds its own state, and tests pass in any order.
- **TEST-08.** A gateway has integration tests over its real transport against a test instance of the backend.
- **TEST-09.** Scheduled contract tests check that a gateway's fake still matches the external system.
- **BLD-09.** A deployment sets limits on memory, CPU, restarts, and file descriptors.

## project-facts

- **ARCH-17.** Two leaves do not read or write the same table, file, or environment variable.
- **HTTP-16.** An entry leaf sends no CORS headers unless the project needs them, and then only for listed origins.
- **HTTP-21.** Each client-facing route has a rate limit that fits its cost.
- **SEC-16.** Each route requires authentication unless a project list names it as public.
- **SEC-17.** Each operation on an object checks that the caller can access that object.
- **SEC-18.** Administrative functions have their own authorization check, separate from the check on ordinary functions.
- **SEC-32.** The project identifies its sensitive flows, such as purchase and reservation, and limits their use per client.
- **SEC-33.** Each paid external service has a spending limit or a billing alert.
- **SEC-34.** The project does not store a sensitive field that it does not need.
- **PAY-05.** Each retryable write entry takes an idempotency key: a client key, an operation ID, or a delivery ID.
- **PAY-09.** A call to a payment processor uses a key that the service stored before the first attempt, and reuses it across each retry and restart.
- **PAY-10.** A timeout or a 5xx from a payment processor marks the operation as unknown, not failed, until the service reconciles it.
- **PAY-11.** A webhook handler verifies the signature on the raw body before it uses the event.
- **PAY-12.** A webhook handler records processed event IDs and accepts events in any order.
- **PAY-13.** A webhook handler stores the event durably, returns 2xx quickly, and does slow work later.
- **PAY-14.** The service does not receive, store, or log raw card numbers, and uses processor tokens.
- **PAY-15.** A booked money entry is never updated or deleted, and a correction is a new entry.
- **PAY-16.** A money operation has named states and rejects a transition that the state table does not allow, including a refund above the captured amount. *The kind `transitions` states this once a team names its status field and its table. A default cannot name them.*
- **PAY-18.** A container that holds money sets `on_contract_failure: exit`. *Which containers hold money is a project fact, and the config key is a declaration.*
- **PAY-19.** A scheduled job compares local records with the processor, and reports each difference.
- **PAY-21.** A service uses processor keys with only the rights that it needs.

## outside-source

- **ARCH-25.** Each change that adds or removes a port, a wire, or an external system shows `vulcan archdiff` output in its review.
- **DATA-11.** Each schema change is a small numbered script, in version control, that automation applies.
- **API-02.** A published API does not remove, rename, or change the type of a field in one major version, and does not add a required request field.
- **API-03.** The build generates API documentation from an open standard.
- **CODE-01.** Code passes `go vet` with its full set of analyzers, and passes `staticcheck`.
- **CODE-10.** Code passes `vulcan fmt` with no change.
- **TEST-11.** A test that fails at random moves to a quarantine with a time limit and an owner.
- **BLD-01.** CI runs `govulncheck`, and a finding blocks a release.
- **BLD-02.** The project uses a supported Go toolchain, and updates its dependencies after review and tests.
- **BLD-03.** The repository holds `go.mod` and `go.sum`, and the build pins its tools in `go.mod`.
- **BLD-04.** A released `go.mod` has no `replace` directive.
- **BLD-05.** A pre-commit hook or CI runs a secret scanner.
- **BLD-06.** CI runs `vulcan check`, `vulcan test`, and `vulcan audit`, and the count of unchecked paths does not grow without review.
- **BLD-08.** A change is small and does one thing.

## declaration

- **ARCH-05.** A new project sets `config { strict: true }`. *`init` writes `strict: true` in the project that it creates. That is a template, not a rule, and a team can change it.*
- **ARCH-23.** The project sets `context_budget`, and a leaf over the budget gets a warning that a reviewer must acknowledge.
- **ARCH-24.** Each `external` declaration states the role of the system and the data that it exchanges.

## deferred

- **SEC-25.** Each call between containers authenticates the caller, and the receiving service checks authorization again.
- **MSG-01.** A handler stores `vulcan.DeliveryID(ctx)` and its effects in one transaction, and ignores a delivery that it already handled.
- **MSG-02.** A state change and its event go through an outbox, not a save followed by a publish.
- **MSG-03.** Each subscription wire names a dead-letter route, and a leaf handles that route and raises an alert.
- **MSG-04.** A handler does not depend on message order, unless the wire states that the transport keeps order.
- **MSG-05.** A message type changes only by adding fields whose zero value passes the invariant, and any other change is a new type.
- **MSG-06.** Each variable-size message field has a `limit`.
- **MSG-07.** Each wire that can block a publisher sets a full-queue policy or a `maxWait`.

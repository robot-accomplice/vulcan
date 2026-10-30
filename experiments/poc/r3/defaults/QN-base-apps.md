# The defaults that fire on the base applications

Plan-r3, section 4: a default that fires on a base application is counted under QN, and is not quietly removed. This draft removes none, and it turns none off. A later decision can change that, in writing.

Measured on copies of `poc/app/vulcan` and `poc/app/golint`, with `init` and then the check of each arm, on the draft of `defaults v1` with the hash `sha256:4d47eceb52365d0cc00b2a21f2db656977dec6519c1bb4d0f678690892224d37`. The draft is not frozen, so this is not the QN measure of plan-r3, section 5. No session has judged these findings. The applications did not change.

V gives 284 findings and L gives 285. `TestDefaultsOnBase` checks that the same rules fire in both. The one difference by rule and part is that L has a finding that V does not: `CODE-08` on `lib/shop`, because `library.vulcan` of V has a comment and the Go files of L have no package comment. That is a difference of the two applications, not of the rule.

Round 1 of this table held 49 findings, of `CTX-06` and `OBS-01`. The other rules below are new with the shape kinds of amendment 6, and `HTTP-01.timeouts` comes from the kind `literal` too.

| Rule | V | L | Wanted rule |
|---|---:|---:|---|
| CODE-08 | 126 | 127 | Each port, topic, `external`, library, and exported name has a doc comment. |
| CODE-09 | 10 | 10 | Near-duplicate functions become one function, in the leaf or in a library. |
| CON-01 | 51 | 51 | A port method that changes state declares a `requires` for the limits of its arguments. |
| CTX-06 | 46 | 46 | Code uses the context forms of calls that can block: `http.NewRequestWithContext`, `QueryContext`, `ExecContext`, `BeginTx`, `DialContext`, `exec.CommandContext`. |
| CTX-08 | 32 | 32 | A port method whose provider reaches a network or acquiring function takes `context.Context` as its first parameter. |
| ERR-01 | 6 | 6 | Code does not drop an `error` result. |
| ERR-03 | 4 | 4 | Each `_ = f()` that discards an error has a comment that gives the reason. |
| ERR-13 | 2 | 2 | Error strings start with a lower-case letter and do not end with punctuation. |
| HTTP-01.timeouts | 3 | 3 | An entry leaf serves through an `http.Server` with `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` set, and does not call `http.ListenAndServe`, `http.ListenAndServeTLS`, or `http.Serve`. |
| HTTP-09 | 1 | 1 | A gateway does not follow a redirect automatically, unless it checks the target against an allow-list. |
| OBS-01 | 3 | 3 | Leaf code and library code log through `log/slog`, not through `log` or `fmt.Print` functions. |

| Rule | Part | V | L |
|---|---|---:|---:|
| CODE-08 | api/billing | 4 | 4 |
| CODE-08 | api/events | 9 | 9 |
| CODE-08 | api/inventory | 8 | 8 |
| CODE-08 | api/orders | 12 | 12 |
| CODE-08 | api/payments | 2 | 2 |
| CODE-08 | api/purchasing | 10 | 10 |
| CODE-08 | api/store | 1 | 1 |
| CODE-08 | api/web | 27 | 27 |
| CODE-08 | lib/shop | 53 | 54 |
| CODE-09 | api/orders | 1 | 1 |
| CODE-09 | api/payments | 1 | 1 |
| CODE-09 | api/store | 3 | 3 |
| CODE-09 | api/web | 5 | 5 |
| CON-01 | api/billing | 2 | 2 |
| CON-01 | api/events | 7 | 7 |
| CON-01 | api/inventory | 6 | 6 |
| CON-01 | api/orders | 9 | 9 |
| CON-01 | api/purchasing | 8 | 8 |
| CON-01 | api/web | 19 | 19 |
| CTX-06 | api/payments | 1 | 1 |
| CTX-06 | api/store | 44 | 44 |
| CTX-06 | api/webhooks | 1 | 1 |
| CTX-08 | wires | 32 | 32 |
| ERR-01 | api/events | 1 | 1 |
| ERR-01 | api/store | 5 | 5 |
| ERR-03 | api/store | 2 | 2 |
| ERR-03 | api/web | 1 | 1 |
| ERR-03 | api/webhooks | 1 | 1 |
| ERR-13 | api/orders | 1 | 1 |
| ERR-13 | api/store | 1 | 1 |
| HTTP-01.timeouts | api/web | 3 | 3 |
| HTTP-09 | api/payments | 1 | 1 |
| OBS-01 | api/events | 1 | 1 |
| OBS-01 | api/web | 2 | 2 |

`CTX-08` reports at the wires: `system.vulcan` in V, and the main package in L. No rule at level `warn` fires. No other rule of the draft fires on these applications. A rule that a model check already enforces could not fire here, because both applications pass their model checks. The rules that read test files, such as `TEST-04`, have nothing to read, because `poc/app` holds no test files of its own.

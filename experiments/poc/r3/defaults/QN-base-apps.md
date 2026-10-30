# The defaults that fire on the base applications

Plan-r3, section 4: a default that fires on a base application is counted under QN, and is not quietly removed. This draft removes none, and it turns none off. A later decision can change that, in writing.

Measured on copies of `poc/app/vulcan` and `poc/app/golint`, with `init` and then the check of each arm, on the draft of `defaults v1` with the hash `sha256:30bce5a80c2cc248838e258c5a078924e07bcf4bf550cd30669f7dc577854c9b`. The draft is not frozen, so this is not the QN measure of plan-r3, section 5. No session has judged these findings. The applications did not change.

Both arms give the same findings: 49 in all, the same rule, part, and function, the same number of times.

| Rule | Part | Function | Findings |
|---|---|---|---:|
| CTX-06 | api/payments | `net/http.NewRequest` | 1 |
| CTX-06 | api/store | `(*database/sql.DB).Begin` | 7 |
| CTX-06 | api/store | `(*database/sql.DB).Exec` | 8 |
| CTX-06 | api/store | `(*database/sql.DB).Query` | 9 |
| CTX-06 | api/store | `(*database/sql.DB).QueryRow` | 5 |
| CTX-06 | api/store | `(*database/sql.Tx).Exec` | 12 |
| CTX-06 | api/store | `(*database/sql.Tx).QueryRow` | 3 |
| CTX-06 | api/webhooks | `net/http.NewRequest` | 1 |
| OBS-01 | api/events | `log.Printf` | 1 |
| OBS-01 | api/web | `log.Printf` | 2 |

| Rule | Findings | Wanted rule |
|---|---:|---|
| CTX-06 | 46 | Code uses the context forms of calls that can block: `http.NewRequestWithContext`, `QueryContext`, `ExecContext`, `BeginTx`, `DialContext`, `exec.CommandContext`. |
| OBS-01 | 3 | Leaf code and library code log through `log/slog`, not through `log` or `fmt.Print` functions. |

No other rule of the draft fires on these applications, and that includes the rules of plan-r3, section 3. A rule that a model check already enforces could not fire here, because both applications pass their model checks.

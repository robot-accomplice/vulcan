# Wishlist for a default ruleset for new service projects

Date: 2026-10-30. I checked each web source on this date. The file `wishlist.json` holds the same rules as data.

This list states each rule that a good team would want on a new service. I did not limit it to what a tool can do. It has no quota by topic.

## How to read this list

- **Scope.** A rule speaks only of kinds of part (entry, gateway, leaf, library) and of the standard library. It names no part of a project. Where a rule needs a project's own facts, the entry says so.
- **Plain leaf.** A leaf with no `gateway for` line and no `entry for` line. This list uses the term. The spec does not.
- **Before run.** A tool can decide the rule, or find most breaches, from source and declarations.
- **Only at run time.** A test, a probe, or the deployment shows a breach only when it runs.
- **By review only.** Only a person or a reviewing model can judge the rule.
- **Confidence.** My confidence (low, medium, high) in the checkability judgement, not in the value of the rule.
- **Spec status.** My reading of how `input/spec.md` treats the rule. It does not narrow the list. I kept rules that the language already enforces.
- **Deferred parts.** Topics, remote ports, and transports are deferred in release 1 of the spec. The MSG rules and SEC-25 apply when they exist.
- **Limits of my checks.** A fetch tool read each web source and summarized it. I checked the claims that I cite against those readings, not against each full page. Stripe pages came back in German, so my locators paraphrase them. Source text on money types, and on transactions and network calls, is the weakest. DATA-07 rests on one incident report.

## Counts

Total rules: **210**.

| Checkability | Rules | High | Medium | Low |
|---|---:|---:|---:|---:|
| before run | 118 | 66 | 47 | 5 |
| only at run time | 16 | 7 | 9 | 0 |
| by review only | 76 | 33 | 42 | 1 |

Rules that need a project's own facts: **43**. Rules where practice disagrees: **41**.

| Spec status | Rules |
|---|---:|
| The spec says that the compiler enforces this in ordinary mode. | 8 |
| The spec enforces this only in strict mode. | 8 |
| The spec's tools warn or report on this, and the build continues. | 14 |
| The spec names this as a duty of the application. | 9 |
| The spec does not enforce this. | 171 |

| Category | Rules |
|---|---:|
| Architecture and layering (ARCH) | 25 |
| Errors (ERR) | 15 |
| Context (CTX) | 9 |
| HTTP and network (HTTP) | 21 |
| Security (SEC) | 34 |
| Data and transactions (DATA) | 15 |
| Payment and money (PAY) | 21 |
| Messages between containers (MSG) | 7 |
| Resilience (RES) | 7 |
| Concurrency (CONC) | 7 |
| Observability (OBS) | 7 |
| API surface (API) | 7 |
| Contracts (CON) | 4 |
| Code quality (CODE) | 11 |
| Tests (TEST) | 11 |
| Build and delivery (BLD) | 9 |

## Where practice disagrees

Each rule below states the positions and my choice in its own entry.

ARCH-01, ARCH-02, ARCH-03, ARCH-17, ARCH-20, ARCH-23, ERR-03, ERR-05, ERR-07, ERR-11, ERR-12, ERR-14, CTX-08, HTTP-01, HTTP-02, HTTP-12, HTTP-20, SEC-03, SEC-07, SEC-09, SEC-25, SEC-27, DATA-05, DATA-15, PAY-07, PAY-13, PAY-15, PAY-18, RES-03, RES-06, CONC-03, OBS-03, API-07, CON-03, CODE-03, CODE-07, TEST-03, TEST-04, TEST-07, BLD-04, BLD-08.

## Rules that need a project's own facts

| Rule | Facts |
|---|---|
| ARCH-15 | A check needs the vendor package names and vocabulary. |
| ARCH-16 | The host list of each external system. |
| ARCH-17 | Which leaf owns each table and file. |
| HTTP-10 | The allow-list of targets. |
| HTTP-12 | Where TLS ends, and which targets are loopback or inside a mesh. |
| HTTP-16 | The allowed origins. |
| HTTP-18 | Which routes use cookies. |
| HTTP-19 | The deployment platform. |
| HTTP-20 | The limit, and whether a proxy enforces it. |
| HTTP-21 | The limits for each route. |
| SEC-03 | The secret store of the deployment platform. |
| SEC-12 | Only for a project that stores passwords. |
| SEC-15 | Only for a project that accepts uploads. |
| SEC-16 | The list of public routes and the authentication mechanism. |
| SEC-17 | The ownership model of each object. |
| SEC-18 | The role model. |
| SEC-23 | Only for a project that uses JWT. |
| SEC-24 | Only for a project with its own login. |
| SEC-29 | The deployment accounts. |
| SEC-32 | The list of sensitive flows. |
| SEC-33 | The list of paid services. |
| SEC-34 | The data classification. |
| DATA-11 | The migration tool and directory. |
| PAY-01 | The name of the project's money type gives a stronger check. |
| PAY-02 | The money type. |
| PAY-05 | Which routes and port methods change money or other state that must not repeat. |
| PAY-09 | Which gateway is the payment gateway. |
| PAY-10 | The payment gateway. |
| PAY-11 | Which entry leaf receives webhooks, and the signature scheme. |
| PAY-12 | The webhook entries. |
| PAY-13 | The webhook entries. |
| PAY-14 | Only for a project that takes cards. |
| PAY-15 | The ledger table. |
| PAY-16 | The state table of each operation. |
| PAY-18 | Which containers hold money. |
| PAY-19 | The processor and its report API. |
| PAY-21 | The processor's key model. |
| OBS-07 | The trace system. Some external systems ignore the header. |
| API-02 | The published schema, as a baseline for comparison. |
| API-03 | The API format. |
| TEST-08 | A test instance of each backend. |
| TEST-09 | A sandbox of each external system. |
| BLD-09 | The deployment platform. |

## Rules

### Architecture and layering

**ARCH-01.** A plain leaf does not import `net/http`, `net`, `database/sql`, or `os/exec`, even for a type.

- Reason: Inner code must not know the web or the database.
- Source: [Robert C. Martin, The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) (The Dependency Rule); [Alistair Cockburn, Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/) (rule on inside and outside)
- Check: before run, confidence high.
- Applies to: plain leaf. Spec status: The spec does not enforce this.
- Practice disagrees: The spec allows any import and restricts only reached functions. Fowler's basic layering lets the domain depend on the data layer, and he notes that substitution is rarely used. Martin and Cockburn keep the inside free of these details. **My choice:** I went strict. A team can exempt a package, for example for a shared type.

**ARCH-02.** A library does not import `net/http` or `database/sql`.

- Reason: A shared library that knows transport or storage spreads that knowledge to each leaf.
- Source: [Robert C. Martin, The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) (The Dependency Rule); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4a rule 2)
- Check: before run, confidence high.
- Applies to: library. Spec status: The spec does not enforce this.
- Practice disagrees: The spec lets a library query through a `*sql.DB` that its caller gives it. Clean Architecture keeps database details out of shared inner code. **My choice:** I went strict. A team that wants a query helper library can exempt it.

**ARCH-03.** A port signature does not carry `*http.Request`, `http.ResponseWriter`, `*http.Response`, `*sql.DB`, `*sql.Tx`, `*sql.Rows`, or `net.Conn`.

- Reason: Only simple data crosses a boundary, so a consumer stays free of provider details.
- Source: [Robert C. Martin, The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) (data crossing boundaries); [Microsoft Azure Architecture Center, Anti-Corruption Layer pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/anti-corruption-layer) (solution section)
- Check: before run, confidence high.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Practice disagrees: A shared `*sql.Tx` across a port lets two leaves join one transaction. The spec shows a `*sql.Tx` parameter in 7.4, on a method that is not stated to be a port. Clean Architecture forbids database rows at a boundary. **My choice:** I banned it at ports. The provider owns its transaction. Use a saga or an outbox across providers.

**ARCH-04.** A port signature carries no function, channel, or interface type other than `error` and `context.Context`.

- Reason: A callback through a port lets the callee run the caller's code and hides a dependency.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4d strict rule 1, K0123)
- Check: before run, confidence high.
- Applies to: leaf. Spec status: The spec enforces this only in strict mode.

**ARCH-05.** A new project sets `config { strict: true }`.

- Reason: Strict mode closes most known paths around the architecture rules.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4d Strict mode)
- Check: before run, confidence high.
- Applies to: system. Spec status: The spec enforces this only in strict mode.

**ARCH-06.** Each entry in `strict_exempt`, `strict_interfaces`, `strict_shared`, `allow_network`, and `allow_acquiring` has a comment that gives the reason.

- Reason: A reviewer must see why a guard is off.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4c and 5.4d, audit lists each entry); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (nolintlint)
- Check: before run, confidence medium.
- Applies to: system. Spec status: The spec's tools warn or report on this, and the build continues.

**ARCH-07.** Code in a leaf uses another leaf only through a port, a topic, or a library.

- Reason: Declared boundaries are the base of each other architecture rule.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4 rule 1); [Alistair Cockburn, Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/) (communication only through ports)
- Check: before run, confidence high.
- Applies to: leaf. Spec status: The spec says that the compiler enforces this in ordinary mode.

**ARCH-08.** The wires of a container form no cycle.

- Reason: Dependencies must have one direction, so that a reader can find an order and a change stays local.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4 rule 8); [Martin Fowler, Presentation Domain Data Layering](https://martinfowler.com/bliki/PresentationDomainDataLayering.html) (dependency direction)
- Check: before run, confidence high.
- Applies to: system. Spec status: The spec says that the compiler enforces this in ordinary mode.

**ARCH-09.** A type that crosses a boundary comes from a library or the standard library.

- Reason: Each side then compiles the same type and neither side owns the other.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4 rule 9); [Robert C. Martin, The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) (data crossing boundaries)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec says that the compiler enforces this in ordinary mode.

**ARCH-10.** Only a gateway leaf or an entry leaf reaches a network function.

- Reason: A reviewer finds all traffic in known places.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4c rule 1); [Alistair Cockburn, Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/) (adapters at the outside)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec says that the compiler enforces this in ordinary mode.

**ARCH-11.** An entry leaf reaches only serve functions, and only a gateway leaf reaches a dial function.

- Reason: A leaf that both serves and dials hides an outbound path from the diagram.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4d strict rule 3, K0125)
- Check: before run, confidence high.
- Applies to: entry, gateway. Spec status: The spec enforces this only in strict mode.

**ARCH-12.** Each container has one gateway leaf for each external system that it talks to.

- Reason: All rules about one external system stay in one place.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4b rule 1); [Microsoft Azure Architecture Center, Anti-Corruption Layer pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/anti-corruption-layer) (solution section)
- Check: before run, confidence high.
- Applies to: gateway, system. Spec status: The spec says that the compiler enforces this in ordinary mode.

**ARCH-13.** A gateway leaf holds translation only, and holds no business rule or orchestration.

- Reason: Business rules in an adapter are hidden from the core and from its tests.
- Source: [Microsoft Azure Architecture Center, Anti-Corruption Layer pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/anti-corruption-layer) ('Avoid placing business rules or orchestration in the layer')
- Check: by review only, confidence high.
- Applies to: gateway. Spec status: The spec does not enforce this.

**ARCH-14.** An entry leaf only decodes, validates, calls a port, and encodes, and holds no business rule.

- Reason: A second driver, such as a test or a batch job, must reach the same logic.
- Source: [Alistair Cockburn, Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/) (no logic in the UI adapter)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Note: A size limit on entry leaves can give a weak signal before run.

**ARCH-15.** A port of a gateway uses the words of the consumer, and exposes no name, status code, or error type of the external system.

- Reason: Vendor terms in a port bind the core to that vendor.
- Source: [Microsoft Azure Architecture Center, Anti-Corruption Layer pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/anti-corruption-layer) (layer uses the data model of subsystem A); [Go Blog, Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors) ('Do not wrap an error when doing so would expose implementation details')
- Check: by review only, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.
- Project facts: A check needs the vendor package names and vocabulary.

**ARCH-16.** A gateway connects only to the hosts of its declared external system.

- Reason: An unexpected target is a leak, or an unmodeled dependency.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4d G11, out of scope); [OWASP Server Side Request Forgery Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) (allow-lists)
- Check: only at run time, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.
- Project facts: The host list of each external system.
- Note: The spec states that no static check decides this. An egress policy can enforce it at run time.

**ARCH-17.** Two leaves do not read or write the same table, file, or environment variable.

- Reason: A shared store is a hidden coupling that no port shows.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4d path 7, G12); [Chris Richardson, Database per Service](https://microservices.io/patterns/data/database-per-service.html) (private data, access through the API)
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Project facts: Which leaf owns each table and file.
- Practice disagrees: Database per service applies between services. The spec's containers can share one database, and many teams share tables inside one process. **My choice:** I apply the rule between leaves. A team can relax it inside one container.

**ARCH-18.** A package-level `var` in a library holds no pointer, slice, map, channel, function, or interface, except an `error`.

- Reason: Library state that changes is hidden shared memory between leaves.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4d strict rule 4, K0210); [Google Go Style Guide, Best Practices](https://google.github.io/styleguide/go/best-practices) (global state); [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Avoid Mutable Globals)
- Check: before run, confidence high.
- Applies to: library. Spec status: The spec enforces this only in strict mode.

**ARCH-19.** Leaf code and library code do not call a global mutator such as `os.Setenv`, `http.Handle`, `slog.SetDefault`, or `flag.Parse`.

- Reason: Process-wide settings couple leaves that have no declared link.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4d strict rule 5, K0211, Appendix D); [Google Go Style Guide, Best Practices](https://google.github.io/styleguide/go/best-practices) (global state)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec enforces this only in strict mode.

**ARCH-20.** A leaf does not keep state in process memory that must survive a restart.

- Reason: Any process can stop at any time, and a second instance cannot see the first one's memory.
- Source: [The Twelve-Factor App, Processes](https://12factor.net/processes) ('stateless and share-nothing')
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Practice disagrees: The spec's example has an in-memory ledger. Twelve-factor says that processes are stateless and keep persistent data in a backing service. **My choice:** I allow memory for caches and for test or demo providers only.

**ARCH-21.** A library name is not `util`, `common`, `misc`, `helpers`, `shared`, or `base`.

- Reason: Such a name tells the reader nothing, and the package collects unrelated code and dependencies.
- Source: [Go Blog, Package names](https://go.dev/blog/package-names) (Bad package names)
- Check: before run, confidence high.
- Applies to: library. Spec status: The spec does not enforce this.
- Note: The source names `util`, `common`, and `misc`. I added the other three by analogy.

**ARCH-22.** A leaf declares no port method that its code does not call.

- Reason: A port states what the consumer uses, so a wider port hides the true dependency.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Interfaces); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 9, unused ports)
- Check: before run, confidence high.
- Applies to: leaf. Spec status: The spec's tools warn or report on this, and the build continues.

**ARCH-23.** The project sets `context_budget`, and a leaf over the budget gets a warning that a reviewer must acknowledge.

- Reason: Small leaves fit in one reader's context, human or model.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4 rule 10, R1); [Google Go Style Guide, Best Practices](https://google.github.io/styleguide/go/best-practices) (package size)
- Check: before run, confidence high.
- Applies to: system, leaf. Spec status: The spec's tools warn or report on this, and the build continues.
- Practice disagrees: Size limits are a dispute. The Google guide warns only against very large files and sets no number, and the spec refuses a hard error. **My choice:** I keep a warning with a tunable number, not a build failure.

**ARCH-24.** Each `external` declaration states the role of the system and the data that it exchanges.

- Reason: An inventory of integrations is a named control against forgotten exposure.
- Source: [OWASP API9:2023 Improper Inventory Management](https://api-security.owasp.org/editions/2023/en/0xa9-improper-inventory-management) ('Inventory integrated services')
- Check: before run, confidence medium.
- Applies to: system. Spec status: The spec does not enforce this.
- Note: A tool can check that a description exists. Only review can check that it is true.

**ARCH-25.** Each change that adds or removes a port, a wire, or an external system shows `vulcan archdiff` output in its review.

- Reason: A reviewer must see an architecture change as such, and not find it in a code diff.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 9, Architecture diffs)
- Check: before run, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.
- Note: CI can attach the report. Only review can judge it.

### Errors

**ERR-01.** Code does not drop an `error` result.

- Reason: A dropped error hides a failure.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Handle Errors); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (errcheck); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G104); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 6, K0701)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec's tools warn or report on this, and the build continues.

**ERR-02.** A branch that receives a non-nil error reads the error value.

- Reason: A branch that returns success on failure masks the error.
- Source: [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (nilerr); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 6, K0702)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec's tools warn or report on this, and the build continues.

**ERR-03.** Each `_ = f()` that discards an error has a comment that gives the reason.

- Reason: A deliberate discard must be visible and defended.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) ('Do not discard errors using _ variables'); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 6, audit lists each one); [gosec README](https://github.com/securego/gosec) (G307 retired)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec's tools warn or report on this, and the build continues.
- Practice disagrees: Go Code Review Comments forbids discards. The common `defer tx.Rollback()` idiom drops an error, and gosec retired its deferred-close rule as more trouble than value. **My choice:** I allow a bare deferred `Rollback` and a deferred `Close` on a read handle. Each other discard needs a comment.

**ERR-04.** Code compares errors with `errors.Is` or `errors.As`, not with `==`, a type assertion, or the error text.

- Reason: A direct comparison fails on a wrapped error, and the text is not an API.
- Source: [Go Blog, Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors) (errors.Is and errors.As); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (errorlint); [Google Go Style Guide, Best Practices](https://google.github.io/styleguide/go/best-practices) ('Do not attempt to distinguish errors based on their string form')
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**ERR-05.** A gateway returns its own error values at the port, and does not wrap a driver error such as `sql.ErrNoRows` with `%w`.

- Reason: A wrapped driver error becomes part of the port's API.
- Source: [Go Blog, Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors) ('When not to wrap')
- Check: by review only, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.
- Practice disagrees: The Go blog says not to wrap when it exposes implementation details. The `wrapcheck` linter asks to wrap errors from external packages. **My choice:** I wrap with `%w` inside a leaf and translate at the port. Text may keep the cause with `%v`.

**ERR-06.** Code handles an error once: it logs the error or returns it, not both.

- Reason: Log and return gives duplicate lines for one failure.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Handle Errors Once); [Google Go Style Guide, Best Practices](https://google.github.io/styleguide/go/best-practices) (logging errors)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Note: A tool can find a log and a return of the same error in one branch.

**ERR-07.** Code does not call `panic` for an expected failure.

- Reason: A caller can handle a returned error, and cannot plan for a panic.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Don't Panic); [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Don't Panic)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Practice disagrees: Uber asks to avoid panic in production. Contracts in the spec panic by design, and `Must` functions at startup are common. **My choice:** I exempt contract failures and failures in `setup`, a constructor, or a library initializer.

**ERR-08.** Leaf code and library code do not call `os.Exit`, `log.Fatal`, or `log.Panic` functions.

- Reason: Only `main` may end the process, and `main` is generated here.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Exit in Main); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 6, generated main)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**ERR-09.** Each type assertion uses the two-value form.

- Reason: A failed one-value assertion panics.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Handle Type Assertion Failures); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (forcetypeassert)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**ERR-10.** A function reports failure with an `error` or a `bool` result, not with a special value such as `-1` or `""`.

- Reason: A caller can forget a special value, and the compiler cannot help.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (In-Band Errors)
- Check: by review only, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**ERR-11.** A function does not return a nil value together with a nil error, unless its doc comment says so.

- Reason: The caller then dereferences nil after a success.
- Source: [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (nilnil); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 6, K0703)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Practice disagrees: `nilnil` flags the form. Many Go lookups use nil, nil for 'not found'. **My choice:** I allow it only with a doc comment. A sentinel error is the default.

**ERR-12.** Code calls `recover` only at the top of a request or a goroutine, and that code logs the panic and returns a generic failure.

- Reason: A quiet recover can hide a broken state.
- Source: [Sourcegraph Blog, When is it OK to recover from panics in Go?](https://sourcegraph.com/blog/go-when-is-it-ok-to-recover) (Symonds quote and takeaway); [Go issue 76763, net/http built-in panic recovery documentation](https://github.com/golang/go/issues/76763) (title); [OWASP Error Handling Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Error_Handling_Cheat_Sheet.html) (global handler)
- Check: before run, confidence medium.
- Applies to: entry, leaf. Spec status: The spec does not enforce this.
- Practice disagrees: OWASP asks for a global handler that returns a generic response. Symonds and Go issue 76763 call the recovery in `net/http` a poor practice, and prefer a crash and a proxy retry. **My choice:** I allow one top-level recovery, with a log line. A container that holds money uses PAY-18.

**ERR-13.** Error strings start with a lower-case letter and do not end with punctuation.

- Reason: Callers append context to errors.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Error Strings); [Staticcheck, Checks](https://staticcheck.dev/docs/checks/) (ST1005)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**ERR-14.** An entry leaf does not send an error's text, a stack trace, a SQL message, or a file path to a client.

- Reason: Internal detail helps an attacker.
- Source: [OWASP Error Handling Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Error_Handling_Cheat_Sheet.html) (Context section and global handler); [RFC 9457, Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457.html) (Security Considerations); [OWASP API8:2023 Security Misconfiguration](https://api-security.owasp.org/editions/2023/en/0xa8-security-misconfiguration) ('Control error responses')
- Check: before run, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Practice disagrees: The spec says that a response that shows an error is a question for review. A direct `http.Error(w, err.Error(), ...)` is easy to detect, and indirect flows are not. **My choice:** I treat it as checkable for direct flows and as review for the rest.

**ERR-15.** An entry leaf maps a client fault to 4xx and an unexpected failure to 5xx.

- Reason: Clients and alerts depend on this split.
- Source: [OWASP Error Handling Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Error_Handling_Cheat_Sheet.html) (5xx only for failures that the application cannot handle); [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html) (HTTP status codes)
- Check: by review only, confidence high.
- Applies to: entry. Spec status: The spec does not enforce this.

### Context

**CTX-01.** A function that takes a `context.Context` takes it as the first parameter, named `ctx`.

- Reason: One form makes context flow easy to read and to check.
- Source: [Go standard library, package context](https://pkg.go.dev/context) (package overview)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CTX-02.** A struct does not have a `context.Context` field.

- Reason: A stored context hides its lifetime.
- Source: [Go standard library, package context](https://pkg.go.dev/context) ('Do not store Contexts inside a struct type'); [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Contexts)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CTX-03.** Code does not pass a nil `context.Context`.

- Reason: A callee may dereference it.
- Source: [Go standard library, package context](https://pkg.go.dev/context) ('Do not pass a nil Context'); [Staticcheck, Checks](https://staticcheck.dev/docs/checks/) (SA1012)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CTX-04.** Code calls the cancel function of each derived context on all paths.

- Reason: An uncalled cancel leaks the child until the parent ends.
- Source: [Go standard library, package context](https://pkg.go.dev/context) (CancelFunc); [Go command documentation, cmd/vet](https://pkg.go.dev/cmd/vet) (lostcancel); [Go documentation, Canceling in-progress database operations](https://go.dev/doc/database/cancel-operations) (always defer cancel)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CTX-05.** A function that has a `ctx` passes it on, and does not call `context.Background` or `context.TODO`.

- Reason: A new root context drops cancellation and the deadline.
- Source: [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (contextcheck); [Go standard library, package context](https://pkg.go.dev/context) (Background); [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (deadline propagation)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Note: For work that must outlive the request, use `context.WithoutCancel`, and say why in a comment.

**CTX-06.** Code uses the context forms of calls that can block: `http.NewRequestWithContext`, `QueryContext`, `ExecContext`, `BeginTx`, `DialContext`, `exec.CommandContext`.

- Reason: A call without a context cannot stop when its caller stops.
- Source: [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (noctx); [Go documentation, Canceling in-progress database operations](https://go.dev/doc/database/cancel-operations) (timeouts); [Go standard library, package database/sql](https://pkg.go.dev/database/sql) (QueryContext)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CTX-07.** Context values use an unexported key type and hold only request-scoped data, not optional parameters.

- Reason: A hidden parameter breaks the visible signature.
- Source: [Go standard library, package context](https://pkg.go.dev/context) (WithValue); [Staticcheck, Checks](https://staticcheck.dev/docs/checks/) (SA1029)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Note: The key type is checkable. The purpose of a value is for review.

**CTX-08.** A port method whose provider reaches a network or acquiring function takes `context.Context` as its first parameter.

- Reason: A deadline cannot pass through a port that has no context.
- Source: [Go standard library, package context](https://pkg.go.dev/context) (pass a Context explicitly); [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (deadline propagation)
- Check: before run, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Practice disagrees: The spec's examples show port methods with no context, because a port call is an ordinary call. The context and deadline guidance needs one on each path that can block. **My choice:** I require it. A check follows the wire to the provider.

**CTX-09.** A long loop or a multi-stage operation checks `ctx` between stages.

- Reason: Work past a deadline wastes capacity on a result that nobody will read.
- Source: [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (check the remaining deadline at each stage)
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.

### HTTP and network

**HTTP-01.** An entry leaf serves through an `http.Server` with `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` set, and does not call `http.ListenAndServe`, `http.ListenAndServeTLS`, or `http.Serve`.

- Reason: The package-level functions run with no timeouts, so a slow client can hold resources.
- Source: [Cloudflare Blog, The complete guide to Go net/http timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/) (server timeouts); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G112, G114); [Go standard library, package net/http](https://pkg.go.dev/net/http) (type Server)
- Check: before run, confidence high.
- Applies to: entry. Spec status: The spec does not enforce this.
- Practice disagrees: A fixed `WriteTimeout` breaks streaming responses, and some teams leave it to a proxy. Cloudflare says to set both for untrusted clients. **My choice:** I require all four. A streaming route needs a written exemption.

**HTTP-02.** A gateway uses an `http.Client` with a non-zero `Timeout`, and does not call `http.Get`, `http.Post`, `http.Head`, `http.PostForm`, or `http.DefaultClient`.

- Reason: `DefaultClient` has no timeout, so a stuck peer holds a goroutine.
- Source: [Cloudflare Blog, The complete guide to Go net/http timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/) (client side); [Go standard library, package net/http](https://pkg.go.dev/net/http) ('A Timeout of zero means no timeout')
- Check: before run, confidence high.
- Applies to: gateway. Spec status: The spec does not enforce this.
- Practice disagrees: A context deadline on each call can replace `Client.Timeout`, and `Timeout` does not suit a stream. **My choice:** I require `Timeout` by default. A deadline on each call is an accepted alternative.

**HTTP-03.** Each call from a gateway to a remote system runs under a deadline, including database calls.

- Reason: A call with no deadline holds resources without limit.
- Source: [AWS Builders' Library, Timeouts, retries, and backoff with jitter](https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter) (timeouts on every remote call); [Go documentation, Canceling in-progress database operations](https://go.dev/doc/database/cancel-operations) (timeouts); [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (set deadlines)
- Check: before run, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.

**HTTP-04.** Code closes each `http.Response.Body` on each path.

- Reason: An open body leaks a connection.
- Source: [Go standard library, package net/http](https://pkg.go.dev/net/http) ('The caller must close the response body'); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (bodyclose); [Go command documentation, cmd/vet](https://pkg.go.dev/cmd/vet) (httpresponse)
- Check: before run, confidence high.
- Applies to: gateway, entry. Spec status: The spec does not enforce this.

**HTTP-05.** An entry leaf limits a request body with `http.MaxBytesReader` before it reads or decodes the body.

- Reason: A large body exhausts memory.
- Source: [Go standard library, package net/http](https://pkg.go.dev/net/http) (MaxBytesReader); [OWASP API4:2023 Unrestricted Resource Consumption](https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption) (maximum sizes); [OWASP Denial of Service Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Denial_of_Service_Cheat_Sheet.html) (request size); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G120)
- Check: before run, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.

**HTTP-06.** A gateway bounds each response body and each decompressed stream that it reads, and does not call `io.ReadAll` or `io.Copy` on it without a limit.

- Reason: A hostile or broken peer can send data without end.
- Source: [OWASP API10:2023 Unsafe Consumption of APIs](https://api-security.owasp.org/editions/2023/en/0xaa-unsafe-consumption-of-apis) ('Validate incoming data'); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G110); [OWASP File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html) (decompression size)
- Check: before run, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.

**HTTP-07.** An entry leaf registers routes on its own `http.ServeMux`, not on `http.DefaultServeMux`.

- Reason: Any imported package can add a route to the default mux.
- Source: [Go standard library, package net/http](https://pkg.go.dev/net/http) (DefaultServeMux); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (Appendix D, global mutators); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G108)
- Check: before run, confidence high.
- Applies to: entry. Spec status: The spec enforces this only in strict mode.

**HTTP-08.** An entry leaf calls `Server.Shutdown` when its startup context ends, and treats `http.ErrServerClosed` as a normal end.

- Reason: A deploy must not cut active requests.
- Source: [Go standard library, package net/http](https://pkg.go.dev/net/http) (Server.Shutdown); [The Twelve-Factor App, Disposability](https://12factor.net/disposability) (graceful shutdown on SIGTERM); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 6, stop steps)
- Check: before run, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.

**HTTP-09.** A gateway does not follow a redirect automatically, unless it checks the target against an allow-list.

- Reason: A redirect can move a permitted request to an internal target.
- Source: [OWASP API10:2023 Unsafe Consumption of APIs](https://api-security.owasp.org/editions/2023/en/0xaa-unsafe-consumption-of-apis) ('Restrict redirects'); [OWASP Server Side Request Forgery Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) (Redirects)
- Check: before run, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.

**HTTP-10.** A URL or a host that comes from request data does not reach a dial without an allow-list check.

- Reason: This is server-side request forgery.
- Source: [OWASP API Security Top 10, 2023](https://api-security.owasp.org/editions/2023/en/0x11-t10) (API7:2023); [OWASP Server Side Request Forgery Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) (allow-lists); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G107, G704)
- Check: before run, confidence medium.
- Applies to: gateway, entry. Spec status: The spec does not enforce this.
- Project facts: The allow-list of targets.

**HTTP-11.** Code does not set `InsecureSkipVerify: true` or a `MinVersion` below TLS 1.2, except in test files.

- Reason: Without verification, TLS does not stop a man in the middle.
- Source: [Go standard library, package crypto/tls](https://pkg.go.dev/crypto/tls) (InsecureSkipVerify, MinVersion); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G402); [Stripe documentation, Integration security guide](https://docs.stripe.com/security/guide) (TLS 1.2 or higher)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**HTTP-12.** Each connection to an external system and each public endpoint uses TLS.

- Reason: Plain text exposes credentials and data.
- Source: [OWASP API10:2023 Unsafe Consumption of APIs](https://api-security.owasp.org/editions/2023/en/0xaa-unsafe-consumption-of-apis) ('Encrypt all integrations'); [OWASP API8:2023 Security Misconfiguration](https://api-security.owasp.org/editions/2023/en/0xa8-security-misconfiguration) (TLS for internal and public traffic); [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html) (HTTPS)
- Check: only at run time, confidence medium.
- Applies to: gateway, entry. Spec status: The spec does not enforce this.
- Project facts: Where TLS ends, and which targets are loopback or inside a mesh.
- Practice disagrees: OWASP asks for TLS on internal and public traffic. Many teams end TLS at a proxy or a mesh and use plain HTTP behind it. **My choice:** I require TLS by default. A project records each exception.

**HTTP-13.** Each entry route accepts only the methods that it lists, and answers others with 405.

- Reason: Unplanned methods widen the attack surface.
- Source: [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html) (HTTP method restriction); [OWASP API8:2023 Security Misconfiguration](https://api-security.owasp.org/editions/2023/en/0xa8-security-misconfiguration) ('Limit HTTP verbs')
- Check: before run, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.

**HTTP-14.** An entry route accepts only the content types that it declares.

- Reason: A parser for an unplanned format is added attack surface.
- Source: [OWASP API8:2023 Security Misconfiguration](https://api-security.owasp.org/editions/2023/en/0xa8-security-misconfiguration) ('Restrict content types')
- Check: only at run time, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.

**HTTP-15.** An API response sets an accurate `Content-Type` and `X-Content-Type-Options: nosniff`.

- Reason: A browser must not guess the type of a response.
- Source: [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html) (Security headers)
- Check: only at run time, confidence high.
- Applies to: entry. Spec status: The spec does not enforce this.

**HTTP-16.** An entry leaf sends no CORS headers unless the project needs them, and then only for listed origins.

- Reason: A wide CORS policy lets other sites call the API.
- Source: [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html) (CORS); [OWASP API8:2023 Security Misconfiguration](https://api-security.owasp.org/editions/2023/en/0xa8-security-misconfiguration) ('Protect browser-facing APIs')
- Check: before run, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: The allowed origins.

**HTTP-17.** Each session cookie sets `Secure`, `HttpOnly`, and `SameSite` with the value `Strict` or `Lax`.

- Reason: These attributes limit theft and cross-site use.
- Source: [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) (Cookie attributes)
- Check: before run, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.

**HTTP-18.** An entry route that uses cookie authentication and changes state has CSRF protection, and no GET route changes state.

- Reason: A browser attaches cookies by itself.
- Source: [OWASP Cross-Site Request Forgery Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html) (CSRF tokens, GET requests)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: Which routes use cookies.

**HTTP-19.** An entry leaf exposes a readiness check and a liveness check, and the liveness check does not depend on any external system.

- Reason: A liveness check that fails with a dependency restarts healthy processes and cascades.
- Source: [Kubernetes documentation, Liveness, Readiness, and Startup Probes](https://kubernetes.io/docs/concepts/configuration/liveness-readiness-startup-probes/) (caution about liveness probes)
- Check: only at run time, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: The deployment platform.

**HTTP-20.** The service rejects new requests with 503 when in-flight work exceeds a set limit.

- Reason: A server near overload must shed work to serve what it can.
- Source: [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (Load Shedding)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: The limit, and whether a proxy enforces it.
- Practice disagrees: The SRE book places load shedding in the server. Many teams use a proxy or a mesh for it. **My choice:** I accept either, but a project must state where it is.

**HTTP-21.** Each client-facing route has a rate limit that fits its cost.

- Reason: An unlimited route can exhaust resources or run up cost.
- Source: [OWASP API4:2023 Unrestricted Resource Consumption](https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption) (rate limiting); [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html) (return 429); [OWASP Denial of Service Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Denial_of_Service_Cheat_Sheet.html) (Rate limiting)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: The limits for each route.

### Security

**SEC-01.** Source files, declaration files, and test fixtures hold no real secret, key, or password.

- Reason: Anyone who reads the repository gets the secret, and history keeps it.
- Source: [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G101); [Stripe documentation, Best practices for managing secret API keys](https://docs.stripe.com/keys-best-practices) (never in source code); [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html) (hard-coded secrets); [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) (key storage)
- Check: before run, confidence medium.
- Applies to: leaf, library, system. Spec status: The spec's tools warn or report on this, and the build continues.
- Note: A scanner finds likely values. It misses some and flags some false values.

**SEC-02.** A settings field that holds a credential has the tag `vulcan:"secret"`.

- Reason: The tag makes a literal value a compile error.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.3, K0605)
- Check: before run, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Note: A tool can match on field names such as `Key`, `Token`, and `Password`. It cannot prove that it found them all.

**SEC-03.** In production, a secret comes from a secret store or a mounted file, and an environment variable is the fallback only.

- Reason: A process and its dumps can expose the environment, and a store adds audit and rotation.
- Source: [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html) (environment variables); [The Twelve-Factor App, Config](https://12factor.net/config) (config in the environment); [Stripe documentation, Best practices for managing secret API keys](https://docs.stripe.com/keys-best-practices) (secrets vault first, then environment)
- Check: by review only, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.
- Project facts: The secret store of the deployment platform.
- Practice disagrees: Twelve-factor stores config in environment variables. OWASP does not recommend them for secrets unless other methods are not possible. Stripe prefers a vault and accepts environment variables as a fallback. The spec's `flag` reads the environment. **My choice:** I went with OWASP and Stripe. The ban on secrets in source stays absolute, and the environment stays acceptable for small projects.

**SEC-04.** The project has a written way to rotate each secret, and can rotate one without a code change.

- Reason: A stolen key must have a short useful life.
- Source: [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html) (Rotation); [Stripe documentation, Best practices for managing secret API keys](https://docs.stripe.com/keys-best-practices) (rotate regularly)
- Check: by review only, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.

**SEC-05.** Logs and error messages hold no password, token, key, session ID, connection string, or card data.

- Reason: Log stores are wide-reach and long-lived.
- Source: [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) (data to exclude); [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html) (secrets in logs); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G117)
- Check: before run, confidence low.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Note: A taint check needs to know which values are secret. A type with a redacting `slog.LogValuer` helps.

**SEC-06.** A type that holds a secret redacts itself with `slog.LogValuer` and `String`.

- Reason: The type then cannot leak by accident in a log line.
- Source: [Go standard library, package log/slog](https://pkg.go.dev/log/slog) (LogValuer, Secret example)
- Check: by review only, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**SEC-07.** A SQL statement is a constant string, and each value goes in as a parameter.

- Reason: A built statement is a SQL injection risk.
- Source: [Go documentation, Avoiding SQL injection risk](https://go.dev/doc/database/sql-injection) (rule on parameters); [OWASP SQL Injection Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html) (primary defense); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G201, G202)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Practice disagrees: Parameters cannot bind identifiers such as a sort column, so some teams build that part of the text. **My choice:** I keep the rule. Dynamic text must come from a fixed allow-list in the same function, with a comment.

**SEC-08.** `exec.Command` runs no shell, and untrusted data reaches it only as a checked argument.

- Reason: A shell turns data into commands.
- Source: [OWASP OS Command Injection Defense Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/OS_Command_Injection_Defense_Cheat_Sheet.html) (Avoid shells, Parameterize); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G204)
- Check: before run, confidence medium.
- Applies to: gateway, entry. Spec status: The spec does not enforce this.

**SEC-09.** Code that makes a key, token, session ID, or nonce uses `crypto/rand`.

- Reason: `math/rand` output is predictable.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Crypto Rand); [Go standard library, package math/rand/v2](https://pkg.go.dev/math/rand/v2) (warning on security-sensitive work); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G404); [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) (Randomness)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Practice disagrees: gosec flags each use of `math/rand`. Retry jitter and sampling need no secure source. **My choice:** I allow `math/rand/v2` for jitter and sampling, with a comment.

**SEC-10.** Security code does not use MD5, SHA-1, DES, or RC4.

- Reason: These algorithms are broken.
- Source: [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G401, G501, G505); [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) (Algorithms)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Note: A non-security checksum can use a suppression with a reason, under CODE-02.

**SEC-11.** Code uses reviewed crypto packages and authenticated modes such as AES-GCM, and defines no algorithm of its own.

- Reason: Home-grown crypto fails in ways that tests do not show.
- Source: [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) ('Don't do this')
- Check: by review only, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**SEC-12.** Password storage uses Argon2id, or scrypt or bcrypt where Argon2id is not available.

- Reason: A fast hash lets an attacker test guesses at speed.
- Source: [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) (algorithm order)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Project facts: Only for a project that stores passwords.

**SEC-13.** HTML output uses `html/template`, and code does not call `template.HTML` on untrusted data.

- Reason: `text/template` does not escape output.
- Source: [Go standard library, package text/template](https://pkg.go.dev/text/template) (package overview); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G203)
- Check: before run, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.

**SEC-14.** A filename from outside the process opens through `os.Root`, `os.OpenInRoot`, or a `filepath.IsLocal` check.

- Reason: An unchecked path can escape the planned directory.
- Source: [Go Blog, Traversal-resistant file APIs](https://go.dev/blog/osroot) (Advice); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G304, G305)
- Check: before run, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.

**SEC-15.** A file upload has an allow-list of types, a generated name, a size limit, and storage outside the served tree.

- Reason: Each one closes a known upload attack.
- Source: [OWASP File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html) (all four points)
- Check: by review only, confidence high.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: Only for a project that accepts uploads.

**SEC-16.** Each route requires authentication unless a project list names it as public.

- Reason: Access must be denied by default.
- Source: [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html) (Deny by Default); [OWASP API Security Top 10, 2023](https://api-security.owasp.org/editions/2023/en/0x11-t10) (API2:2023)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: The list of public routes and the authentication mechanism.

**SEC-17.** Each operation on an object checks that the caller can access that object.

- Reason: A missing check is the top API risk.
- Source: [OWASP API1:2023 Broken Object Level Authorization](https://api-security.owasp.org/editions/2023/en/0xa1-broken-object-level-authorization) (How To Prevent); [OWASP Insecure Direct Object Reference Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Insecure_Direct_Object_Reference_Prevention_Cheat_Sheet.html) (per-object checks); [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html) (Validate Permissions on Every Request)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Project facts: The ownership model of each object.

**SEC-18.** Administrative functions have their own authorization check, separate from the check on ordinary functions.

- Reason: Unclear role separation lets users reach functions that they must not reach.
- Source: [OWASP API Security Top 10, 2023](https://api-security.owasp.org/editions/2023/en/0x11-t10) (API5:2023); [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html) (Least Privilege)
- Check: by review only, confidence high.
- Applies to: entry, leaf. Spec status: The spec does not enforce this.
- Project facts: The role model.

**SEC-19.** An entry leaf decodes a request into a type that holds only the fields the client can set, not into a stored or core type.

- Reason: Binding to a stored type lets a client set fields such as a role.
- Source: [OWASP Mass Assignment Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Mass_Assignment_Cheat_Sheet.html) (DTOs and allow-listing); [OWASP API Security Top 10, 2023](https://api-security.owasp.org/editions/2023/en/0x11-t10) (API3:2023)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.

**SEC-20.** An entry leaf returns only the fields that the caller may read, through a response type.

- Reason: A serialized core type exposes each new field by default.
- Source: [OWASP API Security Top 10, 2023](https://api-security.owasp.org/editions/2023/en/0x11-t10) (API3:2023)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.

**SEC-21.** An entry leaf validates each input on the server, by allow-list, for type and format and for meaning.

- Reason: All data from outside is untrusted, and a browser check is not a control.
- Source: [OWASP Input Validation Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html) (allow-list, syntax, semantics)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Note: A `requires` clause on the first function that takes the value can make the rule visible.

**SEC-22.** A gateway validates data from an external system before it returns the data.

- Reason: An attacker can use a partner system as a path in.
- Source: [OWASP API10:2023 Unsafe Consumption of APIs](https://api-security.owasp.org/editions/2023/en/0xaa-unsafe-consumption-of-apis) ('Validate incoming data'); [Microsoft Azure Architecture Center, Anti-Corruption Layer pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/anti-corruption-layer) (input validation at this boundary)
- Check: by review only, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.

**SEC-23.** Token verification rejects `alg: none` and takes the algorithm from configuration, not from the token.

- Reason: A token must not choose how it is verified.
- Source: [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html) (JWT)
- Check: by review only, confidence high.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: Only for a project that uses JWT.

**SEC-24.** A login gives one generic failure message, takes about equal time for known and unknown users, and limits failed attempts per account.

- Reason: Differences in responses reveal valid accounts.
- Source: [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html) (Generic error messages, Timing, Login throttling)
- Check: by review only, confidence high.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: Only for a project with its own login.

**SEC-25.** Each call between containers authenticates the caller, and the receiving service checks authorization again.

- Reason: A gateway check does not prove that a downstream operation is allowed.
- Source: [OWASP Microservices Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Microservices_Security_Cheat_Sheet.html) (edge and service level authorization)
- Check: by review only, confidence medium.
- Applies to: entry, gateway. Spec status: The spec does not enforce this.
- Practice disagrees: Many teams trust their private network. The OWASP sheet asks each service to enforce access, including on internal calls. **My choice:** I went with OWASP.

**SEC-26.** No password, token, or API key appears in a URL.

- Reason: URLs reach server logs.
- Source: [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html) (Sensitive information in URLs)
- Check: by review only, confidence medium.
- Applies to: entry, gateway. Spec status: The spec does not enforce this.

**SEC-27.** An ID that a client sees is random and hard to guess.

- Reason: It adds a barrier if an access check fails.
- Source: [OWASP API1:2023 Broken Object Level Authorization](https://api-security.owasp.org/editions/2023/en/0xa1-broken-object-level-authorization) ('random and unpredictable values'); [OWASP Insecure Direct Object Reference Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Insecure_Direct_Object_Reference_Prevention_Cheat_Sheet.html) (defense in depth)
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Practice disagrees: OWASP API1 prefers random IDs. The IDOR sheet says that they do not fix a missing check. Sequential keys perform better in many databases. **My choice:** I keep the rule for public IDs only, and SEC-17 stays the real control.

**SEC-28.** An integer conversion that can narrow a value has a range check first.

- Reason: A wrapped value breaks later checks.
- Source: [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G115); [MITRE CWE-190, Integer Overflow or Wraparound](https://cwe.mitre.org/data/definitions/190.html) (description)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**SEC-29.** The database account of a service has only the rights that it needs, and is not an administrator.

- Reason: Least privilege limits the damage of an injection.
- Source: [OWASP SQL Injection Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html) (least privilege)
- Check: only at run time, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.
- Project facts: The deployment accounts.

**SEC-30.** Leaf code and library code do not import `unsafe` or `syscall`, do not use cgo, and do not call `reflect.Value.Call` or `reflect.Value.Method`.

- Reason: These paths escape each static check.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4d strict rule 2, K0124); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G103)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec enforces this only in strict mode.

**SEC-31.** An entry leaf or a gateway does not decode outside data with `encoding/gob`.

- Reason: Native serial formats do not suit untrusted data.
- Source: [OWASP Deserialization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Deserialization_Cheat_Sheet.html) (general rule)
- Check: before run, confidence medium.
- Applies to: entry, gateway. Spec status: The spec does not enforce this.
- Note: The source does not discuss Go. I apply its general rule to `encoding/gob`.

**SEC-32.** The project identifies its sensitive flows, such as purchase and reservation, and limits their use per client.

- Reason: Automated use can harm the business with no code bug.
- Source: [OWASP API6:2023 Unrestricted Access to Sensitive Business Flows](https://api-security.owasp.org/editions/2023/en/0xa6-unrestricted-access-to-sensitive-business-flows) (How To Prevent)
- Check: by review only, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.
- Project facts: The list of sensitive flows.

**SEC-33.** Each paid external service has a spending limit or a billing alert.

- Reason: A loop or an attack can run up cost without limit.
- Source: [OWASP API4:2023 Unrestricted Resource Consumption](https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption) (spending limits)
- Check: by review only, confidence high.
- Applies to: project. Spec status: The spec does not enforce this.
- Project facts: The list of paid services.

**SEC-34.** The project does not store a sensitive field that it does not need.

- Reason: Data that a system does not hold cannot leak.
- Source: [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) ('the best protection')
- Check: by review only, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.
- Project facts: The data classification.

### Data and transactions

**DATA-01.** Code calls `defer tx.Rollback()` immediately after `BeginTx` succeeds.

- Reason: Each path must end the transaction.
- Source: [Go documentation, Executing transactions](https://go.dev/doc/database/execute-transactions) (example); [Go standard library, package database/sql](https://pkg.go.dev/database/sql) (Tx must end with Commit or Rollback); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec names this as a duty of the application.

**DATA-02.** Inside a transaction, code uses `tx` methods and not `db` methods.

- Reason: A `db` call runs outside the transaction and can deadlock.
- Source: [Go documentation, Executing transactions](https://go.dev/doc/database/execute-transactions) (quoted warning)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**DATA-03.** SQL text does not hold `BEGIN`, `COMMIT`, or `ROLLBACK`.

- Reason: Manual statements can leave the database in an unpredictable state.
- Source: [Go documentation, Executing transactions](https://go.dev/doc/database/execute-transactions) (quoted warning)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**DATA-04.** Code closes each `sql.Rows` and `sql.Stmt`, and checks `Rows.Err` after the loop.

- Reason: An unchecked iteration error looks like a short result.
- Source: [Go standard library, package database/sql](https://pkg.go.dev/database/sql) (Rows.Err); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (rowserrcheck, sqlclosecheck)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**DATA-05.** A database handle sets `SetMaxOpenConns`.

- Reason: The default is unlimited, so a load spike can overload the database.
- Source: [Go standard library, package database/sql](https://pkg.go.dev/database/sql) (SetMaxOpenConns); [Go documentation, Managing connections](https://go.dev/doc/database/manage-connections) (overview and warning)
- Check: before run, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.
- Practice disagrees: The Go docs say that most programs need not tune the pool, and warn that a limit can deadlock. The default has no limit at all. **My choice:** I require a limit, and the project chooses the number.

**DATA-06.** A process opens each database once, in a constructor, and shares the handle.

- Reason: A `sql.DB` is a long-lived pool.
- Source: [Go standard library, package database/sql](https://pkg.go.dev/database/sql) (Open once); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.3, one instance; 5.4d K0607)
- Check: before run, confidence medium.
- Applies to: gateway. Spec status: The spec enforces this only in strict mode.

**DATA-07.** A transaction makes no network call and no port call that can reach a network function.

- Reason: A slow peer then holds a connection and row locks, and the pool runs out.
- Source: [BestHub, API latency spike: long transactions holding DB connections](https://www.besthub.dev/articles/api-latency-spike-to-8s-not-slow-sql-but-long-transactions-holding-db-connections-e9a1370f7180) (cause and fix); [Go documentation, Managing connections](https://go.dev/doc/database/manage-connections) (a limit acts like a semaphore)
- Check: before run, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Note: My source is an incident report, not a standard. A check follows calls between `BeginTx` and `Commit`.

**DATA-08.** A change that depends on a read uses a conditional update, a row lock, a version column, or a unique constraint.

- Reason: State can change between a check and a write.
- Source: [MITRE CWE-367, Time-of-check Time-of-use Race Condition](https://cwe.mitre.org/data/definitions/367.html) (description); [Martin Fowler, Optimistic Offline Lock (PoEAA catalog)](https://martinfowler.com/eaaCatalog/optimisticOfflineLock.html) (pattern)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec does not enforce this.

**DATA-09.** One state change, its audit line, and each event to publish go in one database transaction.

- Reason: The process can stop between separate writes.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 Responsibilities); [Chris Richardson, Transactional Outbox](https://microservices.io/patterns/data/transactional-outbox.html) (solution); [AWS Builders' Library, Making retries safe with idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/) (atomic record with mutation)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec names this as a duty of the application.

**DATA-10.** Each check that guards a write runs before `Commit`, and `Commit` is the last step of the function.

- Reason: A check after commit finds a fault but cannot undo it.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 items 2 and 3, K0525)
- Check: before run, confidence medium.
- Applies to: leaf. Spec status: The spec's tools warn or report on this, and the build continues.

**DATA-11.** Each schema change is a small numbered script, in version control, that automation applies.

- Reason: A hand-made change cannot be repeated or traced.
- Source: [Pramod Sadalage and Martin Fowler, Evolutionary Database Design](https://martinfowler.com/articles/evodb.html) (main practices)
- Check: before run, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.
- Project facts: The migration tool and directory.

**DATA-12.** A schema change that breaks old code goes in steps: add the new form, move readers and writers, then remove the old form.

- Reason: Old and new versions of a service run together during a deploy.
- Source: [Martin Fowler, Parallel Change](https://martinfowler.com/bliki/ParallelChange.html) (expand, migrate, contract); [Pramod Sadalage and Martin Fowler, Evolutionary Database Design](https://martinfowler.com/articles/evodb.html) (transition phase)
- Check: by review only, confidence high.
- Applies to: project. Spec status: The spec does not enforce this.

**DATA-13.** Each list query has a limit.

- Reason: A table grows after the code ships.
- Source: [OWASP API4:2023 Unrestricted Resource Consumption](https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption) (parameters that control record counts); [Google AIP-158, Pagination](https://google.aip.dev/158) (finite default page size)
- Check: by review only, confidence medium.
- Applies to: gateway, leaf. Spec status: The spec does not enforce this.
- Note: A tool can flag a `SELECT` that has no `LIMIT` and is not a lookup by key. Expect false findings.

**DATA-14.** Code uses `time.Time` for instants and `time.Duration` for periods, not integers or strings.

- Reason: A bare number has no unit, and a mistake compiles.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Use time.Time, time.Duration); [Google AIP-142, Time and duration](https://google.aip.dev/142) (timestamps and durations); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (durationcheck)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Note: Field names and types give a heuristic.

**DATA-15.** A plain leaf or a library does not call `time.Now` for a business decision, and takes time through a port.

- Reason: A test must control the clock.
- Source: [Martin Fowler, Eradicating Non-Determinism in Tests](https://martinfowler.com/articles/nonDeterminism.html) (Time); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.3, SystemClock wire)
- Check: before run, confidence high.
- Applies to: plain leaf, library. Spec status: The spec does not enforce this.
- Practice disagrees: Fowler asks to wrap the clock. `testing/synctest` now gives a fake clock in a test bubble, so some teams drop the wrapper. **My choice:** I keep the port for decisions. Timing for logs and metrics may use `time.Now` and `time.Since`. A tool must see the difference, so expect some suppressions.

### Payment and money

**PAY-01.** A money amount is an integer in the currency's smallest unit, or an exact decimal, and is never `float32` or `float64`.

- Reason: Binary floats do not add up exactly.
- Source: [Stripe documentation, Supported currencies](https://docs.stripe.com/currencies) (amounts in smallest unit); [The Floating-Point Guide, Exact Types](https://floating-point-gui.de/formats/exact/) (exact types for money); [Martin Fowler, Money (PoEAA catalog)](https://martinfowler.com/eaaCatalog/money.html) (rounding)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Project facts: The name of the project's money type gives a stronger check.
- Note: Without that name, a tool uses field names such as `amount` and `price`.

**PAY-02.** An amount travels with its currency, and code does not add, subtract, or compare two currencies.

- Reason: Adding dollars to yen gives a wrong value that still looks valid.
- Source: [Martin Fowler, Money (PoEAA catalog)](https://martinfowler.com/eaaCatalog/money.html) (currency mix)
- Check: by review only, confidence high.
- Applies to: library, leaf. Spec status: The spec does not enforce this.
- Project facts: The money type.
- Note: A `requires` on the money type's operations can enforce this at run time.

**PAY-03.** Rounding of money happens in named functions in one library, with a stated rule.

- Reason: Scattered rounding loses or creates minor units.
- Source: [Martin Fowler, Money (PoEAA catalog)](https://martinfowler.com/eaaCatalog/money.html) (rounding); [Stripe documentation, Supported currencies](https://docs.stripe.com/currencies) (zero-decimal currencies and special cases)
- Check: by review only, confidence medium.
- Applies to: library. Spec status: The spec does not enforce this.

**PAY-04.** A guard on money before a write cannot overflow, for example `amount > charged - refunded` and not `refunded + amount > charged`.

- Reason: Go integer arithmetic wraps, and a wrapped guard passes.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 Responsibilities); [MITRE CWE-190, Integer Overflow or Wraparound](https://cwe.mitre.org/data/definitions/190.html) (description)
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec names this as a duty of the application.

**PAY-05.** Each retryable write entry takes an idempotency key: a client key, an operation ID, or a delivery ID.

- Reason: A lost reply makes a client retry, and the retry must not charge twice.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 Responsibilities); [Stripe API reference, Idempotent requests](https://docs.stripe.com/api/idempotent_requests) (overview); [Google AIP-155, Request identification](https://google.aip.dev/155) (request IDs); [AWS Builders' Library, Making retries safe with idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/) (client request token)
- Check: by review only, confidence high.
- Applies to: entry, leaf. Spec status: The spec names this as a duty of the application.
- Project facts: Which routes and port methods change money or other state that must not repeat.

**PAY-06.** The key and the outcome are stored with the change, in one transaction.

- Reason: A key without the change, or a change without the key, breaks the retry.
- Source: [AWS Builders' Library, Making retries safe with idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/) (atomic record with mutation); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 Responsibilities)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec names this as a duty of the application.

**PAY-07.** A retry with a known key returns the stored outcome and does not run the change again.

- Reason: The first reply may have been lost after the commit.
- Source: [Google AIP-155, Request identification](https://google.aip.dev/155) (duplicate requests); [AWS Builders' Library, Making retries safe with idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/) (semantically equivalent response); [Stripe API reference, Idempotent requests](https://docs.stripe.com/api/idempotent_requests) (saved response); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 Responsibilities)
- Check: only at run time, confidence high.
- Applies to: leaf. Spec status: The spec names this as a duty of the application.
- Practice disagrees: Stripe API v1 replays the first response, including a 500. Stripe API v2 retries failed work to completion. AIP-155 speaks of the previously successful request. **My choice:** I store and replay each final outcome, including a business refusal. A failure with no effect, such as a validation error, stores nothing, and the client can retry.
- Note: A test sends one request twice and compares the effects.

**PAY-08.** A known key with different parameters returns an error.

- Reason: The caller probably meant a different operation.
- Source: [Stripe API reference, Idempotent requests](https://docs.stripe.com/api/idempotent_requests) (parameter comparison); [AWS Builders' Library, Making retries safe with idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/) (parameter mismatch)
- Check: only at run time, confidence high.
- Applies to: leaf. Spec status: The spec does not enforce this.

**PAY-09.** A call to a payment processor uses a key that the service stored before the first attempt, and reuses it across each retry and restart.

- Reason: A new key on a retry can start a second charge.
- Source: [Stripe documentation, Advanced error handling](https://docs.stripe.com/error-low-level) (Sending idempotency keys, Server errors)
- Check: by review only, confidence high.
- Applies to: gateway, leaf. Spec status: The spec does not enforce this.
- Project facts: Which gateway is the payment gateway.

**PAY-10.** A timeout or a 5xx from a payment processor marks the operation as unknown, not failed, until the service reconciles it.

- Reason: The charge may have succeeded.
- Source: [Stripe documentation, Advanced error handling](https://docs.stripe.com/error-low-level) (Network errors, Server errors); [AWS Builders' Library, Timeouts, retries, and backoff with jitter](https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter) (a timeout does not prove no effect)
- Check: by review only, confidence high.
- Applies to: gateway, leaf. Spec status: The spec does not enforce this.
- Project facts: The payment gateway.

**PAY-11.** A webhook handler verifies the signature on the raw body before it uses the event.

- Reason: Anyone can post to a public URL.
- Source: [Stripe documentation, Webhooks](https://docs.stripe.com/webhooks) (signature verification); [Stripe documentation, Integration security guide](https://docs.stripe.com/security/guide) (webhooks)
- Check: by review only, confidence high.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: Which entry leaf receives webhooks, and the signature scheme.

**PAY-12.** A webhook handler records processed event IDs and accepts events in any order.

- Reason: A processor can send one event twice and does not promise order.
- Source: [Stripe documentation, Webhooks](https://docs.stripe.com/webhooks) (duplicate events, order of events)
- Check: by review only, confidence high.
- Applies to: entry, leaf. Spec status: The spec does not enforce this.
- Project facts: The webhook entries.

**PAY-13.** A webhook handler stores the event durably, returns 2xx quickly, and does slow work later.

- Reason: A slow handler times out, and the sender then retries.
- Source: [Stripe documentation, Webhooks](https://docs.stripe.com/webhooks) (return 2xx quickly)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: The webhook entries.
- Practice disagrees: Stripe says to return 2xx before complex logic. A reply before any durable write loses the event if the process stops. **My choice:** I store first and reply second. The store must be fast.

**PAY-14.** The service does not receive, store, or log raw card numbers, and uses processor tokens.

- Reason: Direct handling of card data brings a far larger set of PCI DSS controls.
- Source: [Stripe documentation, Integration security guide](https://docs.stripe.com/security/guide) (low-risk integrations); [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) (card holder data)
- Check: by review only, confidence high.
- Applies to: entry, leaf. Spec status: The spec does not enforce this.
- Project facts: Only for a project that takes cards.

**PAY-15.** A booked money entry is never updated or deleted, and a correction is a new entry.

- Reason: The history of money must stay readable and auditable.
- Source: [Martin Fowler, Accounting Entry](https://martinfowler.com/eaaDev/AccountingEntry.html) (entries may be immutable); [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) (higher-risk functionality)
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Project facts: The ledger table.
- Practice disagrees: Fowler says that entries may be immutable always, or mutable until booked. He does not make it a rule. **My choice:** I require immutability after booking.

**PAY-16.** A money operation has named states and rejects a transition that the state table does not allow, including a refund above the captured amount.

- Reason: Order-state logic is a common defect that no architecture rule covers.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (1.1 third hypothesis, 7.4 refund guard)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Project facts: The state table of each operation.
- Note: A struct `invariant` can state part of this.

**PAY-17.** A business operation across more than one system has a designed compensating step for each step that can succeed before a later step fails.

- Reason: No shared transaction exists, so nothing undoes the work by itself.
- Source: [Chris Richardson, Saga](https://microservices.io/patterns/data/saga.html) (compensating transactions)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec does not enforce this.

**PAY-18.** A container that holds money sets `on_contract_failure: exit`.

- Reason: A broken value must not serve a later request.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 item 5 and Responsibilities); [Sourcegraph Blog, When is it OK to recover from panics in Go?](https://sourcegraph.com/blog/go-when-is-it-ok-to-recover) (quiet catch leaves an inconsistent state)
- Check: before run, confidence high.
- Applies to: system. Spec status: The spec does not enforce this.
- Project facts: Which containers hold money.
- Practice disagrees: The spec's default is `panic`, and it only says to consider `exit`. Availability favors a recover. Consistency favors a crash. **My choice:** I chose `exit` for money, and the default for each other container.

**PAY-19.** A scheduled job compares local records with the processor, and reports each difference.

- Reason: Unknown outcomes and lost webhooks must surface.
- Source: [Stripe documentation, Advanced error handling](https://docs.stripe.com/error-low-level) (reconcile the outcome)
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Project facts: The processor and its report API.

**PAY-20.** Each change to money writes an audit record with the actor, the amount, and the reason.

- Reason: An auditor must be able to explain each balance.
- Source: [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) (higher-risk functionality); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 Responsibilities)
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec names this as a duty of the application.

**PAY-21.** A service uses processor keys with only the rights that it needs.

- Reason: A restricted key limits the damage of a leak.
- Source: [Stripe documentation, Best practices for managing secret API keys](https://docs.stripe.com/keys-best-practices) (restricted API keys)
- Check: by review only, confidence high.
- Applies to: project. Spec status: The spec does not enforce this.
- Project facts: The processor's key model.

### Messages between containers

**MSG-01.** A handler stores `vulcan.DeliveryID(ctx)` and its effects in one transaction, and ignores a delivery that it already handled.

- Reason: A transport can deliver a message more than once, and the `idempotent` marker alone proves nothing.
- Source: [Chris Richardson, Idempotent Consumer](https://microservices.io/patterns/communication-style/idempotent-consumer.html) (solution); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.5.0 delivery ID; 5.5 idempotency is declared)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec names this as a duty of the application.

**MSG-02.** A state change and its event go through an outbox, not a save followed by a publish.

- Reason: The process can stop between the two lines, and the event is lost.
- Source: [Chris Richardson, Transactional Outbox](https://microservices.io/patterns/data/transactional-outbox.html) (solution); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.5 not atomic, 7.4)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec names this as a duty of the application.
- Note: A tool can flag `Publish` in a function that also commits. It cannot be sure.

**MSG-03.** Each subscription wire names a dead-letter route, and a leaf handles that route and raises an alert.

- Reason: A message that cannot succeed must not vanish.
- Source: [Enterprise Integration Patterns, Dead Letter Channel](https://www.enterpriseintegrationpatterns.com/patterns/messaging/DeadLetterChannel.html) (pattern); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.5.0, 5.5.1)
- Check: before run, confidence high.
- Applies to: system. Spec status: The spec does not enforce this.
- Note: The `direct` transport already requires the route. The spec example shows a `kafka` wire with none, so the rule asks for more than the language does.

**MSG-04.** A handler does not depend on message order, unless the wire states that the transport keeps order.

- Reason: The language promises no order.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.5.0); [Stripe documentation, Webhooks](https://docs.stripe.com/webhooks) (order of events)
- Check: by review only, confidence high.
- Applies to: leaf. Spec status: The spec does not enforce this.

**MSG-05.** A message type changes only by adding fields whose zero value passes the invariant, and any other change is a new type.

- Reason: Old and new subscribers run together.
- Source: [Vulcan primer (input/guide.md)](input/guide.md) (section 4, topics.lock); [Google AIP-180, Backwards compatibility](https://google.aip.dev/180) (compatibility)
- Check: before run, confidence high.
- Applies to: library. Spec status: The spec says that the compiler enforces this in ordinary mode.

**MSG-06.** Each variable-size message field has a `limit`.

- Reason: A transport queue has a fixed memory budget.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.5.1); [OWASP API4:2023 Unrestricted Resource Consumption](https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption) (maximum sizes)
- Check: before run, confidence high.
- Applies to: library. Spec status: The spec says that the compiler enforces this in ordinary mode.

**MSG-07.** Each wire that can block a publisher sets a full-queue policy or a `maxWait`.

- Reason: Under the default policy, a slow subscriber can slow the publisher.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.5 publish rules); [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (overload)
- Check: before run, confidence medium.
- Applies to: system. Spec status: The spec does not enforce this.

### Resilience

**RES-01.** Code retries only an operation that is idempotent.

- Reason: A timeout does not prove that the effect did not happen.
- Source: [AWS Builders' Library, Timeouts, retries, and backoff with jitter](https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter) (retry idempotent operations only)
- Check: by review only, confidence high.
- Applies to: leaf, gateway. Spec status: The spec does not enforce this.

**RES-02.** A retry loop has a limit on attempts, exponential backoff, and jitter.

- Reason: Unlimited or synchronized retries deepen an overload.
- Source: [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (Retries); [AWS Builders' Library, Timeouts, retries, and backoff with jitter](https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter) (backoff and jitter)
- Check: before run, confidence low.
- Applies to: leaf, gateway. Spec status: The spec does not enforce this.
- Note: A tool can flag a loop with `time.Sleep` around a call. It will miss many forms.

**RES-03.** One layer of a call path retries, and each other layer does not.

- Reason: Retries at each layer multiply load.
- Source: [AWS Builders' Library, Timeouts, retries, and backoff with jitter](https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter) (single layer); [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (stacking retries)
- Check: by review only, confidence high.
- Applies to: leaf, gateway. Spec status: The spec does not enforce this.
- Practice disagrees: Sources agree on one layer and do not say which one. **My choice:** I chose the gateway by default, because it knows the protocol. The spec's runtime does not retry remote calls.

**RES-04.** Code does not retry a permanent error or a malformed request.

- Reason: Neither will ever succeed.
- Source: [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (permanent errors)
- Check: by review only, confidence high.
- Applies to: leaf, gateway. Spec status: The spec does not enforce this.

**RES-05.** A downstream call gets the remaining deadline of its caller, not a new and longer one.

- Reason: Work past the caller's deadline is waste.
- Source: [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (deadline propagation)
- Check: before run, confidence low.
- Applies to: leaf, gateway. Spec status: The spec does not enforce this.
- Note: A tool can flag `context.WithTimeout` on a fresh root. It cannot judge the length.

**RES-06.** A gateway to a dependency that can stay down for long has a defined way to fail fast, such as a circuit breaker.

- Reason: Blocked calls hold resources and spread a failure.
- Source: [Microsoft Azure Architecture Center, Circuit Breaker pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker) (pattern and when to use)
- Check: by review only, confidence low.
- Applies to: gateway. Spec status: The spec does not enforce this.
- Practice disagrees: Azure lists cases where a breaker adds needless complexity, such as message-driven systems or platform-level recovery. **My choice:** I make this a question for review, not a required pattern.

**RES-07.** A constructor returns an error for an invalid setting, and does not fall back to a silent default.

- Reason: A failed start with a report is easier to fix than a wrong run.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.3 A constructor can fail; section 6 step 2)
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.

### Concurrency

**CONC-01.** Each goroutine has a known stop condition, and something waits for it.

- Reason: A leaked goroutine holds memory and outlives shutdown.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Goroutine Lifetimes); [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Don't fire-and-forget goroutines, Wait for goroutines to exit)
- Check: before run, confidence low.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Note: Static proof is hard. A test can detect leaks at run time for covered paths.

**CONC-02.** A library initializer starts no goroutine.

- Reason: Import must not start background work.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (No goroutines in init); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4 rule 12, leaves have no init)
- Check: before run, confidence high.
- Applies to: library. Spec status: The spec does not enforce this.

**CONC-03.** A channel is unbuffered or has size one, unless a comment defends a larger size.

- Reason: A large buffer hides a capacity decision.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Channel Size is One or None)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Practice disagrees: Uber asks for this. Worker pools and queues often use larger buffers for good reasons. **My choice:** I keep it as a warning, and a comment removes it.

**CONC-04.** CI runs each test with the race detector on a host that supports it.

- Reason: A race fails only when the path runs, and some hosts cannot run the detector.
- Source: [Go documentation, Data Race Detector](https://go.dev/doc/articles/race_detector) (tests); [Go documentation, Security Best Practices for Go Developers](https://go.dev/doc/security/best-practices) (race detector); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 6, K0705)
- Check: only at run time, confidence high.
- Applies to: project. Spec status: The spec's tools warn or report on this, and the build continues.

**CONC-05.** A function is synchronous, unless asynchrony is its purpose.

- Reason: A caller can add concurrency, and cannot easily remove it.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Synchronous Functions)
- Check: by review only, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CONC-06.** Code copies a slice or a map that it stores from a caller or returns from its own state.

- Reason: A shared backing array lets another caller change the value, and a port call passes references.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Copy Slices and Maps at Boundaries); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.3 port call semantics)
- Check: by review only, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CONC-07.** Work per item runs under a set concurrency limit, not in one goroutine for each item.

- Reason: Input size must not decide resource use.
- Source: [OWASP API4:2023 Unrestricted Resource Consumption](https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption) (limit resources); [Google SRE Book, Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/) (overload)
- Check: by review only, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.

### Observability

**OBS-01.** Leaf code and library code log through `log/slog`, not through `log` or `fmt.Print` functions.

- Reason: Structured records are machine-readable and resist log injection.
- Source: [Go standard library, package log/slog](https://pkg.go.dev/log/slog) (overview); [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) (structured formats); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (sloglint)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**OBS-02.** A log message text is constant, and variable data goes in attributes.

- Reason: Untrusted text in a message can forge lines.
- Source: [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) (log injection); [gosec README](https://github.com/securego/gosec) (G7xx log injection)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**OBS-03.** A process writes logs to standard output, and does not manage log files.

- Reason: The platform routes and stores the stream.
- Source: [The Twelve-Factor App, Logs](https://12factor.net/logs) (rule)
- Check: before run, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Practice disagrees: Twelve-factor gives the stream to the platform. A compliance audit log can need a separate, protected store. **My choice:** I apply this to operational logs. An audit trail goes in a database or a named store.

**OBS-04.** Code logs authentication results, authorization failures, input validation failures, and use of high-risk functions.

- Reason: An incident needs this record.
- Source: [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) (events that should be logged)
- Check: by review only, confidence high.
- Applies to: entry, leaf. Spec status: The spec does not enforce this.

**OBS-05.** Code uses the `Context` forms of `slog` calls when a `ctx` is in scope, so each line can carry a correlation ID.

- Reason: One request must be traceable across lines and services.
- Source: [Go standard library, package log/slog](https://pkg.go.dev/log/slog) (context-aware methods); [OWASP Microservices Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Microservices_Security_Cheat_Sheet.html) (correlation IDs)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**OBS-06.** Each entry route and each gateway call reports latency as a histogram, traffic, errors, and saturation.

- Reason: These four signals answer most service questions, and averages hide the tail.
- Source: [Google SRE Book, Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/) (golden signals, tail latency); [OWASP Error Handling Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Error_Handling_Cheat_Sheet.html) (monitor 5xx)
- Check: only at run time, confidence medium.
- Applies to: entry, gateway. Spec status: The spec does not enforce this.

**OBS-07.** A gateway passes W3C `traceparent` on outbound calls, and an entry leaf accepts it.

- Reason: A portable trace ID links services.
- Source: [W3C Recommendation, Trace Context](https://www.w3.org/TR/trace-context/) (Recommendation, purpose)
- Check: only at run time, confidence medium.
- Applies to: entry, gateway. Spec status: The spec does not enforce this.
- Project facts: The trace system. Some external systems ignore the header.

### API surface

**API-01.** Each route that returns a collection pages from its first release, with a finite default and a maximum page size.

- Reason: Adding pages later is a breaking change.
- Source: [Google AIP-158, Pagination](https://google.aip.dev/158) (rules); [OWASP API4:2023 Unrestricted Resource Consumption](https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption) (record-count parameters)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.

**API-02.** A published API does not remove, rename, or change the type of a field in one major version, and does not add a required request field.

- Reason: Clients depend on each published name.
- Source: [Google AIP-180, Backwards compatibility](https://google.aip.dev/180) (compatibility rules)
- Check: before run, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Project facts: The published schema, as a baseline for comparison.

**API-03.** The build generates API documentation from an open standard.

- Reason: Hand-written docs drift.
- Source: [OWASP API9:2023 Improper Inventory Management](https://api-security.owasp.org/editions/2023/en/0xa9-improper-inventory-management) (generate documentation)
- Check: before run, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.
- Project facts: The API format.

**API-04.** A production build exposes no debug or profiling route, and each old API version has an owner and an end date.

- Reason: Forgotten endpoints are an attack surface.
- Source: [OWASP API9:2023 Improper Inventory Management](https://api-security.owasp.org/editions/2023/en/0xa9-improper-inventory-management) (inventory and retire); [gosec rule list (RULES.md)](https://raw.githubusercontent.com/securego/gosec/master/RULES.md) (G108)
- Check: before run, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Note: Importing `net/http/pprof` is checkable. Ownership of versions is for review.

**API-05.** Non-production environments use no production data, unless they have the same controls.

- Reason: Test systems are weaker.
- Source: [OWASP API9:2023 Improper Inventory Management](https://api-security.owasp.org/editions/2023/en/0xa9-improper-inventory-management) (production data)
- Check: by review only, confidence high.
- Applies to: project. Spec status: The spec does not enforce this.

**API-06.** A route's method matches its semantics: GET and HEAD are safe, and PUT and DELETE are idempotent.

- Reason: Clients and proxies retry on the basis of the method.
- Source: [RFC 9110, HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html) (sections 9.2.1, 9.2.2); [OWASP Cross-Site Request Forgery Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html) (no state change on GET)
- Check: by review only, confidence high.
- Applies to: entry. Spec status: The spec does not enforce this.
- Note: A repeat test can check this at run time for covered routes.

**API-07.** Error responses use one documented format, by default RFC 9457 problem details.

- Reason: Clients must parse errors in one way.
- Source: [RFC 9457, Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457.html) (purpose); [OWASP API8:2023 Security Misconfiguration](https://api-security.owasp.org/editions/2023/en/0xa8-security-misconfiguration) (response schemas include errors)
- Check: by review only, confidence medium.
- Applies to: entry. Spec status: The spec does not enforce this.
- Practice disagrees: RFC 9457 offers a standard. Large APIs such as Stripe keep their own format. **My choice:** I want one format, and RFC 9457 is only the default.

### Contracts

**CON-01.** A port method that changes state declares a `requires` for the limits of its arguments.

- Reason: A boundary is the best place to state what a caller must meet.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 2 Contracts are code, 7.4 item 2)
- Check: before run, confidence medium.
- Applies to: leaf. Spec status: The spec does not enforce this.
- Note: A tool can check that a clause exists. Only review can judge if it is enough.

**CON-02.** A rule that must stop an effect is a `requires` or a check before the effect, not an `ensures` or an invariant.

- Reason: Only `requires` runs before the effect.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 item 2)
- Check: before run, confidence medium.
- Applies to: leaf, library. Spec status: The spec's tools warn or report on this, and the build continues.

**CON-03.** A `return` in a function with `ensures` does not itself cause an effect, such as `return tx.Commit()`.

- Reason: Go runs the effect first, and the check then fires too late.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (7.4 item 3, K0525)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec's tools warn or report on this, and the build continues.
- Practice disagrees: The spec's own `Refund` example ends in `return tx.Commit()` on a function with no `ensures`, and says that a warning stays if a developer accepts the order. **My choice:** I keep it as a warning, and a comment records the choice.

**CON-04.** A fake bound in a test meets the contract of its port.

- Reason: A fake that breaks the contract tests a false world.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 8, G8); [Software Engineering at Google, Chapter 13, Test Doubles](https://abseil.io/resources/swe-book/html/ch13.html) (fidelity)
- Check: only at run time, confidence high.
- Applies to: test. Spec status: The spec says that the compiler enforces this in ordinary mode.

### Code quality

**CODE-01.** Code passes `go vet` with its full set of analyzers, and passes `staticcheck`.

- Reason: These tools catch real faults such as copied locks.
- Source: [Go command documentation, cmd/vet](https://pkg.go.dev/cmd/vet) (analyzers); [Staticcheck, Checks](https://staticcheck.dev/docs/checks/) (checks); [Go documentation, Security Best Practices for Go Developers](https://go.dev/doc/security/best-practices) (vet)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Note: Vulcan code compiles to Go, so a run over generated Go must map lines back.

**CODE-02.** Each suppression of a check names the check and gives a reason.

- Reason: An unexplained suppression hides a defect.
- Source: [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (nolintlint)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CODE-03.** A function above a set complexity gets a warning.

- Reason: Complex functions are hard to review.
- Source: [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (cyclop, gocognit, funlen)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.
- Practice disagrees: Linters offer limits, and teams dispute the numbers. The Google guide avoids hard size conventions. **My choice:** I keep a warning with a tunable number. It does not stop a build.

**CODE-04.** A `switch` over a named constant set covers each value, or has a `default` with a reason.

- Reason: A new state must not fall through quietly.
- Source: [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (exhaustive)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CODE-05.** A struct that code marshals has a field tag on each marshaled field.

- Reason: A rename must not change the wire format.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Use field tags in marshaled structs); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (musttag)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec does not enforce this.

**CODE-06.** An exported struct in a library does not embed another type.

- Reason: Embedding leaks methods into the API.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Avoid Embedding Types in Public Structs)
- Check: before run, confidence high.
- Applies to: library. Spec status: The spec does not enforce this.

**CODE-07.** A library `init` is deterministic, and reads no file, environment, or clock.

- Reason: Import order and environment must not change behavior.
- Source: [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (Avoid init()); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4 rule 12)
- Check: before run, confidence medium.
- Applies to: library. Spec status: The spec does not enforce this.
- Practice disagrees: The spec allows `init` in a library. Uber asks to avoid it where possible. **My choice:** I allow a deterministic `init`, such as a table build.

**CODE-08.** Each port, topic, `external`, library, and exported name has a doc comment.

- Reason: Diagrams and generated docs read these comments.
- Source: [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Doc Comments); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.3 examples, section 2 Self-documenting)
- Check: before run, confidence high.
- Applies to: leaf, library, system. Spec status: The spec does not enforce this.

**CODE-09.** Near-duplicate functions become one function, in the leaf or in a library.

- Reason: Duplicate code diverges, and models write much of it.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 9, K0412, R2)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec's tools warn or report on this, and the build continues.

**CODE-10.** Code passes `vulcan fmt` with no change.

- Reason: One format removes a class of review comment.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 2 Imposed structure); [Go Wiki, Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (Gofmt)
- Check: before run, confidence high.
- Applies to: leaf, library, system. Spec status: The spec does not enforce this.

**CODE-11.** Code does not use a deprecated function, and has no unused name.

- Reason: Deprecated code loses support, and dead code misleads a reader.
- Source: [Staticcheck, Checks](https://staticcheck.dev/docs/checks/) (SA1019); [golangci-lint, Linters](https://golangci-lint.run/docs/linters/) (unused); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 9, dead code)
- Check: before run, confidence high.
- Applies to: leaf, library. Spec status: The spec's tools warn or report on this, and the build continues.

### Tests

**TEST-01.** A change of behavior comes with a test in the same change, and the test fails when the code is broken.

- Reason: Google's review practice asks for tests in the same change, except in an emergency.
- Source: [Google Engineering Practices, What to look for in a code review](https://google.github.io/eng-practices/review/reviewer/looking-for.html) (Tests)
- Check: by review only, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.
- Note: A tool can check that tests changed. Mutation testing can give a run-time signal.

**TEST-02.** A test names one behavior, and calls the leaf the way a real caller does.

- Reason: A test of internals breaks on a harmless change.
- Source: [Software Engineering at Google, Chapter 12, Unit Testing](https://abseil.io/resources/swe-book/html/ch12.html) (Test behaviors; Test via public APIs)
- Check: by review only, confidence medium.
- Applies to: test. Spec status: The spec does not enforce this.

**TEST-03.** A test checks the resulting state, and checks calls only where the call is the effect.

- Reason: Call checks break on refactors and miss bugs.
- Source: [Software Engineering at Google, Chapter 12, Unit Testing](https://abseil.io/resources/swe-book/html/ch12.html) (Test state, not interactions); [Software Engineering at Google, Chapter 13, Test Doubles](https://abseil.io/resources/swe-book/html/ch13.html) (interaction testing)
- Check: by review only, confidence medium.
- Applies to: test. Spec status: The spec does not enforce this.
- Practice disagrees: The Google book prefers state checks. A gateway test has only a call as its effect, and `vulcan.Published` checks a call by design. **My choice:** I keep state as the default, and accept call checks at effects.

**TEST-04.** A test body has no branch or loop, apart from a loop over a table of cases.

- Reason: A test must be plainly correct on reading.
- Source: [Software Engineering at Google, Chapter 12, Unit Testing](https://abseil.io/resources/swe-book/html/ch12.html) (Don't put logic in tests); [Go Wiki, Table Driven Tests](https://go.dev/wiki/TableDrivenTests) (overview)
- Check: before run, confidence medium.
- Applies to: test. Spec status: The spec does not enforce this.
- Practice disagrees: The Google book says no logic in tests. The Go wiki recommends table-driven tests, which loop. **My choice:** I allow the one table loop.

**TEST-05.** A test does not call `time.Sleep`.

- Reason: A sleep wastes time or fails on a slow host.
- Source: [Martin Fowler, Eradicating Non-Determinism in Tests](https://martinfowler.com/articles/nonDeterminism.html) (Asynchronous behavior); [Go standard library, package testing/synctest](https://pkg.go.dev/testing/synctest) (fake clock)
- Check: before run, confidence high.
- Applies to: test. Spec status: The spec does not enforce this.

**TEST-06.** Each test builds its own state, and tests pass in any order.

- Reason: Shared state makes failures depend on order.
- Source: [Martin Fowler, Eradicating Non-Determinism in Tests](https://martinfowler.com/articles/nonDeterminism.html) (Isolation)
- Check: only at run time, confidence high.
- Applies to: test. Spec status: The spec does not enforce this.

**TEST-07.** A leaf test uses a fake at its ports, and the suite also has tests that use the real provider where the real one is fast and deterministic.

- Reason: A port fake gives speed, and a real provider gives fidelity.
- Source: [Software Engineering at Google, Chapter 13, Test Doubles](https://abseil.io/resources/swe-book/html/ch13.html) (real implementations, fakes); [Alistair Cockburn, Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/) (tests without a database); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 8)
- Check: by review only, confidence medium.
- Applies to: test. Spec status: The spec does not enforce this.
- Practice disagrees: The Google book prefers real implementations. Hexagonal design puts a fake at each port. **My choice:** I use both, at different levels.

**TEST-08.** A gateway has integration tests over its real transport against a test instance of the backend.

- Reason: A fake cannot show that SQL or a wire format is right.
- Source: [Google Go Style Guide, Best Practices](https://google.github.io/styleguide/go/best-practices) (Use real transports)
- Check: only at run time, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.
- Project facts: A test instance of each backend.

**TEST-09.** Scheduled contract tests check that a gateway's fake still matches the external system.

- Reason: The vendor can change, and the fake then lies.
- Source: [Martin Fowler, Contract Test](https://martinfowler.com/bliki/ContractTest.html) (idea, timing)
- Check: only at run time, confidence medium.
- Applies to: gateway. Spec status: The spec does not enforce this.
- Project facts: A sandbox of each external system.

**TEST-10.** Code that parses untrusted input has a fuzz test.

- Reason: Fuzzing reaches edge cases that people miss.
- Source: [Go documentation, Go Fuzzing](https://go.dev/doc/security/fuzz/) (purpose); [Go documentation, Security Best Practices for Go Developers](https://go.dev/doc/security/best-practices) (fuzzing)
- Check: by review only, confidence medium.
- Applies to: entry, gateway, library. Spec status: The spec does not enforce this.
- Note: The source states the method. Choosing parsers as targets is my reading of it.

**TEST-11.** A test that fails at random moves to a quarantine with a time limit and an owner.

- Reason: A flaky test must not hide the signal of healthy tests.
- Source: [Martin Fowler, Eradicating Non-Determinism in Tests](https://martinfowler.com/articles/nonDeterminism.html) (Quarantine)
- Check: by review only, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.

### Build and delivery

**BLD-01.** CI runs `govulncheck`, and a finding blocks a release.

- Reason: Known flaws in reached code must not ship.
- Source: [Go documentation, Security Best Practices for Go Developers](https://go.dev/doc/security/best-practices) (scan for vulnerabilities)
- Check: before run, confidence high.
- Applies to: project. Spec status: The spec does not enforce this.

**BLD-02.** The project uses a supported Go toolchain, and updates its dependencies after review and tests.

- Reason: Point releases carry security fixes, and each update can bring faults.
- Source: [Go documentation, Security Best Practices for Go Developers](https://go.dev/doc/security/best-practices) (keep up to date); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 6, Go versions and pinning)
- Check: before run, confidence medium.
- Applies to: project. Spec status: The spec does not enforce this.

**BLD-03.** The repository holds `go.mod` and `go.sum`, and the build pins its tools in `go.mod`.

- Reason: The build must be repeatable and must verify what it downloads.
- Source: [Go documentation, Managing dependencies](https://go.dev/doc/modules/managing-dependencies) (go.sum, tool dependencies); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (section 6, One go.mod pins everything)
- Check: before run, confidence high.
- Applies to: project. Spec status: The spec does not enforce this.

**BLD-04.** A released `go.mod` has no `replace` directive.

- Reason: Go does not authenticate replaced modules.
- Source: [Go documentation, Managing dependencies](https://go.dev/doc/modules/managing-dependencies) (replace)
- Check: before run, confidence high.
- Applies to: project. Spec status: The spec does not enforce this.
- Practice disagrees: A `replace` is useful during development, and for a forked fix under urgent need. **My choice:** I ban it at release only. A temporary fork needs a comment.

**BLD-05.** A pre-commit hook or CI runs a secret scanner.

- Reason: A leak in history is costly to fix.
- Source: [Stripe documentation, Best practices for managing secret API keys](https://docs.stripe.com/keys-best-practices) (secret scanner, pre-commit hook)
- Check: before run, confidence high.
- Applies to: project. Spec status: The spec does not enforce this.

**BLD-06.** CI runs `vulcan check`, `vulcan test`, and `vulcan audit`, and the count of unchecked paths does not grow without review.

- Reason: A rule that CI does not run is a wish.
- Source: [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.4d Paths around the rules, section 9)
- Check: before run, confidence high.
- Applies to: project. Spec status: The spec's tools warn or report on this, and the build continues.

**BLD-07.** Each deployment-specific value comes from a flag or from the environment, and leaf code holds no address or hostname literal.

- Reason: One build must run in each environment.
- Source: [The Twelve-Factor App, Config](https://12factor.net/config) (rule); [The Twelve-Factor App, Backing services](https://12factor.net/backing-services) (attached resources); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (5.3 Flags)
- Check: before run, confidence medium.
- Applies to: leaf, system. Spec status: The spec does not enforce this.
- Note: A tool can flag URL and host literals. Expect some false findings.

**BLD-08.** A change is small and does one thing.

- Reason: A small change is easier to review, and review is a main cost in AI-assisted work.
- Source: [Google Engineering Practices, Small CLs](https://google.github.io/eng-practices/review/developer/small-cls.html) (definition); [Vulcan Language Specification, Draft 0 (input/spec.md)](input/spec.md) (1.1)
- Check: before run, confidence low.
- Applies to: project. Spec status: The spec does not enforce this.
- Practice disagrees: Google gives about 100 lines as reasonable and 1000 as usually too large, and leaves the decision to reviewers. **My choice:** I keep a warning, not a gate.

**BLD-09.** A deployment sets limits on memory, CPU, restarts, and file descriptors.

- Reason: A process must not take down its host.
- Source: [OWASP API4:2023 Unrestricted Resource Consumption](https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption) (limit resources)
- Check: only at run time, confidence high.
- Applies to: project. Spec status: The spec does not enforce this.
- Project facts: The deployment platform.

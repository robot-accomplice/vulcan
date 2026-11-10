# Architecture linter for Stockroom

## Tool
- go-arch-lint (github.com/fe3dback/go-arch-lint), version v1.19.1 (`go-arch-lint version` prints v1.19.1; config file format version 3).
- Chosen over depguard because the README asks for component boundaries and dependency directions. go-arch-lint states these directly. depguard only denies or allows import paths per file group.
- Go toolchain: go1.27.3 windows/amd64.

## Install
    go install github.com/fe3dback/go-arch-lint@latest

## Run (from the app root, C:\Users\jonat\p-linter-work\app)
    go-arch-lint check --arch-file ../.go-arch-lint.yml --project-path .

The config sits one level above the app, because the app must not change. Exit code 0 = pass, 1 = findings.
Every package in the module is mapped to a component (11 packages, 11 components), so the whole module is checked.

## Rules (8; sources are also in the config comments)
| Id | Rule | Source |
|---|---|---|
| R1 | lib/shop (shared domain types) may be imported by every component (commonComponents) | https://www.gobeyond.dev/standard-package-layout/ (rule 1) |
| R2 | cmd/stockroom (main) may import any project package; nothing imports main | https://www.gobeyond.dev/standard-package-layout/ (rule 4); https://go.dev/doc/modules/layout |
| R3 | lib/shop imports no other project component (no lib -> api or cmd) | gobeyond (rule 1) |
| R4 | Domain services (inventory, orders, billing, purchasing, events) import only shop: not adapters, not each other | gobeyond (rule 2); https://alistair.cockburn.us/hexagonal-architecture/ |
| R5 | Adapters (web, store, payments, webhooks) import only shop: not each other, not domain services | gobeyond (rule 2) |
| R6 | web does not import store or the outbound clients directly | gobeyond (rule 2) |
| R7 | The SQLite driver (modernc.org/sqlite) is used by the store component only; no other third-party vendor is allowed anywhere (allow.depOnAnyVendor: false) | gobeyond (rule 2) |
| R8 | No import cycles | https://go.dev/ref/spec#Program_initialization_and_execution |

R4, R5 and R6 are expressed by the same mechanism: a component with no `deps` entry may import no other project component.
R3 and R8 are partly enforced by the compiler (cycles) and by the same mechanism.

## Base findings
None. `OK - No warnings found`, exit 0. The app was not changed.

## Check that the linter can fail
In a throwaway copy (deleted afterward), I added `import _ "stockroom/api/store"` to lib/shop. The linter reported a notice. A first run of the real app also found one real config error (see below), so the check does fail when it should.

## Choices
1. Layout from the README: 9 api packages + cmd + lib/shop. The README does not say which api packages are "domain" and which are "adapters". I took: inventory, orders, billing, purchasing, events = domain services; web, store, payments, webhooks = adapters. This follows the README descriptions (store = persistence, payments/webhooks = clients, web = HTTP API).
2. Strictest reading: peer packages may not import each other. The base app already meets it (every package imports only lib/shop, and main imports all). No business-logic rule was added.
3. Config location: outside app/ (the app is not to be changed), so `--arch-file ../.go-arch-lint.yml` is needed.
4. Empty `mayDependOn: []` is rejected by the tool. The equivalent is to omit the component from `deps`.
5. Config fix made: the vendor mask `modernc.org/sqlite/**` did not match the import `modernc.org/sqlite` itself (a blank import in api/store/store.go). The first run reported it as a violation. I added the plain path to the vendor `in` list. This was a config error, not a weakened rule.
6. deepScan left at its default (on). It checks method calls and injection across components. The standard library is not restricted (go-arch-lint checks project and vendor imports).
7. R8 is mainly the compiler's job. It is listed so the rule set shows the "no cycles" practice. The tool did not check test files separately; I did not verify how it treats `_test.go` files (the app has none in this copy).
8. The depguard route was not tried.

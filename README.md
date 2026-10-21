# Vulcan

Vulcan is a compiled, garbage-collected language. Humans and LLMs use it to build software
together. It is Go with an enforced architecture layer, contracts, and tools around the code.

- **Function bodies are plain Go**, with Go's syntax and behavior.
- **Architecture is part of the language:** C4 systems, containers, components, ports, and
  topics. The compiler enforces the boundaries.
- **Contracts** (`requires`, `ensures`, `invariant`) are part of declarations. They run in every
  build, and reviewers can audit them.
- **The tools teach the model** how to use the language, from a primer with a fixed token budget.

**Status:** specification draft. Nothing is implemented. There is no compiler yet. Source files
end in `.vulcan`, and the command is `vulcan`. The experiments keep the earlier working name, Keel.
See [Status and handoff](docs/status.md).

## Why Vulcan

Most failures in human and AI co-development come from missing structure. The model cannot see
the architecture. It invents APIs, duplicates code, and masks errors. The human cannot review all
that the model wrote. Vulcan puts the structure in the language, so the compiler can see it,
enforce it, draw it, and teach it.

The premise (spec §1.1): with an AI model, a developer spends most time on three tasks. The
developer states the requirements, reads the code, and validates and corrects it. Vulcan makes the
second and third tasks shorter. Two early writing experiments suggest where the gain is. Their
data is lost and no compiler existed, so these are hypotheses that a compiler must test:

- Vulcan takes the architecture rules out of review. The compiler checks them on every change.
- Vulcan rejects rule-breaking changes before review, so a model fixes them against the compiler.
- Vulcan does not make a thorough reviewer read less code. It does not reduce logic defects that
  no rule covers.

Spec §3 maps each observed LLM failure (lost context, duplication, error masking, invented APIs)
to the feature that answers it, with sources in Appendix A.

## A short example

A leaf declares the ports that it needs, with their contracts:

```go
// api/orders/component.vulcan
import "shop/lib/money"

component Orders {
    // Checkout takes payment for a confirmed order.
    needs port Checkout {
        func Pay(customer money.CustomerID, amount money.Money) (money.Receipt, error)
            requires amount > 0 && amount < money.MaxOrder
            ensures  err == nil => result.Amount == amount
    }
}
```

Any ordinary Go type with a matching method set can provide the port. The provider never names
its consumers. One architecture file wires the leaves together:

```go
// system.vulcan
system Shop {
    external Stripe "Card payment processor"

    container Api {
        Web, Billing, Orders, StripeGateway

        Orders.Checkout:     Billing.NewPayer
        Web.Checkout:        Billing.NewPayer
        Billing.CardNetwork: StripeGateway.NewCards{ APIKey: flag("stripe-key") }
    }
}
```

Inside `Orders`, the code calls `Checkout.Pay(c, amt)` like a Go package. A port call is an
ordinary function call: one interface dispatch, with the contract checked at the boundary. The
compiler rejects a reference to another leaf that does not go through a port, and a port that
`system.vulcan` does not wire.

## What Vulcan adds to Go

| Area | What it gives | Spec |
|---|---|---|
| Architecture | Systems, containers, groups, leaves, libraries, gateways, and consumer-owned ports. The compiler enforces the rules between them, and lists each path around them that it cannot check. A strict mode rejects most of those paths. | §5, §5.4d |
| Crossing processes | Topics and remote ports between containers, with transports, dead letters, and a `topics.lock` for compatibility. | §5.5 |
| Contracts | `requires`, `ensures`, `old()`, and struct invariants, with exact arithmetic in contract expressions. | §7 |
| Testing | Fakes bound to port names, contract-aware fakes, and per-leaf tests. | §8 |
| Compiler intelligence | Diagnostics that explain why and propose fixes, plus warnings for duplication, masked errors, and races. | §9, Appendix E |
| Tools for models | `guide` (a primer of at most 4,000 tokens), `context` (a minimal context pack), `query` (the exact symbol index), and `audit`. | §10 |
| Runtime and backend | Lowering to Go source first, with LLVM later. The compiler is written in Go. | §11, Appendix F |

Non-goals for now (§12): manual memory management, inheritance, macros, and a general effect
system.

## Repository layout

| Path | Contents |
|---|---|
| [docs/spec.md](docs/spec.md) | The specification: design, evidence, grammar (Appendix B), runtime API (C), network rules (D), diagnostics (E), and lowering to Go (F). Open questions are in §13. |
| [docs/guide.md](docs/guide.md) | The primer that `vulcan guide` prints for a model. |
| [docs/status.md](docs/status.md) | Where the work stands, the design rules for new decisions, and the open work in order. |
| [docs/development-plan.md](docs/development-plan.md) | The plan (accepted): test the rules as Go tools first, check that the final evaluation can give an answer, then build a compiler and a real application, and decide by fixed rules whether the language earns its cost. |
| [experiments/](experiments/) | Spec reviews 8 and 9 with their resolutions, writing test 3, audit 2 of the spec and the test 3 code, audits 3 to 10 of the development plan, and the sizing check of its evaluation. |
| [experiments/test3/](experiments/test3/) | The same shop system in Go and in Keel, the feature brief, the writers' work, and the hidden grading rules. |

## Reading order

1. This README, then [docs/guide.md](docs/guide.md) for the language in brief.
2. [docs/spec.md](docs/spec.md) §1 to §3 for the purpose and the evidence, then §5 and §7.
3. [docs/status.md](docs/status.md) for the decisions so far and what comes next.

## Running the experiment code

Only the Go fixtures of test 3 run today. They need Go 1.22 or later.

```bash
cd experiments/test3/base-go && go test ./...
```

```bash
cd experiments/test3/work/go && go test ./...
```

The Keel versions (`base-keel`, `work/keel`, written under the working name) cannot build,
because no compiler exists. They were
checked by hand against the spec (`base-keel/SELF-CHECK.md`).

## Contributing

- Read the design rules in [docs/status.md](docs/status.md) before you propose a change.
- Write prose in Simplified Technical English: short active sentences, one idea each. Do not use
  em-dashes.
- Keep experiment data and reports in `experiments/`. Do not change the old experiment records.
- Commits and comments carry no AI attribution.
- Each question to the owner states its context, the options, and a recommendation.

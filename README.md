# Vulcan

Vulcan is a compiled, garbage-collected language. Humans and LLMs use it to build software
together. It is Go with an enforced architecture layer, contracts, and tools around the code.

- **Function bodies are plain Go**, with Go's syntax and behavior.
- **Architecture is part of the language:** C4 systems, containers, components, ports, and
  topics. The compiler enforces the boundaries.
- **Contracts** (`requires`, `ensures`, `invariant`) are part of declarations. They run in every
  build, and reviewers can audit them.
- **The tools teach the model** how to use the language, from a primer with a fixed token budget.

**Status:** specification draft. Nothing is implemented. The spec text still uses the working name
"Keel"; see [docs/status.md](docs/status.md).

- [Specification](docs/spec.md): the design, the research behind it (§3 and Appendix A), and the
  open questions (§13).
- [Primer](docs/guide.md): what `vulcan guide` prints for a model, at most 4,000 tokens.
- [Status and handoff](docs/status.md): where the work stands and what comes next.
- [Experiments](experiments/): the writing test fixtures and the spec reviews with their
  resolutions.

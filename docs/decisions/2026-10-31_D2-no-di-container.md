# D2 — No DI Container (samber/do Declined)

**Date:** 2026-10-31
**Status:** Accepted
**Source:** architecture review `docs/architecture-understanding/2026-10-31_14-41_service-orientation-resilience-selfhealth.md` (verdict R5)

## Context

The repo is a Go SDK (pkg/vision) plus one daemon (internal/reviewd, wired
from cmd/visionreviewd). The question was whether service orientation
deserved a DI container (samber/do v2). The DO-1 → DO-6 anti-pattern audit
found **samber/do is not used at all** — only `samber/lo` arrives
transitively — while the existing manual constructor wiring already passes
every analog: single composition root (`openPipeline` /
`runDaemonCommand`), validated constructors with sentinel errors
(`NewDaemon`, `NewPipeline`, `NewHealthGate`, `NewReviewer`), interfaces at
the seams consumers actually substitute (`PassRunner`, `Analyzer`,
`ServiceNotifier`), zero global injector state, and self-contained
`Close` paths.

## Decision

**Keep manual constructor injection. Do not adopt samber/do** (or any DI
container) absent a triggering need below.

Rationale: the daemon has ~5 services wired in one place, once, at process
start. A container would add a dependency, a lifecycle API, and an
indirection layer to solve problems (lazy resolution, graph-wide health
reporting, runtime re-wiring) this system does not have. The health-gate
work proved the seam model works: `HealthGate` decorates `PassRunner`
without any container support, and the NixOS watchdog rides the same
`ServiceNotifier` interface.

## Consequences

- Wiring stays greppable: `openPipeline` in cmd/visionreviewd/commands.go
  is the only place the graph is assembled.
- Tests substitute dependencies through constructor args and interfaces,
  not through an injector.
- samber/do stays OFF the dependency list; `samber/lo` usage is transitive
  only and not an adoption signal.

## Revisit triggers

Adopt a DI container (re-audit against samber/do v2 best practices) when
ANY of these becomes true:

1. The wiring graph exceeds ~10 services or grows a second composition
   root (e.g. a second long-running binary sharing internal/reviewd).
2. A service needs lazy/runtime resolution (plugins, on-demand providers).
3. Circular or near-circular dependency ordering appears and starts
   requiring init-order hacks.
4. Graph-wide health reporting or graceful shutdown coordination gets
   duplicated in more than one binary.

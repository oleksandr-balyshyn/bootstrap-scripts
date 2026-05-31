---
name: go-refactoring-review
description: Use when reviewing, refactoring, or improving a Go codebase with emphasis on maintainability, testability, SOLID/GRASP responsibility assignment, evolutionary architecture, continuous delivery, and safe incremental change. Applies to Go CLIs, services, libraries, modular monoliths, and legacy systems.
metadata:
  short-description: Go refactoring and architecture review
---

# Go Refactoring Review

Use this skill to analyze and improve Go projects through small, safe, behavior-preserving steps. Favor the practices associated with Fowler, Beck, Feathers, Rainsberger, Farley, GeePaw Hill, Belshee, Troy, and Bellotti: fast feedback, refactoring before redesign, explicit dependencies, operability, and simple designs that can evolve.

## Operating Principles

- Preserve behavior unless the user explicitly requests a behavior change.
- Prefer characterization tests before changing risky or poorly understood code.
- Prefer small reversible changes over broad rewrites.
- Prefer Go idioms over class-heavy OOP translations.
- Introduce interfaces only at useful boundaries: external systems, side effects, policy variation, tests, or dependency inversion.
- Avoid speculative patterns. If a function or struct is enough, use it.
- Keep packages cohesive and dependencies pointing inward toward domain/application logic.
- Treat build, test, deployability, logging, and failure modes as design concerns.

## Initial Reconnaissance

Start by understanding the project before prescribing structure.

Run or inspect, as appropriate:

```bash
go test ./...
go vet ./...
go list ./...
go list -deps ./...
rg --files
rg -n "TODO|FIXME|panic\\(|os\\.Exit|log\\.Fatal|http\\.DefaultClient|exec\\.Command|time\\.Now\\(|interface \\{|go "
```

Inspect:

- `go.mod`, `go.work`, build scripts, CI workflows, Dockerfiles, Makefiles/Justfiles.
- `cmd/`, `internal/`, `pkg/`, service entrypoints, handlers, adapters, config loading, persistence, and external clients.
- Test files, test helpers, fixtures, integration tests, and slow tests.

Produce a compact system map:

```text
entrypoints -> application orchestration -> domain/core -> adapters/infrastructure
```

For CLIs:

```text
main -> app.Run(args, streams) -> domain/application services -> filesystem/network/process adapters
```

For services:

```text
main -> server wiring -> handlers -> application services -> domain -> repositories/clients
```

## Design Review Checklist

### Code Smells

Look for:

- God files/packages
- long functions
- feature envy
- primitive obsession
- hidden dependencies
- temporal coupling
- global mutable state
- broad interfaces
- duplicated orchestration
- shell/stringly typed command construction
- configuration coupled to business behavior
- tests that assert implementation details

### SOLID In Go Terms

- SRP: Does a package/type/function have one coherent reason to change?
- OCP: Can new behavior be added by adding a focused type/function/config entry, or must central code be repeatedly edited?
- LSP: If interfaces exist, do all implementations obey the same behavioral contract?
- ISP: Are interfaces small and consumer-owned?
- DIP: Do core packages depend on abstractions while infrastructure depends on details?

Avoid creating interfaces for every concrete type. In Go, consumers usually define the interface they need.

### GRASP In Go Terms

- Information Expert: Put behavior near the data and rules it uses.
- Creator: Let construction live near configuration/wiring, not inside domain logic.
- Controller: Keep entrypoints thin; use application services for workflows.
- Low Coupling: Isolate filesystem, clock, network, database, subprocess, environment, and randomness.
- High Cohesion: Split packages by responsibility, not by generic technical bucket alone.
- Polymorphism: Use interfaces or function parameters when real variation exists.
- Indirection: Add seams only where change or test pressure justifies it.
- Protected Variations: Wrap unstable APIs and external systems behind adapters.

## Testing Strategy

Classify existing tests:

- Unit tests: fast, deterministic, isolated, domain/application behavior.
- Characterization tests: capture current behavior before refactoring legacy code.
- Integration tests: verify real adapters, persistence, subprocesses, or network boundaries.
- End-to-end tests: minimal coverage for critical happy paths.

Prefer:

- table-driven tests for pure rules
- fake adapters for side effects
- golden tests only for stable text/format output
- explicit clocks/readers/writers/filesystems where needed
- `app.Run(args, streams)` or equivalent seam for CLIs

Watch for:

- sleeps in tests
- tests requiring global host state
- tests that depend on execution order
- large integration suites substituting for unit tests
- mocks that mirror implementation instead of behavior

## Refactoring Workflow

Use this sequence:

1. Establish a safety net: run tests, add characterization tests for risky behavior.
2. Make the smallest structure-preserving improvement.
3. Run fast checks.
4. Commit or prepare a small reviewable diff.
5. Repeat.

Preferred refactorings:

- Extract Function for long methods.
- Move Function to the package that owns the responsibility.
- Extract Package for cohesive infrastructure/adapters.
- Introduce Parameter Object when arguments travel together repeatedly.
- Replace package-level mutable vars with explicit dependencies.
- Split command-line `main` into `main` plus testable `app.Run`.
- Extract side-effect adapters for filesystem, process execution, HTTP, DB, clock, and env.
- Replace central switches with a registry or strategy only after installer/handler/type growth proves the need.

## Architecture Guidance

Prefer modular monolith first. Reach for hexagonal/clean architecture only where there are meaningful boundaries.

Good Go package boundaries usually look like:

```text
cmd/<binary>        process boundary only
internal/app        orchestration/use cases
internal/domain     business concepts and rules
internal/adapters   database, HTTP clients, filesystem, subprocess, queues
internal/config     config parsing and validation
```

Do not force this layout if the project is small. A few cohesive files in one package can be better than premature architecture.

## Pattern Heuristics

Before applying a pattern, answer:

- What concrete change pressure exists?
- What coupling does this reduce?
- What complexity does this add?
- Can a smaller function, package split, or explicit dependency solve it?
- Will tests become simpler?
- Will deploy/build/operate become simpler?

Usually justified:

- Adapter for external systems.
- Facade for complex third-party APIs.
- Strategy for real algorithm/provider variation.
- Command for queued/deferred/retryable operations.
- Value Object for validated domain values.
- Application Service for workflows that coordinate domain and adapters.

Usually suspicious:

- Factories without construction complexity.
- Repositories for simple file reads or static config.
- Event-driven architecture without independent lifecycle or scale pressure.
- CQRS/Event Sourcing without strong audit/history/query-model needs.
- Service locator or hidden global registries.

## Continuous Delivery Review

Check:

- `go test ./...`, `go vet ./...`, linting, race checks where valuable.
- CI permissions are minimal.
- Actions are pinned or intentionally versioned.
- Build artifacts are reproducible enough for the project risk.
- Vulnerability scanning exists for dependencies.
- Config validation and smoke tests run in CI.
- Rollback or failure behavior is explicit.
- Logs/errors expose actionable failure context.

For CLIs, add smoke checks such as:

```bash
go build -o bin/tool ./cmd/tool
./bin/tool --help
./bin/tool validate
```

## Output Format For Reviews

When asked to analyze a project, return:

1. Executive summary
2. System overview and dependency map
3. Highest-risk design smells with file references
4. SOLID/GRASP assessment, concise scores if helpful
5. Testability and CI/CD assessment
6. Prioritized refactoring roadmap
7. First 1-3 safe implementation steps
8. Tradeoffs and what not to do yet

Lead with actionable risks, not generic design theory.

## Implementation Rules

When asked to implement:

- Keep the diff narrow.
- Do not mix behavior change with structural refactoring unless unavoidable.
- Add or update tests with each risky change.
- Prefer package extraction before interface extraction.
- Keep `main` thin.
- Make dependencies explicit at boundaries.
- Preserve public APIs unless the user approves a breaking change.
- Run relevant checks and report exactly what passed or could not be run.

## Stop Conditions

Pause or ask before:

- broad rewrites
- changing persistence schemas
- changing public APIs
- replacing frameworks
- adding large dependencies
- introducing async/event-driven designs
- altering security-sensitive behavior without a test/rollback path

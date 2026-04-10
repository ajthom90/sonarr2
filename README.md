# sonarr2

A feature-complete rewrite of [Sonarr](https://github.com/Sonarr/Sonarr) focused on performance.

Status: **early design phase.** See [the design doc](./docs/superpowers/specs/2026-04-10-sonarr-rewrite-design.md) for the full plan.

## Why another Sonarr?

Sonarr is an excellent PVR, but its current .NET implementation struggles with large libraries (tens of thousands of series, shows with thousands of episodes like Jeopardy). sonarr2 is a ground-up rewrite in Go targeting:

- **Performance on large libraries** — responsive UI on 29,000+ series, 500,000+ episode files.
- **Low-resource deployment** — runs comfortably on a 2 CPU core / 4 GB RAM homelab.
- **Drop-in replacement** — wire-compatible with Sonarr's v3 REST API so existing integrations keep working, plus a one-shot migration tool for importing an existing Sonarr database.
- **Multi-arch** — first-class support for `linux/amd64`, `linux/arm64`, `linux/arm/v7`, macOS, Windows.
- **Open source, MIT licensed** — freely reusable without GPL restrictions.

## Relationship to upstream Sonarr

sonarr2 is a **clean-room reimplementation**. It re-implements the Sonarr v3 REST API surface (APIs themselves are not copyrightable — see _Google v. Oracle_, 2021), but does not copy any source code from Sonarr. The Sonarr project is licensed under GPL-3.0; sonarr2 is licensed under MIT.

sonarr2 is not affiliated with the Sonarr project.

## License

MIT — see [LICENSE](./LICENSE).

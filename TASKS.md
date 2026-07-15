# Tasks

Only the product father or integrator changes priority. Each Agent takes one item on its own branch.

## Now

- [x] Give every indexed record a stable provider-qualified ID, migrate ctx metadata safely, and add strict `search --project/--provider` filters for RC1.2.
- [x] Fix, format, build, and test the Go bootstrap.
- [x] Verify Codex scanning against synthetic fixtures and the local read-only catalog.
- [x] Verify Claude Code scanning against synthetic fixtures and the local read-only catalog.
- [x] Test `scan → search → show → name → snapshot → diff` end to end.

## Next

- [x] Add incremental scan without a daemon.
- [x] Freeze versioned JSON output and add golden tests.
- [x] Fix the public product name as Clewkeep and align the module and release archives.
- [x] Add release archives and checksums (delivered as six platforms: windows/darwin/linux × amd64/arm64).
- [ ] Recruit 20 private testers and record clean-install failures.
- [ ] Create a 30-second demo and honest competitor comparison.

## Later

- [ ] Explicit selection into a custom context repo.
- [ ] Content-addressed packs and deterministic metadata merge.
- [ ] Remote push/pull of selected encrypted objects.
- [ ] Model-assisted merge proposals with evidence.

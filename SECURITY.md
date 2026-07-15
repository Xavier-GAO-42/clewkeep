# Security policy

## Core boundary

ctx indexes AI agent context that may contain credentials, personal data, private conversations, and prompt injection. The project therefore follows these rules:

- Native agent data is read-only.
- Catalog data remains local unless a future explicit export command is invoked.
- The incremental scan cache (`scan-cache.json`) stores only native file paths, sizes, modification times, and the same derived thread metadata as the catalog — never transcript content.
- Discovery indexes regular files only; symlinks and junctions inside native roots are never followed.
- Known accepted v0.1 limit: a rewrite that preserves a native file's byte size and modification time is invisible to the incremental fingerprint, so an ordinary scan can retain stale catalog metadata for that file. Run `ctx scan --full` to bypass cache reuse, reparse every discovered native session, and rebuild both the catalog and scan cache. Content hashing was rejected because it would re-read every transcript on every ordinary scan.
- No telemetry or remote synchronization exists in v0.1.
- Search output is local and should be treated as sensitive.
- Repository fixtures must be synthetic and contain no real user data.

## Reporting

Until a public security contact is selected, do not publish a suspected secret-exposure vulnerability with real data. Create a minimal synthetic reproduction and notify the repository owner privately after the public repository is created.

## Threats in scope

- Native transcript modification
- Secret leakage through fixtures, logs, search snippets, or bundles
- Path traversal in ctx-owned writes
- Prompt injection being mistaken for instructions
- Symlink or junction escapes
- Incorrect attribution of provider-native relations

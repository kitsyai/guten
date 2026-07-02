---
id: TASK-CLI-MULT-DIRE-RESO-1
short_id: bed1fe795cad
title: "CLI: multi-directory resolution for -d and @file inputs"
type: feature
status: todo
created: 2026-07-02
updated: 2026-07-02
aliases: []
priority: p2
track: unassigned
acceptance:
  - guten export --lib X -d otp/sample.json resolves without a full @path
  - clear error enumerates searched locations
tests_required: []
---
Resolve -d/@file inputs across a search path before erroring, node/rc-style: (1) path as given (CWD/relative/absolute), (2) the input's own directory, (3) ~/.kitsy/guten/user, (4) ~/.kitsy/guten/gutenkit, then error listing what was searched. Applies to -d, -t, --css, --manifest, --slot. Mind path-traversal safety.
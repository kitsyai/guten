---
id: TASK-NPM-WRAP-KITS-GUTE-1
short_id: 1f6b9841093f
title: npm wrapper @kitsy/guten-cli that installs/updates the native binary
type: feature
status: todo
created: 2026-07-02
updated: 2026-07-02
aliases: []
priority: p2
track: unassigned
acceptance:
  - npm i -g @kitsy/guten-cli then 'guten version' works on win/mac/linux
  - wrapper selects correct platform archive + verifies checksum
tests_required: []
---
npm i -g @kitsy/guten-cli installs a thin JS wrapper that downloads the matching guten CLI binary from the cli/vX goreleaser GitHub release for the host platform/arch, caches it, and execs it. Removes the manual download/extract/PATH steps.
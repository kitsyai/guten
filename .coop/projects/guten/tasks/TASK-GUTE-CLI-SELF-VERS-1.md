---
id: TASK-GUTE-CLI-SELF-VERS-1
short_id: f4e0d0b0f0d8
title: guten CLI self version-check + auto-update
type: feature
status: todo
created: 2026-07-02
updated: 2026-07-02
aliases: []
priority: p2
track: unassigned
acceptance:
  - "'guten update' replaces the binary with the latest release"
  - version check is cached and never blocks normal commands
tests_required: []
---
guten checks the latest cli/vX release via GitHub API (cached ~daily, non-blocking), warns when the running binary is outdated, and provides 'guten update' to fetch+replace the binary in place.
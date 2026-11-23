---
title: Worktrunk status markers for pi sessions
type: feature
author: devopstales
created: 2026-11-22T00:00:00.000Z
---

The plugin updates Worktrunk branch markers so `wt list` shows whether pi is
actively working or waiting for input: session start sets waiting (💬), agent
start sets working (🤖), agent end sets waiting (💬) again, and session
shutdown clears the marker.

If a session is killed before shutdown, the marker can remain stale; the plugin
reports it and it is removed manually with `wt config state marker clear`.

This plugin is the sole writer of Worktrunk marker state while installed. If
Worktrunk's own pi plugin is also installed, both write the same markers — the
README notes the conflict (no runtime detection).

*By @devopstales.*

---
name: tui-stories
description: Add, change, check, or look at uncors TUI widgets (internal/uncors_app) through stories — the Storybook-like catalog in stories.go, the tuiplay playground, story snapshots, and frame renders. Use when changing how the TUI looks or behaves, adding a widget or widget state, reviewing a TUI diff, or when asked to show or verify what the TUI looks like.
---

# TUI stories

A story is one widget (or the whole `UncorsApp`) in one state with fixed data.
`internal/uncors_app/stories.go` lists them; `Stories()` returns them in
display order, named `<widget>/<state>`.

The same list feeds three tools:

| Tool          | Command                                                      | Purpose                                                                                                                         |
| ------------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| Snapshot test | `go test ./internal/uncors_app/ -run TestStories`            | Every story at 40/80/120 columns, plain text then styled, in `testing/snapshots/internal/uncors_app/stories_internal_test.snap` |
| Render        | `go run ./tools/tuiplay -render <story> -width <n> [-plain]` | One frame to stdout, cut like Bubble Tea cuts it                                                                                |
| Playground    | `make playground`                                            | For the user: pick a story, see it full screen, rebuilt on every Go change                                                      |

The playground is for the user. You cannot drive it: it needs a terminal that
answers Bubble Tea's startup queries. Use `-render` instead.

## Looking at the TUI

1. `go run ./tools/tuiplay -list` for the names.
2. `go run ./tools/tuiplay -render <story> -width 80 -plain` to read the layout.
   Try 40 too: narrow widths show cut text and hidden parts.
3. For colours, render without `-plain` and read the escape codes
   (`| cat -v`), or compare the styled half of the story's snapshot.

## Adding a story

- Widget: `widgetStory(widget, resize, msgs...)`. `resize` is for widgets whose
  height the app sets (see `historyStory`); pass nil otherwise.
- Whole app: `appStory(msgs)`; it replaces the memory widget with a fixed one.
- Feed state as messages the widget really receives: `requestStartedMsg`,
  `requestCompletedMsg` and `lifecycleMsg` (`storyLifecycle` wraps events),
  `outputLineMsg` (`storyRendered` renders events like the app), keys with
  `storyKey`. Reuse `storyPending`, `storyTraffic`, `storyStarted`.
- Set `Height` when the widget fills the terminal (history, app); leave 0 to fit
  the content.
- Data must be fixed: no `time.Now`, real memory, random IDs, or map order in
  the output. A flaky snapshot means the widget or the story depends on one of
  these; fix the cause (inject it, sort it).

Then run `make stories` (or `UPDATE_SNAPS=true go test ./internal/uncors_app/ -run TestStories`)
and read the new entries in the `.snap` file before keeping them.

## Changing a widget

1. Render the affected stories at 40 and 80 before the change.
2. Make the change.
3. `go test ./internal/uncors_app/`: a failing `TestStories` shows the visual
   diff. Check that only the stories you meant to change changed, then
   `make stories`.
4. When colours changed, check the styled snapshot diff, not only the plain one.
5. Mention the stories that changed when you report back.

Stories live in production code (package `uncorsapp`) because they need its
unexported message types; the linker drops them from the uncors binary.

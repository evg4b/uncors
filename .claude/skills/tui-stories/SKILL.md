---
name: tui-stories
description: Add, change, check, or look at uncors TUI widgets (internal/tui) through stories — the Storybook-like catalog in testing/stories, the tuiplay playground, story snapshots, and frame renders. Use when changing how the TUI looks or behaves, adding a widget or widget state, reviewing a TUI diff, or when asked to show or verify what the TUI looks like.
---

# TUI stories

A story is one widget (or the whole `UncorsApp`) in one state with fixed data.
`testing/stories` lists them; `stories.Stories()` returns them in
display order, named `<widget>/<state>`.

The same list feeds three tools:

| Tool          | Command                                                      | Purpose                                                                                                            |
| ------------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------ |
| Snapshot test | `go test ./testing/stories/`                                 | Every story at 40/80/120 columns, plain text then styled, in `testing/snapshots/testing/stories/stories_test.snap` |
| Render        | `go run ./tools/tuiplay -render <story> -width <n> [-plain]` | One frame to stdout, cut like Bubble Tea cuts it                                                                   |
| Playground    | `make playground`                                            | For the user: pick a story, see it full screen, rebuilt on every Go change                                         |

The playground is for the user. You cannot drive it: it needs a terminal that
answers Bubble Tea's startup queries. Use `-render` instead.

## Looking at the TUI

1. `go run ./tools/tuiplay -list` for the names.
2. `go run ./tools/tuiplay -render <story> -width 80 -plain` to read the layout.
   Try 40 too: narrow widths show cut text and hidden parts.
3. For colours, render without `-plain` and read the escape codes
   (`| cat -v`), or compare the styled half of the story's snapshot.

## Adding a story

Stories use only the public API of `internal/tui`; add a small public
seam there (an option, a method) rather than reaching into internals.

- Widget: `widgetStory(widget, resize, msgs...)` in `models.go`. `resize` is
  for widgets whose height the app sets (see `historyStory`); pass nil
  otherwise.
- Whole app: `appStory(msgs)`; it passes a fixed memory widget through
  `tui.WithMemoryWidget`.
- Feed state as the app does: engine events through `tui.EventMsg`
  (`eventMsgs`, `started`), history output with `HistoryWidget.AppendLine`
  (`rendered` renders events like the app), keys with `key`. Reuse the
  fixtures in `fixtures.go`: `pending`, `traffic`, `serverStarted`.
- Set `Height` when the widget fills the terminal (history, app); leave 0 to fit
  the content.
- Data must be fixed: no `time.Now`, real memory, random IDs, or map order in
  the output. A flaky snapshot means the widget or the story depends on one of
  these; fix the cause (inject it, sort it).

Then run `make stories` (or `UPDATE_SNAPS=true go test ./testing/stories/`)
and read the new entries in the `.snap` file before keeping them.

## Changing a widget

1. Render the affected stories at 40 and 80 before the change.
2. Make the change.
3. `go test ./testing/stories/`: a failing `TestStories` shows the visual
   diff. Check that only the stories you meant to change changed, then
   `make stories`.
4. When colours changed, check the styled snapshot diff, not only the plain one.
5. Mention the stories that changed when you report back.

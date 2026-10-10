# Change Flow

Change Flow turns code changes into a navigable request path. Open it with **Flow** in the editor toolbar, `Cmd/Ctrl+Shift+G`, or **View: Change Flow** in the command palette.

## Open from any repository

Run `make git-px0-flow` once from the px0-flow checkout. It installs a global git alias, so `git px0-flow` inside any repository rebuilds px0-flow from that checkout and opens Change Flow on the repository's changes. Flags are passed through, for example `git px0-flow -port 0` or `git px0-flow -agent claude`. The alias always runs with `-no-update`, because the self-updater downloads upstream px0 and would replace the px0-flow build. Remove it with `git config --global --unset alias.px0-flow`.

## Scopes

- **Working tree** — staged, unstaged, and untracked changes compared with `HEAD`.
- **Pull request** — files changed between the PR merge base and checked-out PR head. Available in PR review sessions.
- **Thread** — files recorded on an agent thread turn. The default is the most recent finished turn that touched files; a specific thread can be selected in the view.

The graph groups changed symbols into UI, API, service, and database layers. It intersects diff line markers with px0's regex symbol outline and falls back to a file node when a changed file has no matching symbols. Deleted files remain visible as file nodes.

## Relationships

Edges are inferred from references between changed symbols, common `fetch` / Axios URL patterns, and references to database symbols. These text-based edges are approximate. When a language server is already running, px0 adds one outgoing call-hierarchy level; opening Change Flow never starts a language server.

Node colors distinguish added, modified, deleted, and contextual code. Select a node to open its file at the relevant line in the existing diff view. The graph re-renders from the Git status stream while it is open.

## Plan and compare

Select **Thread**, choose a thread, then click **Plan flow** to ask its selected harness for a read-only `flowjson` plan. The request is saved in that thread transcript under px0's normal config directory. The plan is overlaid on the actual thread change map: planned work remains dashed, matching changes use their actual status, and unplanned changes are highlighted. The plan prompt tells the harness not to edit or create files; as with any harness response, review the thread reply.

## API

`GET /api/flow?scope=working|pr|thread&id=<thread-id>` returns nodes and edges. Thread scope also accepts `turn=<1-based-turn-number>`; without it, the latest finished turn with recorded changed files is used.

```json
{
  "scope": "working",
  "nodes": [
    {"id": "src/ui/AddressForm.tsx#submitAddress", "label": "submitAddress", "layer": "ui", "path": "src/ui/AddressForm.tsx", "line": 42, "status": "modified"}
  ],
  "edges": [
    {"from": "src/ui/AddressForm.tsx#submitAddress", "to": "src/api/checkout.ts#POST", "kind": "fetch"}
  ]
}
```

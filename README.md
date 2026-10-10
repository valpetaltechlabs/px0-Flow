<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://px0.ai/logo/px0-logo-dark.png">
  <source media="(prefers-color-scheme: light)" srcset="https://px0.ai/logo/px0-logo-light.png">
  <img alt="px0" src="https://px0.ai/logo/px0-logo-dark.png" width="120">
</picture>

# px0-flow

---

px0-flow extends [px0](https://github.com/px0-ai/px0). It keeps px0's core review IDE and adds the features listed in [What px0-flow adds](#what-px0-flow-adds).

px0-flow shows you how a change moves through your system instead of only file by file. Its Change Flow view maps every changed symbol onto a UI → API → service → database graph and links the pieces by calls, fetches, and queries. It opens by default in the middle pane whenever the workspace has changes. Click any node to jump to its code.

On top of that, px0-flow adds Threads, which are long-running conversations with your AI coding harness that can plan a flow before any code is written and then compare that plan with what actually landed. It also adds call trails, which show who calls a function and what it calls, and a review of unpushed commits so you can read each one before pushing.

See full upstream px0 performance benchmarks and comparisons at [px0.ai/benchmarks](https://px0.ai/benchmarks).

## Demo

Change Flow (`Cmd/Ctrl+Shift+G`) tracing a discount-code change from the UI through the API and service layers to the database:

![px0-flow Change Flow view](docs/assets/demo-change-flow.png)

## What px0-flow adds

- Change Flow: Visualize changes from the working tree, a pull request, or an agent thread as a connected UI → API → service → database graph. It opens by default when px0-flow starts on a workspace with changes, and from **Flow** in the Source | Diff | Flow switch, **View: Change Flow** in the command palette, or `Cmd/Ctrl+Shift+G`; click a node to inspect its diff. See [Change Flow](docs/features/change-flow.md).
- Threads: Long-running conversations with your coding harness, anchored to a selection, a line, or the whole workspace (`Alt+T`). Each turn lists the files it changed, and thread scope in Change Flow compares a read-only planned flow with the changes that landed. See [Threads](docs/features/threads.md).
- Unpushed commit review: Read the commits your branch is ahead of its tracking branch by, one file diff at a time, before you push them. See [Git integration](docs/features/git-integration.md).
- Call trails: See who calls a function and what it calls, expanding one level at a time, from the **Calls** tab in the right sidebar (powered by the language server's call hierarchy).
- `make flow-demo WORKSPACE=/path/to/repo`: Build and open Change Flow for any repository in one command.
- `git px0-flow`: Run `make git-px0-flow` once to install a global git alias. After that, `git px0-flow` inside any repository rebuilds px0-flow from this checkout and opens Change Flow on that repository's changes. Remove it with `git config --global --unset alias.px0-flow`. See [Change Flow](docs/features/change-flow.md#open-from-any-repository).

## Features

Inherited from upstream px0:

- GitHub PR reviews & Git panel: Review pull requests directly (`px0 <pr-url>`), inspect scoped merge-base diffs, draft inline review comments, expand recent commits to browse what each one changed, and stage, commit, or push from the browser.
- AI coding harness integration: Dispatch edits directly to Claude Code, Gemini CLI, Cursor Agent, Antigravity, OpenCode, Codex, Aider, or Goose with live reloading.
- Fast navigation: Fuzzy file search, symbol outline, and workspace regex search in milliseconds.
- Remote-first: Run on any remote server, VM, or container and browse locally without SSH key setups or remote daemons.
- Virtual rendering: Opens 400,000-line files smoothly by mounting only visible rows; frees memory back to the OS after 15 seconds of inactivity.
- Rich code viewer: 14 built-in themes, rendered Markdown with Mermaid diagrams, CSV/TSV table view, image inspector, and optional zero-config LSP for Go-to-Definition and hover.

## Roadmap

Planned px0-flow features, not yet built:

- Accurate code flow visualization:
  - Build call edges from the language server's call hierarchy instead of name matching.
  - Draw edges the language server confirmed as solid lines and guessed ones as dotted lines.
  - Scope matching to each function's body, so a name that appears elsewhere in the file doesn't create an edge.
  - Refine the graph automatically once the language server finishes starting.
- Readable large graphs: filter by layer (UI, API, service, database), collapse nodes by file, focus on one symbol and its neighbours, and show a node's diff in a side panel without leaving Flow.
- Multi-repository workspaces: open several repositories in one px0-flow session and trace a change across them, such as a frontend repo calling an API repo that writes to a shared schema repo.
## License

[MIT License](LICENSE). px0 © Arpit Bhayani. px0-flow modifications © ValPetal Tech Labs.

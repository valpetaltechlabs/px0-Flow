<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://px0.ai/logo/px0-logo-dark.png">
  <source media="(prefers-color-scheme: light)" srcset="https://px0.ai/logo/px0-logo-light.png">
  <img alt="px0" src="https://px0.ai/logo/px0-logo-dark.png" width="120">
</picture>

# px0-flow

---

px0-flow extends [px0](https://github.com/px0-ai/px0) by Arpit Bhayani. It keeps px0's core review IDE and adds the features listed in [What px0-flow adds](#what-px0-flow-adds).

px0-flow is an IDE built for reviewing AI-generated code, optimized for speed. It turns your browser into a zero-latency console with native Git and GitHub integrations, instant search across massive codebases, and seamless handoff to local AI coding harnesses.

As AI agents author more code directly from the terminal, engineering productivity is no longer constrained by how fast you type - it is constrained by how fast you can review, navigate, and verify changes. px0-flow replaces heavy, multi-gigabyte editing suites with an instant, distraction-free environment built specifically for this review loop.

- < 1 ms cold start
- ~20–30 MB server daemon RAM (~100–180 MB total with browser tab, vs. VS Code's ~1,440 MB)
- Single static Go binary with zero runtime dependencies (no Electron, no Node, no CGO)
- ~280 languages tokenized natively via Chroma

See full upstream px0 performance benchmarks and comparisons at [px0.ai/benchmarks](https://px0.ai/benchmarks).

## Demo

Change Flow (`Cmd/Ctrl+Shift+G`) tracing a discount-code change from the UI through the API and service layers to the database:

![px0-flow Change Flow view](docs/assets/demo-change-flow.png)

## What px0-flow adds

- Change Flow: Visualize changes from the working tree, a pull request, or an agent thread as a connected UI → API → service → database graph. Open it from **Flow**, **View: Change Flow** in the command palette, or with `Cmd/Ctrl+Shift+G`; click a node to inspect its diff. See [Change Flow](docs/features/change-flow.md).
- Threads: Long-running conversations with your coding harness, anchored to a selection, a line, or the whole workspace (`Alt+T`). Each turn lists the files it changed, and thread scope in Change Flow compares a read-only planned flow with the changes that landed. See [Threads](docs/features/threads.md).
- Unpushed commit review: Read the commits your branch is ahead of its tracking branch by, one file diff at a time, before you push them. See [Git integration](docs/features/git-integration.md).
- Call trails: See who calls a function and what it calls, expanding one level at a time, from the **Calls** tab in the right sidebar (powered by the language server's call hierarchy).
- `make flow-demo WORKSPACE=/path/to/repo`: Build and open Change Flow for any repository in one command.

## Features

Inherited from upstream px0:

- GitHub PR reviews & Git panel: Review pull requests directly (`px0 <pr-url>`), inspect scoped merge-base diffs, draft inline review comments, expand recent commits to browse what each one changed, and stage, commit, or push from the browser.
- AI coding harness integration: Dispatch edits directly to Claude Code, Gemini CLI, Cursor Agent, Antigravity, OpenCode, Codex, Aider, or Goose with live reloading.
- Fast navigation: Fuzzy file search, symbol outline, and workspace regex search in milliseconds.
- Remote-first: Run on any remote server, VM, or container and browse locally without SSH key setups or remote daemons.
- Virtual rendering: Opens 400,000-line files smoothly by mounting only visible rows; frees memory back to the OS after 15 seconds of inactivity.
- Rich code viewer: 14 built-in themes, rendered Markdown with Mermaid diagrams, CSV/TSV table view, image inspector, and optional zero-config LSP for Go-to-Definition and hover.

## Installation

Build px0-flow from source (requires Go 1.24+ and Node.js or Bun). The binary is named `px0`:

```bash
git clone https://github.com/valpetaltechlabs/px0-Flow.git
cd px0-Flow
make build
install -d ~/.local/bin && install px0 ~/.local/bin/
```

Quick install of upstream px0 only, without the px0-flow additions (macOS, Linux, BSD):

```bash
curl -fsSL https://px0.ai/install.sh | sh
```

## Usage

```bash
# Inspect current directory
px0

# Inspect a specific directory or file
px0 ~/workspace/project
px0 main.go:42

# Review a GitHub pull request
px0 https://github.com/owner/repo/pull/123

# Remote or headless server mode
px0 -host 0.0.0.0 -port 7777 ~/workspace

# Behind a reverse proxy under a subpath
px0 -base-path /rev-123/ -host 0.0.0.0 -port 7777 ~/workspace
```

When px0 binds to `0.0.0.0`, it prints a `network` URL for every unique
non-loopback IPv4 address on the machine, using the port selected by the
listener:

```text
  url:        http://0.0.0.0:7777
  network:    http://10.0.0.15:7777
  network:    http://192.168.1.42:7777
```

Open the address that is reachable from your local machine. The list can
include LAN, VPN, and container-network addresses, depending on the remote
host's interfaces.

Access securely over Tailscale, WireGuard, reverse proxy, or Cloudflare Tunnel
with zero remote setup overhead and strict read-only sandboxing
(path traversal protection & DNS rebinding checks).

## Development

```bash
git clone https://github.com/valpetaltechlabs/px0-Flow.git
cd px0-Flow

# Run tests
make test

# Live frontend development (serves web/ assets directly from disk)
go run . -dev .

# Build binary
make build

# Build and launch px0 against a repo to inspect its changes in Change Flow
make flow-demo WORKSPACE=/path/to/repository
```

## Benchmarks

```bash
# Clone the benchmark corpus (~3 GB: Linux, Kubernetes, TypeScript, etc.)
./benchmark.sh --clone

# Run benchmarks across the corpus
./benchmark.sh

# Compare px0 directly against VS Code on the current workspace
./benchmark.sh --vscode .

# Profile memory lifecycle (index, search, idle memory release)
./benchmark.sh --memory bench-repos/linux
```

> **Architecture & Memory Accounting**: px0 separates the lightweight native Go daemon from the browser-rendered UI. The Go server consumes ~20–30 MB resident RAM (RSS). The active browser tab allocates ~80–150 MB for DOM nodes, V8 JS runtime, and GPU compositing. Combined, the total local memory footprint is ~100–180 MB (~85–90% lighter than Electron IDEs like VS Code at ~1,440 MB, which bundle dedicated Chromium and Node runtimes). In remote or container environments (`px0 -host 0.0.0.0`), the remote host pays strictly the ~20–30 MB server cost. See [BENCHMARKS.md](BENCHMARKS.md) for full methodology and breakdowns.

## Documentation

- [px0-flow docs](docs/README.md)

Upstream px0:

- [Docs](https://px0.ai/docs)
- [Benchmarks](https://px0.ai/#bench)
- [Blog](https://px0.ai/blog)
- [Changelog](https://px0.ai/changelog)
- [Partners](https://px0.ai/design-partners)

## Design Partners

When AI coding agents generate large diffs across repositories every day, code authoring is no longer the bottleneck. Verification and review are.

We are looking for engineering teams (10+ engineers or teams running active agent workflows) as design partners. Partners receive direct Slack Connect access to the core team, rapid turnaround on custom harness integrations, and direct input on the roadmap.

Learn more on the [Design Partners page](https://px0.ai/design-partners).

## License

[MIT License](LICENSE). px0 © Arpit Bhayani. px0-flow modifications © ValPetal Tech Labs.

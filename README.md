# Bundle

Bundle turns a Go web server and its web UI into a native desktop application —
native windowing, icons, and a controlled server lifecycle — for macOS, Windows,
and Linux.

Your server keeps being an ordinary Go HTTP server. Bundle starts it on a
loopback port, waits for it to answer, and opens its UI in a native WebView.

## Delivery modes

- **Native bundle** (`bundle build <target>`) — a self-contained, target-specific
  application (`.app`, a Windows `.zip`, or a Debian `.deb`). It embeds a
  precompiled, platform-specific *core* that manages the server and hosts the
  WebView, so you can cross-package without building native WebView dependencies
  yourself.
- **Runtime-managed bundle** (`bundle pack <target>`) — a portable `.bundl`
  directory containing only your server, config, and assets — never a core. The
  separately installed **Bundle Runtime** supplies the platform's core and opens
  `.bundl` packages in place, the way a JRE opens a JAR.

## Quickstart

```sh
bundle init                 # write a bundle.toml template
# edit bundle.toml, then build your server binary for the target
bundle build darwin-arm64   # -> ./out/<App>.app
bundle pack  darwin-arm64   # -> ./out/<App>.bundl
```

Supported targets (all six of `{darwin,windows,linux}-{amd64,arm64}`):
`darwin-amd64`, `darwin-arm64`, `windows-amd64`, `windows-arm64`, `linux-amd64`,
`linux-arm64`. You supply the matching server binary for each; Bundle wraps it.

`bundle.toml` declares app metadata, the prebuilt server binary, the icon, the
server port (`auto` selects a free loopback port and substitutes `{port}` into
`runtime_flags` and `webview.url`), window settings, and `shutdown_grace`. Run
`bundle init` for a fully commented template.

When the window closes, Bundle stops the server the way any launcher would — a
standard OS signal, then a hard kill after `shutdown_grace` if it hasn't exited.
Your server needs no Bundle-specific code: just handle the signal you already
handle (`SIGTERM` on macOS/Linux, `Ctrl+Break` / `os.Interrupt` on Windows).

## Repository layout

| Path | What it is |
| --- | --- |
| `cmd/bundle` | the `bundle` CLI |
| `cmd/bundle-core` | the embedded core: manages the server + WebView for native bundles |
| `cmd/bundle-runtime-runner` | opens one `.bundl` package for the Bundle Runtime |
| `internal/manifest` | `bundle.toml` / runtime config parsing, defaults, validation |
| `internal/packager` | native `.app` / `.zip` / `.deb` builders |
| `internal/bundl` | `.bundl` descriptor and open/validate logic |
| `internal/runner` | starts the server, waits for readiness, drives the WebView |
| `internal/coreassets` | precompiled cores embedded into the CLI |
| `runtime/` | the platform-native Bundle Runtime launchers |
| `e2e/app` | a sample client+server used as a smoke-test fixture |

## Building from source

The CLI embeds a core for every target (`internal/coreassets/<target>/`).
Building the CLI locally only produces the **current platform's** core; the
cores for other platforms are CGO-linked against native WebView libraries and
are produced by CI (`.github/workflows/artifacts.yml`).

```sh
mise run core:darwin-arm64   # build this platform's core
mise run build               # build the CLI
mise run test                # go test -race ./...
mise run check               # test + build the macOS Runtime + vet + git diff --check
```

To cross-package for another platform from your machine, fetch the CI-built
cores first:

```sh
mise run artifacts:download  # cores + runtimes into out/downloads
```

Bundle uses [`mise`](https://mise.jdx.dev/) for tools and tasks — see
`mise.toml`. There is deliberately no Makefile.

## Known limitations

- Produced applications are **not code-signed or notarized**; recipients may
  need to clear Gatekeeper / SmartScreen manually.
- Both delivery modes cover all six targets. The runtime-managed mode ships one
  single-arch Bundle Runtime per target (a separate macOS `.app` for each Mac
  architecture, plus per-arch Linux and Windows runtimes).
- The Linux native target produces a Debian `.deb` only (no AppImage / RPM). The
  `.deb` declares its WebView runtime dependencies so `apt install ./app.deb`
  pulls them on a clean machine. Because `webview_go` links `webkit2gtk-4.0`, the
  core is built on Ubuntu 22.04 and the `.deb` targets the 4.0 runtime — i.e.
  Debian 12 (bookworm), Ubuntu 22.04, and derivatives. Ubuntu 24.04 (noble)
  dropped the 4.0 line; supporting it is a follow-up that requires bumping
  `webview_go` to the 4.1 backend.

## License

Released under the [MIT License](LICENSE).

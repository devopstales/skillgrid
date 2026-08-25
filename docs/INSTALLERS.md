# Installer Plan

## Assumptions

- Binary: statically-linked Go CLI (`skillgrid`), built from this repo (`aiskillgrid-v2`).
- Runtime deps: `git`, `node`/`npm` (auto-installed by `scripts/install_node.sh` if missing), network access.
- Config source: the binary clones `https://github.com/devopstales/skillgrid.git` at install time to obtain `config.d/`. The `aiskillgrid-v2` repo is the builder; it contains the `config.d/` source tree for development/sync only.
- Cross-compile targets: `darwin-amd64`, `darwin-arm64`, `linux-amd64`, `linux-arm64`. No Windows.
- Release artifacts: `dist/skillgrid-release-<platform>.tar.gz` produced by `task release-build`.

## Prerequisites (all installers)

1. Standardize Go version in `go.mod` to `1.23` to match docs.
2. Add a GitHub Actions workflow (`.github/workflows/release.yml`) that:
   - Triggers on tag push (`v*`).
   - Runs `task test` and `task release-build`.
   - Uploads `dist/*.tar.gz` and per-platform raw binaries as GitHub Release assets.
   - Optionally generates a SHA256SUMS file.
3. Parameterize `VERSION` in `Taskfile.yml` from `GITHUB_REF_NAME` / `git describe --tags` so release artifacts carry real versions.

---

## 1. Bash Script (`install.sh`)

**Location:** `scripts/install.sh` (or standalone in repo root).

**Behavior:**
- Detect OS (`uname -s`) and arch (`uname -m`), map to Go platform strings.
- Check for existing `skillgrid` binary; warn or skip.
- Download the matching `dist/skillgrid-release-<platform>.tar.gz` from the latest GitHub Release.
- Extract to `/usr/local/bin` (or `~/.local/bin` if `/usr/local/bin` is not writable).
- Verify checksum if `SHA256SUMS` is available.
- Optionally pre-clone the config repo into `~/.skillgrid/config.d` so `skillgrid install --skip-clone` can run offline.

**Entry points:**
```bash
# Recommended
curl -fsSL https://raw.githubusercontent.com/devopstales/aiskillgrid-v2/main/scripts/install.sh | bash

# Or download then run
curl -fsSL -o install.sh https://.../install.sh
bash install.sh
```

**Caveats to print at end:**
```bash
export PATH="$HOME/.skillgrid/bin:$PATH"
export PATH="$HOME/.skillgrid/npm/node_modules/.bin:$PATH"
skillgrid install --yes
```

**Pros:** Zero-dependency, works on any POSIX system with curl/tar.
**Cons:** No native uninstall, no version pinning, no PATH management.

---

## 2. Homebrew Formula

**Tap:** `devopstales/homebrew-skillgrid` (separate repo or under this repo's tap).

**Formula file:** `Formula/skillgrid.rb`.

**Skeleton:**
```ruby
class Skillgrid < Formula
  desc "Single CLI that installs and configures AI coding agents"
  homepage "https://github.com/devopstales/aiskillgrid-v2"
  url "https://github.com/devopstales/aiskillgrid-v2/releases/download/v#{version}/skillgrid-release-#{os}-#{arch}.tar.gz"
  version "1.0.0"
  sha256 "<per-platform or use livecheck>"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/devopstales/aiskillgrid-v2/releases/download/v#{version}/skillgrid-release-darwin-arm64.tar.gz"
      sha256 "<arm64 sha>"
    else
      url "https://github.com/devopstales/aiskillgrid-v2/releases/download/v#{version}/skillgrid-release-darwin-amd64.tar.gz"
      sha256 "<amd64 sha>"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/devopstales/aiskillgrid-v2/releases/download/v#{version}/skillgrid-release-linux-arm64.tar.gz"
      sha256 "<arm64 sha>"
    else
      url "https://github.com/devopstales/aiskillgrid-v2/releases/download/v#{version}/skillgrid-release-linux-amd64.tar.gz"
      sha256 "<amd64 sha>"
    end
  end

  def install
    bin.install "skillgrid"
  end

  def caveats
    <<~EOS
      Run the installer to set up agents:
        skillgrid install --yes

      Add to your shell profile:
        export PATH="$HOME/.skillgrid/bin:$PATH"
        export PATH="$HOME/.skillgrid/npm/node_modules/.bin:$PATH"
    EOS
  end
end
```

**Livecheck:**
Use GitHub releases to auto-detect new versions.

**Dependencies:**
`depends_on "git"` — required at runtime because the binary clones the config repo.

**Pros:** Native macOS/Linux package management, easy upgrades (`brew upgrade`), checksum verification.
**Cons:** Requires maintaining a tap, no post-install network actions (user must run `skillgrid install`).

---

## 3. Nix Flake

**File:** `flake.nix` in repo root.

**Skeleton:**
```nix
{
  description = "skillgrid CLI — install AI agents from one config";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages = {
          skillgrid = pkgs.buildGoModule {
            pname = "skillgrid";
            version = "1.0.0"; # derive from tag
            src = ./.;
            vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
            # OR use prebuilt binary for faster evaluation:
            # src = pkgs.fetchurl {
            #   url = "https://github.com/devopstales/aiskillgrid-v2/releases/download/v${version}/skillgrid-release-${system}.tar.gz";
            #   sha256 = "...";
            # };
          };
          default = self.packages.${system}.skillgrid;
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.skillgrid}/bin/skillgrid";
        };

        # Optional: dev shell with Go toolchain
        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.go ];
          shellHook = ''
            export PATH="$PWD/bin:$PATH"
          '';
        };
      });
}
```

**Two implementation paths:**
- **Build from source (`buildGoModule`):** Reproducible, Nix-native. Requires Go module deps to fetch (they are small and standard). Best for NixOS purity.
- **Fetch prebuilt binary (`fetchurl`):** Faster evaluation, smaller closure. Less reproducible if upstream binaries change. Preferred by some Nix users for Go CLIs.

**Runtime deps to declare:**
- `git` — the binary shell-outs to `git clone`.
- `nodejs` and `npm` — required for `npm install` steps inside `~/.skillgrid`. We can wrap the binary with a script that ensures node is on PATH, or document it. NixOS users often have node in their environment already.

**Config distribution:**
Because the binary clones `devopstales/skillgrid` at runtime, the flake does not strictly need to ship `config.d/`. However, for air-gapped or offline Nix installs:
- Add a `configDir` output or a separate derivation that copies `./config.d` to `$out/share/skillgrid/config.d`.
- Users can run: `skillgrid install --sync-repo ${./config.d}` (or point `SKILLGRID_REPO_URL` to the store path).

**Pros:** Reproducible builds, atomic upgrades/rollbacks, works in NixOS declarative configs.
**Cons:** Go module vendoring/hashing needs maintenance, runtime `git`/`npm` still required.

---

## Decision Matrix

| Criterion | Bash script | Homebrew | Nix flake |
|-----------|-------------|----------|-----------|
| Setup friction | Lowest | Low | Medium |
| Upgrade path | Manual | `brew upgrade` | `nix flake update` |
| Checksum/verification | Optional | Built-in | Built-in |
| Offline / air-gap | Partial (with pre-clone) | No | Yes (with `configDir` output) |
| macOS support | Yes | Yes | Yes |
| Linux support | Yes | Yes | Yes |
| Reproducibility | Low | Medium | High |

---

## Recommended Order of Implementation

1. **Release CI** — create `.github/workflows/release.yml` so we have binaries to point installers at.
2. **Bash script** — unblocks manual installs immediately; lowest maintenance.
3. **Homebrew formula** — standard macOS distribution; requires a tap but minimal ongoing work.
4. **Nix flake** — highest reproducibility value; requires `vendorHash` capture and optional config output.

## Open Questions / Risks

- The `scripts/install_node.sh` has a known typo (`githubusercontent.com` without `raw.`). Should be fixed before any installer relies on it.
- `GOTOOLCHAIN: local` in `Taskfile.yml` means builds fail if the local Go version mismatches `go.mod`. CI should pin a specific Go version.
- Windows is not in the cross-compile matrix; if needed, add `build-windows-amd64` task and Windows installer (PowerShell / winget).
- The binary hardcodes `release/2` branch in docs but uses `DefaultRepoURL` without branch in code — `Clone` does not specify `-b release/2`. Need to verify the default branch of `devopstales/skillgrid` is `release/2`, or update `Clone` to pin it.

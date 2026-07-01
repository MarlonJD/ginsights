# Install ginsights

`ginsights` is distributed as a single Go binary. It does not require Node, Vite, a database, a daemon, or a GitHub token for core local use.

## Homebrew

The intended tap is `marlonjd/tap`.

```bash
brew tap marlonjd/tap
brew install ginsights
```

If Homebrew refuses to load the formula from an untrusted tap, trust this tap explicitly and retry:

```bash
brew trust marlonjd/tap
brew install ginsights
```

The Homebrew formula source lives in this repository at:

```text
packaging/homebrew/Formula/ginsights.rb
```

Tap maintenance flow:

```bash
git clone https://github.com/marlonjd/homebrew-tap.git
mkdir -p homebrew-tap/Formula
cp packaging/homebrew/Formula/ginsights.rb homebrew-tap/Formula/ginsights.rb
```

The current formula builds from source from `https://github.com/MarlonJD/ginsights.git` on `main`. That keeps Homebrew installation available before prebuilt release artifacts exist. When tagged releases are cut, the formula can switch to a versioned tarball and SHA256.

## Shell Installer

Install from the default GitHub source:

```bash
curl -fsSL https://raw.githubusercontent.com/MarlonJD/ginsights/main/scripts/install.sh | bash
```

By default, this installs to:

```text
~/.local/bin/ginsights
```

Options:

```bash
curl -fsSL https://raw.githubusercontent.com/MarlonJD/ginsights/main/scripts/install.sh | bash -s -- --install-dir /usr/local/bin
curl -fsSL https://raw.githubusercontent.com/MarlonJD/ginsights/main/scripts/install.sh | bash -s -- --ref main
curl -fsSL https://raw.githubusercontent.com/MarlonJD/ginsights/main/scripts/install.sh | bash -s -- --dry-run
```

Environment overrides:

```bash
GINSIGHTS_INSTALL_DIR=/usr/local/bin \
GINSIGHTS_REF=main \
GINSIGHTS_REPO_URL=https://github.com/MarlonJD/ginsights.git \
bash scripts/install.sh
```

Requirements:

- `git`
- `go`

## From Source

```bash
git clone https://github.com/MarlonJD/ginsights.git
cd ginsights
go build -o bin/ginsights ./cmd/ginsights
./bin/ginsights help
```

## Verify Install

```bash
ginsights help
ginsights serve . --port 43117
ginsights build . --out report
ginsights json .
```

# gha-freeze

Pin GitHub Actions to specific SHA commits for better security.

## Why?

Pinning actions to commit SHAs prevents tag manipulation attacks while keeping version info readable.

## Installation

```bash
# Quick install (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/thinesjs/gha-freeze/main/install.sh | bash

# Manual install (example for macOS ARM)
curl -L https://github.com/thinesjs/gha-freeze/releases/download/v0.0.3/gha-freeze_x.x.x_macOS_arm64.tar.gz | tar xz
sudo mv gha-freeze /usr/local/bin/

# Download binary
https://github.com/thinesjs/gha-freeze/releases/latest

# With Go
go install github.com/thinesjs/gha-freeze/cmd/gha-freeze@latest
```

## Usage

```bash
cd your-repo
gha-freeze
```

### Commands

```bash
gha-freeze                  # Pin actions in workflows
gha-freeze --dry-run        # Preview changes
gha-freeze version          # Show version
gha-freeze update           # Update to latest version
gha-freeze auth TOKEN       # Save GitHub token
```

## Example

**Before:**
```yaml
- uses: actions/checkout@v4
```

**After:**
```yaml
- uses: actions/checkout@cd7d8d697e10461458bc61a30d094dc601a8b017 # v4
```

## GitHub Token

Unauthenticated: 60 requests/hour
Authenticated: 5,000 requests/hour

When rate limited, the tool shows a link to create a token. Save it with:
```bash
gha-freeze auth YOUR_TOKEN
```

Token is stored in `~/.config/gha-freeze/token` or use `GITHUB_TOKEN` / `GHA_FREEZE_TOKEN` env var.

## Backups

Backups saved to `.github/workflows/.backup-TIMESTAMP/`

After pinning:
- `d` - Delete backup
- `r` - Restore from backup
- `q` - Quit

## Development

```bash
git clone https://github.com/thinesjs/gha-freeze
cd gha-freeze
go build -o gha-freeze ./cmd/gha-freeze
```

Release:
```bash
./scripts/release.sh patch   # or minor/major/auto
```

## License

MIT

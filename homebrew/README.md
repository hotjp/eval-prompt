# Homebrew Tap for eval-prompt

This directory contains the Homebrew formula for installing `ep` via Homebrew.

## Installation

```bash
# Add the tap
brew tap eval-prompt/tap

# Install ep
brew install eval-prompt/tap/ep
```

## For Maintainers

### Building Release Binaries

Before releasing a new version, build the binaries and calculate their SHA256 checksums:

```bash
# Build for all platforms
make build

# Calculate SHA256 for each binary
sha256sum ./bin/ep-darwin-arm64
sha256sum ./bin/ep-darwin-amd64
sha256sum ./bin/ep-linux-arm64
sha256sum ./bin/ep-linux-amd64
```

Update the `ep.rb` formula with the correct SHA256 checksums.

### Releasing

1. Tag the release in GitHub
2. Upload the binaries to the GitHub release
3. Update the version and SHA256 checksums in `ep.rb`
4. Push changes to the tap repository

## Development

To test the formula locally:

```bash
# Install from local formula
brew install --debug ./homebrew/ep.rb

# Or use the test block
brew test ./homebrew/ep.rb
```

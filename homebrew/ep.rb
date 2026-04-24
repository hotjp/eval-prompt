class Ep < Formula
  desc "Prompt asset management and evaluation tool"
  homepage "https://github.com/eval-prompt/eval-prompt"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/eval-prompt/eval-prompt/releases/download/v0.1.0/ep-darwin-arm64"
      sha256 "TODO: Add SHA256 for darwin-arm64 binary"
    else
      url "https://github.com/eval-prompt/eval-prompt/releases/download/v0.1.0/ep-darwin-amd64"
      sha256 "TODO: Add SHA256 for darwin-amd64 binary"
    end
  end

  on_linux do
    if Hardware::CPU.arm64?
      url "https://github.com/eval-prompt/eval-prompt/releases/download/v0.1.0/ep-linux-arm64"
      sha256 "TODO: Add SHA256 for linux-arm64 binary"
    else
      url "https://github.com/eval-prompt/eval-prompt/releases/download/v0.1.0/ep-linux-amd64"
      sha256 "TODO: Add SHA256 for linux-amd64 binary"
    end
  end

  def install
    bin.install "ep"
  end

  test do
    system "#{bin}/ep", "--version"
  end
end

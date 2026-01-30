# typed: false
# frozen_string_literal: true

# Agent Orchestrator - CLI to orchestrate Cursor Agent (Headless Mode).
# To update after a new release:
#   curl -fSL -o /tmp/ao "https://github.com/kokjohn0824/agent_orchestrator/releases/download/vX.Y.Z/agent-orchestrator-OS-ARCH"
#   shasum -a 256 /tmp/ao
class AgentOrchestrator < Formula
  desc "CLI to orchestrate multiple Cursor Agents (Headless Mode)"
  homepage "https://github.com/kokjohn0824/agent_orchestrator"
  version "0.2.0"

  on_macos do
    on_intel do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.2.0/agent-orchestrator-darwin-amd64"
      sha256 "855125a0158848fdac1cd6f643faf9e3d65685521e71ba52fb8a6bcf5b5b6052"
    end
    on_arm do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.2.0/agent-orchestrator-darwin-arm64"
      sha256 "96ee14cefe435ab169a40e845699c634d7d059e73cbadc9eb0ea71655453826f"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.2.0/agent-orchestrator-linux-amd64"
      sha256 "0baed43412c67c3e27ac635a1d049cabbea669ceaa84936b8e975fb434cb400e"
    end
    on_arm do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.2.0/agent-orchestrator-linux-arm64"
      sha256 "b4c10405e1659810e28dd608049f23103d4531591fe65c313fb87a40dee73700"
    end
  end

  def install
    name = "agent-orchestrator-#{OS.mac? ? "darwin" : "linux"}-#{Hardware::CPU.arch}"
    bin.install name => "agent-orchestrator"
  end

  test do
    assert_match "agent-orchestrator", shell_output("#{bin}/agent-orchestrator --help", 0)
  end
end

# typed: false
# frozen_string_literal: true

# Agent Orchestrator - CLI to orchestrate Cursor Agent (Headless Mode).
# After the first release, update version and sha256 for each platform:
#   curl -fSL -o /tmp/ao "https://github.com/kokjohn0824/agent_orchestrator/releases/download/vX.Y.Z/agent-orchestrator-OS-ARCH"
#   shasum -a 256 /tmp/ao
class AgentOrchestrator < Formula
  desc "CLI to orchestrate multiple Cursor Agents (Headless Mode)"
  homepage "https://github.com/kokjohn0824/agent_orchestrator"
  version "0.0.0"

  on_macos do
    on_intel do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.0.0/agent-orchestrator-darwin-amd64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_arm do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.0.0/agent-orchestrator-darwin-arm64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.0.0/agent-orchestrator-linux-amd64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_arm do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.0.0/agent-orchestrator-linux-arm64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
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

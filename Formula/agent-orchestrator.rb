# typed: false
# frozen_string_literal: true

# Agent Orchestrator - CLI to orchestrate Cursor Agent (Headless Mode).
# To update after a new release:
#   curl -fSL -o /tmp/ao "https://github.com/kokjohn0824/agent_orchestrator/releases/download/vX.Y.Z/agent-orchestrator-OS-ARCH"
#   shasum -a 256 /tmp/ao
class AgentOrchestrator < Formula
  desc "CLI to orchestrate multiple Cursor Agents (Headless Mode)"
  homepage "https://github.com/kokjohn0824/agent_orchestrator"
  version "0.1.0"

  on_macos do
    on_intel do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.1.0/agent-orchestrator-darwin-amd64"
      sha256 "8b5fced792c8dd58c983ae976fae642a6a5346c231056249d2b9f8e803d58bdc"
    end
    on_arm do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.1.0/agent-orchestrator-darwin-arm64"
      sha256 "2b7b0077799fd04dfdcefa58fa50ee0663f21562f70c5666b6ba44e80a1e0ee3"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.1.0/agent-orchestrator-linux-amd64"
      sha256 "1d4af3d1ddf87368e324041a6e09937a38a48498f35a263fb1de6d8268b2174b"
    end
    on_arm do
      url "https://github.com/kokjohn0824/agent_orchestrator/releases/download/v0.1.0/agent-orchestrator-linux-arm64"
      sha256 "50178d431e989ca8492bef065fbe5f9460db3c6a35082209baa4b73cfa81c817"
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

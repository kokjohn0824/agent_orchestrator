package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "產生 shell 自動補全腳本",
	Long: `產生指定 shell 的自動補全腳本。

Bash:
  # Linux
  agent-orchestrator completion bash > /etc/bash_completion.d/agent-orchestrator
  
  # macOS
  agent-orchestrator completion bash > $(brew --prefix)/etc/bash_completion.d/agent-orchestrator

Zsh:
  # 如果 shell completion 尚未啟用，需要先執行:
  echo "autoload -U compinit; compinit" >> ~/.zshrc
  
  # 產生補全腳本
  agent-orchestrator completion zsh > "${fpath[1]}/_agent-orchestrator"
  
  # 或者放到自訂目錄
  agent-orchestrator completion zsh > ~/.zsh/completions/_agent-orchestrator

Fish:
  agent-orchestrator completion fish > ~/.config/fish/completions/agent-orchestrator.fish

PowerShell:
  agent-orchestrator completion powershell > agent-orchestrator.ps1
  # 然後在 PowerShell profile 中 source 這個檔案`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

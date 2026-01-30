# Agent Orchestrator

使用 Cursor Agent (Headless Mode) 協調多個 AI Agents 的 CLI 工具。

## 功能特色

- **互動式專案初始化** (`init`): 透過問答產生專案 milestone
- **現有專案分析** (`analyze`): 分析程式碼品質、效能、安全性等問題
- **智慧規劃** (`plan`): 將 milestone 分解為可執行的 tickets
- **自動化開發** (`work`): 平行處理 tickets，自動實作程式碼
- **完整 Pipeline** (`run`): 一鍵執行 plan → work → test → review → commit

## 安裝

### 從原始碼建置

```bash
# Clone 專案 (請替換為實際的 repository URL)
# git clone https://github.com/YOUR_ORG/agent-orchestrator.git
# cd agent-orchestrator

# 或從本地目錄建置
make build

# 安裝到 GOPATH/bin
make install
```

### 前置需求

1. Go 1.21+
2. Cursor CLI (確保 `agent` 指令可用)

```bash
# 檢查 agent 指令
which agent
```

## 快速開始

### 1. 互動式初始化新專案

```bash
agent-orchestrator init "建立一個 Log 分析工具，使用 Drain 演算法"
```

這會：
1. 詢問一系列問題了解需求
2. 產生詳細的 milestone 文件
3. 可選擇直接產生 tickets

### 2. 從 Milestone 產生 Tickets

```bash
agent-orchestrator plan docs/milestone-001.md
```

### 3. 處理 Tickets

```bash
# 處理所有 pending tickets (預設 3 個並行)
agent-orchestrator work

# 使用 5 個並行 agents
agent-orchestrator work -p 5

# 處理單一 ticket
agent-orchestrator work TICKET-001
```

### 4. 分析現有專案

```bash
# 分析所有面向
agent-orchestrator analyze

# 只分析效能和安全性
agent-orchestrator analyze --scope performance,security

# 自動產生 tickets
agent-orchestrator analyze --auto
```

### 5. 執行完整 Pipeline

```bash
agent-orchestrator run docs/milestone-001.md
```

## 完整指令列表

```
agent-orchestrator
├── init <goal>          # 互動式專案初始化，產生 milestone
├── analyze              # 分析現有專案，產生改進 issues/tickets
├── plan <milestone>     # 解析 milestone 產生 tickets
├── work [ticket-id]     # 處理 tickets (單一或全部)
├── review               # 程式碼審查
├── test                 # 執行測試
├── commit [ticket-id]   # 提交變更
├── run <milestone>      # 完整 pipeline
├── status               # 查看狀態
├── retry                # 重試失敗
├── clean                # 清除資料
├── config               # 設定管理
├── completion           # 產生 shell 補全
└── version              # 版本資訊
```

## 設定

### 設定檔

建立 `.agent-orchestrator.yaml`:

```bash
agent-orchestrator config init
```

設定檔範例：

```yaml
# Agent 設定
agent_command: agent           # Cursor Agent CLI 指令
agent_output_format: text      # 輸出格式: text, json, stream-json
agent_force: true              # 是否使用 --force 允許修改檔案
agent_timeout: 600             # Agent 執行超時秒數

# 路徑設定
tickets_dir: .tickets          # Tickets 儲存目錄
logs_dir: .agent-logs          # Agent 執行日誌目錄
docs_dir: docs                 # 文件目錄

# 執行設定
max_parallel: 3                # 最大並行 Agent 數量

# 分析範圍
analyze_scopes:
  - all
```

### 環境變數

```bash
export AGENT_CMD=agent                    # Agent 指令
export AGENT_OUTPUT_FORMAT=stream-json    # 輸出格式
export AGENT_FORCE=true                   # Force 模式
```

### 設定說明

以下為設定檔各欄位的預設值與建議情境；程式內預設以 `DefaultConfig()` 為準，設定檔與環境變數會覆寫對應欄位。

| 欄位 | 預設值 | 說明與建議情境 |
|------|--------|----------------|
| **agent_command** | `agent` | 呼叫 Cursor Agent 的 CLI 指令名稱或路徑。**何時調整**：Cursor CLI 安裝在非 PATH 或使用自訂執行檔時，改為完整路徑或別名。 |
| **agent_output_format** | `text` | 輸出格式：`text`、`json`、`stream-json`。**何時調整**：需要程式化解析輸出時用 `json` 或 `stream-json`；一般使用 `text` 即可。 |
| **agent_force** | `true` | 是否在呼叫 agent 時加上 `--force`，允許寫入/修改檔案。**何時調整**：僅想預覽不寫入時設為 `false`；多數情境建議保持 `true`。 |
| **agent_timeout** | `600` | 單次 agent 呼叫的超時秒數（10 分鐘）。**何時調整**：任務較大或環境較慢時可提高；想提早中止卡住任務時可降低。 |
| **tickets_dir** | `.tickets` | Tickets 儲存目錄（可為相對路徑，相對於專案根目錄）。 | 
| **logs_dir** | `.agent-logs` | Agent 執行日誌目錄；日誌可能含 prompt 與輸出內容。 |
| **docs_dir** | `docs` | 文件（如 milestone）輸出目錄。 |
| **max_parallel** | `3` | `work` 指令同時執行的 agent 數量上限。**何時調整**：機器資源足夠且想加快處理時可提高；資源有限或避免過載時可降低。 |
| **disable_detailed_log** | `false` | 設為 `true` 時**停用詳細日誌**：不會在 `logs_dir` 寫入含 prompt 與 agent 輸出的日誌檔。**副作用**：無法從日誌還原對話內容。**何時調整**：在含機密或專屬程式碼的環境、或需符合資安/合規要求時，建議設為 `true`。 |
| **analyze_scopes** | `["all"]` | `analyze` 指令的預設分析範圍；可選 `performance`、`refactor`、`security`、`test`、`docs`、`all`。指令列 `--scope` 會覆寫此預設。**何時調整**：若經常只分析部分面向（例如僅 performance、security），可在此設定以省去每次下 `--scope`。 |

## Tickets 生命週期

```
                    ┌─────────┐
                    │ pending │
                    └────┬────┘
                         │
                         ▼
                  ┌─────────────┐
                  │ in_progress │
                  └──────┬──────┘
                         │
              ┌──────────┴──────────┐
              ▼                     ▼
        ┌───────────┐         ┌────────┐
        │ completed │         │ failed │
        └───────────┘         └────────┘
              │                     │
              │    ┌────────┐       │
              └───▶│ commit │◀──────┘ (retry)
                   └────────┘
```

## Shell 自動補全

### Bash

```bash
# Linux
agent-orchestrator completion bash > /etc/bash_completion.d/agent-orchestrator

# macOS
agent-orchestrator completion bash > $(brew --prefix)/etc/bash_completion.d/agent-orchestrator
```

### Zsh

```bash
agent-orchestrator completion zsh > "${fpath[1]}/_agent-orchestrator"
```

### Fish

```bash
agent-orchestrator completion fish > ~/.config/fish/completions/agent-orchestrator.fish
```

## 常用工作流程

### 流程 1: 從零開始的新專案

```bash
# 1. 互動式初始化
agent-orchestrator init "我的專案想法"

# 2. 處理 tickets
agent-orchestrator work

# 3. 查看狀態
agent-orchestrator status

# 4. 提交變更
agent-orchestrator commit --all
```

### 流程 2: 改進現有專案

```bash
# 1. 分析專案
agent-orchestrator analyze --scope performance,refactor --auto

# 2. 處理改進 tickets
agent-orchestrator work

# 3. 審查變更
agent-orchestrator review

# 4. 提交
agent-orchestrator commit --all
```

### 流程 3: 完整自動化

```bash
# 一鍵執行所有步驟
agent-orchestrator run docs/milestone.md --analyze-first
```

## 故障排除

### Agent 指令找不到

```bash
# 確認 Cursor CLI 已安裝
which agent

# 如果使用自訂路徑，設定環境變數
export AGENT_CMD=/path/to/agent
```

### 重試失敗的 Tickets

```bash
# 查看失敗的 tickets
agent-orchestrator status

# 重試
agent-orchestrator retry
agent-orchestrator work
```

### 清除並重新開始

```bash
agent-orchestrator clean --force
agent-orchestrator plan docs/milestone.md
```

## 開發

```bash
# 建置
make build

# 測試
make test

# 格式化程式碼
make fmt

# Lint
make lint
```

## 外部連結與文件

本專案參考了以下外部資源：

| 資源 | 用途 | 狀態 |
|------|------|------|
| [Cursor CLI 文件](https://cursor.com/docs/cli/headless) | Cursor Headless Mode 使用指南 | 官方文件 |

### 連結維護說明

- 外部連結可能會隨時間失效或變更
- 建議定期驗證連結的有效性
- 關鍵的 CLI 使用方法已在本文件中說明，減少對外部連結的依賴
- 如發現失效連結，請提交 Issue 或 PR 更新

### 驗證連結

```bash
# 使用 curl 驗證連結是否有效
curl -I https://cursor.com/docs/cli/headless
```

## 授權

MIT License

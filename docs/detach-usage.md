# Detach 使用說明

`work` 指令支援 **背景執行（detach）**：在背景啟動子 process 執行 work，不佔用當前 terminal，關閉終端機也不會中斷執行。

## 指令用法

### work --detach（處理全部 tickets）

```bash
agent-orchestrator work --detach
```

- 啟動一個背景子 process 處理所有 pending tickets。
- 父 process 印出 **PID** 與 **日誌路徑** 後立即結束。
- 子 process 會寫入 PID 檔，並將 stdout/stderr 導向日誌檔。

### work [ticket-id] --detach（處理單一 ticket）

```bash
agent-orchestrator work TICKET-001 --detach
```

- 同上，但只處理指定的 ticket。

## 日誌路徑：--log-file 與 work_detach_log_dir

### 指令列：--log-file

使用 `--log-file` 可**直接指定**當次 detach 的日誌檔路徑（覆寫設定檔與預設規則）：

```bash
agent-orchestrator work --detach --log-file /var/log/my-work.log
agent-orchestrator work TICKET-001 --detach --log-file ./custom/detach.log
```

- 可為絕對路徑或相對路徑（相對路徑以專案根目錄為準）。
- 未指定時，依設定與時間戳決定路徑（見下方）。

### 設定檔：work_detach_log_dir

在 `.agent-orchestrator.yaml` 中可設定：

```yaml
work_detach_log_dir: .detach-logs   # 或任意目錄，可為相對路徑
```

- **有設定**：detach 日誌寫入此目錄，檔名為 `work-YYYYMMDD-HHMMSS.log`（依啟動時間）。
- **未設定**：使用 `logs_dir`（預設 `.agent-logs`）作為目錄，同樣使用 `work-YYYYMMDD-HHMMSS.log`。

**優先順序**：`--log-file` 指定路徑 > `work_detach_log_dir` 或 `logs_dir` + 時間戳檔名。

## PID 檔路徑

背景 work 會寫入一個 **PID 檔**，用來讓 `status` 判斷是否有背景 work 在跑，並在 process 結束或收到 SIGTERM/SIGINT 時自動刪除。

- **預設路徑**：`tickets_dir/.work.pid`（例如 `.tickets/.work.pid`）。
- **自訂路徑**：在設定檔中設定 `work_pid_file`：

```yaml
work_pid_file: /var/run/agent-orchestrator-work.pid
```

相對路徑會依專案根目錄解析為絕對路徑。

## status 顯示

執行 `agent-orchestrator status` 時：

- 若 **PID 檔存在** 且 **該 PID 的 process 仍存活**，會顯示：
  - 「背景工作: 執行中 (PID xxxxx)」
  - 「日誌路徑: \<目錄\>」（目錄為 `work_detach_log_dir` 若已設定，否則為 `logs_dir`）
- 若 PID 檔存在但 process 已結束，會視為過期並**自動刪除 PID 檔**，不會顯示為執行中。

因此可透過 `status` 快速確認背景 work 是否還在跑，以及日誌所在目錄。

## 流程摘要

1. 執行 `work --detach` 或 `work [ticket-id] --detach`（可選加 `--log-file`）。
2. 父 process 印出 PID 與日誌路徑後結束。
3. 子 process 寫入 PID 檔（`work_pid_file` 或 `.tickets/.work.pid`），並將輸出寫入日誌檔。
4. 使用 `status` 查看是否仍在執行及日誌路徑。
5. 子 process 結束或收到中斷訊號時會刪除 PID 檔。

## 建議 .gitignore

若使用預設路徑，建議在專案 `.gitignore` 中加入：

- `.tickets/.work.pid`
- `.agent-logs/work-*.log`

若設定了 `work_detach_log_dir`，請一併忽略該目錄下的 `work-*.log`。

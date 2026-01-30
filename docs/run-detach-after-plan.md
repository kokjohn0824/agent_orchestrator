# Run --detach-after-plan 流程說明

`run --detach-after-plan` 讓你在 **Planning（Step 1）完成後** 改為在背景執行 Coding（work），CLI 印出 PID 與日誌路徑後立即返回，不佔用當前 terminal，也不執行後續的 test、review、commit。

## 使用情境

- 希望先完成 plan、產生 tickets，但 **Coding 階段想在背景跑**，以便繼續做其他事或關閉終端機。
- 想 **手動控制** 何時執行 test、review、commit，而不是一鍵跑完整 pipeline。
- 規劃與實作分開：先確認 tickets 沒問題，再讓 work 在背景跑，稍後再決定是否跑 test/review/commit 或再次 `run`。

## 指令用法

```bash
agent-orchestrator run docs/milestone-001.md --detach-after-plan
```

**實際流程：**

1. **Step 1 Planning**：依 milestone 產生 tickets 並寫入 store（與一般 `run` 相同）。
2. **啟動背景 work**：改為以 detach 模式啟動 work（等同 `work --detach`），處理所有 pending tickets。
3. **CLI 立即返回**：印出「Coding 已分離」、**PID** 與 **日誌路徑**，以及提示「可稍後執行 test、review、commit。」後結束。
4. **不執行** Step 2 之後的 test、review、commit（皆在背景 work 完成後由你手動執行或再次 `run`）。

背景 work 的 PID 檔、日誌路徑規則與 `work --detach` 相同，詳見 [Detach 使用說明](detach-usage.md)。

## 後續手動流程建議

Plan 完成並 detach 後，可依下列方式接續：

### 1. 查看背景 work 狀態

```bash
agent-orchestrator status
```

- 若顯示「背景工作: 執行中 (PID xxxxx)」與日誌路徑，表示 work 仍在跑；可從日誌路徑查看輸出。
- 若不再顯示執行中，表示背景 work 已結束（成功或失敗皆可從日誌與 ticket 狀態判斷）。

### 2. 背景 work 完成後：手動 test、review、commit

當 `status` 顯示背景 work 已結束後，可依序執行：

```bash
# 執行測試
agent-orchestrator test

# 程式碼審查
agent-orchestrator review

# 提交變更（可指定 ticket 或 --all）
agent-orchestrator commit --all
```

依專案需求可只執行其中幾步，或調整順序（例如先 review 再 test）。

### 3. 再次執行完整 pipeline（可選）

若希望在同一個 milestone 上再跑一次「plan 已存在，從 work 到 commit」的流程，可先確認 tickets 狀態後再執行：

```bash
agent-orchestrator status
agent-orchestrator run docs/milestone-001.md
```

注意：再次 `run` 會重新執行 plan（會覆寫/更新 tickets），若只想從 work 開始，應使用 `agent-orchestrator work` 而非 `run`。

### 4. 僅重跑 work（不重新 plan）

若 plan 與 tickets 已就緒，只想重新或繼續跑 work，可直接：

```bash
agent-orchestrator work
# 或背景執行
agent-orchestrator work --detach
```

## 流程摘要

| 階段           | 說明 |
|----------------|------|
| 執行指令       | `run <milestone> --detach-after-plan` |
| Plan 完成後    | 啟動背景 work，CLI 印出 PID、日誌與下一步提示後結束 |
| 監看進度       | 使用 `status` 與日誌檔 |
| 背景 work 結束 | 手動執行 `test` → `review` → `commit`，或依需只執行部分步驟 |
| 可選           | 再次 `run`（重新 plan）或僅執行 `work` / `work --detach` |

## 相關文件

- [Detach 使用說明](detach-usage.md)：`work --detach` 的日誌路徑、PID 檔、`status` 顯示等。
- README「完整 Pipeline」與「常用工作流程」：其他 run/work 用法。

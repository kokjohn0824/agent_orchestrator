# Milestone: Work Command Detach Mode

**文件日期：** 2026-01-30  
**開發目標：** 在 `work` command 新增 detach mode，使呼叫 `work` 後可立即執行其他指令。

---

## 1. 開發目標概述

### 1.1 目標說明

讓使用者能以「背景模式」執行 `work`：啟動後 CLI 立即結束並回傳 shell，實際的 ticket 處理在子 process 中持續進行。使用者可同時執行其他指令（如 `status`、`plan`、或另一個 `work`），並透過固定路徑的 PID file 與 log 得知背景 work 的狀態與輸出。

### 1.2 核心決策摘要

| 項目 | 決策 |
|------|------|
| 背景 process 啟動方式 | 同 binary 再 exec 自己並傳遞參數（如 `--detach-child`） |
| ticket.Store 並行策略 | 優先以樂觀鎖解決 race condition，允許同時修改；若不可行再 fallback 為僅查詢 |
| run pipeline 支援 | 支援「plan 完成後 detach work」，run 新增 flag（如 `--detach-after-plan`） |
| Detach 輸出 | 全部寫入 log 檔；可設定 log 路徑（config + `--log-file`），config 新增欄位（如 `WorkDetachLogDir`） |
| 背景 work 狀態 | 固定路徑 PID file（如 `.tickets/.work.pid`），`status` 顯示「background work: running (PID 12345)」 |
| 測試策略 | 非 detach 路徑維持現有單元測試；detach 路徑以整合測試（真正 fork、檢查 PID file / log） |
| 發佈與文件 | 建議將 agent-orchestrator 放在 PATH；暫不特別註明 detach 需在專案目錄下執行 |

---

## 2. 現有架構分析（與新功能的關聯）

### 2.1 專案結構與職責

- **語言與框架：** Go、Cobra（CLI）、Viper（設定）、Charm（bubbles/bubbletea/lipgloss TUI）
- **入口：** `cmd/agent-orchestrator/main.go` → `cli.Execute()` → Cobra `rootCmd`
- **相關模組：**
  - **internal/cli/**：`work.go`（work 主流程）、`run.go`（pipeline 含 Step 2 Coding）、`status.go`（狀態顯示）、`root.go`（config 載入、子指令註冊）
  - **internal/config/**：`config.go`（Config 結構、Load、路徑解析）
  - **internal/ticket/**：`store.go`（Save/Load/Count 等）、`ticket.go`（Ticket 結構）
  - **internal/agent/**：CodingAgent、Caller（實際呼叫 Cursor Agent）
  - **internal/ui/**：Spinner、MultiSpinner（TUI 輸出到 stdout）

### 2.2 與 Detach 的關聯

| 現有元件 | 關聯 |
|----------|------|
| **work.go** | 需分支：`--detach` 時父 process 只負責 exec 子 process 並寫 PID file，子 process 以 `--detach-child` 進入「無 TUI、輸出導向 log」的 work 邏輯 |
| **run.go** | Step 2（Coding）目前同步呼叫 work 邏輯；需支援 `--detach-after-plan`，在 plan 完成後改為啟動 detach work 並 return |
| **status.go** | 需讀取 PID file，顯示「background work: running (PID N)」或「無背景 work」；可顯示 log 路徑 |
| **config** | 新增 `WorkDetachLogDir`（或類似）及 PID file 路徑約定；Load/Validate 需涵蓋新欄位 |
| **ticket.Store** | 樂觀鎖或並行安全：若採用 version/ETag，需在 Ticket 與 Store 上擴充；否則 fallback 為偵測「有背景 work 時禁止寫入」 |
| **ui (Spinner/MultiSpinner)** | Detach 子 process 內不使用 TUI；所有輸出寫入單一 log 檔（例如純文字或簡化格式） |
| **root PersistentPreRunE** | 子 process 以 `--detach-child` 啟動時仍需載入 config（或從父 process 傳遞必要參數）；需決定是否跳過部分 global flags |

### 2.3 依賴關係與整合順序

1. **Config 與路徑**：先擴充 config（WorkDetachLogDir、PID 路徑約定），再實作 detach 啟動與 log 寫入。
2. **Detach 啟動與 child 分支**：在 `work` 與 `main`/root 層辨識 `--detach` / `--detach-child`，再實作「父 exec 子、子跑 work 並寫 log」。
3. **PID file 與 status**：子 process 啟動時寫 PID file，結束時刪除；`status` 讀取並顯示。
4. **Store 並行**：在「多 process 可能同時寫入」的前提下，先設計樂觀鎖或檔案鎖，再決定 fallback。
5. **run --detach-after-plan**：依賴 work detach 與 PID file 穩定後，在 run 的 Coding 階段改為啟動 detach 並 return。

---

## 3. 功能需求清單

### 3.1 Work Detach Mode

- **W1** 使用者可執行 `work --detach`（或等價 flag），當前 process 啟動一個「detach child」process 後立即 exit 0。
- **W2** Child process 以同一 binary、參數包含 `--detach-child`（內部用，可不對外文件）執行，並脫離 terminal（例如 `setsid` 或雙 fork），避免 terminal 關閉時被 kill。
- **W3** Child 內不啟動 TUI（無 MultiSpinner/Spinner 到 stdout）；所有輸出寫入單一 log 檔（路徑見 L1、L2）。
- **W4** 支援 `work [ticket-id] --detach`：僅處理指定 ticket 的 detach 語意一致。

### 3.2 Log 與設定

- **L1** Log 路徑可由 config 指定（例如 `WorkDetachLogDir`），預設可為 `.agent-logs` 下之 `work-YYYYMMDD-HHMMSS.log` 或專用子目錄。
- **L2** 支援 `--log-file`（work 或 run 使用 detach 時）覆寫當次 log 路徑。
- **L3** Config 新增欄位（如 `work_detach_log_dir`），並在 Load/DefaultConfig/Validate 中處理；若為空則用預設規則。

### 3.3 PID File 與狀態可見性

- **P1** 固定路徑 PID file：例如 `.tickets/.work.pid`（或由 config 指定），內容為單行 PID。
- **P2** Child 啟動成功後寫入 PID file；正常或異常結束時刪除 PID file。
- **P3** `status` 讀取 PID file：若存在且 process 存活則顯示「background work: running (PID N)」及 log 路徑（若可知）；若不存在或 process 已死則顯示無背景 work，並可清理過期 PID file。

### 3.4 Ticket Store 並行

- **S1** 優先以樂觀鎖（或等價機制）讓多 process 可同時修改 ticket.Store（例如 version/ETag + Save 時檢查，衝突則 retry 或回報）。
- **S2** 若評估後無法安全並行寫入，則 fallback：偵測到「有背景 work 在跑」時，禁止會改動 store 的指令（plan、work、run 等），僅允許查詢（如 status）。

### 3.5 Run Pipeline

- **R1** `run` 新增 flag（如 `--detach-after-plan`）：在 Step 1 Planning 完成後，Step 2 改為啟動 detach work（等同 `work --detach`）然後 return，不執行後續 test/review/commit。
- **R2** 使用者可之後手動執行 `test`、`review`、`commit` 或再次 `run`（跳過 plan）完成剩餘步驟；文件需說明此流程。

### 3.6 發佈與文件

- **D1** 建議使用者將 agent-orchestrator 放在 PATH，以便 detach 時子 process 能正確找到同一 binary。
- **D2** PID file 與預設 log 路徑列於文件或 .gitignore 範例（如 `.tickets/.work.pid`、`.agent-logs/work-*.log`）；暫不強制註明「detach 需在專案目錄下執行」。

---

## 4. 實作階段規劃

### Phase 0：準備（Config、路徑、i18n）

- **目標：** 新增 detach 相關設定與路徑約定，不改變現有行為。
- **修改模組：** `internal/config`、`internal/i18n`（若有新訊息）。
- **產出：** Config 新欄位、DefaultConfig/Load/Validate、路徑常數或 helper；.gitignore 範例更新（可選）。

### Phase 1：Detach 啟動與 Child 分支

- **目標：** 實作「父 process 在 `work --detach` 時 exec 自己並傳 `--detach-child`，子 process 辨識後執行 work 邏輯並寫 PID file」。
- **修改模組：** `internal/cli/work.go`、`internal/cli/root.go`（或 main 層辨識 `--detach-child`）、`cmd/agent-orchestrator/main.go`（若需提早解析 detach-child）。
- **依賴：** Phase 0（config 與 PID/log 路徑）。

### Phase 2：Child 內無 TUI、輸出導向 Log

- **目標：** 當以 `--detach-child` 執行時，不建立 MultiSpinner/Spinner；所有原本寫到 stdout/stderr 的內容改寫入單一 log 檔（路徑依 config 與 `--log-file`）。
- **修改模組：** `internal/cli/work.go`（分支依 detach-child 選擇 writer）、`internal/ui` 僅在非 detach 時使用（或傳入 io.Writer 抽象 log）。
- **依賴：** Phase 0（WorkDetachLogDir、--log-file）、Phase 1（child 分支）。

### Phase 3：PID File 生命週期與 Status 顯示

- **目標：** Child 啟動後寫 PID file，結束時刪除；`status` 讀取 PID file 並顯示「background work: running (PID N)」或無。
- **修改模組：** `internal/cli/work.go`（寫/刪 PID file）、`internal/cli/status.go`（讀取並顯示）、可選 `internal/detach` 或 `internal/cli/detach.go` 封裝 PID 邏輯。
- **依賴：** Phase 1、Phase 2。

### Phase 4：Ticket Store 並行策略

- **目標：** 實作樂觀鎖或檔案鎖，使多 process 可安全寫入 ticket.Store；若不可行則實作 fallback（偵測背景 work 時禁止寫入）。
- **修改模組：** `internal/ticket`（Store、Ticket 若需 version）、`internal/cli`（work/plan/run 在寫入前檢查鎖或 PID）。
- **依賴：** Phase 3（PID file 存在即表示背景 work 在跑）。

### Phase 5：Run --detach-after-plan

- **目標：** `run --detach-after-plan <milestone>` 在 Step 1 Planning 完成後，改為啟動 detach work 並 return，不執行 test/review/commit。
- **修改模組：** `internal/cli/run.go`（新增 flag、Coding 階段分支）。
- **依賴：** Phase 1～3 穩定。

### Phase 6：文件與發佈

- **目標：** README 或 docs 說明 detach 用法、PID file、log 路徑、run --detach-after-plan 流程；.gitignore 範例；建議 PATH 安裝。
- **修改模組：** `docs/`、`README.md`、`.gitignore`（可選）。

---

## 5. 每個階段的具體任務

### Phase 0：準備

| # | 任務 | 說明 |
|---|------|------|
| 0.1 | Config 新增欄位 | 在 `Config` 增加 `WorkDetachLogDir`（或 `work_detach_log_dir`）、約定 PID file 路徑為 `TicketsDir/.work.pid` 或單獨欄位。 |
| 0.2 | DefaultConfig / Load / Validate | 設定預設值（如 WorkDetachLogDir 為 `""` 表示用 `LogsDir` 下 `work-*.log`）；Validate 若需檢查路徑格式則一併加入。 |
| 0.3 | 路徑 helper | 提供「解析 detach log 路徑」的函式（考慮 config + 時間戳或 --log-file），供 Phase 2 使用。 |
| 0.4 | i18n | 新增 detach、background work、log 路徑等訊息 key（中英文依現有慣例）。 |
| 0.5 | .gitignore | 將 `.tickets/.work.pid`、`.agent-logs/work-*.log` 加入 .gitignore 範例或說明。 |

### Phase 1：Detach 啟動與 Child 分支

| # | 任務 | 說明 |
|---|------|------|
| 1.1 | 早期解析 --detach-child | 在 `main` 或 root 層於 Cobra 執行前檢查 os.Args 是否含 `--detach-child`，若有則設定內部 flag，供後續跳過「再次 detach」或調整 config 載入。 |
| 1.2 | work --detach flag | 在 `workCmd` 新增 `--detach`；`runWork` 中若 `--detach` 則不呼叫 `workSingleTicket`/`workAllTickets`，改為準備參數並 exec 自己。 |
| 1.3 | Exec 子 process | 使用 `exec.Command` 或等同方式，傳入 binary 路徑（`os.Executable()`）、子參數（如 `work`、可選 `ticket-id`、`--detach-child`、`--config`、`--log-file` 等），並 setsid/雙 fork 使子脫離 terminal（依 OS）。 |
| 1.4 | 父 process 行為 | 父 process 在成功啟動子 process 後印出「Detached. PID: N, log: …」（若已知 log 路徑），然後 exit 0；不等待子結束。 |

### Phase 2：Child 內無 TUI、輸出導向 Log

| # | 任務 | 說明 |
|---|------|------|
| 2.1 | 偵測 detach-child 並選 writer | 在 work 執行路徑中，若為 detach-child，則建立 log 檔（依 Phase 0 helper 與 --log-file），將 stdout/stderr 導向該檔（或使用同一 io.Writer 寫入）。 |
| 2.2 | 跳過 TUI | 在 detach-child 分支中不建立 MultiSpinner/Spinner；改為純文字進度輸出（例如 "Processing ticket X"）寫入 log writer。 |
| 2.3 | 錯誤與結束 | 所有錯誤訊息與 summary 皆寫入 log；process 結束時關閉 log 檔。 |

### Phase 3：PID File 與 Status

| # | 任務 | 說明 |
|---|------|------|
| 3.1 | 寫入 PID file | Child process 進入 work 邏輯前，將 `os.Getpid()` 寫入約定路徑（.tickets/.work.pid）；確保目錄存在。 |
| 3.2 | 刪除 PID file | Child 正常結束或 defer 中刪除 PID file；需處理 signal（SIGTERM/SIGINT）時也刪除。 |
| 3.3 | status 讀取 PID file | 在 `runStatus` 中讀取 PID file；若存在則檢查該 PID 是否存活（例如 `os.FindProcess` + 或 platform 特定檢查）；若存活則顯示「background work: running (PID N)」及 log 路徑（若可從 config 或固定命名推得）。 |
| 3.4 | 過期 PID 清理 | 若 PID file 存在但 process 已不存在，視為過期，刪除 PID file 並顯示無背景 work。 |

### Phase 4：Store 並行策略

| # | 任務 | 說明 |
|---|------|------|
| 4.1 | 設計樂觀鎖方案 | 決定是否在 Ticket 增加 version/ETag 欄位，Save 時「讀取-比較-寫入」或檔案鎖；評估 Store 與 CLI 改動範圍。 |
| 4.2 | 實作樂觀鎖或鎖機制 | 在 `ticket.Store`（及必要時 `Ticket`）實作；Save 衝突時 retry 或回傳明確錯誤給上層。 |
| 4.3 | Fallback：禁止寫入 | 若樂觀鎖不可行，則在 plan/work/run 等會寫入 store 的指令開頭檢查「是否存在背景 work（PID file 且存活）」；若存在則拒絕執行並提示使用者。 |
| 4.4 | 僅查詢允許 | status 等僅讀指令不檢查 PID file 或允許與背景 work 並存。 |

### Phase 5：Run --detach-after-plan

| # | 任務 | 說明 |
|---|------|------|
| 5.1 | run 新增 --detach-after-plan | 在 `runCmd` 增加 flag。 |
| 5.2 | Pipeline 分支 | 在 Step 1 Planning 完成、寫入 tickets 後，若 `--detach-after-plan` 為 true，則呼叫與「work --detach」等價的啟動邏輯（exec 子 process），然後 return，不執行 Step 2 同步 Coding 及後續 test/review/commit。 |
| 5.3 | 輸出說明 | 印出「Coding detached. PID: N, log: …。可稍後執行 test/review/commit。」等提示。 |

### Phase 6：文件與發佈

| # | 任務 | 說明 |
|---|------|------|
| 6.1 | Detach 使用說明 | 在 README 或 docs 中說明 `work --detach`、`--log-file`、config `work_detach_log_dir`、PID file 路徑、status 顯示。 |
| 6.2 | Run --detach-after-plan 流程 | 說明 plan 完成後 detach、後續手動 test/review/commit 的建議流程。 |
| 6.3 | PATH 與安裝 | 註明建議將 agent-orchestrator 放在 PATH（與現有 install 說明一致）；暫不特別註明 detach 須在專案目錄執行。 |
| 6.4 | .gitignore / 範例 | 列出 `.tickets/.work.pid`、`.agent-logs/work-*.log` 等於文件或 .gitignore。 |

---

## 6. 測試計畫

### 6.1 非 Detach 路徑（現有風格）

- **範圍：** `runWork` 在未使用 `--detach` 時行為不變；現有 `work_test.go` 之測試仍全部通過。
- **方式：** 維持現有作法（替換 `os.Stdout`、mock config、dry-run）；不啟動子 process。
- **重點：** `TestRunWork_StoreInitFails`、`TestRunWork_NoArgs_EmptyStore`、`TestWorkSingleTicket_*`、`TestWorkCmd_Flags` 等；新增「有 --detach flag 但未使用時」之行為測試（可選）。

### 6.2 Detach 路徑（整合測試）

- **範圍：** 真正 fork/exec 子 process，驗證 PID file、log 檔、status 顯示。
- **環境：** 使用 temp dir 作為專案根目錄、獨立 config、.tickets 與 log 目錄。
- **案例：**
  1. **啟動 detach：** 執行 `work --detach`（或等價），父 process 立即 exit 0；檢查 PID file 存在且內容為子 process PID；檢查 log 檔被建立（可選：內容含預期字串）。
  2. **Status 顯示：** 在 detach 運行中執行 `status`，輸出含「background work: running (PID N)」；子 process 結束後再執行 `status`，PID file 已刪除，輸出無背景 work。
  3. **Child 輸出進 log：** 子 process 在 dry-run 或 mock 下跑完，檢查 log 檔內有預期之 ticket 處理或完成訊息。
  4. **過期 PID：** 手動寫入一筆已不存在 process 的 PID file，執行 `status`，確認 PID file 被刪除且不顯示為 running。
- **注意：** 整合測試可能需較長 timeout、或限制並行數，避免 CI 負載過大。

### 6.3 Store 並行（若 Phase 4 實作樂觀鎖）

- **單元：** Store Save 在 version 衝突時 retry 或回傳錯誤；多 goroutine 同時 Save 不同 ticket 不損壞資料。
- **整合（可選）：** 兩 process 同時寫入不同 ticket，結果皆正確；或一 process 寫入、另一讀取，無 race。

### 6.4 Run --detach-after-plan

- **整合：** 執行 `run --detach-after-plan <milestone>`，plan 完成後 process 結束，且 PID file 存在、log 有 detach work 的紀錄；不執行 test/review/commit。

---

## 7. 驗收標準

### 7.1 功能

- [ ] `work --detach` 與 `work [ticket-id] --detach` 會啟動子 process 並立即結束；子 process 在無 TUI 下將輸出寫入指定 log 檔。
- [ ] 子 process 寫入並在結束時刪除 PID file（.tickets/.work.pid 或約定路徑）。
- [ ] `status` 在背景 work 運行時顯示「background work: running (PID N)」及 log 路徑（若可取得）；無背景 work 時不顯示該行，且過期 PID file 被清理。
- [ ] Log 路徑可由 config（WorkDetachLogDir）與 `--log-file` 指定；預設行為符合文件說明。
- [ ] Ticket.Store 在設計範圍內可多 process 並行寫入（樂觀鎖），或 fallback 下「有背景 work 時禁止寫入」且僅查詢指令可執行。
- [ ] `run --detach-after-plan <milestone>` 在 plan 完成後啟動 detach work 並 return，不執行後續步驟；文件說明後續手動步驟。

### 7.2 非功能

- [ ] 現有非 detach 的 work/run/status 行為與測試不退化。
- [ ] Detach 子 process 脫離 terminal，關閉 terminal 後子 process 仍可繼續運行（依環境驗證）。
- [ ] 文件已更新（detach 用法、PID、log、run --detach-after-plan、PATH 建議）；.gitignore 或範例已涵蓋新檔案路徑。

### 7.3 測試

- [ ] 所有既有單元測試通過。
- [ ] Detach 相關整合測試通過（PID file、log、status、過期 PID 清理）。
- [ ] 若實作 Store 並行，其對應單元/整合測試通過。

---

*本文件由需求問答與現有程式碼分析產生，實作時可依進度微調任務順序與細項。*

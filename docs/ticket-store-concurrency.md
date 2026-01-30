# Ticket Store 並行策略（TICKET-017）

## 設計決策

**採用 fallback 策略**：不在 Ticket 增加 version/ETag 欄位，Store.Save 不做「讀取-比較-寫入」或檔案鎖；改由 **偵測背景 work（PID 檔 + process 存活）時禁止寫入** 來避免並行寫入。

- **樂觀鎖（version/ETag）**：未採用。  
  若採用，需在 Ticket 增加 `Version`（或 ETag），Save 時讀取現有檔案、比對 version、一致才寫入並遞增。  
  - 優點：可做 per-ticket 並行控制。  
  - 缺點：既有 JSON 無 version，需遷移或預設 0；所有呼叫 Save 的 CLI 都要處理 conflict（重試或失敗）；MoveToStatus / MoveFailed 等多檔寫入流程較複雜。改動範圍大（Ticket、Store、所有寫入端）。

- **檔案鎖**：未採用。  
  Store 需取得鎖檔路徑（與 config 或 PID 路徑耦合），Save/Delete 前 lock、後 unlock；跨 process 時需 advisory lock，實作與平台行為較複雜。

- **Fallback（檢查 PID 檔）**：採用。  
  背景 work（`work --detach`）已寫入 PID 檔（如 `.tickets/.work.pid`）且 process 存活即視為「有背景寫入者」。  
  - 優點：不改 Ticket 結構、不改 Store 介面；改動集中在 CLI 寫入入口；與現有 detach/PID 機制一致。  
  - 行為：有背景 work 時，**禁止** 會寫入 store 的指令執行；僅查詢（如 `status`）允許與背景 work 並存。

## 介面與改動範圍評估

| 項目 | 變更 |
|------|------|
| **Ticket** | 無。不新增 version/ETag 欄位。 |
| **Store** | 無介面變更。Save、Load、Delete、MoveToStatus、MoveFailed、SaveGeneratedTickets 等簽名與行為不變。不在 Store 內做 PID 檢查（避免 ticket 依賴 config/CLI）。 |
| **CLI** | 會寫入 store 的指令（plan、add、edit、drop、run、work、retry、analyze、commit、clean 等）應在執行寫入前檢查：若 work PID 檔存在且該 process 存活，則拒絕執行並提示使用者。僅讀指令（如 status、部分 read-only 查詢）不檢查 PID，可與背景 work 並存。 |

## 實作要點

1. **Store / Ticket**  
   - 在 `internal/ticket/store.go` 與 `internal/ticket/ticket.go` 以註解記錄上述決策與 concurrency contract（寫入端須自行確保無並行寫入，例如透過 PID 檢查）。

2. **CLI 寫入端**  
   - 在會呼叫 `Store.Save`、`Delete`、`MoveToStatus`、`MoveFailed`、`SaveGeneratedTickets` 的指令開頭，使用既有 `ReadWorkPIDFile(cfg.WorkPIDFilePath())` 與 `IsProcessAlive(pid)` 判斷是否有背景 work；若存在則回傳錯誤並提示（例如「背景 work 執行中，請稍後再試或先停止背景 work」）。

3. **僅讀**  
   - `status`、`Load`、`LoadByStatus`、`Count` 等僅讀操作不檢查 PID，可與背景 work 並存。

## 參考

- 背景 work 與 PID 檔：`docs/detach-usage.md`、`internal/cli/detach.go`、`internal/config/config.go`（`WorkPIDFilePath()`）。
- 若未來需改為樂觀鎖，可再評估在 Ticket 增加 `Version`、Store.Save 改為 read-compare-write 的改動範圍與遷移步驟。

// Package i18n provides internationalization support for the agent orchestrator.
// All user-facing strings are centralized here for future localization.
package i18n

// Message keys organized by functional area.
// The current implementation uses Traditional Chinese (zh-TW) as the default.
// Future implementations can load these from locale-specific message files.

// Common messages
const (
	// General
	MsgSuccess    = "成功"
	MsgFailed     = "失敗"
	MsgCompleted  = "完成"
	MsgCancelled  = "已取消"
	MsgSkipped    = "跳過"
	MsgPending    = "等待中"
	MsgInProgress = "進行中"
	MsgNoData     = "沒有資料"
	MsgConfirm    = "確定"
	MsgYes        = "是"
	MsgNo         = "否"

	// Input prompts
	MsgInputEndHint        = "(輸入空行結束)"
	MsgTextareaPlaceholder  = "在此輸入內容..."
	MsgTextareaSubmitHint   = "(Ctrl+D 完成輸入，Ctrl+C 取消)"
	MsgTextinputPlaceholder = "在此輸入..."
	MsgTextinputSubmitHint  = "(Enter 確認，Esc/Ctrl+C 取消)"
	MsgSelectRange        = "選擇 (1-%d): "
	MsgInvalidSelection   = "無效的選擇: %s"
)

// Command descriptions
const (
	// Root command
	CmdRootShort = "協調多個 Cursor Agent 的 CLI 工具"
	CmdRootLong  = `Agent Orchestrator - 使用 Cursor Agent (Headless Mode) 作為 Subagents

這個工具可以幫助你：
  • 透過互動式問答初始化專案規劃 (init)
  • 分析現有專案並產生改進建議 (analyze)
  • 將 milestone 分解為可執行的 tickets (plan)
  • 自動執行 coding、review、test、commit 等任務

參考文件: https://cursor.com/docs/cli/headless`

	// Version command
	CmdVersionShort = "顯示版本資訊"

	// Init command
	CmdInitShort = "互動式專案初始化，產生 milestone"
	CmdInitLong  = `透過一系列問題來了解專案需求，然後產生對應的 milestone 文件。

範例:
  agent-orchestrator init "建立一個 Log 分析工具，使用 Drain 演算法"
  agent-orchestrator init  # 互動模式輸入目標`

	// Analyze command
	CmdAnalyzeShort = "分析現有專案，產生改進 issues 和 tickets"
	CmdAnalyzeLong  = `分析現有專案的程式碼，找出可改進的地方，包括效能問題、重構建議、安全性問題等。

範例:
  agent-orchestrator analyze
  agent-orchestrator analyze --scope performance,refactor
  agent-orchestrator analyze --scope security --auto`

	// Plan command
	CmdPlanShort = "分析 milestone 並產生 tickets"
	CmdPlanLong  = `分析 milestone 文件，將其分解為可執行的 tickets。

範例:
  agent-orchestrator plan docs/milestone-001.md
  agent-orchestrator plan docs/milestone.md --dry-run`

	// Work command
	CmdWorkShort = "處理 pending tickets"
	CmdWorkLong  = `處理所有 pending 狀態的 tickets，或指定單一 ticket 處理。

範例:
  agent-orchestrator work              # 處理所有 pending tickets
  agent-orchestrator work TICKET-001   # 處理指定 ticket
  agent-orchestrator work -p 5         # 使用 5 個並行 agents`

	// Review command
	CmdReviewShort = "執行程式碼審查"
	CmdReviewLong  = `對變更的檔案執行程式碼審查。如果沒有指定檔案，會自動取得 git 變更的檔案。

範例:
  agent-orchestrator review
  agent-orchestrator review src/main.go src/util.go`

	// Test command
	CmdTestShort = "執行專案測試"
	CmdTestLong  = `執行專案的測試並分析結果。

範例:
  agent-orchestrator test`

	// Commit command
	CmdCommitShort = "提交變更"
	CmdCommitLong  = `為完成的 ticket 建立 git commit。

範例:
  agent-orchestrator commit TICKET-001
  agent-orchestrator commit --all`

	// Run command
	CmdRunShort = "執行完整 pipeline"
	CmdRunLong  = `執行完整的開發 pipeline: plan -> work -> test -> review -> commit

範例:
  agent-orchestrator run docs/milestone.md
  agent-orchestrator run docs/milestone.md --analyze-first
  agent-orchestrator run docs/milestone.md --skip-test --skip-review`

	// Status command
	CmdStatusShort = "顯示 tickets 狀態"
	CmdStatusLong  = `顯示所有 tickets 的狀態統計和列表。

範例:
  agent-orchestrator status`

	// Retry command
	CmdRetryShort = "重試失敗的 tickets"
	CmdRetryLong  = `將所有失敗的 tickets 移回 pending 狀態，以便重新處理。

範例:
  agent-orchestrator retry
  agent-orchestrator retry && agent-orchestrator work`

	// Clean command
	CmdCleanShort = "清除所有 tickets 和 logs"
	CmdCleanLong  = `清除所有 tickets 和 agent 執行日誌。

範例:
  agent-orchestrator clean
  agent-orchestrator clean --force  # 不詢問直接清除`

	// Config command
	CmdConfigShort     = "設定管理"
	CmdConfigShowShort = "顯示目前設定"
	CmdConfigInitShort = "產生預設設定檔"
	CmdConfigPathShort = "顯示設定檔路徑"
	CmdConfigLong      = `顯示或管理 agent-orchestrator 設定。

範例:
  agent-orchestrator config           # 顯示目前設定
  agent-orchestrator config init      # 產生預設設定檔
  agent-orchestrator config path      # 顯示設定檔路徑`

	// Add command
	CmdAddShort = "新增 ticket"
	CmdAddLong  = `透過互動式問答或直接參數新增 ticket。

範例:
  agent-orchestrator add                              # 互動模式
  agent-orchestrator add --title "實作登入功能"        # 直接模式
  agent-orchestrator add --title "新增快取" --enhance  # AI 預處理
  agent-orchestrator add --title "重構" --type refactor --priority 2`

	// Edit command
	CmdEditShort = "修改 ticket"
	CmdEditLong  = `修改現有 ticket 的內容。

範例:
  agent-orchestrator edit TICKET-001                    # 互動模式
  agent-orchestrator edit TICKET-001 --title "新標題"   # 修改標題
  agent-orchestrator edit TICKET-001 --priority 1       # 修改優先級
  agent-orchestrator edit TICKET-001 --enhance          # AI 重新分析`

	// Drop command
	CmdDropShort = "刪除 ticket"
	CmdDropLong  = `刪除指定的 ticket。

範例:
  agent-orchestrator drop TICKET-001
  agent-orchestrator drop TICKET-001 --force  # 不詢問直接刪除`
)

// Flag descriptions
const (
	FlagConfig       = "設定檔路徑 (預設: .agent-orchestrator.yaml)"
	FlagDryRun       = "不實際執行 agent，只顯示會做什麼"
	FlagVerbose      = "詳細輸出"
	FlagDebug        = "除錯模式"
	FlagQuiet        = "安靜模式，只顯示錯誤"
	FlagOutput       = "Agent 輸出格式: text, json, stream-json"
	FlagParallel     = "最大並行 agents 數量 (預設使用設定值)"
	FlagScope        = "分析範圍: all, performance, refactor, security, test, docs (可用逗號分隔多個)"
	FlagAuto         = "自動產生 tickets 不詢問"
	FlagCommitAll    = "批次提交所有 completed tickets"
	FlagAnalyzeFirst = "先執行 analyze 分析現有專案"
	FlagSkipTest     = "跳過測試步驟"
	FlagSkipReview   = "跳過審查步驟"
	FlagSkipCommit   = "跳過提交步驟"
	FlagForce        = "不詢問直接執行"

	// Add/Edit ticket flags
	FlagTitle       = "Ticket 標題"
	FlagType        = "Ticket 類型: feature, bugfix, refactor, test, docs, performance, security"
	FlagPriority    = "優先級 (1-5，1 最高)"
	FlagDescription = "詳細描述"
	FlagDeps        = "依賴的 ticket IDs (逗號分隔)"
	FlagEnhance     = "使用 AI 預處理補充 ticket 內容"
	FlagCriteria    = "驗收條件 (逗號分隔)"
)

// UI messages
const (
	// Headers
	UIProjectInit      = "專案初始化"
	UIProjectAnalyze   = "專案分析"
	UIPlanning         = "規劃階段"
	UIProcessTickets   = "處理 Tickets"
	UIProcessTicket    = "處理 Ticket"
	UICodeReview       = "程式碼審查"
	UIRunTests         = "執行測試"
	UICommitChanges    = "提交變更"
	UIBatchCommit      = "批次提交"
	UICommitComplete   = "提交完成"
	UITicketStatus     = "Tickets 狀態"
	UIAnalysisReport   = "分析報告"
	UIRetryFailed      = "重試失敗的 Tickets"
	UICleanData        = "清除資料"
	UICurrentConfig    = "目前設定"
	UIFullPipeline     = "執行完整 Pipeline"
	UIPipelineComplete = "Pipeline 完成!"
	UIProcessComplete  = "處理完成"
	UICommonCommands   = "常用指令:"
	UIAddTicket        = "新增 Ticket"
	UIEditTicket       = "修改 Ticket"
	UIDropTicket       = "刪除 Ticket"

	// Info messages
	MsgProjectGoal             = "專案目標: %s"
	MsgAnalyzeProject          = "分析專案: %s"
	MsgAnalyzeScope            = "分析範圍: %s"
	MsgAnalyzeMilestone        = "分析 Milestone: %s"
	MsgProjectDir              = "專案目錄: %s"
	MsgMilestone               = "Milestone: %s"
	MsgDetectedExistingProject = "偵測到現有專案"
	MsgProjectSummary          = "專案摘要:"
	MsgScanComplete            = "掃描完成"
	MsgMaxParallel             = "最大並行數: %d"
	MsgIteration               = "迭代 %d: 處理 %d 個 tickets"
	MsgTicketInfo              = "ID: %s"
	MsgTicketTitle             = "標題: %s"
	MsgTicket                  = "Ticket: %s - %s"
	MsgChanges                 = "變更:"
	MsgReviewFiles             = "審查檔案:"
	MsgTestResult              = "測試結果:"
	MsgSummary                 = "摘要: %s"
	MsgFullOutput              = "完整輸出:"
	MsgDependencies            = "依賴: %v"
	MsgErrorDetail             = "錯誤: %s"
	MsgErrorLog                 = "詳細日誌: %s"
	MsgConfigFilePath          = "設定檔路徑: %s"
	MsgEditConfigHint          = "你可以編輯此檔案來自訂設定"

	// Counts and statistics
	MsgFoundIssues        = "共發現 %d 個問題"
	MsgGeneratedTickets   = "已產生 %d 個 tickets"
	MsgToDirectory        = "已產生 %d 個 tickets 到 %s"
	MsgPrepareCommit      = "準備提交 %d 個 tickets"
	MsgFoundFailedTickets = "找到 %d 個失敗的 tickets"
	MsgMovedToPending     = "已將 %d 個 tickets 移回 pending"
	MsgCountCompleted     = "完成: %d"
	MsgCountFailed        = "失敗: %d"
	MsgCountSkipped       = "跳過: %d"
	MsgCountSuccess       = "成功: %d"
	MsgCommitCount        = "提交 %d 個 commits"

	// Prompts
	PromptProjectGoal     = "請描述你的專案目標"
	PromptGenerateTickets = "要產生對應的 tickets 嗎？"
	PromptContinuePlan    = "要立即執行 plan 產生 tickets 嗎？"
	PromptConfirmClean    = "確定要清除所有資料嗎？"
	PromptOverwrite       = "要覆蓋嗎？"

	// Add/Edit ticket prompts
	PromptTicketTitle    = "請輸入 ticket 標題"
	PromptTicketDesc     = "請輸入詳細描述 (可多行)"
	PromptTicketType     = "請選擇 ticket 類型"
	PromptTicketPriority = "請輸入優先級 (1-5，1 最高)"
	PromptTicketDeps     = "請輸入依賴的 ticket IDs (逗號分隔，可留空)"
	PromptTicketCriteria = "請輸入驗收條件 (可多行)"
	PromptConfirmDrop    = "確定要刪除 ticket %s 嗎？"
	PromptEditField      = "選擇要修改的欄位"

	// Spinner messages
	SpinnerGeneratingQuestions = "產生問題中..."
	SpinnerGeneratingMilestone = "產生 milestone 文件中..."
	SpinnerAnalyzing           = "分析專案中..."
	SpinnerPlanning            = "分析並產生 tickets..."
	SpinnerReviewing           = "審查程式碼中..."
	SpinnerTesting             = "執行測試中..."
	SpinnerCommitting          = "產生並執行 commit..."
	SpinnerProcessing          = "處理 %s: %s"
	SpinnerEnhancing           = "AI 分析並補充 ticket 內容..."
	SpinnerScanningProject     = "掃描專案結構中..."

	// Success messages
	MsgQuestionsGenerated = "已產生問題"
	MsgMilestoneGenerated = "已產生 milestone"
	MsgMilestoneCreated   = "已產生 milestone: %s"
	MsgAnalysisComplete   = "分析完成"
	MsgPlanningComplete   = "規劃完成"
	MsgReviewApproved     = "審查通過"
	MsgReviewComplete     = "審查完成"
	MsgTestComplete       = "測試完成"
	MsgCommitSuccess      = "提交成功"
	MsgTicketCreated      = "建立 ticket: %s - %s"
	MsgNoIssuesFound      = "沒有發現問題！"
	MsgDataCleared        = "已清除所有資料"
	MsgConfigGenerated    = "已產生設定檔: %s"
	MsgProcessingComplete = "%s 完成"
	MsgTicketAdded        = "已新增 ticket: %s"
	MsgTicketUpdated      = "已更新 ticket: %s"
	MsgTicketDropped      = "已刪除 ticket: %s"
	MsgEnhanceComplete    = "AI 預處理完成"

	// Warning messages
	MsgNoTicketsGenerated  = "沒有產生任何 tickets"
	MsgDependencyWarning   = "依賴驗證警告: %s"
	MsgCircularDependency  = "警告: 發現循環依賴"
	MsgTicketStatusWarning = "Ticket %s 狀態為 %s，建議只提交已完成的 tickets"
	MsgTicketCannotProcess = "Ticket %s 狀態為 %s，無法處理"
	MsgPendingBlocked      = "還有 %d 個 tickets 但依賴未滿足"
	MsgProcessInterrupted  = "處理已中斷"
	MsgPipelineInterrupted = "Pipeline 已中斷"
	MsgConfigExists        = "設定檔已存在: %s"
	MsgAboutToDelete       = "即將刪除以下資料:"
	MsgTicketsDir          = "Tickets 目錄: %s"
	MsgLogsDir             = "Logs 目錄: %s"
	MsgCurrentStatus       = "目前狀態:"
	MsgInterruptSignal     = "\n收到中斷信號，正在優雅關閉..."

	// Error messages
	ErrAgentNotFound        = "找不到 agent 指令，請確保已安裝 Cursor CLI"
	ErrAgentCommand         = "找不到 agent 指令"
	ErrMilestoneNotFound    = "Milestone 檔案不存在: %s"
	ErrTicketNotFound       = "找不到 ticket: %s"
	ErrDeleteTicketFailed   = "刪除 ticket 失敗"
	ErrLoadConfigFailed     = "載入設定失敗: %s"
	ErrInitStoreFailed      = "初始化 ticket store 失敗: %w"
	ErrSaveTicketFailed     = "儲存 ticket 失敗: %s"
	ErrCleanTicketsFailed   = "清除 tickets 失敗: %s"
	ErrCleanLogsFailed      = "清除 logs 失敗: %s"
	ErrGenerateConfigFailed = "產生設定檔失敗: %s"

	// Spinner fail messages
	SpinnerFailQuestions   = "產生問題失敗"
	SpinnerFailMilestone   = "產生 milestone 失敗"
	SpinnerFailAnalysis    = "分析失敗"
	SpinnerFailPlanning    = "規劃失敗"
	SpinnerFailReview      = "審查失敗"
	SpinnerFailReviewNeeds = "審查需要修改"
	SpinnerFailTest        = "測試執行失敗"
	SpinnerFailTestHas     = "測試有失敗"
	SpinnerFailCommit      = "提交失敗"
	SpinnerFailTicket      = "%s 失敗"

	// Hints
	HintRunPlanLater = "你可以稍後執行: agent-orchestrator plan %s"
	HintRunWork      = "執行 'agent-orchestrator work' 開始處理 tickets"
	HintRunStatus    = "執行 'agent-orchestrator status' 查看狀態"
	HintRunWorkCmd   = "agent-orchestrator work        # 處理 pending tickets"
	HintRunRetryCmd  = "agent-orchestrator retry       # 重試失敗的 tickets"
	HintRunCommitCmd = "agent-orchestrator commit --all  # 提交所有完成的 tickets"

	// Status page messages
	MsgNoTickets         = "沒有任何 tickets"
	MsgNoDataToClean     = "沒有資料需要清除"
	MsgNoChangesToCommit = "沒有變更需要提交"
	MsgNoFilesToReview   = "沒有檔案需要審查"
	MsgNoFailedToRetry   = "沒有失敗的 tickets 需要重試"
	MsgNoCompletedCommit = "沒有 completed tickets 需要提交"
	MsgSkipNoChanges     = "沒有變更需要提交 (跳過)"

	// Getting started messages
	MsgGettingStarted        = "使用以下指令開始:"
	MsgGettingStartedInit    = "  agent-orchestrator init \"專案目標\"   # 互動式初始化"
	MsgGettingStartedPlan    = "  agent-orchestrator plan <milestone>  # 從 milestone 產生 tickets"
	MsgGettingStartedAnalyze = "  agent-orchestrator analyze           # 分析現有專案"
	MsgGettingStartedAdd     = "  agent-orchestrator add               # 直接新增 ticket"

	// Analysis categories
	CategoryPerformance = "效能問題"
	CategoryRefactor    = "重構建議"
	CategorySecurity    = "安全性問題"
	CategoryTest        = "測試覆蓋"
	CategoryDocs        = "文件缺失"

	// Pipeline steps
	StepAnalyze    = "Analyze - 分析現有專案..."
	StepPlanning   = "Planning - 分析 milestone 產生 tickets..."
	StepCoding     = "Coding - 處理 tickets..."
	StepTesting    = "Testing - 執行測試..."
	StepReview     = "Review - 程式碼審查..."
	StepCommitting = "Committing - 提交變更..."
)

// Agent prompts and messages (caller, coding, planning, enhance)
const (
	// Caller
	AgentContextFilesLabel = "相關檔案: %s"
	AgentWriteJSONToFile    = "請將結果以 JSON 格式寫入檔案: %s"
	AgentDryRunSkipCall     = "[DRY RUN] 跳過實際 agent 呼叫"
	AgentModelInUse         = "使用模型: %s"
	AgentWriteFile          = "寫入檔案: %s"
	AgentReadFile           = "讀取檔案: %s"
	AgentDurationMs = "完成，耗時 %.0fms"

	// Coding agent prompt
	AgentCodingIntro           = "你是一個專業的開發 Agent。請根據以下 ticket 實作程式碼。\n\n"
	AgentCodingProjectRoot     = "專案根目錄: %s\n\n"
	AgentCodingSectionTicket   = "## Ticket 資訊\n"
	AgentCodingTicketId        = "- ID: %s\n"
	AgentCodingTicketTitle     = "- 標題: %s\n"
	AgentCodingTicketDesc      = "- 描述: %s\n"
	AgentCodingTicketType      = "- 類型: %s\n"
	AgentCodingTicketComplexity = "- 複雜度: %s\n\n"
	AgentCodingSectionFilesCreate = "## 需要建立的檔案\n"
	AgentCodingSectionFilesModify = "## 需要修改的檔案\n"
	AgentCodingSectionAcceptance  = "## 驗收標準\n"
	AgentCodingSteps = `## 請執行以下步驟:
1. 閱讀相關的現有程式碼 (如果有)
2. 實作 ticket 所描述的功能
3. 確保程式碼符合最佳實踐
4. 新增必要的 import 語句
5. 確保程式碼可以編譯
6. 如果適當，新增對應的單元測試

完成後，說明你所做的變更。`

	// Analyze agent prompt
	AgentAnalyzeIntro       = "你是一個程式碼分析專家。請分析當前專案的程式碼，找出可改進的地方。\n\n"
	AgentAnalyzeProjectDir  = "專案目錄: %s\n\n"
	AgentAnalyzeAspects     = "請分析以下方面：\n"
	AgentAnalyzePerf        = "- **效能問題**: N+1 查詢、不必要的迴圈、記憶體浪費等\n"
	AgentAnalyzeRefactor    = "- **重構建議**: 過長的方法、重複程式碼、缺少抽象等\n"
	AgentAnalyzeSecurity    = "- **安全性問題**: 硬編碼密碼、SQL 注入、XSS 等\n"
	AgentAnalyzeTest        = "- **測試覆蓋**: 缺少測試的關鍵功能\n"
	AgentAnalyzeDocs        = "- **文件缺失**: 缺少重要文件或註解\n"
	AgentAnalyzeJSONOutput  = `
請以 JSON 格式輸出分析結果：
{
  "issues": [
    {
      "id": "ISSUE-001",
      "category": "performance|refactor|security|test|docs",
      "severity": "HIGH|MED|LOW",
      "title": "問題標題",
      "description": "詳細描述",
      "location": "檔案路徑:行號",
      "suggestion": "建議修復方式"
    }
  ]
}

請將結果寫入 .tickets/analysis-result.json`

	// Planning agent prompt
	AgentPlanningPromptTemplate = `你是一個專案規劃 Agent。請分析 milestone 文件並產生 tickets。

請讀取檔案 %s 的內容，然後產生 JSON 格式的 tickets 列表。

每個 ticket 包含:
- id: 唯一識別碼 (格式: TICKET-xxx-描述)
- title: 簡短標題
- description: 詳細描述
- type: 類型 (feature/test/refactor/docs/bugfix/performance/security)
- priority: 優先級 (1-5, 1最高)
- estimated_complexity: 複雜度 (low/medium/high)
- dependencies: 依賴的其他 ticket ID 列表
- acceptance_criteria: 驗收標準列表
- files_to_create: 需要建立的檔案
- files_to_modify: 需要修改的檔案

請確保：
1. Tickets 之間的依賴關係正確
2. 每個 ticket 都是獨立可完成的工作單元
3. 複雜的任務要拆分成多個小 tickets
4. 按照優先級排序

請將結果以 JSON 格式寫入檔案: %s
格式為: {"tickets": [...]}`

	// Enhance agent prompt
	AgentEnhanceIntro     = "你是一個專案分析專家。請根據以下 ticket 資訊和專案結構，補充更詳細的實作細節。\n\n"
	AgentEnhanceProjectDir = "專案目錄: %s\n\n"
	AgentEnhanceSection    = "## 原始 Ticket 資訊\n"
	AgentEnhanceId         = "- ID: %s\n"
	AgentEnhanceTitle      = "- 標題: %s\n"
	AgentEnhanceType       = "- 類型: %s\n"
	AgentEnhancePriority   = "- 優先級: P%d\n"
	AgentEnhanceDesc       = "- 描述: %s\n"
	AgentEnhanceDeps       = "- 依賴: %s\n"
	AgentEnhanceCriteria   = "- 驗收條件:\n"
	AgentEnhanceJSONBlock  = `## 請分析專案結構並補充以下資訊

請以 JSON 格式輸出分析結果：
{
  "description": "補充或改進的詳細描述",
  "estimated_complexity": "low|medium|high",
  "acceptance_criteria": ["驗收條件1", "驗收條件2"],
  "files_to_create": ["可能需要建立的檔案路徑"],
  "files_to_modify": ["可能需要修改的檔案路徑"],
  "implementation_hints": ["實作建議1", "實作建議2"]
}

分析要點:
1. 根據專案結構推斷需要修改或建立的檔案
2. 評估實作複雜度 (low/medium/high)
3. 補充具體可測試的驗收條件
4. 提供實作建議

請將結果寫入 .tickets/enhance-result.json`

	// Init/Planning agent prompts (planning.go init-related)
	AgentInitScanIntro     = "你是一個專案分析專家。請分析當前目錄的專案結構。\n\n專案目錄: %s\n\n請掃描專案並回答：\n1. 主要使用的程式語言\n2. 使用的框架或工具（如果有）\n3. 專案結構（主要資料夾）\n4. 是否有測試檔案\n5. 是否有文件（README, docs/）\n6. 簡短描述這個專案的功能\n\n請以 JSON 格式輸出：\n{\n  \"language\": \"主要語言\",\n  \"framework\": \"框架名稱（沒有則空字串）\",\n  \"structure\": \"主要資料夾，如 cmd/, internal/, pkg/\",\n  \"main_files\": [\"重要檔案1\", \"重要檔案2\"],\n  \"has_tests\": true/false,\n  \"has_docs\": true/false,\n  \"description\": \"專案功能簡述\"\n}"
	AgentInitQuestionsExisting = "你是一個專案規劃助手。使用者想要在現有專案上進行以下開發：\n\n## 開發目標\n\"%s\"\n\n## 現有專案資訊\n- 語言: %s\n- 框架: %s\n- 結構: %s\n- 專案描述: %s\n- 已有測試: %v\n- 已有文件: %v\n\n請產生 5-7 個針對性問題，幫助我了解更多細節以便產生完整的 milestone。\n因為這是現有專案，問題應該聚焦在：\n1. 新功能如何與現有架構整合\n2. 是否需要修改現有模組\n3. 與現有功能的互動方式\n4. 相容性考量\n5. 測試策略\n6. 部署/遷移考量\n\n請以 JSON 格式輸出：{\"questions\": [\"問題1\", \"問題2\", ...]}"
	AgentInitQuestionsNew   = "你是一個專案規劃助手。使用者想要建立以下專案：\n\n\"%s\"\n\n請產生 5-7 個關鍵問題，幫助我了解更多細節以便產生完整的 milestone。\n問題應該涵蓋：\n1. 技術選型（程式語言、框架等）\n2. 目標使用者\n3. 關鍵功能需求\n4. 效能/規模需求\n5. 部署環境\n6. 整合需求\n\n請以 JSON 格式輸出：{\"questions\": [\"問題1\", \"問題2\", ...]}"
	AgentInitMilestoneExisting = "你是一個專案規劃專家。請根據以下資訊產生詳細的 milestone 文件。\n\n## 開發目標\n%s\n\n## 現有專案資訊\n- 語言: %s\n- 框架: %s\n- 專案結構: %s\n- 專案描述: %s\n- 已有測試: %v\n- 已有文件: %v\n\n## 需求細節\n%s\n\n請產生一個 Markdown 格式的 milestone 文件，包含：\n1. 開發目標概述\n2. 現有架構分析（與新功能的關聯）\n3. 功能需求清單\n4. 實作階段規劃（分成多個 phase）\n   - 考慮與現有程式碼的整合順序\n   - 標註需要修改的現有模組\n5. 每個階段的具體任務\n6. 測試計畫（包含整合測試）\n7. 驗收標準\n\n請將結果寫入檔案: %s"
	AgentInitMilestoneNew   = "你是一個專案規劃專家。請根據以下資訊產生詳細的 milestone 文件。\n\n## 專案目標\n%s\n\n## 需求細節\n%s\n\n請產生一個 Markdown 格式的 milestone 文件，包含：\n1. 專案概述\n2. 技術架構\n3. 功能需求清單\n4. 實作階段規劃（分成多個 phase）\n5. 每個階段的具體任務\n6. 驗收標準\n\n請將結果寫入檔案: %s"
)

// Agent error messages (coding, planning, enhance, init)
const (
	ErrAgentMkdirOutput   = "無法建立輸出目錄: %w"
	ErrAgentMkdirDocs     = "無法建立文件目錄: %w"
	ErrAgentAnalyzeFailed = "分析失敗: %w"
	ErrAgentAnalyzeOutput = "分析失敗: %s"
	ErrAgentInvalidIssues = "無效的 issues 格式"
	ErrAgentReadMilestone = "無法讀取 milestone 檔案: %w"
	ErrAgentPlanningFailed = "規劃失敗: %w"
	ErrAgentPlanningOutput = "規劃失敗: %s"
	ErrAgentInvalidTickets = "無效的 tickets 格式"
	ErrAgentEnhanceFailed  = "AI 預處理失敗: %w"
	ErrAgentEnhanceOutput  = "AI 預處理失敗: %s"
	ErrAgentScanFailed     = "掃描專案失敗: %w"
	ErrAgentWriteMilestone = "無法寫入 milestone 檔案: %w"
	ErrAgentCreateMilestone = "產生 milestone 失敗: %s"
)

// Error messages for the errors package
const (
	// Error operation names
	ErrOpAgent    = "agent"
	ErrOpFile     = "file"
	ErrOpStore    = "store"
	ErrOpAnalyze  = "analyze"
	ErrOpTest     = "test"
	ErrOpReview   = "review"
	ErrOpPlanning = "planning"

	// Error messages
	ErrMsgAgentNotAvailable = "agent command not available"
	ErrMsgFileNotFound      = "file not found: %s"
	ErrMsgSaveTicket        = "failed to save ticket %s"
	ErrMsgAnalysisFailed    = "analysis failed"
	ErrMsgTestFailed        = "test execution failed"
	ErrMsgReviewFailed      = "code review failed"
	ErrMsgPlanningFailed    = "planning failed"
	ErrMsgStoreInit         = "failed to initialize store"
)

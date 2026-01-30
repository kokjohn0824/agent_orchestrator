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

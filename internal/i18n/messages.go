package i18n

// Message key constants for i18n translations.
//
// Usage: i18n.T(i18n.MsgAssetCreateSuccess, pongo2.Context{"id": id})
const (
	// Asset operations
	MsgAssetCmdShort        = "asset_cmd_short"
	MsgAssetCreateSuccess   = "asset_create_success"
	MsgAssetArchiveSuccess  = "asset_archive_success"
	MsgAssetRestoreSuccess  = "asset_restore_success"
	MsgAssetDeleteSuccess   = "asset_delete_success"
	MsgAssetNotFound        = "asset_not_found"
	MsgAssetStateConflict   = "asset_state_conflict"

	// Asset list
	MsgAssetListShort       = "asset_list_short"
	MsgAssetListHeader      = "asset_list_header"
	MsgAssetSearchFailed    = "asset_search_failed"

	// Asset show
	MsgAssetShowShort       = "asset_show_short"
	MsgAssetGetFailed       = "asset_get_failed"

	// Asset cat
	MsgAssetCatShort        = "asset_cat_short"

	// Asset create
	MsgAssetCreateShort     = "asset_create_short"
	MsgAssetNameRequired    = "asset_name_required"
	MsgAssetContentFileConflict = "asset_content_file_conflict"
	MsgAssetFileReadFailed  = "asset_file_read_failed"
	MsgAssetStdinReadFailed = "asset_stdin_read_failed"
	MsgAssetInputRequired   = "asset_input_required"
	MsgAssetInvalidIDFormat = "asset_invalid_id_format"
	MsgAssetDirCreateFailed = "asset_dir_create_failed"
	MsgAssetMarkdownFormatFailed = "asset_markdown_format_failed"
	MsgAssetFileWriteFailed = "asset_file_write_failed"
	MsgAssetSaveFailed      = "asset_save_failed"
	MsgAssetReconcileWarn   = "asset_reconcile_warn"

	// Asset edit
	MsgAssetEditShort       = "asset_edit_short"

	// Asset rm
	MsgAssetRmShort         = "asset_rm_short"
	MsgAssetFileNotFound    = "asset_file_not_found"
	MsgAssetFileReadError   = "asset_file_read_error"
	MsgAssetFrontmatterParseError = "asset_frontmatter_parse_error"
	MsgAssetPleaseArchiveFirst = "asset_please_archive_first"
	MsgAssetFileDeleteFailed = "asset_file_delete_failed"
	MsgAssetIndexRemoveWarn = "asset_index_remove_warn"

	// Asset archive
	MsgAssetArchiveShort    = "asset_archive_short"

	// Asset restore
	MsgAssetRestoreShort    = "asset_restore_short"

	// Asset promote/demote
	MsgAssetPromoteShort    = "asset_promote_short"
	MsgAssetDemoteShort     = "asset_demote_short"

	// Label operations
	MsgLabelCmdShort        = "label_cmd_short"
	MsgLabelListShort       = "label_list_short"
	MsgLabelSetShort        = "label_set_short"
	MsgLabelUnsetShort      = "label_unset_short"
	MsgLabelGetFailed       = "label_get_failed"
	MsgLabelSetOutput       = "label_set_output"
	MsgLabelUnsetOutput     = "label_unset_output"

	// Snapshot operations
	MsgSnapshotCmdShort     = "snapshot_cmd_short"
	MsgSnapshotListShort    = "snapshot_list_short"
	MsgSnapshotDiffShort    = "snapshot_diff_short"
	MsgSnapshotCheckoutShort = "snapshot_checkout_short"
	MsgSnapshotOpenRepoFailed = "snapshot_open_repo_failed"
	MsgSnapshotHistoryFailed = "snapshot_history_failed"
	MsgSnapshotDiffFailed   = "snapshot_diff_failed"
	MsgSnapshotCheckoutStarted = "snapshot_checkout_started"

	// Eval operations
	MsgEvalCmdShort         = "eval_cmd_short"
	MsgEvalRunShort         = "eval_run_short"
	MsgEvalRunStarted       = "eval_run_started"
	MsgEvalRunFailed        = "eval_run_failed"
	MsgEvalRunStatus        = "eval_run_status"
	MsgEvalRunConcurrency   = "eval_run_concurrency"
	MsgEvalRunModel         = "eval_run_model"
	MsgEvalRunTemperature   = "eval_run_temperature"
	MsgEvalCancelSuccess    = "eval_cancel_success"
	MsgEvalCancelShort      = "eval_cancel_short"
	MsgEvalCancelFailed     = "eval_cancel_failed"
	MsgEvalCancelStarted    = "eval_cancel_started"
	MsgEvalExecutionStarted = "eval_execution_started"
	MsgEvalExecutionCancelled = "eval_execution_cancelled"
	MsgEvalCompareTitle     = "eval_compare_title"
	MsgEvalScoreDelta       = "eval_score_delta"
	MsgEvalCasesShort       = "eval_cases_short"
	MsgEvalCasesFailed      = "eval_cases_failed"
	MsgEvalCompareShort     = "eval_compare_short"
	MsgEvalCompareFailed    = "eval_compare_failed"
	MsgEvalReportShort      = "eval_report_short"
	MsgEvalReportFailed     = "eval_report_failed"
	MsgEvalReportComplete   = "eval_report_complete"
	MsgEvalDiagnoseShort    = "eval_diagnose_short"
	MsgEvalDiagnoseFailed   = "eval_diagnose_failed"
	MsgEvalDiagnoseComplete = "eval_diagnose_complete"
	MsgEvalDiagnoseSeverity = "eval_diagnose_severity"
	MsgEvalDiagnoseStrategy = "eval_diagnose_strategy"
	MsgEvalDiagnoseLocation = "eval_diagnose_location"
	MsgEvalDiagnoseProblem  = "eval_diagnose_problem"
	MsgEvalDiagnoseSuggestion = "eval_diagnose_suggestion"
	MsgEvalSetupShort       = "eval_setup_short"
	MsgEvalSetupInvalidID   = "eval_setup_invalid_id"
	MsgEvalSetupCreateDirFailed = "eval_setup_create_dir_failed"
	MsgEvalSetupAlreadyExists = "eval_setup_already_exists"
	MsgEvalSetupFormatFailed  = "eval_setup_format_failed"
	MsgEvalSetupWriteFailed    = "eval_setup_write_failed"
	MsgEvalSetupComplete    = "eval_setup_complete"
	MsgEvalSetupModel       = "eval_setup_model"
	MsgEvalListShort        = "eval_list_short"
	MsgEvalListAssetIDRequired = "eval_list_asset_id_required"
	MsgEvalListFailed       = "eval_list_failed"
	MsgEvalRunComplete      = "eval_run_complete"
	MsgEvalCasesList        = "eval_cases_list"
	MsgEvalCompareComplete  = "eval_compare_complete"
	MsgEvalListComplete     = "eval_list_complete"

	// Server operations
	MsgServeStarting        = "serve_starting"
	MsgServeStarted         = "serve_started"
	MsgServeAPIEndpoint     = "serve_api_endpoint"
	MsgServeSSEEndpoint     = "serve_sse_endpoint"
	MsgServeOpeningBrowser  = "serve_opening_browser"

	// Init operations
	MsgInitTitle            = "init_title"
	MsgInitStart            = "init_start"
	MsgInitGitComplete      = "init_git_complete"
	MsgInitGitWarn          = "init_git_warn"
	MsgInitLockReadWarn     = "init_lock_read_warn"
	MsgInitLockWriteWarn    = "init_lock_write_warn"
	MsgInitLockAdded        = "init_lock_added"
	MsgInitComplete         = "init_complete"
	MsgInitDBDirFail        = "init_db_dir_fail"
	MsgInitGitPath          = "init_git_path"
	MsgInitSQLitePath       = "init_sqlite_path"
	MsgInitServeHint        = "init_serve_hint"

	// Repo operations
	MsgRepoCmd              = "repo_cmd"
	MsgRepoCmdShort         = "repo_cmd_short"
	MsgRepoCmdLong          = "repo_cmd_long"
	MsgRepoList             = "repo_list"
	MsgRepoListShort        = "repo_list_short"
	MsgRepoLockReadFail     = "repo_lock_read_fail"
	MsgRepoNoRepos          = "repo_no_repos"
	MsgRepoCurrent          = "repo_current"
	MsgRepoMarkerCurrent     = "repo_marker_current"
	MsgRepoMarkerCurrentNotFound = "repo_marker_current_not_found"
	MsgRepoMarkerCurrentInvalid  = "repo_marker_current_invalid"
	MsgRepoNotFound         = "repo_not_found"
	MsgRepoNotGit           = "repo_not_git"
	MsgRepoSwitchCmd        = "repo_switch_cmd"
	MsgRepoSwitchCmdShort   = "repo_switch_cmd_short"
	MsgRepoSwitchPathNotFound   = "repo_switch_path_not_found"
	MsgRepoSwitchAskCreate  = "repo_switch_ask_create"
	MsgRepoSwitchCancel     = "repo_switch_cancel"
	MsgRepoSwitchDirFail    = "repo_switch_dir_fail"
	MsgRepoSwitchGitWarn    = "repo_switch_git_warn"
	MsgRepoSwitchNotGit     = "repo_switch_not_git"
	MsgRepoSwitchAskGit     = "repo_switch_ask_git"
	MsgRepoSwitchGitFail    = "repo_switch_git_fail"
	MsgRepoSwitchLockWriteFail   = "repo_switch_lock_write_fail"
	MsgRepoSwitchComplete   = "repo_switch_complete"

	// Adapt operations
	MsgAdaptCmd             = "adapt_cmd"
	MsgAdaptCmdShort        = "adapt_cmd_short"
	MsgAdaptAsset           = "adapt_asset"
	MsgAdaptFromTo          = "adapt_from_to"
	MsgAdaptSaveAs          = "adapt_save_as"
	MsgAdaptAutoEval        = "adapt_auto_eval"

	// Optimize operations
	MsgOptimizeCmd          = "optimize_cmd"
	MsgOptimizeCmdShort     = "optimize_cmd_short"
	MsgOptimizeAsset        = "optimize_asset"
	MsgOptimizeStrategy     = "optimize_strategy"
	MsgOptimizeMaxIterations    = "optimize_max_iterations"
	MsgOptimizeThresholdDelta   = "optimize_threshold_delta"
	MsgOptimizeAutoPromoteNote  = "optimize_auto_promote_note"
	MsgOptimizeFlagStrategy     = "optimize_flag_strategy"
	MsgOptimizeFlagIterations   = "optimize_flag_iterations"
	MsgOptimizeFlagThreshold    = "optimize_flag_threshold"
	MsgOptimizeFlagAutoPromote  = "optimize_flag_auto_promote"

	// Diff operations
	MsgDiffCmd              = "diff_cmd"
	MsgDiffCmdShort         = "diff_cmd_short"
	MsgDiffSummary          = "diff_summary"
	MsgDiffImpact           = "diff_impact"
	MsgDiffChange           = "diff_change"

	// Rewrite operations
	MsgRewriteCmd           = "rewrite_cmd"
	MsgRewriteCmdShort      = "rewrite_cmd_short"

	// Sync operations
	MsgSyncCmdShort         = "sync_cmd_short"
	MsgSyncReconcileDone    = "sync_reconcile_done"
	MsgSyncReconcileShort   = "sync_reconcile_short"
	MsgSyncReconcileFailed  = "sync_reconcile_failed"
	MsgSyncAdded            = "sync_added"
	MsgSyncUpdated          = "sync_updated"
	MsgSyncDeleted          = "sync_deleted"
	MsgSyncError            = "sync_error"
	MsgSyncRepoSwitch       = "sync_repo_switch"
	MsgSyncReconcileStart   = "sync_reconcile_start"
	MsgSyncReconcileWarning = "sync_reconcile_warning"
	MsgSyncExportStart      = "sync_export_start"
	MsgSyncExportShort      = "sync_export_short"
	MsgSyncExportFailed     = "sync_export_failed"
	MsgSyncOpenRepoFailed   = "sync_open_repo_failed"
	MsgSyncResolvePathFailed = "sync_resolve_path_failed"
	MsgSyncReadLockFailed   = "sync_read_lock_failed"
	MsgSyncWriteLockFailed  = "sync_write_lock_failed"
	MsgSyncNoRepoSet        = "sync_no_repo_set"

	// Common messages
	MsgCommonCancel         = "common_cancel"
	MsgCommonConfirm        = "common_confirm"
	MsgCommonError          = "common_error"
	MsgCommonLoading        = "common_loading"
	MsgCommonSuccess        = "common_success"
	MsgCommonWarning        = "common_warning"

	// Trigger operations
	MsgTriggerCmdShort      = "trigger_cmd_short"
	MsgTriggerMatchShort    = "trigger_match_short"
	MsgTriggerMatchComplete = "trigger_match_complete"
	MsgTriggerMatchFound    = "trigger_match_found"
	MsgTriggerMatchFailed   = "trigger_match_failed"

	// Import operations
	MsgImportCmdShort       = "import_cmd_short"
	MsgImportCmdLong        = "import_cmd_long"
	MsgImportReportHeader   = "import_report_header"
	MsgImportReportTotal    = "import_report_total"
	MsgImportReportImported = "import_report_imported"
	MsgImportReportSkipped  = "import_report_skipped"
	MsgImportReportDetails  = "import_report_details"
	MsgImportDryRun         = "import_dry_run"

	// Flag descriptions
	MsgFlagSnapshot    = "flag_snapshot"
	MsgFlagCase        = "flag_case"
	MsgFlagConcurrency = "flag_concurrency"
	MsgFlagModel       = "flag_model"
	MsgFlagTemperature = "flag_temperature"
	MsgFlagNoSync      = "flag_no_sync"
	MsgFlagEvalsDir    = "flag_evals_dir"
	MsgFlagDir         = "flag_dir"
	MsgFlagJsonOutput  = "flag_json_output"
	MsgFlagFormat      = "flag_format"
	MsgFlagRepo        = "flag_repo"
	MsgFlagTop         = "flag_top"
	MsgFlagBizLine     = "flag_biz_line"
	MsgFlagDryRun      = "flag_dry_run"
	MsgFlagPromptsDir  = "flag_prompts_dir"
	MsgFlagOutput      = "flag_output"

	// Asset flag descriptions
	MsgFlagAssetID       = "flag_asset_id"
	MsgFlagAssetName     = "flag_asset_name"
	MsgFlagAssetFile     = "flag_asset_file"
	MsgFlagAssetContent  = "flag_asset_content"
	MsgFlagAssetTag      = "flag_asset_tag"
	MsgFlagAssetBizLine  = "flag_asset_biz_line"

	// Snapshot flag descriptions
	MsgFlagSnapshotLimit = "flag_snapshot_limit"

	// Error messages
	MsgErrAssetNotFound     = "err_asset_not_found"
	MsgErrInvalidID         = "err_invalid_id"
	MsgErrGitNotInitialized = "err_git_not_initialized"
	MsgErrStorageNotConfigured = "err_storage_not_configured"
)

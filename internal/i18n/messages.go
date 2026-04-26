package i18n

// Message key constants for i18n translations.
//
// Usage: i18n.T(i18n.MsgAssetCreateSuccess, id)
const (
	// Asset operations
	MsgAssetCreateSuccess   = "asset_create_success"
	MsgAssetArchiveSuccess  = "asset_archive_success"
	MsgAssetRestoreSuccess  = "asset_restore_success"
	MsgAssetDeleteSuccess   = "asset_delete_success"
	MsgAssetNotFound        = "asset_not_found"
	MsgAssetStateConflict   = "asset_state_conflict"

	// Eval operations
	MsgEvalRunStarted       = "eval_run_started"
	MsgEvalCancelSuccess    = "eval_cancel_success"
	MsgEvalCompareTitle     = "eval_compare_title"
	MsgEvalScoreDelta       = "eval_score_delta"

	// Server operations
	MsgServeStarting        = "serve_starting"
	MsgServeStarted         = "serve_started"
	MsgServeAPIEndpoint     = "serve_api_endpoint"
	MsgServeSSEEndpoint     = "serve_sse_endpoint"
	MsgServeOpeningBrowser  = "serve_opening_browser"

	// Init operations
	MsgInitTitle            = "init_title"
	MsgInitGitComplete      = "init_git_complete"
	MsgInitLockAdded        = "init_lock_added"
	MsgInitComplete         = "init_complete"
	MsgInitServeHint        = "init_serve_hint"

	// Sync operations
	MsgSyncReconcileDone    = "sync_reconcile_done"
	MsgSyncAdded            = "sync_added"
	MsgSyncUpdated          = "sync_updated"
	MsgSyncDeleted          = "sync_deleted"
	MsgSyncError            = "sync_error"

	// Common messages
	MsgCommonCancel         = "common_cancel"
	MsgCommonConfirm        = "common_confirm"
	MsgCommonError          = "common_error"
	MsgCommonLoading        = "common_loading"
	MsgCommonSuccess        = "common_success"
	MsgCommonWarning        = "common_warning"

	// Error messages
	MsgErrAssetNotFound     = "err_asset_not_found"
	MsgErrInvalidID         = "err_invalid_id"
	MsgErrGitNotInitialized = "err_git_not_initialized"
	MsgErrStorageNotConfigured = "err_storage_not_configured"
)

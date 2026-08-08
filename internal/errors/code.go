package apperr

type Code string
type message string

const (
	CodeConfigBroken              Code = "CONFIG_BROKEN"
	CodeConfigNotFound            Code = "CONFIG_NOT_FOUND"
	CodeConfigReadFailed          Code = "CONFIG_READ_FAILED"
	CodeConfigWriteFailed         Code = "CONFIG_WRITE_FAILED"
	CodeSecureStoreFailed         Code = "SECURE_STORE_FAILED"
	CodeDBConnectFailed           Code = "DB_CONNECT_FAILED"
	CodeValidationFailed          Code = "VALIDATION_FAILED"
	CodeProfileNotFound           Code = "PROFILE_NOT_FOUND"
	CodeCredentialUnavailable     Code = "CREDENTIAL_UNAVAILABLE" // #nosec G101 -- エラーコードであり資格情報ではない。
	CodeConnectionFailed          Code = "CONNECTION_FAILED"
	CodeCredentialSaveFailed      Code = "CREDENTIAL_SAVE_FAILED"   // #nosec G101 -- エラーコードであり資格情報ではない。
	CodeCredentialDeleteFailed    Code = "CREDENTIAL_DELETE_FAILED" // #nosec G101 -- エラーコードであり資格情報ではない。
	CodeConfigSaveFailed          Code = "CONFIG_SAVE_FAILED"
	CodeConsistencyRecoveryFailed Code = "CONSISTENCY_RECOVERY_FAILED"
	CodeSchemaLoadFailed          Code = "SCHEMA_LOAD_FAILED"
	CodeStatsLoadFailed           Code = "STATS_LOAD_FAILED"
	CodeDataLoadFailed            Code = "DATA_LOAD_FAILED"
	CodeRowAddFailed              Code = "ROW_ADD_FAILED"
	CodeCellUpdateFailed          Code = "CELL_UPDATE_FAILED"
	CodeRowDeleteFailed           Code = "ROW_DELETE_FAILED"
	CodeFilterApplyFailed         Code = "FILTER_APPLY_FAILED"
	CodeSortApplyFailed           Code = "SORT_APPLY_FAILED"
	CodeScenarioStoreFailed       Code = "SCENARIO_STORE_FAILED"
	CodeScenarioNotFound          Code = "SCENARIO_NOT_FOUND"
	CodePrimaryKeyRequired        Code = "PRIMARY_KEY_REQUIRED"
	CodeOperationTimeout          Code = "OPERATION_TIMEOUT"
	CodeOperationCanceled         Code = "OPERATION_CANCELED"
	CodeUnexpected                Code = "UNEXPECTED"
)

var defaultMessages = map[Code]message{ // #nosec G101 -- 利用者向けエラーメッセージであり資格情報ではない。
	CodeConfigBroken:              "設定ファイルが壊れています",
	CodeConfigNotFound:            "設定ファイルが見つかりません",
	CodeConfigReadFailed:          "設定ファイルの読み込みに失敗しました",
	CodeConfigWriteFailed:         "設定ファイルの保存に失敗しました",
	CodeSecureStoreFailed:         "資格情報ストアへのアクセスに失敗しました",
	CodeDBConnectFailed:           "DB 接続に失敗しました",
	CodeValidationFailed:          "入力内容が正しくありません",
	CodeProfileNotFound:           "接続プロファイルが見つかりません",
	CodeCredentialUnavailable:     "既存の資格情報を利用できません",
	CodeConnectionFailed:          "接続確認に失敗しました",
	CodeCredentialSaveFailed:      "資格情報の保存に失敗しました",
	CodeCredentialDeleteFailed:    "資格情報の削除に失敗しました",
	CodeConfigSaveFailed:          "接続プロファイルの保存に失敗しました",
	CodeConsistencyRecoveryFailed: "保存状態の復旧に失敗しました。再試行してください",
	CodeSchemaLoadFailed:          "スキーマ取得に失敗しました",
	CodeStatsLoadFailed:           "統計取得に失敗しました",
	CodeDataLoadFailed:            "データ取得に失敗しました",
	CodeRowAddFailed:              "行追加に失敗しました",
	CodeCellUpdateFailed:          "セル編集に失敗しました",
	CodeRowDeleteFailed:           "行削除に失敗しました",
	CodeFilterApplyFailed:         "フィルタ適用に失敗しました",
	CodeSortApplyFailed:           "並び替えに失敗しました",
	CodeScenarioStoreFailed:       "検証シナリオの読み込みに失敗しました",
	CodeScenarioNotFound:          "検証シナリオが見つかりません",
	CodePrimaryKeyRequired:        "主対象には一意な生成規則が必要です",
	CodeOperationTimeout:          "処理がタイムアウトしました",
	CodeOperationCanceled:         "処理がキャンセルされました",
	CodeUnexpected:                "予期しないエラーが発生しました",
}

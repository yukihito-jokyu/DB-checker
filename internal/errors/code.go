package apperr

type Code string
type message string

const (
	CodeConfigBroken      Code = "CONFIG_BROKEN"
	CodeDBConnectFailed   Code = "DB_CONNECT_FAILED"
	CodeSchemaLoadFailed  Code = "SCHEMA_LOAD_FAILED"
	CodeStatsLoadFailed   Code = "STATS_LOAD_FAILED"
	CodeDataLoadFailed    Code = "DATA_LOAD_FAILED"
	CodeRowAddFailed      Code = "ROW_ADD_FAILED"
	CodeCellUpdateFailed  Code = "CELL_UPDATE_FAILED"
	CodeRowDeleteFailed   Code = "ROW_DELETE_FAILED"
	CodeFilterApplyFailed Code = "FILTER_APPLY_FAILED"
	CodeSortApplyFailed   Code = "SORT_APPLY_FAILED"
	CodeOperationTimeout  Code = "OPERATION_TIMEOUT"
	CodeOperationCanceled Code = "OPERATION_CANCELED"
	CodeUnexpected        Code = "UNEXPECTED"
)

var defaultMessages = map[Code]message{
	CodeConfigBroken:      "設定ファイルが壊れています",
	CodeDBConnectFailed:   "DB 接続に失敗しました",
	CodeSchemaLoadFailed:  "スキーマ取得に失敗しました",
	CodeStatsLoadFailed:   "統計取得に失敗しました",
	CodeDataLoadFailed:    "データ取得に失敗しました",
	CodeRowAddFailed:      "行追加に失敗しました",
	CodeCellUpdateFailed:  "セル編集に失敗しました",
	CodeRowDeleteFailed:   "行削除に失敗しました",
	CodeFilterApplyFailed: "フィルタ適用に失敗しました",
	CodeSortApplyFailed:   "並び替えに失敗しました",
	CodeOperationTimeout:  "処理がタイムアウトしました",
	CodeOperationCanceled: "処理がキャンセルされました",
	CodeUnexpected:        "予期しないエラーが発生しました",
}

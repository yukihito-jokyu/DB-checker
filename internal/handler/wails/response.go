package wails

import (
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type StatusResponse struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Version string `json:"version"`
}

type ConfigResponse struct {
	Version                   int               `json:"version"`
	ConnectionProfiles        []ProfileResponse `json:"connectionProfiles"`
	ActiveConnectionProfileID *string           `json:"activeConnectionProfileId"`
	FlowStates                map[string]any    `json:"flowStates"`
}

type ProfileResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DBType   string `json:"dbType"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Schema   string `json:"schema"`
	User     string `json:"user"`
}

type ProfileCheckResponse struct {
	Valid        bool `json:"valid"`
	ProfileCount int  `json:"profileCount"`
}

type ConnectionProfilesResponse struct {
	Profiles                  []ProfileResponse `json:"profiles"`
	ActiveConnectionProfileID *string           `json:"activeConnectionProfileId"`
}

type VerificationScenarioSummaryResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	PrimaryTable string `json:"primaryTable"`
	UpdatedAt    string `json:"updatedAt"`
}

type VerificationScenarioResponse struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	PrimaryTable  string         `json:"primaryTable"`
	Definition    map[string]any `json:"definition"`
	WorkspaceName *string        `json:"workspaceName"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	LatestRun     any            `json:"latestRun"`
}

type CreateVerificationScenarioRequest struct {
	Name         string         `json:"name"`
	PrimaryTable string         `json:"primaryTable"`
	Definition   map[string]any `json:"definition"`
}

type UpdateVerificationScenarioRequest struct {
	ScenarioID   string         `json:"scenarioId"`
	Name         string         `json:"name"`
	PrimaryTable string         `json:"primaryTable"`
	Definition   map[string]any `json:"definition"`
}

type DeleteVerificationScenarioRequest struct {
	ScenarioID      string `json:"scenarioId"`
	RemoveWorkspace bool   `json:"removeWorkspace"`
}

type DeleteScenarioResponse struct {
	ScenarioID       string `json:"scenarioId"`
	WorkspaceRemoved bool   `json:"workspaceRemoved"`
}

type PreviewVerificationRunRequest struct {
	ScenarioID string                            `json:"scenarioId"`
	Draft      *VerificationScenarioDraftRequest `json:"draft"`
}

type VerificationScenarioDraftRequest struct {
	Name         string         `json:"name"`
	PrimaryTable string         `json:"primaryTable"`
	Definition   map[string]any `json:"definition"`
}

type VerificationRunPreviewTableResponse struct {
	Name               string   `json:"name"`
	RowCount           int      `json:"rowCount"`
	AutomaticallyAdded bool     `json:"automaticallyAdded"`
	GeneratedColumns   []string `json:"generatedColumns"`
}

type VerificationRunPreviewResponse struct {
	Ready       bool                                  `json:"ready"`
	InsertOrder []VerificationRunPreviewTableResponse `json:"insertOrder"`
	DeleteOrder []VerificationRunPreviewTableResponse `json:"deleteOrder"`
	Warnings    []string                              `json:"warnings"`
}

type VerificationWorkspaceResponse struct {
	ScenarioID    string `json:"scenarioId"`
	WorkspaceName string `json:"workspaceName"`
	Mode          string `json:"mode"`
}

type PrepareVerificationRunRequest struct {
	ScenarioID string `json:"scenarioId"`
	RunID      string `json:"runId"`
}

type UpdateVerificationRunStateRequest struct {
	RunID string `json:"runId"`
	State string `json:"state"`
}

type SaveConnectionProfileRequest struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	DBType   string  `json:"dbType"`
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Database string  `json:"database"`
	Schema   *string `json:"schema,omitempty"`
	User     string  `json:"user"`
	Password string  `json:"password"`
}

type ActiveProfileResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DBType   string `json:"dbType"`
	Database string `json:"database"`
	Schema   string `json:"schema"`
}

type DatabaseColumnResponse struct {
	Name         string `json:"name"`
	DataType     string `json:"dataType"`
	Nullable     bool   `json:"nullable"`
	IsPrimaryKey bool   `json:"isPrimaryKey"`
	IsForeignKey bool   `json:"isForeignKey"`
	IsUnique     bool   `json:"isUnique"`
}

type TableStructureColumnResponse struct {
	Name         string  `json:"name"`
	DataType     string  `json:"dataType"`
	Nullable     bool    `json:"nullable"`
	DefaultValue *string `json:"defaultValue"`
	IsPrimaryKey bool    `json:"isPrimaryKey"`
	IsForeignKey bool    `json:"isForeignKey"`
	IsUnique     bool    `json:"isUnique"`
	IsGenerated  bool    `json:"isGenerated"`
}

type TableStructureTableResponse struct {
	Namespace string                         `json:"namespace"`
	Name      string                         `json:"name"`
	Columns   []TableStructureColumnResponse `json:"columns"`
}

type TableStructureIndexResponse struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Kind    string   `json:"kind"`
}

type TableStructureResponse struct {
	Table       TableStructureTableResponse   `json:"table"`
	ForeignKeys []DatabaseForeignKeyResponse  `json:"foreignKeys"`
	Indexes     []TableStructureIndexResponse `json:"indexes"`
}

type StatisticCountResponse struct {
	Value  *int64  `json:"value"`
	Status string  `json:"status"`
	Reason *string `json:"reason"`
}

type StatisticValueResponse struct {
	Value  *string `json:"value"`
	Status string  `json:"status"`
	Reason *string `json:"reason"`
}

type ColumnStatisticsResponse struct {
	Name           string                 `json:"name"`
	NullCount      StatisticCountResponse `json:"nullCount"`
	DistinctCount  StatisticCountResponse `json:"distinctCount"`
	DuplicateCount StatisticCountResponse `json:"duplicateCount"`
	Min            StatisticValueResponse `json:"min"`
	Max            StatisticValueResponse `json:"max"`
}

type ForeignKeyStatisticsResponse struct {
	Name                  string                 `json:"name"`
	FromColumns           []string               `json:"fromColumns"`
	ToTable               string                 `json:"toTable"`
	ToColumns             []string               `json:"toColumns"`
	SourceRowCount        StatisticCountResponse `json:"sourceRowCount"`
	NullCount             StatisticCountResponse `json:"nullCount"`
	ReferencedRowCount    StatisticCountResponse `json:"referencedRowCount"`
	MissingReferenceCount StatisticCountResponse `json:"missingReferenceCount"`
}

type TableStatisticsResponse struct {
	Table       string                         `json:"table"`
	RowCount    StatisticCountResponse         `json:"rowCount"`
	ColumnCount int                            `json:"columnCount"`
	CollectedAt *string                        `json:"collectedAt"`
	Status      string                         `json:"status"`
	Columns     []ColumnStatisticsResponse     `json:"columns"`
	ForeignKeys []ForeignKeyStatisticsResponse `json:"foreignKeys"`
}

type TableSortRequest struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

type TableFilterRequest struct {
	Column   string   `json:"column"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type FilterGroupRequest struct {
	Operator string               `json:"operator"`
	Filters  []TableFilterRequest `json:"filters"`
	Groups   []FilterGroupRequest `json:"groups"`
}

type ListTableRowsRequest struct {
	Table  string              `json:"table"`
	Page   int                 `json:"page"`
	Sort   *TableSortRequest   `json:"sort"`
	Filter *FilterGroupRequest `json:"filter"`
}

type ColumnValueInputRequest struct {
	Column string  `json:"column"`
	Kind   string  `json:"kind"`
	Value  *string `json:"value,omitempty"`
}

type InsertTableRowRequest struct {
	Table  string                    `json:"table"`
	Values []ColumnValueInputRequest `json:"values"`
}

type UpdateTableCellRequest struct {
	Table   string                    `json:"table"`
	Locator []ColumnValueInputRequest `json:"locator"`
	Column  string                    `json:"column"`
	Value   TableCellResponse         `json:"value"`
}

type DeleteTableRowRequest struct {
	Table   string                    `json:"table"`
	Locator []ColumnValueInputRequest `json:"locator"`
}

type AffectedRowsResponse struct {
	AffectedRows int64 `json:"affectedRows"`
}

type TableCellResponse struct {
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
}

type TableRowResponse struct {
	Cells []TableCellResponse `json:"cells"`
}

type TableRowsResponse struct {
	Rows       []TableRowResponse  `json:"rows"`
	TotalCount int64               `json:"totalCount"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
	Sort       *TableSortRequest   `json:"sort"`
	Filter     *FilterGroupRequest `json:"filter"`
}

type DatabaseTableResponse struct {
	Namespace string                   `json:"namespace"`
	Name      string                   `json:"name"`
	Columns   []DatabaseColumnResponse `json:"columns"`
}

type DatabaseForeignKeyResponse struct {
	Name        string   `json:"name"`
	FromTable   string   `json:"fromTable"`
	FromColumns []string `json:"fromColumns"`
	ToTable     string   `json:"toTable"`
	ToColumns   []string `json:"toColumns"`
}

type DatabaseSchemaResponse struct {
	ActiveProfile ActiveProfileResponse        `json:"activeProfile"`
	Tables        []DatabaseTableResponse      `json:"tables"`
	ForeignKeys   []DatabaseForeignKeyResponse `json:"foreignKeys"`
}

type TableFlowStateResponse struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Expanded bool    `json:"expanded"`
}

type FlowStateResponse struct {
	Version     int                               `json:"version"`
	TableStates map[string]TableFlowStateResponse `json:"tableStates"`
}

type SaveFlowStateRequest struct {
	Version     int                              `json:"version"`
	TableStates map[string]TableFlowStateRequest `json:"tableStates"`
}

type TableFlowStateRequest struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Expanded bool    `json:"expanded"`
}

type Response[T any] struct {
	Data  *T             `json:"data"`
	Error *ErrorResponse `json:"error"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// プロファイルレスポンス変換
func toProfileResponses(profiles []domain.Profile) []ProfileResponse {
	responses := make([]ProfileResponse, 0, len(profiles))
	for _, profile := range profiles {
		responses = append(responses, ProfileResponse{
			ID:       profile.ID,
			Name:     profile.Name,
			DBType:   string(profile.DBType),
			Host:     profile.Host,
			Port:     profile.Port,
			Database: profile.Database,
			Schema:   profile.Schema,
			User:     profile.User,
		})
	}

	return responses
}

// 成功レスポンス生成
func OK[T any](data T) Response[T] {
	return Response[T]{
		Data:  &data,
		Error: nil,
	}
}

// 失敗レスポンス生成
func Fail[T any](err error) Response[T] {
	return Response[T]{
		Data:  nil,
		Error: ToErrorResponse(err),
	}
}

// エラーレスポンス変換
func ToErrorResponse(err error) *ErrorResponse {
	if appErr := apperr.As(err); appErr != nil {
		return &ErrorResponse{
			Code:    string(appErr.Code),
			Message: string(appErr.Message),
		}
	}

	// 未分類エラーの詳細は frontend に漏らさず、共通の想定外エラーへ正規化する。
	appErr := apperr.NewUnexpected(err)

	return &ErrorResponse{
		Code:    string(appErr.Code),
		Message: string(appErr.Message),
	}
}

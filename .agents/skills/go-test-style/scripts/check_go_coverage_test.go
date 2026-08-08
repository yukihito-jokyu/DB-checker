package main

import (
	"bytes"
	"go/ast"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// カバレッジ検査CLI検証
func TestRunCoverageCheck(t *testing.T) {
	root := t.TempDir()
	sourcePath, profilePath := writeCoverageFixture(t, root)
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
		wantError  string
	}{
		{
			name:      "引数不足を拒否する",
			wantCode:  2,
			wantError: "使用方法",
		},
		{
			name: "不正オプションを拒否する",
			args: []string{
				"-unknown",
			},
			wantCode:  2,
			wantError: "flag provided but not defined",
		},
		{
			name: "profile読込失敗を返す",
			args: []string{
				"-profile",
				filepath.Join(root, "missing.out"),
				sourcePath + ":covered:100",
			},
			wantCode:  1,
			wantError: "coverage profileを開けません",
		},
		{
			name: "対象形式不正を返す",
			args: []string{
				"-profile",
				profilePath,
				"invalid",
			},
			wantCode:  1,
			wantError: "coverage対象形式が不正です",
		},
		{
			name: "blockなしを返す",
			args: []string{
				"-profile",
				profilePath,
				sourcePath + ":withoutBlock:100",
			},
			wantCode:  1,
			wantError: "coverage blockがありません",
		},
		{
			name: "未達を返す",
			args: []string{
				"-profile",
				profilePath,
				sourcePath + ":uncovered:100",
			},
			wantCode:   1,
			wantOutput: "raw=0.0%",
			wantError:  "カバレッジが目標未達です",
		},
		{
			name: "実行済み関数を許可する",
			args: []string{
				"-profile",
				profilePath,
				sourcePath + ":covered:100",
			},
			wantCode:   0,
			wantOutput: "raw=100.0% effective=100.0%",
		},
		{
			name: "理由付き到達不能を許可する",
			args: []string{
				"-profile",
				profilePath,
				sourcePath + ":annotated:100",
			},
			wantCode:   0,
			wantOutput: "raw=0.0% effective=100.0%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if got := runCoverageCheck(tt.args, &stdout, &stderr); got != tt.wantCode {
				t.Errorf("runCoverageCheck() = %d, want %d", got, tt.wantCode)
			}
			if !strings.Contains(stdout.String(), tt.wantOutput) {
				t.Errorf("stdout = %q, want contains %q", stdout.String(), tt.wantOutput)
			}
			if !strings.Contains(stderr.String(), tt.wantError) {
				t.Errorf("stderr = %q, want contains %q", stderr.String(), tt.wantError)
			}
		})
	}
}

// coverage profile読込検証
func TestLoadCoverageProfile(t *testing.T) {
	root := t.TempDir()
	validPath := filepath.Join(root, "valid.out")
	writeCoverageFile(t, validPath, "mode: set\nfixture.go:1.1,1.2 1 1\n")
	got, err := loadCoverageProfile(validPath)
	if err != nil {
		t.Fatalf("loadCoverageProfile() error = %v", err)
	}
	if len(got) != 1 {
		t.Errorf("loadCoverageProfile() length = %d, want 1", len(got))
	}

	invalidPath := filepath.Join(root, "invalid.out")
	writeCoverageFile(t, invalidPath, "invalid\n")
	if _, err := loadCoverageProfile(invalidPath); err == nil {
		t.Error("loadCoverageProfile() error = nil, want parse error")
	}

	longPath := filepath.Join(root, "long.out")
	writeCoverageFile(t, longPath, strings.Repeat("x", 70_000))
	if _, err := loadCoverageProfile(longPath); err == nil {
		t.Error("loadCoverageProfile() error = nil, want scanner error")
	}
}

// coverage block解析検証
func TestParseCoverageBlock(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr string
	}{
		{
			name:    "フィールド不足を拒否する",
			line:    "invalid",
			wantErr: "profile形式が不正",
		},
		{
			name:    "範囲不足を拒否する",
			line:    "fixture.go 1 1",
			wantErr: "coverage範囲が不正",
		},
		{
			name:    "開始位置不正を拒否する",
			line:    "fixture.go:x.1,2.1 1 1",
			wantErr: "coverage行番号が不正",
		},
		{
			name:    "終了位置不正を拒否する",
			line:    "fixture.go:1.1,x.1 1 1",
			wantErr: "coverage行番号が不正",
		},
		{
			name:    "statement数不正を拒否する",
			line:    "fixture.go:1.1,2.1 x 1",
			wantErr: "statement数が不正",
		},
		{
			name:    "実行回数不正を拒否する",
			line:    "fixture.go:1.1,2.1 1 x",
			wantErr: "実行回数が不正",
		},
		{
			name: "正常範囲を解析する",
			line: "fixture.go:1.1,2.1 1 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCoverageBlock(tt.line)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("parseCoverageBlock() error = %v, want contains %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCoverageBlock() error = %v", err)
			}
			if got.startLine != 1 || got.endLine != 2 || got.statements != 1 || got.count != 1 {
				t.Errorf("parseCoverageBlock() = %+v, want parsed block", got)
			}
		})
	}
}

// coverage位置行解析検証
func TestParsePositionLine(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{
			name:    "ドットなしを拒否する",
			value:   "1",
			wantErr: true,
		},
		{
			name:    "行番号不正を拒否する",
			value:   "x.1",
			wantErr: true,
		},
		{
			name:  "行番号を返す",
			value: "2.1",
			want:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePositionLine(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePositionLine() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parsePositionLine() = %d, want %d", got, tt.want)
			}
		})
	}
}

// coverage対象解析検証
func TestParseCoverageTarget(t *testing.T) {
	root := t.TempDir()
	validSource, _ := writeCoverageFixture(t, root)
	invalidSource := filepath.Join(root, "invalid.go")
	writeCoverageFile(t, invalidSource, "package fixture\nfunc {\n")
	duplicateSource := filepath.Join(root, "duplicate.go")
	writeCoverageFile(t, duplicateSource, "package fixture\nfunc duplicate() {}\nfunc duplicate() {}\n")
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name:    "区切りなしを拒否する",
			raw:     "invalid",
			wantErr: "対象形式が不正",
		},
		{
			name:    "区切り不足を拒否する",
			raw:     "invalid:100",
			wantErr: "対象形式が不正",
		},
		{
			name:    "閾値文字列を拒否する",
			raw:     validSource + ":covered:x",
			wantErr: "coverage目標が不正",
		},
		{
			name:    "閾値範囲外を拒否する",
			raw:     validSource + ":covered:101",
			wantErr: "coverage目標が不正",
		},
		{
			name:    "存在しないソースを拒否する",
			raw:     filepath.Join(root, "missing.go") + ":covered:100",
			wantErr: "対象ソースを読めません",
		},
		{
			name:    "構文不正ソースを拒否する",
			raw:     invalidSource + ":covered:100",
			wantErr: "対象ソースを解析できません",
		},
		{
			name:    "対象なしを拒否する",
			raw:     validSource + ":missing:100",
			wantErr: "一意に特定できません",
		},
		{
			name:    "対象重複を拒否する",
			raw:     duplicateSource + ":duplicate:100",
			wantErr: "一意に特定できません",
		},
		{
			name: "正常対象を返す",
			raw:  validSource + ":covered:100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCoverageTarget(tt.raw)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("parseCoverageTarget() error = %v, want contains %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCoverageTarget() error = %v", err)
			}
			if got.function != "covered" || got.minimum != 100 {
				t.Errorf("parseCoverageTarget() = %+v, want covered 100", got)
			}
		})
	}
}

// receiver名検証
func TestCoverageFunctionAndReceiverName(t *testing.T) {
	tests := []struct {
		name       string
		expression ast.Expr
		want       string
	}{
		{
			name: "識別子を返す",
			expression: &ast.Ident{
				Name: "Receiver",
			},
			want: "Receiver",
		},
		{
			name: "ポインターを返す",
			expression: &ast.StarExpr{
				X: &ast.Ident{
					Name: "Receiver",
				},
			},
			want: "Receiver",
		},
		{
			name: "単一型引数を返す",
			expression: &ast.IndexExpr{
				X: &ast.Ident{
					Name: "Receiver",
				},
			},
			want: "Receiver",
		},
		{
			name: "複数型引数を返す",
			expression: &ast.IndexListExpr{
				X: &ast.Ident{
					Name: "Receiver",
				},
			},
			want: "Receiver",
		},
		{
			name: "未知式を返す",
			expression: &ast.ArrayType{
				Elt: &ast.Ident{
					Name: "Receiver",
				},
			},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := receiverName(tt.expression); got != tt.want {
				t.Errorf("receiverName() = %q, want %q", got, tt.want)
			}
		})
	}

	freeFunction := &ast.FuncDecl{
		Name: &ast.Ident{
			Name: "Free",
		},
	}
	if got := coverageFunctionName(freeFunction); got != "Free" {
		t.Errorf("coverageFunctionName() = %q, want %q", got, "Free")
	}
	method := &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Type: &ast.Ident{
						Name: "Receiver",
					},
				},
			},
		},
		Name: &ast.Ident{
			Name: "Method",
		},
	}
	if got := coverageFunctionName(method); got != "Receiver.Method" {
		t.Errorf("coverageFunctionName() = %q, want %q", got, "Receiver.Method")
	}
}

// 関数coverage算出検証
func TestCalculateCoverage(t *testing.T) {
	target := coverageTarget{
		file:      "fixture.go",
		function:  "covered",
		startLine: 1,
		endLine:   10,
		sourceLines: []string{
			"func covered() {",
			"",
			"// 単体テスト到達不可: OSが保証する分岐のため",
			"return",
			"}",
		},
	}
	blocks := []coverageBlock{
		{
			file:       "other.go",
			startLine:  1,
			endLine:    1,
			statements: 1,
			count:      1,
		},
		{
			file:       "fixture.go",
			startLine:  2,
			endLine:    2,
			statements: 1,
			count:      1,
		},
		{
			file:       "fixture.go",
			startLine:  4,
			endLine:    4,
			statements: 1,
		},
		{
			file:       "fixture.go",
			startLine:  5,
			endLine:    5,
			statements: 1,
		},
	}
	var stderr bytes.Buffer
	raw, effective, excluded, err := calculateCoverage(target, blocks, &stderr)
	if err != nil {
		t.Fatalf("calculateCoverage() error = %v", err)
	}
	if math.Abs(raw-100.0/3.0) > 0.0001 || math.Abs(effective-200.0/3.0) > 0.0001 || excluded != 1 {
		t.Errorf("calculateCoverage() = %.1f, %.1f, %d, want %.1f, %.1f, 1", raw, effective, excluded, 100.0/3.0, 200.0/3.0)
	}
	if !strings.Contains(stderr.String(), "未到達行: [5]") {
		t.Errorf("stderr = %q, want uncovered line", stderr.String())
	}

	if _, _, _, err := calculateCoverage(target, nil, &stderr); err == nil {
		t.Error("calculateCoverage() error = nil, want no block error")
	}
}

// coverageファイル一致判定検証
func TestSameCoverageFile(t *testing.T) {
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{
			name: "完全一致する",
			got:  sameCoverageFile("internal/file.go", "internal/file.go"),
			want: true,
		},
		{
			name: "module接頭辞付きで一致する",
			got:  sameCoverageFile("example.com/project/internal/file.go", "internal/file.go"),
			want: true,
		},
		{
			name: "別ファイルは一致しない",
			got:  sameCoverageFile("internal/other.go", "internal/file.go"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("sameCoverageFile() = %v, want %v", tt.got, tt.want)
			}
		})
	}
}

// 到達不能コメント判定検証
func TestHasUnreachableComment(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		startLine int
		endLine   int
		want      bool
	}{
		{
			name: "理由付きコメントを許可する",
			lines: []string{
				"func target() {",
				"",
				"// 単体テスト到達不可: OSが保証する分岐のため",
				"return",
			},
			startLine: 4,
			endLine:   4,
			want:      true,
		},
		{
			name: "空行越しの理由付きコメントを許可する",
			lines: []string{
				"func target() {",
				"// 単体テスト到達不可: OSが保証する分岐のため",
				"",
				"return",
			},
			startLine: 4,
			endLine:   4,
			want:      true,
		},
		{
			name: "coverageブロック内の理由付きコメントを許可する",
			lines: []string{
				"func target() {",
				"// 単体テスト到達不可: os.Exitが終了するため",
				"os.Exit(1)",
				"}",
			},
			startLine: 1,
			endLine:   4,
			want:      true,
		},
		{
			name: "理由なしコメントを拒否する",
			lines: []string{
				"func target() {",
				"// 単体テスト到達不可:",
				"return",
			},
			startLine: 3,
			endLine:   3,
		},
		{
			name: "通常コメントを拒否する",
			lines: []string{
				"func target() {",
				"// 通常コメント",
				"return",
			},
			startLine: 3,
			endLine:   3,
		},
		{
			name: "範囲外を拒否する",
			lines: []string{
				"return",
			},
			startLine: 1,
			endLine:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := coverageTarget{
				startLine:   1,
				sourceLines: tt.lines,
			}
			if got := hasUnreachableComment(target, tt.startLine, tt.endLine); got != tt.want {
				t.Errorf("hasUnreachableComment() = %v, want %v", got, tt.want)
			}
		})
	}
}

// coverage fixture作成
func writeCoverageFixture(t *testing.T, root string) (string, string) {
	t.Helper()
	sourcePath := filepath.Join(root, "fixture.go")
	source := `package fixture

func covered() int {
	return 1
}

func annotated() int {
	// 単体テスト到達不可: fixtureで理由付き除外を検証するため
	return 0
}

func uncovered() int {
	return 0
}

func withoutBlock() int {
	return 0
}
`
	writeCoverageFile(t, sourcePath, source)
	profilePath := filepath.Join(root, "coverage.out")
	profile := "mode: set\n" +
		sourcePath + ":4.2,4.10 1 1\n" +
		sourcePath + ":9.2,9.10 1 0\n" +
		sourcePath + ":13.2,13.10 1 0\n"
	writeCoverageFile(t, profilePath, profile)

	return sourcePath, profilePath
}

// coverage fixture書込
func writeCoverageFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

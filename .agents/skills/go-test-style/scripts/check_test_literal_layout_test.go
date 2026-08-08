package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 改行検査CLI検証
func TestRunLayoutCheck(t *testing.T) {
	root := t.TempDir()
	validDir := filepath.Join(root, "valid")
	invalidDir := filepath.Join(root, "invalid")
	brokenDir := filepath.Join(root, "broken")
	writeLayoutFixture(t, filepath.Join(validDir, "valid_test.go"), `package fixture

func external()

func TestValid() {
	_ = []int{
		1,
		2,
	}
}
`)
	writeLayoutFixture(t, filepath.Join(invalidDir, "invalid_test.go"), `package fixture

func TestInvalid() {
	_ = []int{1, 2}
}
`)
	writeLayoutFixture(t, filepath.Join(brokenDir, "broken_test.go"), "package fixture\nfunc {\n")
	writeLayoutFixture(t, filepath.Join(validDir, ".git", "ignored_test.go"), "package ignored\nfunc TestIgnored() { _ = []int{1, 2} }\n")

	emptyBaseline := filepath.Join(root, "empty.baseline")
	writeLayoutFixture(t, emptyBaseline, "# signature\tcount\tfile\tfunction\n")
	invalidBaseline := filepath.Join(root, "invalid.baseline")
	writeLayoutFixture(t, invalidBaseline, "invalid\n")
	blockedParent := filepath.Join(root, "blocked")
	writeLayoutFixture(t, blockedParent, "file")
	t.Chdir(validDir)

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
		wantError  string
	}{
		{
			name:      "baseline指定なしを拒否する",
			wantCode:  2,
			wantError: "-baseline を指定してください",
		},
		{
			name: "既定のカレントディレクトリを検査する",
			args: []string{
				"-baseline",
				emptyBaseline,
			},
			wantCode:   0,
			wantOutput: "改行検査に成功しました",
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
			name: "存在しない対象を拒否する",
			args: []string{
				"-baseline",
				emptyBaseline,
				filepath.Join(root, "missing"),
			},
			wantCode:  1,
			wantError: "検査対象を確認できません",
		},
		{
			name: "構文エラーを拒否する",
			args: []string{
				"-baseline",
				emptyBaseline,
				brokenDir,
			},
			wantCode:  1,
			wantError: "Goテストを解析できません",
		},
		{
			name: "baselineを生成する",
			args: []string{
				"-baseline",
				filepath.Join(root, "generated", "layout.baseline"),
				"-write-baseline",
				invalidDir,
			},
			wantCode:   0,
			wantOutput: "baselineを更新しました: 1件",
		},
		{
			name: "baseline保存失敗を返す",
			args: []string{
				"-baseline",
				filepath.Join(blockedParent, "layout.baseline"),
				"-write-baseline",
				validDir,
			},
			wantCode:  1,
			wantError: "baselineディレクトリを作成できません",
		},
		{
			name: "baseline読込失敗を返す",
			args: []string{
				"-baseline",
				filepath.Join(root, "missing.baseline"),
				validDir,
			},
			wantCode:  1,
			wantError: "baselineを読み込めません",
		},
		{
			name: "baseline形式不正を返す",
			args: []string{
				"-baseline",
				invalidBaseline,
				validDir,
			},
			wantCode:  1,
			wantError: "baseline形式が不正です",
		},
		{
			name: "改行違反を返す",
			args: []string{
				"-baseline",
				emptyBaseline,
				invalidDir,
			},
			wantCode:  1,
			wantError: "複合リテラルの複数フィールドまたは要素を改行してください",
		},
		{
			name: "違反なしを返す",
			args: []string{
				"-baseline",
				emptyBaseline,
				validDir,
			},
			wantCode:   0,
			wantOutput: "改行検査に成功しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if got := runLayoutCheck(tt.args, &stdout, &stderr); got != tt.wantCode {
				t.Errorf("runLayoutCheck() = %d, want %d", got, tt.wantCode)
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

// 違反並び順検証
func TestCollectViolationsOrder(t *testing.T) {
	root := t.TempDir()
	aPath := filepath.Join(root, "a_test.go")
	bPath := filepath.Join(root, "b_test.go")
	writeLayoutFixture(t, aPath, `package fixture
func TestA() {
	_ = [][]int{{1, 2}, {3, 4}}
	_ = []int{5, 6}
}
`)
	writeLayoutFixture(t, bPath, `package fixture
func TestB() {
	_ = []int{1, 2}
}
`)

	got, err := collectViolations([]string{
		bPath,
		aPath,
	})
	if err != nil {
		t.Fatalf("collectViolations() error = %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("collectViolations() length = %d, want 5", len(got))
	}
	for index := 1; index < len(got); index++ {
		previous := got[index-1]
		current := got[index]
		if previous.file > current.file ||
			(previous.file == current.file && previous.line > current.line) ||
			(previous.file == current.file && previous.line == current.line && previous.column > current.column) {
			t.Errorf("collectViolations() order = %#v before %#v", previous, current)
		}
	}
}

// ディレクトリ走査失敗検証
func TestCollectTestFilesWalkFailure(t *testing.T) {
	original := walkDirectory
	walkDirectory = func(root string, callback fs.WalkDirFunc) error {
		return callback(root, nil, errors.New("walk failed"))
	}
	t.Cleanup(func() {
		walkDirectory = original
	})

	if _, err := collectTestFiles([]string{t.TempDir()}); err == nil {
		t.Error("collectTestFiles() error = nil, want walk error")
	}
}

// テストファイル収集検証
func TestCollectTestFiles(t *testing.T) {
	root := t.TempDir()
	testFile := filepath.Join(root, "fixture_test.go")
	nonTestFile := filepath.Join(root, "fixture.go")
	writeLayoutFixture(t, testFile, "package fixture\n")
	writeLayoutFixture(t, nonTestFile, "package fixture\n")

	got, err := collectTestFiles([]string{
		testFile,
		nonTestFile,
		testFile,
	})
	if err != nil {
		t.Fatalf("collectTestFiles() error = %v", err)
	}
	if len(got) != 1 || got[0] != testFile {
		t.Errorf("collectTestFiles() = %v, want [%s]", got, testFile)
	}
}

// 除外ディレクトリ判定検証
func TestIgnoredDirectory(t *testing.T) {
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{
			name: "Gitディレクトリを除外する",
			got:  ignoredDirectory(".git"),
			want: true,
		},
		{
			name: "通常ディレクトリを許可する",
			got:  ignoredDirectory("internal"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("ignoredDirectory() = %v, want %v", tt.got, tt.want)
			}
		})
	}
}

// AST検査検証
func TestInspectNode(t *testing.T) {
	source := `package fixture
func TestFixture() {
	_ = 1
	_ = []int{1}
	_ = []int{
		1,
		2,
	}
	_ = []int{1, 2}
}
`
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "fixture_test.go", source, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	function := parsed.Decls[0].(*ast.FuncDecl)

	got := inspectNode(fset, "fixture_test.go", "TestFixture", function.Body)
	if len(got) != 1 {
		t.Fatalf("inspectNode() length = %d, want 1", len(got))
	}
	if got[0].line != 9 {
		t.Errorf("line = %d, want 9", got[0].line)
	}
}

// 違反署名生成失敗検証
func TestLiteralSignatureFormatFailure(t *testing.T) {
	original := formatASTNode
	formatASTNode = func(io.Writer, *token.FileSet, any) error {
		return errors.New("format failed")
	}
	t.Cleanup(func() {
		formatASTNode = original
	})

	got := literalSignature(token.NewFileSet(), "fixture_test.go", "TestFixture", &ast.CompositeLit{})
	if got == "" {
		t.Error("literalSignature() = empty, want signature")
	}
}

// baseline入出力検証
func TestBaselineInputOutput(t *testing.T) {
	root := t.TempDir()
	baselinePath := filepath.Join(root, "layout.baseline")
	violations := []violation{
		{
			signature: "signature",
			file:      "fixture_test.go",
			function:  "TestFixture",
			line:      1,
			column:    1,
		},
		{
			signature: "signature",
			file:      "fixture_test.go",
			function:  "TestFixture",
			line:      2,
			column:    1,
		},
	}
	if err := saveBaseline(baselinePath, violations); err != nil {
		t.Fatalf("saveBaseline() error = %v", err)
	}
	got, err := loadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("loadBaseline() error = %v", err)
	}
	if got["signature"].count != 2 {
		t.Errorf("count = %d, want 2", got["signature"].count)
	}

	writeLayoutFixture(t, filepath.Join(root, "zero.baseline"), "signature\t0\tfile\tfunction\n")
	if _, err := loadBaseline(filepath.Join(root, "zero.baseline")); err == nil {
		t.Error("loadBaseline() error = nil, want invalid count error")
	}
	writeLayoutFixture(t, filepath.Join(root, "text.baseline"), "signature\ttext\tfile\tfunction\n")
	if _, err := loadBaseline(filepath.Join(root, "text.baseline")); err == nil {
		t.Error("loadBaseline() error = nil, want invalid count error")
	}

	if err := saveBaseline(root, nil); err == nil {
		t.Error("saveBaseline() error = nil, want write error")
	}
}

// baseline比較検証
func TestCompareBaseline(t *testing.T) {
	item := violation{
		signature: "signature",
		file:      "fixture_test.go",
		function:  "TestFixture",
		line:      1,
		column:    1,
	}
	baseline := map[string]baselineEntry{
		"signature": {
			count:    1,
			file:     "fixture_test.go",
			function: "TestFixture",
		},
	}
	if err := compareBaseline([]violation{item}, baseline); err != nil {
		t.Errorf("compareBaseline() error = %v, want nil", err)
	}
	if err := compareBaseline([]violation{
		item,
		item,
	}, baseline); err == nil {
		t.Error("compareBaseline() error = nil, want added violation error")
	}
	if err := compareBaseline(nil, baseline); err == nil {
		t.Error("compareBaseline() error = nil, want resolved baseline error")
	}
}

// fixture書込
func writeLayoutFixture(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type violation struct {
	signature string
	file      string
	function  string
	line      int
	column    int
}

type baselineEntry struct {
	count    int
	file     string
	function string
}

var formatASTNode = format.Node
var walkDirectory = filepath.WalkDir

// テスト複合リテラル改行検査
func main() {
	// 単体テスト到達不可: os.Exitを実行するとテストプロセス自体が終了するため
	os.Exit(runLayoutCheck(os.Args[1:], os.Stdout, os.Stderr))
}

// テスト複合リテラル改行検査実行
func runLayoutCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("check-test-literal-layout", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baselinePath := flags.String("baseline", "", "baselineファイル")
	writeBaseline := flags.Bool("write-baseline", false, "現在の違反をbaselineへ保存")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if *baselinePath == "" {
		fmt.Fprintln(stderr, "-baseline を指定してください")
		return 2
	}

	roots := flags.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	violations, err := collectViolations(roots)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if *writeBaseline {
		if err := saveBaseline(*baselinePath, violations); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}

		fmt.Fprintf(stdout, "テスト複合リテラル改行baselineを更新しました: %d件\n", len(violations))
		return 0
	}

	baseline, err := loadBaseline(*baselinePath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := compareBaseline(violations, baseline); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(stdout, "テスト複合リテラル改行検査に成功しました: baseline %d件\n", len(violations))
	return 0
}

// 違反収集
func collectViolations(roots []string) ([]violation, error) {
	files, err := collectTestFiles(roots)
	if err != nil {
		return nil, err
	}

	var violations []violation
	for _, path := range files {
		fileViolations, err := inspectTestFile(path)
		if err != nil {
			return nil, err
		}

		violations = append(violations, fileViolations...)
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].file != violations[j].file {
			return violations[i].file < violations[j].file
		}
		if violations[i].line != violations[j].line {
			return violations[i].line < violations[j].line
		}
		return violations[i].column < violations[j].column
	})

	return violations, nil
}

// テストファイル収集
func collectTestFiles(roots []string) ([]string, error) {
	fileSet := make(map[string]struct{})
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			return nil, fmt.Errorf("検査対象を確認できません: %s: %w", root, err)
		}

		if !info.IsDir() {
			if strings.HasSuffix(root, "_test.go") {
				fileSet[filepath.Clean(root)] = struct{}{}
			}
			continue
		}

		err = walkDirectory(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && path != root && ignoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
				fileSet[filepath.Clean(path)] = struct{}{}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("テストファイルを収集できません: %s: %w", root, err)
		}
	}

	files := make([]string, 0, len(fileSet))
	for path := range fileSet {
		files = append(files, path)
	}
	sort.Strings(files)

	return files, nil
}

// 除外ディレクトリ判定
func ignoredDirectory(name string) bool {
	switch name {
	case ".git", ".cache", ".local", "coverage", "node_modules", "vendor":
		return true
	default:
		return false
	}
}

// テストファイル検査
func inspectTestFile(path string) ([]violation, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("Goテストを解析できません: %s: %w", path, err)
	}

	relativePath, err := filepath.Rel(".", path)
	if err != nil {
		relativePath = filepath.Clean(path)
	}
	relativePath = filepath.ToSlash(relativePath)

	var violations []violation
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}

		violations = append(violations, inspectNode(fset, relativePath, function.Name.Name, function.Body)...)
	}

	return violations, nil
}

// ASTノード検査
func inspectNode(fset *token.FileSet, path string, function string, node ast.Node) []violation {
	var violations []violation
	ast.Inspect(node, func(current ast.Node) bool {
		literal, ok := current.(*ast.CompositeLit)
		if !ok || len(literal.Elts) < 2 {
			return true
		}

		lineCounts := make(map[int]int)
		for _, element := range literal.Elts {
			lineCounts[fset.Position(element.Pos()).Line]++
		}

		for line, count := range lineCounts {
			if count < 2 {
				continue
			}

			position := fset.Position(literal.Pos())
			violations = append(violations, violation{
				signature: literalSignature(fset, path, function, literal),
				file:      path,
				function:  function,
				line:      line,
				column:    position.Column,
			})
			break
		}

		return true
	})

	return violations
}

// 違反署名生成
func literalSignature(fset *token.FileSet, path string, function string, literal *ast.CompositeLit) string {
	var formatted bytes.Buffer
	if err := formatASTNode(&formatted, fset, literal); err != nil {
		formatted.WriteString("unformattable")
	}
	normalized := strings.Join(strings.Fields(formatted.String()), " ")
	sum := sha256.Sum256([]byte(path + "\n" + function + "\n" + normalized))

	return hex.EncodeToString(sum[:])
}

// baseline保存
func saveBaseline(path string, violations []violation) error {
	counts := make(map[string]int)
	metadata := make(map[string]violation)
	for _, item := range violations {
		counts[item.signature]++
		metadata[item.signature] = item
	}

	signatures := make([]string, 0, len(counts))
	for signature := range counts {
		signatures = append(signatures, signature)
	}
	sort.Strings(signatures)

	var output strings.Builder
	output.WriteString("# signature\tcount\tfile\tfunction\n")
	for _, signature := range signatures {
		item := metadata[signature]
		fmt.Fprintf(&output, "%s\t%d\t%s\t%s\n", signature, counts[signature], item.file, item.function)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("baselineディレクトリを作成できません: %w", err)
	}
	if err := os.WriteFile(path, []byte(output.String()), 0o644); err != nil {
		return fmt.Errorf("baselineを保存できません: %w", err)
	}

	return nil
}

// baseline読込
func loadBaseline(path string) (map[string]baselineEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("baselineを読み込めません: %s: %w", path, err)
	}

	baseline := make(map[string]baselineEntry)
	for lineNumber, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			return nil, fmt.Errorf("baseline形式が不正です: %s:%d", path, lineNumber+1)
		}
		count, err := strconv.Atoi(fields[1])
		if err != nil || count < 1 {
			return nil, fmt.Errorf("baseline件数が不正です: %s:%d", path, lineNumber+1)
		}
		baseline[fields[0]] = baselineEntry{
			count:    count,
			file:     fields[2],
			function: fields[3],
		}
	}

	return baseline, nil
}

// baseline比較
func compareBaseline(violations []violation, baseline map[string]baselineEntry) error {
	currentCounts := make(map[string]int)
	currentMetadata := make(map[string]violation)
	for _, item := range violations {
		currentCounts[item.signature]++
		currentMetadata[item.signature] = item
	}

	var messages []string
	for signature, count := range currentCounts {
		want, found := baseline[signature]
		if found && want.count == count {
			continue
		}

		item := currentMetadata[signature]
		messages = append(messages, fmt.Sprintf(
			"%s:%d:%d: 複合リテラルの複数フィールドまたは要素を改行してください（関数: %s、現在: %d、baseline: %d）",
			item.file,
			item.line,
			item.column,
			item.function,
			count,
			want.count,
		))
	}
	for signature, want := range baseline {
		if _, found := currentCounts[signature]; found {
			continue
		}

		messages = append(messages, fmt.Sprintf(
			"baselineの違反が解消されています: %s（関数: %s、件数: %d）。意図した修正ならbaselineを更新してください",
			want.file,
			want.function,
			want.count,
		))
	}

	if len(messages) == 0 {
		return nil
	}
	sort.Strings(messages)

	return fmt.Errorf("テスト複合リテラル改行検査に失敗しました:\n%s", strings.Join(messages, "\n"))
}

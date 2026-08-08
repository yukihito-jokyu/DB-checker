package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const unreachableCommentPrefix = "// 単体テスト到達不可:"

type coverageBlock struct {
	file       string
	startLine  int
	endLine    int
	statements int
	count      int
}

type coverageTarget struct {
	file        string
	function    string
	minimum     float64
	startLine   int
	endLine     int
	sourceLines []string
}

// 対象関数カバレッジ検査
func main() {
	// 単体テスト到達不可: os.Exitを実行するとテストプロセス自体が終了するため
	os.Exit(runCoverageCheck(os.Args[1:], os.Stdout, os.Stderr))
}

// 対象関数カバレッジ検査実行
func runCoverageCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("check-go-coverage", flag.ContinueOnError)
	flags.SetOutput(stderr)
	profilePath := flags.String("profile", "", "Go coverage profile")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if *profilePath == "" || len(flags.Args()) == 0 {
		fmt.Fprintln(stderr, "使用方法: check_go_coverage.go -profile <path> <file:function:minimum>...")
		return 2
	}

	blocks, err := loadCoverageProfile(*profilePath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	failed := false
	for _, rawTarget := range flags.Args() {
		target, err := parseCoverageTarget(rawTarget)
		if err != nil {
			fmt.Fprintln(stderr, err)
			failed = true
			continue
		}

		rawCoverage, effectiveCoverage, exclusions, err := calculateCoverage(target, blocks, stderr)
		if err != nil {
			fmt.Fprintln(stderr, err)
			failed = true
			continue
		}

		fmt.Fprintf(
			stdout,
			"%s:%s raw=%.1f%% effective=%.1f%% target=%.1f%% unreachable=%d\n",
			target.file,
			target.function,
			rawCoverage,
			effectiveCoverage,
			target.minimum,
			exclusions,
		)
		if effectiveCoverage+0.0001 < target.minimum {
			fmt.Fprintf(stderr, "%s:%s のカバレッジが目標未達です\n", target.file, target.function)
			failed = true
		}
	}

	if failed {
		return 1
	}

	return 0
}

// coverage profile読込
func loadCoverageProfile(path string) ([]coverageBlock, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("coverage profileを開けません: %s: %w", path, err)
	}
	defer file.Close()

	var blocks []coverageBlock
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		block, err := parseCoverageBlock(line)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("coverage profileを読み込めません: %w", err)
	}

	return blocks, nil
}

// coverage block解析
func parseCoverageBlock(line string) (coverageBlock, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return coverageBlock{}, fmt.Errorf("coverage profile形式が不正です: %s", line)
	}

	colon := strings.LastIndex(fields[0], ":")
	comma := strings.Index(fields[0][colon+1:], ",")
	if colon < 0 || comma < 0 {
		return coverageBlock{}, fmt.Errorf("coverage範囲が不正です: %s", line)
	}
	comma += colon + 1

	startLine, err := parsePositionLine(fields[0][colon+1 : comma])
	if err != nil {
		return coverageBlock{}, err
	}
	endLine, err := parsePositionLine(fields[0][comma+1:])
	if err != nil {
		return coverageBlock{}, err
	}
	statements, err := strconv.Atoi(fields[1])
	if err != nil {
		return coverageBlock{}, fmt.Errorf("statement数が不正です: %s", line)
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil {
		return coverageBlock{}, fmt.Errorf("実行回数が不正です: %s", line)
	}

	return coverageBlock{
		file:       filepath.ToSlash(fields[0][:colon]),
		startLine:  startLine,
		endLine:    endLine,
		statements: statements,
		count:      count,
	}, nil
}

// coverage位置行解析
func parsePositionLine(position string) (int, error) {
	dot := strings.Index(position, ".")
	if dot < 1 {
		return 0, fmt.Errorf("coverage位置が不正です: %s", position)
	}

	line, err := strconv.Atoi(position[:dot])
	if err != nil {
		return 0, fmt.Errorf("coverage行番号が不正です: %s", position)
	}

	return line, nil
}

// coverage対象解析
func parseCoverageTarget(raw string) (coverageTarget, error) {
	lastColon := strings.LastIndex(raw, ":")
	if lastColon < 0 {
		return coverageTarget{}, fmt.Errorf("coverage対象形式が不正です: %s", raw)
	}
	secondColon := strings.LastIndex(raw[:lastColon], ":")
	if secondColon < 0 {
		return coverageTarget{}, fmt.Errorf("coverage対象形式が不正です: %s", raw)
	}

	minimum, err := strconv.ParseFloat(raw[lastColon+1:], 64)
	if err != nil || minimum < 0 || minimum > 100 {
		return coverageTarget{}, fmt.Errorf("coverage目標が不正です: %s", raw)
	}

	target := coverageTarget{
		file:     filepath.ToSlash(filepath.Clean(raw[:secondColon])),
		function: raw[secondColon+1 : lastColon],
		minimum:  minimum,
	}
	if err := target.resolveFunction(); err != nil {
		return coverageTarget{}, err
	}

	return target, nil
}

// 対象関数範囲解決
func (target *coverageTarget) resolveFunction() error {
	data, err := os.ReadFile(target.file)
	if err != nil {
		return fmt.Errorf("coverage対象ソースを読めません: %s: %w", target.file, err)
	}
	target.sourceLines = strings.Split(string(data), "\n")

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, target.file, data, parser.SkipObjectResolution)
	if err != nil {
		return fmt.Errorf("coverage対象ソースを解析できません: %s: %w", target.file, err)
	}

	var matches []*ast.FuncDecl
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || coverageFunctionName(function) != target.function {
			continue
		}
		matches = append(matches, function)
	}
	if len(matches) != 1 {
		return fmt.Errorf("coverage対象関数を一意に特定できません: %s:%s（候補: %d）", target.file, target.function, len(matches))
	}

	target.startLine = fset.Position(matches[0].Pos()).Line
	target.endLine = fset.Position(matches[0].End()).Line

	return nil
}

// coverage関数名生成
func coverageFunctionName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return function.Name.Name
	}

	return receiverName(function.Recv.List[0].Type) + "." + function.Name.Name
}

// receiver名生成
func receiverName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverName(value.X)
	case *ast.IndexExpr:
		return receiverName(value.X)
	case *ast.IndexListExpr:
		return receiverName(value.X)
	default:
		return "unknown"
	}
}

// 関数coverage算出
func calculateCoverage(target coverageTarget, blocks []coverageBlock, stderr io.Writer) (float64, float64, int, error) {
	var total int
	var covered int
	var excluded int
	var uncoveredLines []int
	for _, block := range blocks {
		if !sameCoverageFile(block.file, target.file) || block.startLine < target.startLine || block.endLine > target.endLine {
			continue
		}

		total += block.statements
		if block.count > 0 {
			covered += block.statements
			continue
		}
		if hasUnreachableComment(target, block.startLine, block.endLine) {
			excluded += block.statements
			continue
		}
		uncoveredLines = append(uncoveredLines, block.startLine)
	}
	if total == 0 {
		return 0, 0, 0, fmt.Errorf("coverage blockがありません: %s:%s", target.file, target.function)
	}

	rawCoverage := float64(covered) / float64(total) * 100
	effectiveCoverage := float64(covered+excluded) / float64(total) * 100
	if len(uncoveredLines) > 0 {
		sort.Ints(uncoveredLines)
		fmt.Fprintf(
			stderr,
			"%s:%s の未到達行: %v。テスト追加を優先し、到達不能なら直前に %s 具体的な理由 を記載してください\n",
			target.file,
			target.function,
			uncoveredLines,
			unreachableCommentPrefix,
		)
	}

	return rawCoverage, effectiveCoverage, excluded, nil
}

// coverageファイル一致判定
func sameCoverageFile(profileFile string, targetFile string) bool {
	profileFile = filepath.ToSlash(filepath.Clean(profileFile))
	targetFile = filepath.ToSlash(filepath.Clean(targetFile))

	return profileFile == targetFile || strings.HasSuffix(profileFile, "/"+targetFile)
}

// 到達不能コメント判定
func hasUnreachableComment(target coverageTarget, startLine int, endLine int) bool {
	for index := startLine - 2; index >= target.startLine-1 && index < len(target.sourceLines); index-- {
		line := strings.TrimSpace(target.sourceLines[index])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, unreachableCommentPrefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, unreachableCommentPrefix)) != ""
		}

		break
	}

	lastIndex := min(endLine, len(target.sourceLines))
	for index := max(startLine-1, target.startLine-1); index < lastIndex; index++ {
		line := strings.TrimSpace(target.sourceLines[index])
		if strings.HasPrefix(line, unreachableCommentPrefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, unreachableCommentPrefix)) != ""
		}
	}

	return false
}

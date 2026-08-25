// check-file-lines scans project source files and reports those exceeding per-language line limits.
// Replaces the Python script scripts/check-file-lines.py.
//
// Usage:
//
//	go run ./backend/tools/check-file-lines         # flat 800-line check
//	go run ./backend/tools/check-file-lines --strict # graduated per-language limits
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Per-language limits from CLAUDE.md
const (
	goBase     = 300
	tsBase     = 250
	i18nLimit  = 500
	otherLimit = 800
	exemptMul  = 1.5 // gen/ and test files: 50% overage
	errorMul   = 1.5 // >50% over base = error
	warnMul    = 1.2 // 20-50% over base = warning
)

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "dist": true, "build": true,
	".cache": true, "__pycache__": true, ".claude": true, "gen": true,
}

var scopes = []string{
	"backend/internal",
	"backend/cmd",
	"backend/tools", // VM-CODE-HYGIENE-1: now includes VM compiler directory
	"frontend/src",
	"proto",
	"scripts",
}

type fileInfo struct {
	rel   string
	lines int
	cat   string
	limit int
}

func main() {
	strict := len(os.Args) > 1 && os.Args[1] == "--strict"

	root, _ := os.Getwd()
	// Find project root (where go.mod or .git is)
	for {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}

	files := collectFiles(root)
	oversized := make(map[string]fileInfo)

	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		rel = filepath.ToSlash(rel)
		lines := countLines(f)
		cat, limit := classify(rel)
		if !strict {
			if cat == "gen" || cat == "i18n" || cat == "scripts" || cat == "other" {
				continue
			}
			if lines > otherLimit {
				oversized[rel] = fileInfo{rel, lines, cat, otherLimit}
			}
		} else {
			if lines > limit {
				oversized[rel] = fileInfo{rel, lines, cat, limit}
			}
		}
	}

	if len(oversized) == 0 {
		fmt.Println("line check ok: all files within limits ✅")
		os.Exit(0)
	}

	errors, warnings, infos := 0, 0, 0
	for _, fi := range oversized {
		switch severity(fi.lines, fi.limit, fi.cat) {
		case "error":
			errors++
		case "warn":
			warnings++
		default:
			infos++
		}
	}

	for _, fi := range oversized {
		sev := severity(fi.lines, fi.limit, fi.cat)
		icon := map[string]string{"error": "🔴", "warn": "🟡", "info": "🟢"}[sev]
		pct := int(float64(fi.lines-fi.limit) / float64(fi.limit) * 100)
		fmt.Printf("%s %5d/%-4d (+%d%%)  %s\n", icon, fi.lines, fi.limit, pct, fi.rel)
	}

	if errors > 0 {
		fmt.Printf("\nResult: %d ERROR(s), %d warning(s), %d info(s) — must fix before commit\n", errors, warnings, infos)
		os.Exit(1)
	}
	fmt.Printf("\nResult: 0 errors, %d warning(s), %d info(s) — review but not blocking\n", warnings, infos)
}

func classify(rel string) (string, int) {
	ext := strings.ToLower(filepath.Ext(rel))

	if strings.HasPrefix(rel, "scripts/") {
		return "scripts", otherLimit
	}
	if strings.Contains(rel, "/gen/") {
		if ext == ".go" {
			return "gen", int(float64(goBase) * exemptMul)
		}
		if ext == ".ts" || ext == ".tsx" {
			return "gen", int(float64(tsBase) * exemptMul)
		}
		return "gen", otherLimit
	}

	fname := filepath.Base(rel)
	if strings.HasSuffix(fname, "_test.go") || strings.Contains(fname, ".test.ts") {
		if ext == ".go" {
			return "test", int(float64(goBase) * exemptMul)
		}
		if ext == ".ts" || ext == ".tsx" {
			return "test", int(float64(tsBase) * exemptMul)
		}
		return "test", otherLimit
	}

	if ext == ".textproto" || ext == ".proto" {
		return "other", 9999
	}
	if strings.Contains(rel, "/i18n/") && ext == ".json" {
		return "other", 9999
	}
	if strings.Contains(rel, "/i18n/") {
		return "i18n", i18nLimit
	}

	if ext == ".go" {
		return "go", goBase
	}
	if ext == ".ts" || ext == ".tsx" {
		return "ts", tsBase
	}
	return "other", otherLimit
}

func severity(lines, limit int, cat string) string {
	if cat == "gen" || cat == "i18n" || cat == "other" || cat == "scripts" {
		return "info"
	}
	if cat == "test" {
		if lines > int(float64(limit)*1.33) {
			return "warn"
		}
		return "info"
	}
	if lines > int(float64(limit)*errorMul) {
		return "error"
	}
	if lines > int(float64(limit)*warnMul) {
		return "warn"
	}
	return "info"
}

func collectFiles(root string) []string {
	var files []string
	for _, scope := range scopes {
		base := filepath.Join(root, scope)
		_ = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(info.Name(), ".md") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
				if skipDirs[part] {
					return nil
				}
			}
			if isText(path) {
				files = append(files, path)
			}
			return nil
		})
	}
	return files
}

func countLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	text := string(data)
	n := strings.Count(text, "\n")
	if len(text) > 0 && !strings.HasSuffix(text, "\n") {
		n++
	}
	return n
}

func isText(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if len(data) > 4096 {
		data = data[:4096]
	}
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	return true
}

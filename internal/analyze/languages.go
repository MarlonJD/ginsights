package analyze

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var languageByExt = map[string]string{
	".go":    "Go",
	".js":    "JavaScript",
	".jsx":   "JavaScript",
	".ts":    "TypeScript",
	".tsx":   "TypeScript",
	".py":    "Python",
	".rs":    "Rust",
	".java":  "Java",
	".rb":    "Ruby",
	".php":   "PHP",
	".cs":    "C#",
	".c":     "C",
	".h":     "C/C++",
	".cc":    "C++",
	".cpp":   "C++",
	".hpp":   "C++",
	".swift": "Swift",
	".kt":    "Kotlin",
	".sh":    "Shell",
	".bash":  "Shell",
	".zsh":   "Shell",
	".sql":   "SQL",
	".html":  "HTML",
	".css":   "CSS",
	".scss":  "SCSS",
	".md":    "Markdown",
	".yaml":  "YAML",
	".yml":   "YAML",
	".json":  "JSON",
	".toml":  "TOML",
}

func DetectLanguages(repo string) []LanguageStat {
	totals := map[string]int64{}
	var grand int64
	_ = filepath.WalkDir(repo, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules", "report", ".ginsights", ".ginsights-cache", "dist", "bin":
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() <= 0 {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		lang := languageByExt[ext]
		if lang == "" {
			return nil
		}
		totals[lang] += info.Size()
		grand += info.Size()
		return nil
	})

	out := make([]LanguageStat, 0, len(totals))
	for name, size := range totals {
		pct := 0.0
		if grand > 0 {
			pct = float64(size) / float64(grand) * 100
		}
		out = append(out, LanguageStat{Name: name, Bytes: size, Percent: pct})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes == out[j].Bytes {
			return out[i].Name < out[j].Name
		}
		return out[i].Bytes > out[j].Bytes
	})
	return out
}

package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/multica-ai/ginsights/internal/analyze"
)

func Write(outDir string, snap analyze.Snapshot) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	html, err := HTML(snap)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "index.html"), []byte(html), 0o644); err != nil {
		return err
	}
	data, err := JSON(snap)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "data.json"), data, 0o644); err != nil {
		return err
	}
	return nil
}

func JSON(snap analyze.Snapshot) ([]byte, error) {
	return json.MarshalIndent(snap, "", "  ")
}

func HTML(snap analyze.Snapshot) (string, error) {
	view := makeView(snap)
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"formatInt":   formatInt,
		"formatDate":  formatDate,
		"formatPct":   formatPct,
		"barWidth":    barWidth,
		"statusClass": statusClass,
		"statusText":  statusText,
	}).Parse(reportTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type viewModel struct {
	analyze.Snapshot
	MaxWeeklyLines   int
	MaxWeeklyCommits int
	MaxDailyCommits  int
	MaxFileChurn     int
	GeneratedLabel   string
}

func makeView(snap analyze.Snapshot) viewModel {
	v := viewModel{Snapshot: snap, GeneratedLabel: snap.GeneratedAt.Format(time.RFC1123)}
	for _, week := range snap.Weekly {
		if lines := week.Additions + week.Deletions; lines > v.MaxWeeklyLines {
			v.MaxWeeklyLines = lines
		}
		if week.Commits > v.MaxWeeklyCommits {
			v.MaxWeeklyCommits = week.Commits
		}
	}
	for _, day := range snap.Daily {
		if day.Commits > v.MaxDailyCommits {
			v.MaxDailyCommits = day.Commits
		}
	}
	for _, file := range snap.HotFiles {
		if file.Churn > v.MaxFileChurn {
			v.MaxFileChurn = file.Churn
		}
	}
	return v
}

func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02")
}

func formatPct(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}

func barWidth(value, max int) int {
	if value <= 0 || max <= 0 {
		return 1
	}
	pct := int(float64(value) / float64(max) * 100)
	if pct < 1 {
		return 1
	}
	if pct > 100 {
		return 100
	}
	return pct
}

func statusClass(ok bool) string {
	if ok {
		return "ok"
	}
	return "missing"
}

func statusText(ok bool) string {
	if ok {
		return "Present"
	}
	return "Missing"
}

const reportTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.RepoName}} · ginsights</title>
  <style>
    :root { color-scheme: light; --bg:#f6f8fa; --panel:#ffffff; --border:#d0d7de; --muted:#57606a; --text:#24292f; --accent:#0969da; --good:#1a7f37; --bad:#cf222e; --bar:#54aeff; --bar2:#f85149; --heat:#2da44e; }
    * { box-sizing: border-box; }
    body { margin:0; font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; background:var(--bg); color:var(--text); }
    a { color:var(--accent); text-decoration:none; }
    header { background:#24292f; color:#fff; padding:18px 28px; }
    header .repo { font-size:20px; font-weight:650; }
    header .path { color:#afb8c1; margin-top:4px; font-size:13px; }
    nav { display:flex; gap:8px; flex-wrap:wrap; padding:12px 28px; background:var(--panel); border-bottom:1px solid var(--border); position:sticky; top:0; z-index:1; }
    nav a { padding:7px 10px; border-radius:6px; color:var(--text); font-size:14px; }
    nav a:hover { background:#f3f4f6; }
    main { max-width:1180px; margin:0 auto; padding:24px; }
    section { margin:0 0 24px; }
    h2 { margin:0 0 12px; font-size:20px; }
    .cards { display:grid; grid-template-columns:repeat(auto-fit,minmax(160px,1fr)); gap:12px; }
    .card, .panel { background:var(--panel); border:1px solid var(--border); border-radius:10px; padding:16px; box-shadow:0 1px 0 rgba(27,31,36,.04); }
    .metric { font-size:28px; font-weight:700; letter-spacing:-.02em; }
    .label, .muted { color:var(--muted); font-size:13px; }
    .grid { display:grid; grid-template-columns:1fr; gap:16px; }
    @media (min-width:900px){ .grid.two { grid-template-columns:1fr 1fr; } }
    table { width:100%; border-collapse:collapse; font-size:14px; }
    th, td { padding:9px 8px; border-bottom:1px solid var(--border); text-align:left; vertical-align:top; }
    th { color:var(--muted); font-weight:600; background:#f6f8fa; }
    tr:last-child td { border-bottom:0; }
    .barrow { display:flex; align-items:center; gap:8px; min-width:180px; }
    .bartrack { flex:1; height:10px; background:#eaeef2; border-radius:999px; overflow:hidden; }
    .bar { height:100%; background:var(--bar); border-radius:999px; }
    .bar.delete { background:var(--bar2); }
    .language-stack { display:flex; height:14px; border-radius:999px; overflow:hidden; background:#eaeef2; margin-bottom:12px; }
    .language-stack span { display:block; min-width:1px; background:var(--bar); border-right:1px solid rgba(255,255,255,.55); }
    .health { display:grid; grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); gap:10px; }
    .health-item { border:1px solid var(--border); border-radius:8px; padding:10px; background:#fff; }
    .pill { display:inline-flex; align-items:center; border-radius:999px; padding:2px 8px; font-size:12px; font-weight:600; }
    .pill.ok { color:var(--good); background:#dafbe1; }
    .pill.missing { color:var(--bad); background:#ffebe9; }
    .heatmap { display:grid; grid-template-columns:repeat(auto-fill, minmax(10px, 1fr)); gap:3px; max-width:760px; }
    .heat { width:10px; height:10px; border-radius:2px; background:#ebedf0; }
    .heat.on { background:var(--heat); opacity:.35; }
    .heat.l2 { opacity:.55; } .heat.l3 { opacity:.75; } .heat.l4 { opacity:1; }
    .recent li { margin:0 0 8px; }
    footer { color:var(--muted); font-size:12px; margin:36px 0 0; }
    code { background:#f6f8fa; padding:2px 5px; border-radius:5px; }
  </style>
</head>
<body>
  <header>
    <div class="repo">{{.RepoName}} <span class="muted">/ local insights</span></div>
    <div class="path">{{.RepoPath}}</div>
  </header>
  <nav>
    <a href="#pulse">Pulse</a><a href="#contributors">Contributors</a><a href="#commits">Commits</a><a href="#code-frequency">Code frequency</a><a href="#files">Files</a><a href="#languages">Languages</a><a href="#health">Health</a>
  </nav>
  <main>
    <section id="pulse">
      <h2>Pulse</h2>
      <div class="cards">
        <div class="card"><div class="metric">{{formatInt .Totals.Commits}}</div><div class="label">commits</div></div>
        <div class="card"><div class="metric">{{formatInt .Totals.Authors}}</div><div class="label">authors</div></div>
        <div class="card"><div class="metric">{{formatInt .Totals.FilesChanged}}</div><div class="label">files changed</div></div>
        <div class="card"><div class="metric">+{{formatInt .Totals.Additions}}</div><div class="label">lines added</div></div>
        <div class="card"><div class="metric">-{{formatInt .Totals.Deletions}}</div><div class="label">lines deleted</div></div>
      </div>
    </section>

    <section class="grid two">
      <div class="panel" id="commits">
        <h2>Commit activity</h2>
        {{if .Daily}}
        <div class="heatmap" aria-label="commit activity heatmap">
          {{range .Daily}}<span title="{{formatDate .Date}}: {{.Commits}} commits" class="heat {{if gt .Commits 0}}on{{end}} {{if ge .Commits 3}}l3{{else if ge .Commits 2}}l2{{end}}"></span>{{end}}
        </div>
        {{else}}<p class="muted">No commits found.</p>{{end}}
      </div>
      <div class="panel">
        <h2>Recent commits</h2>
        {{if .Recent}}
        <ul class="recent">
          {{range .Recent}}<li><code>{{.ShortHash}}</code> {{.Subject}} <span class="muted">by {{.AuthorName}} on {{formatDate .Date}}</span></li>{{end}}
        </ul>
        {{else}}<p class="muted">No recent commits found.</p>{{end}}
      </div>
    </section>

    <section id="contributors" class="panel">
      <h2>Contributors</h2>
      {{if .Authors}}
      <table>
        <thead><tr><th>Author</th><th>Commits</th><th>Additions</th><th>Deletions</th><th>Files touched</th><th>Active range</th></tr></thead>
        <tbody>{{range .Authors}}
          <tr><td><strong>{{.Name}}</strong><br><span class="muted">{{.Email}}</span></td><td>{{formatInt .Commits}}</td><td>+{{formatInt .Additions}}</td><td>-{{formatInt .Deletions}}</td><td>{{formatInt .FilesTouched}}</td><td>{{formatDate .FirstCommit}} → {{formatDate .LastCommit}}</td></tr>
        {{end}}</tbody>
      </table>
      {{else}}<p class="muted">No contributors found.</p>{{end}}
    </section>

    <section id="code-frequency" class="panel">
      <h2>Code frequency</h2>
      {{if .Weekly}}
      <table>
        <thead><tr><th>Week</th><th>Commits</th><th>Additions</th><th>Deletions</th></tr></thead>
        <tbody>{{range .Weekly}}
          <tr>
            <td>{{formatDate .WeekStart}}</td>
            <td>{{formatInt .Commits}}</td>
            <td><div class="barrow"><span>+{{formatInt .Additions}}</span><div class="bartrack"><div class="bar" style="width:{{barWidth .Additions $.MaxWeeklyLines}}%"></div></div></div></td>
            <td><div class="barrow"><span>-{{formatInt .Deletions}}</span><div class="bartrack"><div class="bar delete" style="width:{{barWidth .Deletions $.MaxWeeklyLines}}%"></div></div></div></td>
          </tr>
        {{end}}</tbody>
      </table>
      {{else}}<p class="muted">No weekly stats found.</p>{{end}}
    </section>

    <section id="files" class="panel">
      <h2>Hot files</h2>
      {{if .HotFiles}}
      <table>
        <thead><tr><th>Path</th><th>Commits</th><th>Churn</th><th>Additions</th><th>Deletions</th></tr></thead>
        <tbody>{{range .HotFiles}}
          <tr><td><code>{{.Path}}</code></td><td>{{formatInt .Commits}}</td><td><div class="barrow"><span>{{formatInt .Churn}}</span><div class="bartrack"><div class="bar" style="width:{{barWidth .Churn $.MaxFileChurn}}%"></div></div></div></td><td>+{{formatInt .Additions}}</td><td>-{{formatInt .Deletions}}</td></tr>
        {{end}}</tbody>
      </table>
      {{else}}<p class="muted">No changed files found.</p>{{end}}
    </section>

    <section id="languages" class="panel">
      <h2>Languages</h2>
      {{if .Languages}}
      <div class="language-stack">{{range .Languages}}<span title="{{.Name}} {{formatPct .Percent}}" style="width:{{printf "%.3f" .Percent}}%"></span>{{end}}</div>
      <table><thead><tr><th>Language</th><th>Bytes</th><th>Share</th></tr></thead><tbody>{{range .Languages}}<tr><td>{{.Name}}</td><td>{{.Bytes}}</td><td>{{formatPct .Percent}}</td></tr>{{end}}</tbody></table>
      {{else}}<p class="muted">No known language files detected.</p>{{end}}
    </section>

    <section id="health" class="panel">
      <h2>Repository health</h2>
      <div class="health">{{range .Health}}
        <div class="health-item"><span class="pill {{statusClass .Present}}">{{statusText .Present}}</span><strong> {{.Name}}</strong><div class="muted">{{.Detail}}</div></div>
      {{end}}</div>
    </section>

    <section class="panel">
      <h2>Metric provenance</h2>
      <table><thead><tr><th>Metric</th><th>Source</th></tr></thead><tbody>{{range .Provenance}}<tr><td>{{.Metric}}</td><td><code>{{.Source}}</code></td></tr>{{end}}</tbody></table>
      <p class="muted">GitHub Traffic metrics such as views, visitors, clones, referrers, and popular content are not derivable from local Git history.</p>
    </section>

    <footer>Generated by ginsights at {{.GeneratedLabel}}. Static data: <a href="data.json">data.json</a>.</footer>
  </main>
</body>
</html>`

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
		"formatInt":       formatInt,
		"formatDate":      formatDate,
		"formatPct":       formatPct,
		"formatSignedInt": formatSignedInt,
		"barWidth":        barWidth,
		"statusClass":     statusClass,
		"statusText":      statusText,
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
	HeatmapDays      []heatmapDay
	HeatmapSummary   string
	WeeklyRows       []weeklyRow
	WeeklySummary    string
}

type heatmapDay struct {
	Date    time.Time
	Commits int
	Class   string
	Tooltip string
}

type weeklyRow struct {
	analyze.WeekStat
	Net      int
	NetClass string
	AddWidth int
	DelWidth int
	Tooltip  string
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
	v.HeatmapDays, v.HeatmapSummary = buildHeatmapDays(snap.Daily)
	v.WeeklyRows, v.WeeklySummary = buildWeeklyRows(snap.Weekly, v.MaxWeeklyLines)
	return v
}

func buildHeatmapDays(daily []analyze.DayStat) ([]heatmapDay, string) {
	if len(daily) == 0 {
		return nil, ""
	}
	counts := map[time.Time]int{}
	var minDate, maxDate time.Time
	maxCommits := 0
	for _, day := range daily {
		if day.Date.IsZero() {
			continue
		}
		date := dayStart(day.Date)
		counts[date] += day.Commits
		if minDate.IsZero() || date.Before(minDate) {
			minDate = date
		}
		if maxDate.IsZero() || date.After(maxDate) {
			maxDate = date
		}
		if counts[date] > maxCommits {
			maxCommits = counts[date]
		}
	}
	if minDate.IsZero() {
		return nil, ""
	}

	start := weekStart(minDate)
	end := weekStart(maxDate).AddDate(0, 0, 6)
	out := make([]heatmapDay, 0, int(end.Sub(start).Hours()/24)+1)
	activeDays := 0
	totalCommits := 0
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		commits := counts[date]
		if commits > 0 {
			activeDays++
			totalCommits += commits
		}
		class := fmt.Sprintf("l%d", heatLevel(commits, maxCommits))
		out = append(out, heatmapDay{
			Date:    date,
			Commits: commits,
			Class:   class,
			Tooltip: fmt.Sprintf("%s: %s", formatDate(date), commitCountLabel(commits)),
		})
	}
	return out, fmt.Sprintf("%s active %s · %s", formatInt(activeDays), plural(activeDays, "day", "days"), commitCountLabel(totalCommits))
}

func buildWeeklyRows(weekly []analyze.WeekStat, maxLines int) ([]weeklyRow, string) {
	if len(weekly) == 0 {
		return nil, ""
	}
	rows := make([]weeklyRow, 0, len(weekly))
	totalAdditions := 0
	totalDeletions := 0
	for _, week := range weekly {
		net := week.Additions - week.Deletions
		totalAdditions += week.Additions
		totalDeletions += week.Deletions
		rows = append(rows, weeklyRow{
			WeekStat: week,
			Net:      net,
			NetClass: netClass(net),
			AddWidth: frequencyWidth(week.Additions, maxLines),
			DelWidth: frequencyWidth(week.Deletions, maxLines),
			Tooltip: fmt.Sprintf("%s: +%s additions, -%s deletions, %s net across %s",
				formatDate(week.WeekStart),
				formatInt(week.Additions),
				formatInt(week.Deletions),
				formatSignedInt(net),
				commitCountLabel(week.Commits),
			),
		})
	}
	net := totalAdditions - totalDeletions
	summary := fmt.Sprintf("%s %s · +%s/-%s · net %s",
		formatInt(len(weekly)),
		plural(len(weekly), "week", "weeks"),
		formatInt(totalAdditions),
		formatInt(totalDeletions),
		formatSignedInt(net),
	)
	return rows, summary
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func weekStart(t time.Time) time.Time {
	date := dayStart(t)
	offset := (int(date.Weekday()) + 6) % 7
	return date.AddDate(0, 0, -offset)
}

func heatLevel(commits, maxCommits int) int {
	if commits <= 0 || maxCommits <= 0 {
		return 0
	}
	level := (commits*4 + maxCommits - 1) / maxCommits
	if level < 1 {
		return 1
	}
	if level > 4 {
		return 4
	}
	return level
}

func frequencyWidth(value, max int) int {
	if value <= 0 || max <= 0 {
		return 0
	}
	pct := int(float64(value) / float64(max) * 100)
	if pct < 2 {
		return 2
	}
	if pct > 100 {
		return 100
	}
	return pct
}

func formatSignedInt(n int) string {
	if n >= 0 {
		return "+" + formatInt(n)
	}
	return "-" + formatInt(-n)
}

func netClass(n int) string {
	if n > 0 {
		return "positive"
	}
	if n < 0 {
		return "negative"
	}
	return "neutral"
}

func commitCountLabel(n int) string {
	return fmt.Sprintf("%s %s", formatInt(n), plural(n, "commit", "commits"))
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
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
    :root { color-scheme: light; --bg:#f6f8fa; --panel:#ffffff; --border:#d0d7de; --muted:#57606a; --text:#24292f; --accent:#0969da; --good:#1a7f37; --bad:#cf222e; --bar:#54aeff; --bar2:#f85149; --heat0:#ebedf0; --heat1:#9be9a8; --heat2:#40c463; --heat3:#30a14e; --heat4:#216e39; }
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
    .section-summary { color:var(--muted); font-size:13px; margin:-4px 0 12px; }
    .code-frequency-table th:nth-child(3), .code-frequency-table td:nth-child(3),
    .code-frequency-table th:nth-child(5), .code-frequency-table td:nth-child(5),
    .code-frequency-table th:nth-child(6), .code-frequency-table td:nth-child(6) { text-align:right; }
    .code-frequency-table .net { font-weight:650; font-variant-numeric:tabular-nums; white-space:nowrap; }
    .code-frequency-table .net.positive { color:var(--good); }
    .code-frequency-table .net.negative { color:var(--bad); }
    .code-frequency-table .net.neutral { color:var(--muted); }
    .frequency-cell { min-width:190px; }
    .frequency-bars { display:flex; align-items:center; height:14px; width:100%; overflow:hidden; border-radius:999px; background:#eaeef2; }
    .frequency-bar { display:block; height:100%; min-width:0; }
    .frequency-bar.additions { background:var(--good); }
    .frequency-bar.deletions { background:var(--bad); }
    .delta { font-variant-numeric:tabular-nums; white-space:nowrap; }
    .delta.additions { color:var(--good); }
    .delta.deletions { color:var(--bad); }
    .language-stack { display:flex; height:14px; border-radius:999px; overflow:hidden; background:#eaeef2; margin-bottom:12px; }
    .language-stack span { display:block; min-width:1px; background:var(--bar); border-right:1px solid rgba(255,255,255,.55); }
    .health { display:grid; grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); gap:10px; }
    .health-item { border:1px solid var(--border); border-radius:8px; padding:10px; background:#fff; }
    .api-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(150px,1fr)); gap:10px; }
    .api-metric { border:1px solid var(--border); border-radius:8px; padding:10px; background:#fff; }
    .warning-list { color:var(--muted); font-size:13px; margin:12px 0 0; }
    .pill { display:inline-flex; align-items:center; border-radius:999px; padding:2px 8px; font-size:12px; font-weight:600; }
    .pill.ok { color:var(--good); background:#dafbe1; }
    .pill.missing { color:var(--bad); background:#ffebe9; }
    .heatmap-shell { display:grid; gap:10px; }
    .heatmap-body { display:flex; align-items:flex-start; gap:6px; max-width:760px; overflow-x:auto; padding-bottom:4px; }
    .heatmap-weekdays { display:grid; grid-template-rows:repeat(7,10px); gap:3px; flex:0 0 28px; color:var(--muted); font-size:10px; line-height:10px; }
    .heatmap-grid { display:grid; grid-auto-flow:column; grid-template-rows:repeat(7,10px); grid-auto-columns:10px; gap:3px; width:max-content; }
    .heat { width:10px; height:10px; border-radius:2px; background:var(--heat0); box-shadow:inset 0 0 0 1px rgba(27,31,36,.06); }
    .heat.l0 { background:var(--heat0); }
    .heat.l1 { background:var(--heat1); }
    .heat.l2 { background:var(--heat2); }
    .heat.l3 { background:var(--heat3); }
    .heat.l4 { background:var(--heat4); }
    .heatmap-legend { display:flex; align-items:center; justify-content:flex-end; gap:5px; color:var(--muted); font-size:12px; max-width:760px; }
    .heatmap-legend .heat { display:inline-block; flex:0 0 auto; }
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
        {{if .HeatmapDays}}
        <div class="heatmap-shell">
          <div class="section-summary">{{.HeatmapSummary}}</div>
          <div class="heatmap-body">
            <div class="heatmap-weekdays" aria-hidden="true"><span>Mon</span><span></span><span>Wed</span><span></span><span>Fri</span><span></span><span></span></div>
            <div class="heatmap-grid" aria-label="commit activity heatmap, {{len .HeatmapDays}} days">
              {{range .HeatmapDays}}<span title="{{.Tooltip}}" aria-label="{{.Tooltip}}" class="heat {{.Class}}"></span>{{end}}
            </div>
          </div>
          <div class="heatmap-legend" aria-label="commit intensity legend"><span>Less</span><span class="heat l0"></span><span class="heat l1"></span><span class="heat l2"></span><span class="heat l3"></span><span class="heat l4"></span><span>More</span></div>
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
      {{if .WeeklyRows}}
      <div class="section-summary">{{.WeeklySummary}}</div>
      <table class="code-frequency-table">
        <thead><tr><th>Week</th><th>Commits</th><th>Net</th><th>Changes</th><th>Additions</th><th>Deletions</th></tr></thead>
        <tbody>{{range .WeeklyRows}}
          <tr>
            <td>{{formatDate .WeekStart}}</td>
            <td>{{formatInt .Commits}}</td>
            <td class="net {{.NetClass}}">{{formatSignedInt .Net}}</td>
            <td class="frequency-cell"><div class="frequency-bars" title="{{.Tooltip}}" aria-label="{{.Tooltip}}"><span class="frequency-bar additions" style="width:{{.AddWidth}}%"></span><span class="frequency-bar deletions" style="width:{{.DelWidth}}%"></span></div></td>
            <td class="delta additions">+{{formatInt .Additions}}</td>
            <td class="delta deletions">-{{formatInt .Deletions}}</td>
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

{{if .GitHub}}
    <section id="github-api" class="panel">
      <h2>GitHub API</h2>
      <p class="section-summary"><code>github_api</code> data for {{.GitHub.Repository}}</p>
      {{if .GitHub.Error}}
      <p class="muted">{{.GitHub.Error}}</p>
      {{else}}
      <div class="api-grid">
        <div class="api-metric"><div class="metric">{{formatInt .GitHub.Stars}}</div><div class="label">stars</div></div>
        <div class="api-metric"><div class="metric">{{formatInt .GitHub.Forks}}</div><div class="label">forks</div></div>
        <div class="api-metric"><div class="metric">{{formatInt .GitHub.OpenIssues}}</div><div class="label">open issues</div></div>
        {{if .GitHub.Views}}<div class="api-metric"><div class="metric">{{formatInt .GitHub.Views.Count}}</div><div class="label">{{formatInt .GitHub.Views.Count}} views · {{formatInt .GitHub.Views.Uniques}} unique</div></div>{{end}}
        {{if .GitHub.Clones}}<div class="api-metric"><div class="metric">{{formatInt .GitHub.Clones.Count}}</div><div class="label">{{formatInt .GitHub.Clones.Count}} clones · {{formatInt .GitHub.Clones.Uniques}} unique</div></div>{{end}}
      </div>
      {{if .GitHub.Warnings}}<ul class="warning-list">{{range .GitHub.Warnings}}<li>{{.}}</li>{{end}}</ul>{{end}}
      {{end}}
    </section>
{{end}}

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

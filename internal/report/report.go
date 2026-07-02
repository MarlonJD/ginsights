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
		"languageColor":   languageColor,
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
	NetLinesLabel    string
	NetLinesClass    string
	TotalLineChanges int
	HealthPresent    int
	HealthTotal      int
	PrimaryLanguage  string
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
	v := viewModel{
		Snapshot:         snap,
		GeneratedLabel:   snap.GeneratedAt.Format(time.RFC1123),
		NetLinesLabel:    formatSignedInt(snap.Totals.NetLines),
		NetLinesClass:    netClass(snap.Totals.NetLines),
		TotalLineChanges: snap.Totals.Additions + snap.Totals.Deletions,
		HealthTotal:      len(snap.Health),
	}
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
	for _, health := range snap.Health {
		if health.Present {
			v.HealthPresent++
		}
	}
	if len(snap.Languages) > 0 {
		lang := snap.Languages[0]
		v.PrimaryLanguage = fmt.Sprintf("%s %s", lang.Name, formatPct(lang.Percent))
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

func languageColor(index int) string {
	colors := []string{"#2da44e", "#0969da", "#8250df", "#bf8700", "#cf222e", "#1f6feb", "#6fdd8b"}
	if index < 0 {
		index = 0
	}
	return colors[index%len(colors)]
}

const reportTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.RepoName}} · ginsights</title>
  <style>
    :root {
      color-scheme: light;
      --bg:#f5f7fb;
      --panel:#ffffff;
      --panel-soft:#f8fafc;
      --border:#d8dee9;
      --border-strong:#b7c2d0;
      --muted:#667085;
      --text:#172033;
      --subtle:#eef2f7;
      --accent:#0969da;
      --accent-soft:#ddf4ff;
      --good:#1a7f37;
      --good-soft:#dafbe1;
      --bad:#cf222e;
      --bad-soft:#ffebe9;
      --warn:#bf8700;
      --violet:#8250df;
      --shadow:0 16px 40px rgba(23,32,51,.08);
      --heat0:#ebedf0;
      --heat1:#9be9a8;
      --heat2:#40c463;
      --heat3:#30a14e;
      --heat4:#216e39;
    }
    * { box-sizing: border-box; }
    html { scroll-behavior:smooth; }
    body { margin:0; font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; background:linear-gradient(180deg,#f8fafc 0,#f5f7fb 340px,#f7f8fb 100%); color:var(--text); }
    a { color:var(--accent); text-decoration:none; }
    .layout { display:grid; grid-template-columns:260px minmax(0,1fr); min-height:100vh; }
    .sidebar { position:sticky; top:0; height:100vh; align-self:start; display:flex; flex-direction:column; gap:20px; padding:24px 18px; background:rgba(255,255,255,.88); border-right:1px solid var(--border); backdrop-filter:saturate(140%) blur(14px); }
    .brand { display:flex; align-items:center; gap:11px; min-width:0; }
    .brand-mark { display:grid; place-items:center; width:36px; height:36px; border-radius:8px; background:#172033; color:#fff; font-weight:800; }
    .brand-title { font-size:17px; font-weight:750; }
    .brand-subtitle { color:var(--muted); font-size:12px; margin-top:1px; }
    .side-nav { display:grid; gap:5px; }
    .side-nav a { display:flex; align-items:center; gap:9px; min-height:34px; padding:7px 10px; border-radius:8px; color:var(--text); font-size:14px; font-weight:600; }
    .side-nav a::before { content:""; width:7px; height:7px; border-radius:50%; background:var(--border-strong); flex:0 0 auto; }
    .side-nav a:hover { background:var(--panel-soft); }
    .side-nav a:focus-visible { outline:2px solid var(--accent); outline-offset:2px; }
    .side-card { margin-top:auto; padding:12px; border:1px solid var(--border); border-radius:8px; background:var(--panel-soft); }
    .side-card strong { display:block; font-size:13px; margin-bottom:4px; }
    .content { width:100%; max-width:1240px; padding:30px 34px 42px; }
    section { margin:0 0 20px; scroll-margin-top:18px; }
    h1, h2, h3, p { margin-top:0; }
    h1 { margin-bottom:8px; font-size:34px; line-height:1.12; letter-spacing:0; }
    h2 { margin:0 0 12px; font-size:20px; line-height:1.25; letter-spacing:0; }
    h3 { margin:0 0 8px; font-size:14px; line-height:1.3; }
    .hero { display:grid; grid-template-columns:minmax(0,1.35fr) minmax(280px,.65fr); gap:18px; align-items:stretch; margin-bottom:18px; }
    .hero-main, .radar { border:1px solid var(--border); border-radius:8px; box-shadow:var(--shadow); }
    .hero-main { display:flex; flex-direction:column; justify-content:flex-start; gap:46px; min-height:210px; padding:24px; background:linear-gradient(135deg,#ffffff 0,#f9fbff 72%,#edf6ff 100%); }
    .repo-path { color:var(--muted); font-size:13px; overflow-wrap:anywhere; margin-bottom:20px; }
    .hero-meta { display:flex; flex-wrap:wrap; gap:8px; }
    .chip { display:inline-flex; align-items:center; gap:7px; min-height:28px; padding:5px 9px; border:1px solid var(--border); border-radius:8px; background:rgba(255,255,255,.72); color:var(--muted); font-size:12px; font-weight:650; }
    .chip::before { content:""; width:7px; height:7px; border-radius:50%; background:var(--accent); }
    .chip.good::before { background:var(--good); }
    .chip.warn::before { background:var(--warn); }
    .radar { display:flex; flex-direction:column; justify-content:space-between; gap:18px; padding:20px; background:#101828; color:#fff; }
    .radar .muted { color:#bac5d5; }
    .radar-title { font-size:13px; font-weight:750; color:#dbe7ff; }
    .radar-value { font-size:32px; line-height:1; font-weight:800; margin:6px 0; }
    .balance-track { display:flex; height:12px; overflow:hidden; border-radius:8px; background:rgba(255,255,255,.16); box-shadow:inset 0 0 0 1px rgba(255,255,255,.12); }
    .balance-track span { display:block; height:100%; }
    .balance-track .add { background:var(--good); }
    .balance-track .del { background:#f85149; }
    .radar-grid { display:grid; grid-template-columns:1fr 1fr; gap:10px; }
    .radar-stat { padding-top:10px; border-top:1px solid rgba(255,255,255,.16); }
    .radar-stat strong { display:block; font-size:18px; }
    .metric-grid { display:grid; grid-template-columns:repeat(5,minmax(0,1fr)); gap:12px; margin-bottom:20px; }
    .metric-card, .panel { background:var(--panel); border:1px solid var(--border); border-radius:8px; box-shadow:0 1px 0 rgba(23,32,51,.04); }
    .metric-card { min-height:106px; padding:15px; display:flex; flex-direction:column; justify-content:space-between; }
    .metric { font-size:30px; font-weight:800; letter-spacing:0; line-height:1; }
    .metric.positive, .net.positive, .delta.additions { color:var(--good); }
    .metric.negative, .net.negative, .delta.deletions { color:var(--bad); }
    .label, .muted { color:var(--muted); font-size:13px; }
    .panel { padding:18px; }
    .grid { display:grid; grid-template-columns:1fr; gap:16px; }
    @media (min-width:980px){ .grid.two { grid-template-columns:1.05fr .95fr; } }
    .section-summary { color:var(--muted); font-size:13px; margin:-4px 0 14px; }
    .table-scroll { width:100%; overflow-x:auto; border:1px solid var(--border); border-radius:8px; }
    table { width:100%; border-collapse:collapse; font-size:14px; }
    th, td { padding:9px 8px; border-bottom:1px solid var(--border); text-align:left; vertical-align:top; }
    th { color:var(--muted); font-weight:700; background:var(--panel-soft); }
    tr:last-child td { border-bottom:0; }
    .table-scroll table { min-width:680px; }
    .contributors-table { min-width:820px; }
    .files-table { min-width:760px; }
    .language-table { min-width:520px; }
    .provenance-table { min-width:560px; }
    .barrow { display:flex; align-items:center; gap:8px; min-width:180px; }
    .bartrack { flex:1; height:10px; background:var(--subtle); border-radius:8px; overflow:hidden; }
    .bar { height:100%; background:var(--accent); border-radius:8px; }
    .code-frequency-table th:nth-child(3), .code-frequency-table td:nth-child(3),
    .code-frequency-table th:nth-child(5), .code-frequency-table td:nth-child(5),
    .code-frequency-table th:nth-child(6), .code-frequency-table td:nth-child(6) { text-align:right; }
    .code-frequency-table .net { font-weight:650; font-variant-numeric:tabular-nums; white-space:nowrap; }
    .code-frequency-table .net.neutral { color:var(--muted); }
    .frequency-cell { min-width:190px; }
    .frequency-bars { display:flex; align-items:center; height:14px; width:100%; overflow:hidden; border-radius:8px; background:var(--subtle); }
    .frequency-bar { display:block; height:100%; min-width:0; }
    .frequency-bar.additions { background:var(--good); }
    .frequency-bar.deletions { background:var(--bad); }
    .delta { font-variant-numeric:tabular-nums; white-space:nowrap; }
    .language-stack { display:flex; height:14px; border-radius:8px; overflow:hidden; background:var(--subtle); margin-bottom:12px; }
    .language-stack span { display:block; min-width:1px; background:var(--bar); border-right:1px solid rgba(255,255,255,.55); }
    .health { display:grid; grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); gap:10px; }
    .health-item { border:1px solid var(--border); border-radius:8px; padding:11px; background:var(--panel-soft); }
    .api-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(150px,1fr)); gap:10px; }
    .api-metric { border:1px solid var(--border); border-radius:8px; padding:11px; background:var(--panel-soft); }
    .warning-list { color:var(--muted); font-size:13px; margin:12px 0 0; }
    .pill { display:inline-flex; align-items:center; border-radius:8px; padding:2px 8px; font-size:12px; font-weight:700; }
    .pill.ok { color:var(--good); background:var(--good-soft); }
    .pill.missing { color:var(--bad); background:var(--bad-soft); }
    .heatmap-shell { display:grid; gap:10px; }
    .heatmap-body { display:flex; align-items:flex-start; gap:6px; max-width:100%; overflow-x:auto; padding:6px 0 4px; }
    .heatmap-weekdays { display:grid; grid-template-rows:repeat(7,10px); gap:3px; flex:0 0 28px; color:var(--muted); font-size:10px; line-height:10px; }
    .heatmap-grid { display:grid; grid-auto-flow:column; grid-template-rows:repeat(7,10px); grid-auto-columns:10px; gap:3px; width:max-content; }
    .heat { width:10px; height:10px; border-radius:2px; background:var(--heat0); box-shadow:inset 0 0 0 1px rgba(27,31,36,.06); }
    .heat.l0 { background:var(--heat0); }
    .heat.l1 { background:var(--heat1); }
    .heat.l2 { background:var(--heat2); }
    .heat.l3 { background:var(--heat3); }
    .heat.l4 { background:var(--heat4); }
    .heatmap-legend { display:flex; align-items:center; justify-content:flex-end; gap:5px; color:var(--muted); font-size:12px; max-width:100%; }
    .heatmap-legend .heat { display:inline-block; flex:0 0 auto; }
    .recent { display:grid; gap:10px; padding:0; margin:0; list-style:none; }
    .recent li { display:grid; grid-template-columns:auto minmax(0,1fr); gap:10px; align-items:start; padding:9px 0; border-bottom:1px solid var(--border); }
    .recent li:last-child { border-bottom:0; }
    .hash { color:var(--accent); font-weight:750; }
    footer { color:var(--muted); font-size:12px; margin:36px 0 0; }
    code { background:var(--panel-soft); padding:2px 5px; border-radius:5px; }
    @media (max-width:1080px) {
      .layout { grid-template-columns:220px minmax(0,1fr); }
      .metric-grid { grid-template-columns:repeat(3,minmax(0,1fr)); }
      .hero { grid-template-columns:1fr; }
    }
    @media (max-width:820px) {
      .layout { display:block; }
      .sidebar { position:sticky; top:0; z-index:2; height:auto; gap:12px; padding:14px 16px; border-right:0; border-bottom:1px solid var(--border); }
      .brand-mark { width:32px; height:32px; }
      .side-nav { display:flex; gap:6px; overflow-x:auto; padding-bottom:2px; }
      .side-nav a { flex:0 0 auto; min-height:32px; }
      .side-card { display:none; }
      .content { padding:18px 16px 34px; }
      h1 { font-size:28px; }
      .metric-grid { grid-template-columns:repeat(2,minmax(0,1fr)); }
      .hero-main, .radar, .panel { padding:16px; }
      .hero-main { gap:38px; min-height:auto; }
    }
    @media (max-width:520px) {
      .metric-grid { grid-template-columns:1fr; }
      .radar-grid { grid-template-columns:1fr; }
      .recent li { grid-template-columns:1fr; }
    }
  </style>
</head>
<body>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand" aria-label="ginsights">
        <div class="brand-mark">gi</div>
        <div>
          <div class="brand-title">ginsights</div>
          <div class="brand-subtitle">offline repository insights</div>
        </div>
      </div>
      <nav class="side-nav" aria-label="Report sections">
        <a href="#pulse">Pulse</a>
        <a href="#commits">Commits</a>
        <a href="#contributors">Contributors</a>
        <a href="#code-frequency">Code frequency</a>
        <a href="#files">Files</a>
        <a href="#languages">Languages</a>
        {{if .GitHub}}<a href="#github-api">GitHub API</a>{{end}}
        <a href="#health">Health</a>
        <a href="#provenance">Provenance</a>
      </nav>
      <div class="side-card">
        <strong>Generated locally</strong>
        <div class="muted">{{.GeneratedLabel}}</div>
      </div>
    </aside>

    <main class="content">
      <section id="pulse" class="hero" aria-labelledby="repo-title">
        <div class="hero-main">
          <div>
            <h1 id="repo-title">{{.RepoName}}</h1>
            <p class="repo-path">{{.RepoPath}}</p>
          </div>
          <div class="hero-meta" aria-label="Repository context">
            <span class="chip good">local Git source</span>
            {{if .PrimaryLanguage}}<span class="chip">{{.PrimaryLanguage}}</span>{{end}}
            {{if .HealthTotal}}<span class="chip warn">{{.HealthPresent}}/{{.HealthTotal}} health signals</span>{{end}}
          </div>
        </div>
        <div class="radar" aria-label="Repository radar">
          <div>
            <div class="radar-title">Repo radar</div>
            <div class="radar-value">{{.NetLinesLabel}}</div>
            <div class="muted">net lines changed across local Git history</div>
          </div>
          {{if .TotalLineChanges}}
          <div>
            <div class="balance-track" aria-label="+{{formatInt .Totals.Additions}} additions and -{{formatInt .Totals.Deletions}} deletions">
              <span class="add" style="width:{{barWidth .Totals.Additions .TotalLineChanges}}%"></span>
              <span class="del" style="width:{{barWidth .Totals.Deletions .TotalLineChanges}}%"></span>
            </div>
          </div>
          {{end}}
          <div class="radar-grid">
            <div class="radar-stat"><strong>{{formatInt .Totals.Commits}}</strong><span class="muted">commits</span></div>
            <div class="radar-stat"><strong>{{formatInt .Totals.FilesChanged}}</strong><span class="muted">files changed</span></div>
          </div>
        </div>
      </section>

      <section class="metric-grid" aria-label="Pulse totals">
        <div class="metric-card"><div class="label">Commits</div><div class="metric">{{formatInt .Totals.Commits}}</div></div>
        <div class="metric-card"><div class="label">Authors</div><div class="metric">{{formatInt .Totals.Authors}}</div></div>
        <div class="metric-card"><div class="label">Files changed</div><div class="metric">{{formatInt .Totals.FilesChanged}}</div></div>
        <div class="metric-card"><div class="label">Lines added</div><div class="metric positive">+{{formatInt .Totals.Additions}}</div></div>
        <div class="metric-card"><div class="label">Lines deleted</div><div class="metric negative">-{{formatInt .Totals.Deletions}}</div></div>
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
            {{range .Recent}}<li><code class="hash">{{.ShortHash}}</code><span>{{.Subject}} <span class="muted">by {{.AuthorName}} on {{formatDate .Date}}</span></span></li>{{end}}
          </ul>
          {{else}}<p class="muted">No recent commits found.</p>{{end}}
        </div>
      </section>

      <section id="contributors" class="panel">
        <h2>Contributors</h2>
        {{if .Authors}}
        <div class="table-scroll">
          <table class="contributors-table">
            <thead><tr><th>Author</th><th>Commits</th><th>Additions</th><th>Deletions</th><th>Files touched</th><th>Active range</th></tr></thead>
            <tbody>{{range .Authors}}
              <tr><td><strong>{{.Name}}</strong><br><span class="muted">{{.Email}}</span></td><td>{{formatInt .Commits}}</td><td>+{{formatInt .Additions}}</td><td>-{{formatInt .Deletions}}</td><td>{{formatInt .FilesTouched}}</td><td>{{formatDate .FirstCommit}} to {{formatDate .LastCommit}}</td></tr>
            {{end}}</tbody>
          </table>
        </div>
        {{else}}<p class="muted">No contributors found.</p>{{end}}
      </section>

      <section id="code-frequency" class="panel">
        <h2>Code frequency</h2>
        {{if .WeeklyRows}}
        <div class="section-summary">{{.WeeklySummary}}</div>
        <div class="table-scroll">
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
        </div>
        {{else}}<p class="muted">No weekly stats found.</p>{{end}}
      </section>

      <section id="files" class="panel">
        <h2>Hot files</h2>
        {{if .HotFiles}}
        <div class="table-scroll">
          <table class="files-table">
            <thead><tr><th>Path</th><th>Commits</th><th>Churn</th><th>Additions</th><th>Deletions</th></tr></thead>
            <tbody>{{range .HotFiles}}
              <tr><td><code>{{.Path}}</code></td><td>{{formatInt .Commits}}</td><td><div class="barrow"><span>{{formatInt .Churn}}</span><div class="bartrack"><div class="bar" style="width:{{barWidth .Churn $.MaxFileChurn}}%"></div></div></div></td><td>+{{formatInt .Additions}}</td><td>-{{formatInt .Deletions}}</td></tr>
            {{end}}</tbody>
          </table>
        </div>
        {{else}}<p class="muted">No changed files found.</p>{{end}}
      </section>

      <section id="languages" class="panel">
        <h2>Languages</h2>
        {{if .Languages}}
        <div class="language-stack">{{range $i, $language := .Languages}}<span title="{{$language.Name}} {{formatPct $language.Percent}}" style="width:{{printf "%.3f" $language.Percent}}%; background:{{languageColor $i}}"></span>{{end}}</div>
        <div class="table-scroll">
          <table class="language-table"><thead><tr><th>Language</th><th>Bytes</th><th>Share</th></tr></thead><tbody>{{range .Languages}}<tr><td>{{.Name}}</td><td>{{.Bytes}}</td><td>{{formatPct .Percent}}</td></tr>{{end}}</tbody></table>
        </div>
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

      <section id="provenance" class="panel">
        <h2>Metric provenance</h2>
        <div class="table-scroll">
          <table class="provenance-table"><thead><tr><th>Metric</th><th>Source</th></tr></thead><tbody>{{range .Provenance}}<tr><td>{{.Metric}}</td><td><code>{{.Source}}</code></td></tr>{{end}}</tbody></table>
        </div>
        <p class="muted">GitHub Traffic metrics such as views, visitors, clones, referrers, and popular content are not derivable from local Git history.</p>
      </section>

      <footer>Generated by ginsights at {{.GeneratedLabel}}. Static data: <a href="data.json">data.json</a>.</footer>
    </main>
  </div>
</body>
</html>`

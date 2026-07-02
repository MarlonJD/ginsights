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
	HeatmapMonths    []heatmapMonth
	HeatmapWeeks     int
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

type heatmapMonth struct {
	Label  string
	Column int
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
	v.HeatmapDays, v.HeatmapMonths, v.HeatmapWeeks, v.HeatmapSummary = buildHeatmapDays(snap.Daily, snap.GeneratedAt)
	v.WeeklyRows, v.WeeklySummary = buildWeeklyRows(snap.Weekly, v.MaxWeeklyLines)
	return v
}

func buildHeatmapDays(daily []analyze.DayStat, anchor time.Time) ([]heatmapDay, []heatmapMonth, int, string) {
	counts := map[time.Time]int{}
	var maxDate time.Time
	maxCommits := 0
	for _, day := range daily {
		if day.Date.IsZero() {
			continue
		}
		date := dayStart(day.Date)
		counts[date] += day.Commits
		if maxDate.IsZero() || date.After(maxDate) {
			maxDate = date
		}
		if counts[date] > maxCommits {
			maxCommits = counts[date]
		}
	}
	if anchor.IsZero() {
		anchor = maxDate
	}
	if anchor.IsZero() {
		return nil, nil, 0, ""
	}

	const weeks = 53
	end := weekStart(anchor).AddDate(0, 0, 6)
	start := end.AddDate(0, 0, -(weeks*7 - 1))
	out := make([]heatmapDay, 0, int(end.Sub(start).Hours()/24)+1)
	months := make([]heatmapMonth, 0, 13)
	activeDays := 0
	totalCommits := 0
	index := 0
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		if date.Day() == 1 {
			months = append(months, heatmapMonth{
				Label:  date.Format("Jan"),
				Column: index/7 + 2,
			})
		}
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
		index++
	}
	return out, months, weeks, fmt.Sprintf("%s active %s · %s in the last year", formatInt(activeDays), plural(activeDays, "day", "days"), commitCountLabel(totalCommits))
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
      --bg:#eef2ef;
      --paper:#fffffe;
      --paper-soft:#f7f9f7;
      --ink:#1d2521;
      --muted:#64706b;
      --quiet:#8a9691;
      --line:#d8e0da;
      --line-strong:#b8c5bd;
      --accent:#0b6bcb;
      --accent-soft:#e6f1fb;
      --good:#1f7a4f;
      --good-soft:#dff3e8;
      --bad:#b33b35;
      --bad-soft:#f8e4e2;
      --amber:#9a6a12;
      --subtle:#edf1ed;
      --shadow:0 24px 70px rgba(28,39,34,.10);
      --heat0:#e6ebe7;
      --heat1:#b8e6c5;
      --heat2:#67c987;
      --heat3:#2f9d63;
      --heat4:#176b45;
    }
    * { box-sizing:border-box; }
    html { scroll-behavior:smooth; }
    body {
      margin:0;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Helvetica Neue",Arial,sans-serif;
      font-size:14px;
      line-height:1.45;
      letter-spacing:0;
      color:var(--ink);
      background:
        linear-gradient(90deg,rgba(29,37,33,.045) 1px,transparent 1px),
        linear-gradient(180deg,rgba(29,37,33,.04) 1px,transparent 1px),
        var(--bg);
      background-size:48px 48px;
    }
    a { color:var(--accent); text-decoration:none; }
    a:focus-visible, button:focus-visible { outline:2px solid var(--accent); outline-offset:2px; }
    h1, h2, h3, p { margin-top:0; }
    h1 { margin-bottom:10px; font-size:clamp(36px,4.25vw,60px); line-height:1; font-weight:730; letter-spacing:0; overflow-wrap:anywhere; }
    h2 { margin:0; font-size:17px; line-height:1.25; font-weight:720; letter-spacing:0; }
    h3 { margin:0 0 8px; font-size:13px; line-height:1.35; font-weight:720; letter-spacing:0; }
    code { border:1px solid var(--line); border-radius:6px; padding:2px 6px; background:var(--paper-soft); color:var(--ink); font-size:.92em; }
    .page { width:min(100% - 32px,1500px); margin:0 auto; padding:18px 0 44px; }
    .topbar {
      position:sticky;
      top:0;
      z-index:4;
      display:grid;
      grid-template-columns:auto minmax(0,1fr);
      align-items:center;
      gap:18px;
      min-height:64px;
      margin-bottom:18px;
      padding:10px 0;
      background:rgba(238,242,239,.86);
      backdrop-filter:saturate(140%) blur(14px);
    }
    .brand { display:flex; align-items:center; gap:10px; min-width:0; color:var(--ink); }
    .brand-mark { display:grid; place-items:center; width:36px; height:36px; border-radius:8px; background:var(--ink); color:#fff; font-weight:850; }
    .brand-title { font-size:17px; line-height:1; font-weight:780; }
    .brand-subtitle { margin-top:3px; color:var(--muted); font-size:12px; white-space:nowrap; }
    .topnav { display:flex; justify-content:flex-end; gap:6px; overflow-x:auto; padding:3px 0; }
    .topnav a { flex:0 0 auto; min-height:32px; padding:7px 10px; border:1px solid transparent; border-radius:8px; color:var(--muted); font-size:13px; font-weight:690; }
    .topnav a:hover { color:var(--ink); background:rgba(255,255,255,.58); border-color:var(--line); }
    section { scroll-margin-top:86px; }
    .atlas {
      overflow:hidden;
      border:1px solid var(--line-strong);
      border-radius:8px;
      background:var(--paper);
      box-shadow:var(--shadow);
    }
    .atlas-main { display:grid; grid-template-columns:minmax(0,1.12fr) minmax(380px,.88fr); min-height:300px; }
    .atlas-copy { display:flex; flex-direction:column; justify-content:space-between; gap:34px; padding:32px 36px 26px; min-width:0; }
    .kicker { margin:0 0 16px; color:var(--accent); font-size:12px; font-weight:760; letter-spacing:.08em; text-transform:uppercase; }
    .repo-path { margin:0; max-width:900px; color:var(--muted); font-size:14px; overflow-wrap:anywhere; }
    .chip-row { display:flex; flex-wrap:wrap; gap:8px; }
    .chip { display:inline-flex; align-items:center; gap:7px; min-height:30px; padding:6px 9px; border:1px solid var(--line); border-radius:8px; background:rgba(255,255,255,.74); color:var(--muted); font-size:12px; font-weight:700; }
    .chip::before { content:""; width:8px; height:8px; border-radius:50%; background:var(--accent); flex:0 0 auto; }
    .chip.local::before { background:var(--good); }
    .chip.health::before { background:var(--amber); }
    .atlas-signal { display:grid; align-content:center; gap:16px; padding:32px 36px; border-left:1px solid var(--line); background:linear-gradient(180deg,#fbfcfb 0,#f5f8f6 100%); }
    .signal-label { color:var(--muted); font-size:12px; font-weight:760; letter-spacing:.08em; text-transform:uppercase; }
    .signal-value { margin:5px 0 4px; font-size:clamp(38px,4.6vw,62px); line-height:.98; font-weight:760; letter-spacing:0; }
    .signal-value.positive, .stat-value.positive, .net.positive, .delta.additions { color:var(--good); }
    .signal-value.negative, .stat-value.negative, .net.negative, .delta.deletions { color:var(--bad); }
    .signal-value.neutral, .net.neutral { color:var(--muted); }
    .signal-note, .muted, .label { color:var(--muted); font-size:13px; }
    .balance-track { display:flex; height:13px; overflow:hidden; border-radius:999px; background:var(--subtle); box-shadow:inset 0 0 0 1px rgba(29,37,33,.06); }
    .balance-track span { display:block; height:100%; }
    .balance-track .add { background:var(--good); }
    .balance-track .del { background:var(--bad); }
    .stat-matrix { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); margin:0; border:1px solid var(--line); border-radius:8px; overflow:hidden; background:rgba(255,255,255,.7); }
    .stat-cell { min-width:0; padding:13px 14px; border-right:1px solid var(--line); }
    .stat-cell:nth-child(3n) { border-right:0; }
    .stat-cell:nth-child(n+4) { border-top:1px solid var(--line); }
    .stat-cell dt { color:var(--muted); font-size:12px; font-weight:760; }
    .stat-cell dd { margin:5px 0 0; color:var(--ink); font-size:19px; line-height:1.1; font-weight:740; overflow-wrap:anywhere; }
    .atlas-activity { border-top:1px solid var(--line); padding:24px 36px 30px; background:#fbfcfb; }
    .activity-map { min-width:0; }
    .section-head { display:flex; align-items:flex-end; justify-content:space-between; gap:18px; margin-bottom:14px; }
    .section-summary { margin:4px 0 0; color:var(--muted); font-size:13px; }
    .heatmap-shell { --heat-size:14px; --heat-gap:4px; --heat-label-width:32px; --heat-column-width:var(--heat-size); display:grid; gap:10px; }
    .heatmap-scroller { max-width:100%; overflow-x:auto; padding:2px 0 5px; }
    .heatmap-calendar { width:max-content; min-width:100%; }
    .heatmap-months { display:grid; gap:var(--heat-gap); margin:0 0 4px; color:var(--quiet); font-size:12px; line-height:16px; }
    .heatmap-months span { min-width:0; white-space:nowrap; }
    .heatmap-body { display:flex; align-items:flex-start; gap:var(--heat-gap); max-width:100%; }
    .heatmap-weekdays { display:grid; grid-template-rows:repeat(7,var(--heat-size)); gap:var(--heat-gap); flex:0 0 var(--heat-label-width); color:var(--quiet); font-size:10px; line-height:var(--heat-size); }
    .heatmap-grid { display:grid; grid-auto-flow:column; grid-template-rows:repeat(7,var(--heat-size)); grid-auto-columns:var(--heat-column-width); gap:var(--heat-gap); width:max-content; }
    .heat { width:100%; height:var(--heat-size); border-radius:3px; background:var(--heat0); box-shadow:inset 0 0 0 1px rgba(29,37,33,.07); }
    .heat.l0 { background:var(--heat0); }
    .heat.l1 { background:var(--heat1); }
    .heat.l2 { background:var(--heat2); }
    .heat.l3 { background:var(--heat3); }
    .heat.l4 { background:var(--heat4); }
    .heatmap-legend { display:flex; align-items:center; justify-content:flex-end; gap:5px; color:var(--muted); font-size:12px; max-width:100%; }
    .heatmap-legend .heat { display:inline-block; flex:0 0 auto; width:var(--heat-size); height:var(--heat-size); }
    .change-tape { margin:18px 0; padding:18px 20px; border:1px solid var(--line); border-radius:8px; background:var(--paper); box-shadow:0 1px 0 rgba(28,39,34,.04); }
    .change-grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:0; margin-top:12px; border:1px solid var(--line); border-radius:8px; overflow:hidden; }
    .change-item { min-width:0; padding:12px 14px; border-right:1px solid var(--line); background:var(--paper-soft); }
    .change-item:nth-child(4n) { border-right:0; }
    .change-item:nth-child(n+5) { display:none; }
    .change-hash { display:inline-flex; margin-bottom:8px; color:var(--accent); font-size:12px; line-height:1; font-weight:720; }
    .change-subject { display:block; color:var(--ink); font-size:13px; line-height:1.32; font-weight:650; overflow-wrap:anywhere; }
    .change-meta { display:block; margin-top:5px; color:var(--muted); font-size:12px; line-height:1.3; }
    .workspace { display:grid; grid-template-columns:minmax(0,.86fr) minmax(0,1.14fr); gap:18px; margin:18px 0; }
    .stack { display:grid; gap:18px; align-content:start; }
    .panel { padding:20px; border:1px solid var(--line); border-radius:8px; background:var(--paper); box-shadow:0 1px 0 rgba(28,39,34,.04); }
    .panel.wide { margin-bottom:18px; }
    .recent { display:grid; gap:0; padding:0; margin:0; list-style:none; }
    .recent li { display:grid; grid-template-columns:78px minmax(0,1fr); gap:14px; align-items:start; padding:12px 0; border-bottom:1px solid var(--line); }
    .recent li:last-child { border-bottom:0; padding-bottom:0; }
    .hash { display:inline-flex; width:max-content; color:var(--accent); font-size:12px; line-height:1; font-weight:790; }
    .commit-subject { display:block; color:var(--ink); font-size:13px; line-height:1.35; font-weight:650; overflow-wrap:anywhere; }
    .commit-meta { display:block; margin-top:3px; color:var(--muted); font-size:12px; line-height:1.35; }
    .table-scroll { width:100%; overflow-x:auto; border:1px solid var(--line); border-radius:8px; }
    table { width:100%; border-collapse:collapse; font-size:13px; }
    th, td { padding:10px 9px; border-bottom:1px solid var(--line); text-align:left; vertical-align:top; }
    th { color:var(--muted); background:var(--paper-soft); font-weight:760; }
    tr:last-child td { border-bottom:0; }
    .table-scroll table { min-width:680px; }
    .contributors-table { min-width:820px; }
    .files-table { min-width:760px; }
    .language-table { min-width:520px; }
    .provenance-table { min-width:560px; }
    .code-frequency-table th:nth-child(3), .code-frequency-table td:nth-child(3),
    .code-frequency-table th:nth-child(5), .code-frequency-table td:nth-child(5),
    .code-frequency-table th:nth-child(6), .code-frequency-table td:nth-child(6) { text-align:right; }
    .code-frequency-table .net { font-weight:760; font-variant-numeric:tabular-nums; white-space:nowrap; }
    .frequency-cell { min-width:190px; }
    .frequency-bars { display:flex; align-items:center; height:14px; width:100%; overflow:hidden; border-radius:999px; background:var(--subtle); }
    .frequency-bar { display:block; height:100%; min-width:0; }
    .frequency-bar.additions { background:var(--good); }
    .frequency-bar.deletions { background:var(--bad); }
    .delta { font-variant-numeric:tabular-nums; white-space:nowrap; }
    .barrow { display:flex; align-items:center; gap:10px; min-width:180px; }
    .bartrack { flex:1; height:10px; background:var(--subtle); border-radius:999px; overflow:hidden; }
    .bar { height:100%; background:var(--accent); border-radius:999px; }
    .language-stack { display:flex; height:16px; border-radius:999px; overflow:hidden; background:var(--subtle); margin:0 0 13px; }
    .language-stack span { display:block; min-width:1px; background:var(--bar); border-right:1px solid rgba(255,255,255,.7); }
    .health-list { display:grid; border:1px solid var(--line); border-radius:8px; overflow:hidden; }
    .health-row { display:grid; grid-template-columns:auto minmax(0,1fr); gap:10px; align-items:start; padding:11px 12px; border-bottom:1px solid var(--line); background:var(--paper); }
    .health-row:last-child { border-bottom:0; }
    .pill { display:inline-flex; align-items:center; justify-content:center; min-width:64px; border-radius:999px; padding:3px 8px; font-size:12px; font-weight:780; }
    .pill.ok { color:var(--good); background:var(--good-soft); }
    .pill.missing { color:var(--bad); background:var(--bad-soft); }
    .api-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(150px,1fr)); gap:10px; }
    .api-metric { min-width:0; padding:13px; border:1px solid var(--line); border-radius:8px; background:var(--paper-soft); }
    .metric { color:var(--ink); font-size:30px; line-height:1; font-weight:830; letter-spacing:0; }
    .warning-list { color:var(--muted); font-size:13px; margin:12px 0 0; }
    .empty { margin:0; color:var(--muted); }
    footer { margin-top:34px; color:var(--muted); font-size:12px; }
    @media (min-width:1121px) {
      .heatmap-shell { --heat-column-width:minmax(0,1fr); }
      .heatmap-calendar, .heatmap-grid { width:100%; }
    }
    @media (min-width:1240px) {
      .heatmap-shell { --heat-size:17px; --heat-gap:5px; --heat-label-width:34px; }
    }
    @media (min-width:1440px) {
      .heatmap-shell { --heat-size:20px; --heat-gap:5px; --heat-label-width:34px; }
    }
    @media (max-width:1120px) {
      .atlas-main, .workspace { grid-template-columns:1fr; }
      .atlas-signal { border-left:0; border-top:1px solid var(--line); }
      .change-grid { grid-template-columns:repeat(2,minmax(0,1fr)); }
      .change-item:nth-child(2n) { border-right:0; }
      .change-item:nth-child(n+3) { border-top:1px solid var(--line); }
    }
    @media (max-width:820px) {
      .page { width:min(100% - 24px,1500px); padding-top:10px; }
      .topbar { grid-template-columns:1fr; gap:10px; min-height:0; }
      .topnav { justify-content:flex-start; }
      .atlas-copy, .atlas-signal, .atlas-activity, .panel { padding:18px; }
      .atlas-main { min-height:0; }
      h1 { font-size:34px; line-height:1.03; }
      .signal-value { font-size:44px; }
      .stat-matrix { grid-template-columns:repeat(2,minmax(0,1fr)); }
      .stat-cell:nth-child(3n) { border-right:1px solid var(--line); }
      .stat-cell:nth-child(2n) { border-right:0; }
      .stat-cell:nth-child(n+3) { border-top:1px solid var(--line); }
      .section-head { align-items:flex-start; flex-direction:column; gap:4px; }
      .recent li { grid-template-columns:1fr; gap:6px; }
    }
    @media (max-width:560px) {
      .stat-matrix { grid-template-columns:repeat(2,minmax(0,1fr)); }
      .change-grid { grid-template-columns:1fr; }
      .change-item { border-right:0; }
      .change-item:nth-child(n+2) { border-top:1px solid var(--line); }
    }
  </style>
</head>
<body>
  <div class="page">
    <header class="topbar">
      <a class="brand" href="#pulse" aria-label="ginsights">
        <span class="brand-mark">gi</span>
        <span>
          <span class="brand-title">ginsights</span>
          <span class="brand-subtitle">offline repository insights</span>
        </span>
      </a>
      <nav class="topnav" aria-label="Report sections">
        <a href="#pulse">Atlas</a>
        <a href="#commits">Activity</a>
        <a href="#code-frequency">Frequency</a>
        <a href="#contributors">Contributors</a>
        <a href="#files">Files</a>
        <a href="#languages">Languages</a>
        {{if .GitHub}}<a href="#github-api">GitHub API</a>{{end}}
        <a href="#health">Health</a>
        <a href="#provenance">Provenance</a>
      </nav>
    </header>

    <main>
      <section id="pulse" class="atlas" aria-labelledby="repo-title">
        <div class="atlas-main">
          <div class="atlas-copy">
            <div>
              <p class="kicker">Repository atlas</p>
              <h1 id="repo-title">{{.RepoName}}</h1>
              <p class="repo-path">{{.RepoPath}}</p>
            </div>
            <div class="chip-row" aria-label="Repository context">
              <span class="chip local">local Git source</span>
              {{if .PrimaryLanguage}}<span class="chip">{{.PrimaryLanguage}}</span>{{end}}
              {{if .HealthTotal}}<span class="chip health">{{.HealthPresent}}/{{.HealthTotal}} health signals</span>{{end}}
              <span class="chip">generated locally</span>
            </div>
          </div>

          <div class="atlas-signal" aria-label="Net local change">
            <div>
              <div class="signal-label">Net local change</div>
              <div class="signal-value {{.NetLinesClass}}">{{.NetLinesLabel}}</div>
              <div class="signal-note">lines changed across local Git history</div>
            </div>
            {{if .TotalLineChanges}}
            <div class="balance-track" aria-label="+{{formatInt .Totals.Additions}} additions and -{{formatInt .Totals.Deletions}} deletions">
              <span class="add" style="width:{{barWidth .Totals.Additions .TotalLineChanges}}%"></span>
              <span class="del" style="width:{{barWidth .Totals.Deletions .TotalLineChanges}}%"></span>
            </div>
            {{end}}
            <dl class="stat-matrix">
              <div class="stat-cell"><dt>Commits</dt><dd>{{formatInt .Totals.Commits}}</dd></div>
              <div class="stat-cell"><dt>Authors</dt><dd>{{formatInt .Totals.Authors}}</dd></div>
              <div class="stat-cell"><dt>Files changed</dt><dd>{{formatInt .Totals.FilesChanged}}</dd></div>
              <div class="stat-cell"><dt>Lines added</dt><dd class="stat-value positive">+{{formatInt .Totals.Additions}}</dd></div>
              <div class="stat-cell"><dt>Lines deleted</dt><dd class="stat-value negative">-{{formatInt .Totals.Deletions}}</dd></div>
              {{if .HealthTotal}}<div class="stat-cell"><dt>Health</dt><dd>{{.HealthPresent}}/{{.HealthTotal}}</dd></div>{{end}}
            </dl>
          </div>
        </div>

        <div id="commits" class="atlas-activity">
          <div class="section-head">
            <div>
              <h2>Commit activity</h2>
              {{if .HeatmapSummary}}<p class="section-summary">{{.HeatmapSummary}}</p>{{end}}
            </div>
          </div>
          <div class="activity-map">
            {{if .HeatmapDays}}
            <div class="heatmap-shell">
              <div class="heatmap-scroller">
                <div class="heatmap-calendar">
                  <div class="heatmap-months" style="grid-template-columns:var(--heat-label-width) repeat({{.HeatmapWeeks}},var(--heat-column-width))" aria-hidden="true">
                    {{range .HeatmapMonths}}<span style="grid-column:{{.Column}} / span 4">{{.Label}}</span>{{end}}
                  </div>
                  <div class="heatmap-body">
                    <div class="heatmap-weekdays" aria-hidden="true"><span>Mon</span><span></span><span>Wed</span><span></span><span>Fri</span><span></span><span></span></div>
                    <div class="heatmap-grid" aria-label="commit activity heatmap, {{len .HeatmapDays}} days">
                      {{range .HeatmapDays}}<span title="{{.Tooltip}}" aria-label="{{.Tooltip}}" class="heat {{.Class}}"></span>{{end}}
                    </div>
                  </div>
                </div>
              </div>
              <div class="heatmap-legend" aria-label="commit intensity legend"><span>Less</span><span class="heat l0"></span><span class="heat l1"></span><span class="heat l2"></span><span class="heat l3"></span><span class="heat l4"></span><span>More</span></div>
            </div>
            {{else}}<p class="empty">No commits found.</p>{{end}}
          </div>
        </div>
      </section>

      <section class="change-tape" aria-labelledby="latest-changes-title">
        <div class="section-head">
          <div>
            <h2 id="latest-changes-title">Latest changes</h2>
            <p class="section-summary">Most recent commits in local Git history</p>
          </div>
        </div>
        {{if .Recent}}
        <div class="change-grid">
          {{range .Recent}}<div class="change-item"><code class="change-hash">{{.ShortHash}}</code><span class="change-subject">{{.Subject}}</span><span class="change-meta">{{.AuthorName}} · {{formatDate .Date}}</span></div>{{end}}
        </div>
        {{else}}<p class="empty">No recent commits found.</p>{{end}}
      </section>

      <div class="workspace">
        <div class="stack">
          <section id="languages" class="panel">
            <div class="section-head"><h2>Languages</h2></div>
            {{if .Languages}}
            <div class="language-stack">{{range $i, $language := .Languages}}<span title="{{$language.Name}} {{formatPct $language.Percent}}" style="width:{{printf "%.3f" $language.Percent}}%; --bar:{{languageColor $i}}"></span>{{end}}</div>
            <div class="table-scroll">
              <table class="language-table"><thead><tr><th>Language</th><th>Bytes</th><th>Share</th></tr></thead><tbody>{{range .Languages}}<tr><td>{{.Name}}</td><td>{{.Bytes}}</td><td>{{formatPct .Percent}}</td></tr>{{end}}</tbody></table>
            </div>
            {{else}}<p class="empty">No known language files detected.</p>{{end}}
          </section>
        </div>

        <div class="stack">
          <section id="code-frequency" class="panel">
            <div class="section-head">
              <div>
                <h2>Code frequency</h2>
                {{if .WeeklySummary}}<p class="section-summary">{{.WeeklySummary}}</p>{{end}}
              </div>
            </div>
            {{if .WeeklyRows}}
            <div class="table-scroll">
              <table class="code-frequency-table">
                <thead><tr><th>Week</th><th>Commits</th><th>Net</th><th>Change mix</th><th>Additions</th><th>Deletions</th></tr></thead>
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
            {{else}}<p class="empty">No weekly stats found.</p>{{end}}
          </section>
        </div>
      </div>

      <section id="contributors" class="panel wide">
        <div class="section-head"><h2>Contributors</h2></div>
        {{if .Authors}}
        <div class="table-scroll">
          <table class="contributors-table">
            <thead><tr><th>Author</th><th>Commits</th><th>Additions</th><th>Deletions</th><th>Files touched</th><th>Active range</th></tr></thead>
            <tbody>{{range .Authors}}
              <tr><td><strong>{{.Name}}</strong><br><span class="muted">{{.Email}}</span></td><td>{{formatInt .Commits}}</td><td>+{{formatInt .Additions}}</td><td>-{{formatInt .Deletions}}</td><td>{{formatInt .FilesTouched}}</td><td>{{formatDate .FirstCommit}} to {{formatDate .LastCommit}}</td></tr>
            {{end}}</tbody>
          </table>
        </div>
        {{else}}<p class="empty">No contributors found.</p>{{end}}
      </section>

      <section id="files" class="panel wide">
        <div class="section-head"><h2>Hot files</h2></div>
        {{if .HotFiles}}
        <div class="table-scroll">
          <table class="files-table">
            <thead><tr><th>Path</th><th>Commits</th><th>Churn</th><th>Additions</th><th>Deletions</th></tr></thead>
            <tbody>{{range .HotFiles}}
              <tr><td><code>{{.Path}}</code></td><td>{{formatInt .Commits}}</td><td><div class="barrow"><span>{{formatInt .Churn}}</span><div class="bartrack"><div class="bar" style="width:{{barWidth .Churn $.MaxFileChurn}}%"></div></div></div></td><td>+{{formatInt .Additions}}</td><td>-{{formatInt .Deletions}}</td></tr>
            {{end}}</tbody>
          </table>
        </div>
        {{else}}<p class="empty">No changed files found.</p>{{end}}
      </section>

{{if .GitHub}}
      <section id="github-api" class="panel wide">
        <div class="section-head">
          <div>
            <h2>GitHub API</h2>
            <p class="section-summary"><code>github_api</code> data for {{.GitHub.Repository}}</p>
          </div>
        </div>
        {{if .GitHub.Error}}
        <p class="empty">{{.GitHub.Error}}</p>
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

      <section id="health" class="panel wide">
        <div class="section-head"><h2>Repository health</h2></div>
        <div class="health-list">{{range .Health}}
          <div class="health-row"><span class="pill {{statusClass .Present}}">{{statusText .Present}}</span><span><strong>{{.Name}}</strong><br><span class="muted">{{.Detail}}</span></span></div>
        {{end}}</div>
      </section>

      <section id="provenance" class="panel wide">
        <div class="section-head"><h2>Metric provenance</h2></div>
        <div class="table-scroll">
          <table class="provenance-table"><thead><tr><th>Metric</th><th>Source</th></tr></thead><tbody>{{range .Provenance}}<tr><td>{{.Metric}}</td><td><code>{{.Source}}</code></td></tr>{{end}}</tbody></table>
        </div>
        <p class="section-summary">GitHub Traffic metrics such as views, visitors, clones, referrers, and popular content are not derivable from local Git history.</p>
      </section>

      <footer>Generated by ginsights at {{.GeneratedLabel}}. Static data: <a href="data.json">data.json</a>.</footer>
    </main>
  </div>
</body>
</html>`

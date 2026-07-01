# Design

## UI direction

Familiar GitHub-style layout without copying GitHub branding:

- top repo header;
- left or horizontal insight navigation;
- stat cards;
- contributor table;
- code-frequency bars;
- commit activity heatmap;
- hot files table;
- language bar;
- repo health checklist.

## Visual constraints

- No frontend build step in MVP.
- Inline CSS is acceptable until the UI becomes large.
- Prefer semantic HTML and simple SVG/CSS charts.
- Every chart should have a nearby table or text summary for accessibility.

## Pages/tabs

1. Pulse
2. Contributors
3. Commits
4. Code frequency
5. Files
6. Languages
7. Health

## Future polish

- sticky nav;
- branch/date selector;
- dark mode;
- shareable static report route fragments;
- screenshot verification fixtures.

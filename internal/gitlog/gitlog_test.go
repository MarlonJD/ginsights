package gitlog

import "testing"

func TestParseLog(t *testing.T) {
	input := []byte("\x1eabc\x1fAda\x1fada@example.com\x1f2026-07-01T12:00:00+03:00\x1fInitial commit\n" +
		"10\t2\tmain.go\n" +
		"-\t-\timage.png\n" +
		"\x1edef\x1fLinus\x1flinus@example.com\x1f2026-07-02T12:00:00+03:00\x1fSecond\n" +
		"3\t4\tREADME.md\n")

	commits, err := parseLog(input)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(commits), 2; got != want {
		t.Fatalf("len(commits) = %d, want %d", got, want)
	}
	if got, want := commits[0].Files[0].Path, "main.go"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if commits[0].Files[1].Additions != 0 || commits[0].Files[1].Deletions != 0 {
		t.Fatalf("binary file stats should become zero: %+v", commits[0].Files[1])
	}
}

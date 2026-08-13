package csvformula

import (
	"encoding/csv"
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "plain", in: "SF123", want: "SF123"},
		{name: "numeric", in: "42", want: "42"},
		{name: "equals", in: "=HYPERLINK(\"http://evil\")", want: "'=HYPERLINK(\"http://evil\")"},
		{name: "plus", in: "+cmd|' /C calc", want: "'+cmd|' /C calc"},
		{name: "minus", in: "-2+3+cmd", want: "'-2+3+cmd"},
		{name: "at", in: "@SUM(A1:A10)", want: "'@SUM(A1:A10)"},
		{name: "tab", in: "\t=cmd", want: "'\t=cmd"},
		{name: "cr", in: "\r=cmd", want: "'\r=cmd"},
		{name: "already quoted", in: "'=CMD", want: "'=CMD"},
		{name: "internal equals", in: "a=b", want: "a=b"},
		{name: "space then equals", in: " =1+1", want: " =1+1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Sanitize(tc.in)
			if got != tc.want {
				t.Fatalf("Sanitize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSanitizeRow(t *testing.T) {
	t.Parallel()

	if got := SanitizeRow(nil); got != nil {
		t.Fatalf("nil row: %#v", got)
	}
	if got := SanitizeRow([]string{}); len(got) != 0 {
		t.Fatalf("empty row: %#v", got)
	}

	in := []string{"ok", "=1+1", "plain"}
	got := SanitizeRow(in)
	want := []string{"ok", "'=1+1", "plain"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d]=%q want %q", i, got[i], want[i])
		}
	}
	if in[1] != "=1+1" {
		t.Fatalf("SanitizeRow must not mutate input, got %q", in[1])
	}
}

func TestSanitizeRow_CSVRoundTrip(t *testing.T) {
	t.Parallel()

	want := SanitizeRow([]string{
		"ok",
		`=HYPERLINK("http://evil")`,
		"+cmd",
		"-1+2",
		"@SUM(A1)",
		"\tcmd",
		"\r=x",
	})

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	if err := w.Write(want); err != nil {
		t.Fatal(err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatal(err)
	}

	got, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("rows=%v", got)
	}
	for i := range want {
		if got[0][i] != want[i] {
			t.Fatalf("[%d]=%q want %q", i, got[0][i], want[i])
		}
		if want[i] != "ok" && (len(want[i]) == 0 || want[i][0] != '\'') {
			t.Fatalf("[%d] missing leading quote: %q", i, want[i])
		}
	}
}

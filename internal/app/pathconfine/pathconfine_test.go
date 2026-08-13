package pathconfine

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestJoinUnder_RejectsWindowsAbsUNCAndDotDot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	if _, err := JoinUnder(root, `C:\Windows\System32`); err == nil {
		t.Fatal("expected reject C:\\...")
	}
	if _, err := JoinUnder(root, `C:/Windows/System32`); err == nil {
		t.Fatal("expected reject C:/...")
	}
	if _, err := JoinUnder(root, `\\server\share\file`); err == nil {
		t.Fatal("expected reject UNC")
	}
	if _, err := JoinUnder(root, `//server/share/file`); err == nil {
		t.Fatal("expected reject // UNC")
	}
	if _, err := JoinUnder(root, `..\..`); err == nil {
		t.Fatal("expected reject ..\\..")
	}
	if _, err := JoinUnder(root, `foo\..\..\bar`); err == nil {
		t.Fatal("expected reject nested ..")
	}

	got, err := JoinUnder(root, "covers")
	if err != nil {
		t.Fatalf("relative covers: %v", err)
	}
	want, err := filepath.Abs(filepath.Join(root, "covers"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("covers = %q, want %q", got, want)
	}
}

func TestConfine_StaysInsideRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	inside := filepath.Join(root, "a", "b.csv")
	got, err := Confine(root, inside)
	if err != nil {
		t.Fatalf("inside: %v", err)
	}
	abs, _ := filepath.Abs(inside)
	if got != abs {
		t.Fatalf("got %q want %q", got, abs)
	}

	if _, err := Confine(root, filepath.Join(root, "..", "outside")); err == nil {
		t.Fatal("expected escape via ..")
	}
}

func TestLooksAbsolute(t *testing.T) {
	t.Parallel()
	if !LooksAbsolute(`C:\Windows`) {
		t.Fatal("C:\\Windows")
	}
	if !LooksAbsolute(`\\server\share`) {
		t.Fatal("UNC")
	}
	if LooksAbsolute("covers") {
		t.Fatal("relative covers")
	}
	if LooksAbsolute(`主图`) {
		t.Fatal("relative CJK")
	}
	if runtime.GOOS == "windows" && !LooksAbsolute(`D:\abs`) {
		t.Fatal("D:\\abs")
	}
}

func TestHasDotDot(t *testing.T) {
	t.Parallel()
	if !HasDotDot(`..\..`) || !HasDotDot(`foo/../bar`) {
		t.Fatal("expected .. detection")
	}
	if HasDotDot("covers") || HasDotDot("a.b") {
		t.Fatal("false positive")
	}
}

func TestUnderRoot_MatchesConfineOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	inside := filepath.Join(root, "exports", "a.csv")
	got, err := UnderRoot(inside, root)
	if err != nil {
		t.Fatal(err)
	}
	want, err := Confine(root, inside)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("UnderRoot = %q, Confine = %q", got, want)
	}
	if _, err := UnderRoot(filepath.Join(root, "..", "outside"), root); err == nil {
		t.Fatal("expected reject")
	}
}

func TestConfine_RootItself(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got, err := Confine(root, root)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(root)
	if got != abs {
		t.Fatalf("got %q want %q", got, abs)
	}
	// Prefix trap: root + sibling suffix must not match as inside.
	if !strings.HasSuffix(root, string(os.PathSeparator)) {
		sibling := root + "-sibling"
		if _, err := Confine(root, sibling); err == nil {
			t.Fatal("separator-aware prefix must reject sibling suffix")
		}
	}
}

package gen

import (
	"testing"
)

func TestFlatcLangsForTarget(t *testing.T) {
	tests := []struct {
		target   string
		wantFlag string
		wantN    int
	}{
		{"android", "--kotlin", 1},
		{"ios", "--swift", 1},
		{"macos", "--swift", 1},
		{"web", "--ts", 1},
		{"windows", "", 0},
		{"linux", "", 0},
		{"unknown", "", 0},
	}

	for _, tt := range tests {
		langs := FlatcLangsForTarget(tt.target)
		if len(langs) != tt.wantN {
			t.Errorf("FlatcLangsForTarget(%q): got %d langs, want %d", tt.target, len(langs), tt.wantN)
			continue
		}
		if tt.wantN > 0 && langs[0].Flag != tt.wantFlag {
			t.Errorf("FlatcLangsForTarget(%q): got flag %q, want %q", tt.target, langs[0].Flag, tt.wantFlag)
		}
	}
}

func TestFlatcLangForImplLang(t *testing.T) {
	tests := []struct {
		implLang string
		wantFlag string
		wantOK   bool
	}{
		{"cpp", "--cpp", true},
		{"rust", "--rust", true},
		{"go", "--go", true},
		{"c", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		lang, ok := FlatcLangForImplLang(tt.implLang)
		if ok != tt.wantOK {
			t.Errorf("FlatcLangForImplLang(%q): got ok=%v, want %v", tt.implLang, ok, tt.wantOK)
			continue
		}
		if ok && lang.Flag != tt.wantFlag {
			t.Errorf("FlatcLangForImplLang(%q): got flag %q, want %q", tt.implLang, lang.Flag, tt.wantFlag)
		}
	}
}

func TestCollectFlatcLangs_Deduplication(t *testing.T) {
	// ios + macos should produce only one --swift entry
	langs := collectFlatcLangs([]string{"ios", "macos"}, "")
	if len(langs) != 1 {
		t.Fatalf("expected 1 deduplicated lang, got %d", len(langs))
	}
	if langs[0].Flag != "--swift" {
		t.Errorf("expected --swift, got %s", langs[0].Flag)
	}
}

func TestCollectFlatcLangs_ImplLangDedup(t *testing.T) {
	// ios target (--swift) + rust impl_lang (--rust) = 2 distinct
	langs := collectFlatcLangs([]string{"ios"}, "rust")
	if len(langs) != 2 {
		t.Fatalf("expected 2 langs, got %d", len(langs))
	}

	flags := map[string]bool{}
	for _, l := range langs {
		flags[l.Flag] = true
	}
	if !flags["--swift"] || !flags["--rust"] {
		t.Errorf("expected --swift and --rust, got %v", flags)
	}
}

func TestCollectFlatcLangs_ImplLangAlreadyCoveredByTarget(t *testing.T) {
	// If impl_lang=cpp and some hypothetical target also needed --cpp,
	// it should still deduplicate. Test with impl_lang only since no target maps to --cpp.
	langs := collectFlatcLangs([]string{}, "cpp")
	if len(langs) != 1 {
		t.Fatalf("expected 1 lang, got %d", len(langs))
	}
	if langs[0].Flag != "--cpp" {
		t.Errorf("expected --cpp, got %s", langs[0].Flag)
	}
}

func TestCollectFlatcLangs_NoLangs(t *testing.T) {
	// windows + linux targets with c impl_lang → no flatc invocations
	langs := collectFlatcLangs([]string{"windows", "linux"}, "c")
	if len(langs) != 0 {
		t.Errorf("expected 0 langs for windows+linux+c, got %d", len(langs))
	}
}

func TestCollectFlatcLangs_AllTargets(t *testing.T) {
	// All 6 targets + cpp impl should produce: --kotlin, --swift, --ts, --cpp (4 unique)
	langs := collectFlatcLangs([]string{"android", "ios", "web", "windows", "macos", "linux"}, "cpp")
	if len(langs) != 4 {
		t.Fatalf("expected 4 langs, got %d: %v", len(langs), langs)
	}

	flags := map[string]bool{}
	for _, l := range langs {
		flags[l.Flag] = true
	}
	for _, expected := range []string{"--kotlin", "--swift", "--ts", "--cpp"} {
		if !flags[expected] {
			t.Errorf("missing expected flag %s in %v", expected, flags)
		}
	}
}

func TestRunFlatc_DryRun(t *testing.T) {
	cfg := &FlatcConfig{
		FlatcPath: "/usr/bin/flatc",
		FBSFiles:  []string{"/tmp/types.fbs"},
		OutputDir: "/tmp/out",
		Targets:   []string{"android", "ios"},
		ImplLang:  "cpp",
		DryRun:    true,
	}

	count, err := cfg.run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// dry-run returns 0 (no actual invocations)
	if count != 0 {
		t.Errorf("expected 0 invocations in dry-run, got %d", count)
	}
}

func TestRunFlatc_NoFBSFiles(t *testing.T) {
	cfg := &FlatcConfig{
		FlatcPath: "/usr/bin/flatc",
		FBSFiles:  []string{"/tmp/types.fbs"},
		OutputDir: "/tmp/out",
		Targets:   []string{"windows"},
		ImplLang:  "c",
		DryRun:    true,
	}

	count, err := cfg.run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 invocations for windows+c, got %d", count)
	}
}

// Helper to call RunFlatc via the config — keeps tests readable.
func (cfg *FlatcConfig) run() (int, error) {
	return RunFlatc(cfg)
}

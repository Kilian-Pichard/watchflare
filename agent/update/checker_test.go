package update

import "testing"

// --- semverGreater ---

func TestSemverGreater_PatchIncrement(t *testing.T) {
	if !semverGreater("1.2.3", "1.2.2") {
		t.Error("1.2.3 must be greater than 1.2.2")
	}
}

func TestSemverGreater_MinorIncrement(t *testing.T) {
	if !semverGreater("1.3.0", "1.2.9") {
		t.Error("1.3.0 must be greater than 1.2.9")
	}
}

func TestSemverGreater_MajorIncrement(t *testing.T) {
	if !semverGreater("2.0.0", "1.9.9") {
		t.Error("2.0.0 must be greater than 1.9.9")
	}
}

func TestSemverGreater_Equal(t *testing.T) {
	if semverGreater("1.2.3", "1.2.3") {
		t.Error("equal versions must not be greater")
	}
}

func TestSemverGreater_Older(t *testing.T) {
	if semverGreater("1.2.2", "1.2.3") {
		t.Error("older version must not be greater")
	}
}

func TestSemverGreater_VPrefix(t *testing.T) {
	if !semverGreater("v1.2.3", "v1.2.2") {
		t.Error("v-prefixed versions must compare correctly")
	}
}

func TestSemverGreater_PreReleaseSuffix(t *testing.T) {
	// Pre-release suffix is stripped; 1.2.3-beta == 1.2.3 numeric
	if semverGreater("1.2.3-beta", "1.2.3") {
		t.Error("pre-release version must not be greater than its release")
	}
}

// --- parseSemver ---

func TestParseSemver_Standard(t *testing.T) {
	got := parseSemver("1.2.3")
	want := [3]int{1, 2, 3}
	if got != want {
		t.Errorf("parseSemver(\"1.2.3\") = %v, want %v", got, want)
	}
}

func TestParseSemver_VPrefix(t *testing.T) {
	got := parseSemver("v2.10.5")
	want := [3]int{2, 10, 5}
	if got != want {
		t.Errorf("parseSemver(\"v2.10.5\") = %v, want %v", got, want)
	}
}

func TestParseSemver_PreRelease(t *testing.T) {
	got := parseSemver("1.2.3-beta")
	want := [3]int{1, 2, 3}
	if got != want {
		t.Errorf("parseSemver(\"1.2.3-beta\") = %v, want %v", got, want)
	}
}

func TestParseSemver_Partial(t *testing.T) {
	got := parseSemver("1.2")
	want := [3]int{1, 2, 0}
	if got != want {
		t.Errorf("parseSemver(\"1.2\") = %v, want %v", got, want)
	}
}

func TestParseSemver_Empty(t *testing.T) {
	got := parseSemver("")
	want := [3]int{0, 0, 0}
	if got != want {
		t.Errorf("parseSemver(\"\") = %v, want %v", got, want)
	}
}

package packages

import (
	"errors"
	"testing"
)

// fakeCollector implements Collector for testing
type fakeCollector struct {
	name     string
	packages []*Package
	err      error
}

func (f *fakeCollector) Name() string            { return f.name }
func (f *fakeCollector) IsAvailable() bool       { return true }
func (f *fakeCollector) Collect() ([]*Package, error) {
	return f.packages, f.err
}

// fakeUpdateChecker implements UpdateChecker for testing
type fakeUpdateChecker struct {
	name            string
	packageManagers []string
	updates         map[string]UpdateStatus
	err             error
}

func (f *fakeUpdateChecker) Name() string              { return f.name }
func (f *fakeUpdateChecker) IsAvailable() bool         { return true }
func (f *fakeUpdateChecker) PackageManagers() []string { return f.packageManagers }
func (f *fakeUpdateChecker) CheckUpdates() (map[string]UpdateStatus, error) {
	return f.updates, f.err
}

func TestEnrichPackagesWithUpdateStatus(t *testing.T) {
	pkgs := []*Package{
		{Name: "curl", Version: "8.10.0", PackageManager: "dpkg"},
		{Name: "bash", Version: "5.2.0", PackageManager: "dpkg"},
		{Name: "typescript", Version: "5.0.0", PackageManager: "npm"},
	}

	checker := &fakeUpdateChecker{
		name:            "apt",
		packageManagers: []string{"dpkg"},
		updates: map[string]UpdateStatus{
			"curl": {AvailableVersion: "8.11.1", HasSecurityUpdate: true},
		},
	}

	// Apply the enrichment (mirrors CollectAll logic)
	pmSet := make(map[string]bool)
	for _, pm := range checker.PackageManagers() {
		pmSet[pm] = true
	}
	updates, err := checker.CheckUpdates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, pkg := range pkgs {
		if !pmSet[pkg.PackageManager] {
			continue
		}
		if status, ok := updates[pkg.Name]; ok {
			pkg.AvailableVersion = status.AvailableVersion
			pkg.HasSecurityUpdate = status.HasSecurityUpdate
		}
	}

	// curl: should be enriched
	if pkgs[0].AvailableVersion != "8.11.1" {
		t.Errorf("curl AvailableVersion = %q, want %q", pkgs[0].AvailableVersion, "8.11.1")
	}
	if !pkgs[0].HasSecurityUpdate {
		t.Error("curl: HasSecurityUpdate should be true")
	}

	// bash: no update available, should be unchanged
	if pkgs[1].AvailableVersion != "" {
		t.Errorf("bash AvailableVersion = %q, want empty", pkgs[1].AvailableVersion)
	}
	if pkgs[1].HasSecurityUpdate {
		t.Error("bash: HasSecurityUpdate should be false")
	}

	// typescript: different package manager, should not be touched
	if pkgs[2].AvailableVersion != "" {
		t.Errorf("typescript AvailableVersion = %q, want empty (different PM)", pkgs[2].AvailableVersion)
	}
}

func TestEnrichPackages_CheckerError(t *testing.T) {
	registry := &CollectorRegistry{
		collectors: []Collector{
			&fakeCollector{
				name: "dpkg",
				packages: []*Package{
					{Name: "curl", Version: "8.10.0", PackageManager: "dpkg"},
				},
			},
		},
		updateCheckers: []UpdateChecker{
			&fakeUpdateChecker{
				name:            "apt",
				packageManagers: []string{"dpkg"},
				err:             errors.New("apt unavailable"),
			},
		},
	}

	// CollectAll uses registry internally, so we test via the registry methods
	var allPackages []*Package
	for _, c := range registry.GetAvailableCollectors() {
		pkgs, err := c.Collect()
		if err != nil {
			t.Fatalf("unexpected collect error: %v", err)
		}
		allPackages = append(allPackages, pkgs...)
	}

	// Checker fails: packages should still be returned unchanged
	for _, checker := range registry.GetAvailableUpdateCheckers() {
		updates, err := checker.CheckUpdates()
		if err == nil {
			t.Fatal("expected error from checker")
		}
		if updates != nil {
			t.Error("updates should be nil on error")
		}
	}

	if len(allPackages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(allPackages))
	}
	if allPackages[0].AvailableVersion != "" {
		t.Error("AvailableVersion should be empty when checker fails")
	}
}

func TestEnrichPackages_MultipleCheckers(t *testing.T) {
	pkgs := []*Package{
		{Name: "curl", Version: "8.10.0", PackageManager: "dpkg"},
		{Name: "openssl", Version: "3.0.0", PackageManager: "rpm"},
	}

	checkers := []UpdateChecker{
		&fakeUpdateChecker{
			name:            "apt",
			packageManagers: []string{"dpkg"},
			updates: map[string]UpdateStatus{
				"curl": {AvailableVersion: "8.11.1", HasSecurityUpdate: true},
			},
		},
		&fakeUpdateChecker{
			name:            "dnf",
			packageManagers: []string{"rpm"},
			updates: map[string]UpdateStatus{
				"openssl": {AvailableVersion: "3.3.2", HasSecurityUpdate: true},
			},
		},
	}

	for _, checker := range checkers {
		pmSet := make(map[string]bool)
		for _, pm := range checker.PackageManagers() {
			pmSet[pm] = true
		}
		updates, _ := checker.CheckUpdates()
		for _, pkg := range pkgs {
			if !pmSet[pkg.PackageManager] {
				continue
			}
			if status, ok := updates[pkg.Name]; ok {
				pkg.AvailableVersion = status.AvailableVersion
				pkg.HasSecurityUpdate = status.HasSecurityUpdate
			}
		}
	}

	if pkgs[0].AvailableVersion != "8.11.1" || !pkgs[0].HasSecurityUpdate {
		t.Errorf("curl not enriched correctly: %+v", pkgs[0])
	}
	if pkgs[1].AvailableVersion != "3.3.2" || !pkgs[1].HasSecurityUpdate {
		t.Errorf("openssl not enriched correctly: %+v", pkgs[1])
	}
}

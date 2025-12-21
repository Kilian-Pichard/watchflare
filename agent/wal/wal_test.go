package wal

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBasicOperations tests basic WAL operations
func TestBasicOperations(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	// Create WAL
	wal, err := New(walPath, 10)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Test append
	testData := []byte("test metric data")
	if err := wal.Append(testData); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Test read
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	if string(records[0]) != string(testData) {
		t.Fatalf("Data mismatch: expected %s, got %s", testData, records[0])
	}

	t.Logf("✅ Basic operations: PASS")
}

// TestTruncateNormal tests normal truncate operation (atomic rename)
func TestTruncateNormal(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	// Create WAL
	wal, err := New(walPath, 10)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append 10 records
	for i := 0; i < 10; i++ {
		data := []byte("metric data record " + string(rune('0'+i)))
		if err := wal.Append(data); err != nil {
			t.Fatalf("Failed to append record %d: %v", i, err)
		}
	}

	// Verify we have 10 records
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read before truncate: %v", err)
	}
	if len(records) != 10 {
		t.Fatalf("Expected 10 records before truncate, got %d", len(records))
	}

	// Perform truncate (should keep 50% = 5 most recent)
	if err := wal.Truncate(); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Verify truncate kept 50% most recent records
	records, err = wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read after truncate: %v", err)
	}

	if len(records) != 5 {
		t.Fatalf("Expected 5 records after truncate (50%%), got %d", len(records))
	}

	// Verify no temp file left behind
	tmpPath := walPath + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Fatalf("Temp file should not exist after successful truncate")
	}

	t.Logf("✅ Truncate normal operation: PASS")
	t.Logf("   - 10 records → 5 records (50%% kept)")
	t.Logf("   - No temp file left behind")
}

// TestTruncateAtomicity tests that WAL is never corrupted during truncate
func TestTruncateAtomicity(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	// Create WAL and add records
	wal, err := New(walPath, 10)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	for i := 0; i < 10; i++ {
		data := []byte("test data " + string(rune('0'+i)))
		if err := wal.Append(data); err != nil {
			t.Fatalf("Failed to append: %v", err)
		}
	}

	// Get file size before truncate
	sizeBefore, _ := wal.Size()
	t.Logf("WAL size before truncate: %d bytes", sizeBefore)

	// Close WAL
	wal.Close()

	// Simulate crash scenario: create a temp file as if truncate was interrupted
	tmpPath := walPath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.WriteString("interrupted truncate data")
	tmpFile.Close()

	t.Logf("Simulated crash: created orphaned temp file")

	// Reopen WAL - should cleanup temp file
	wal, err = New(walPath, 10)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	// Verify temp file was cleaned up
	if _, err := os.Stat(tmpPath); err == nil {
		t.Fatalf("Temp file should be cleaned up on restart")
	}

	// Verify original WAL is intact (all 10 records)
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read after restart: %v", err)
	}

	if len(records) != 10 {
		t.Fatalf("Expected 10 records after crash recovery, got %d", len(records))
	}

	t.Logf("✅ Truncate atomicity (crash recovery): PASS")
	t.Logf("   - Temp file cleaned up")
	t.Logf("   - Original WAL intact (10 records)")
}

// TestCleanupTempOnStartup tests that orphaned .tmp files are cleaned
func TestCleanupTempOnStartup(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")
	tmpPath := walPath + ".tmp"

	// Create WAL first
	wal, err := New(walPath, 10)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	wal.Append([]byte("initial data"))
	wal.Close()

	// Create orphaned temp file (simulating crash during truncate)
	if err := os.WriteFile(tmpPath, []byte("orphaned data"), 0644); err != nil {
		t.Fatalf("Failed to create orphaned temp file: %v", err)
	}

	// Verify temp file exists
	if _, err := os.Stat(tmpPath); err != nil {
		t.Fatalf("Temp file should exist before cleanup")
	}

	t.Logf("Created orphaned temp file: %s", tmpPath)

	// Reopen WAL - should trigger cleanup
	wal, err = New(walPath, 10)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	// Verify temp file was removed
	if _, err := os.Stat(tmpPath); err == nil {
		t.Fatalf("Temp file should be cleaned up on startup")
	}

	t.Logf("✅ Cleanup temp on startup: PASS")
	t.Logf("   - Orphaned .tmp file removed")
}

// TestClear tests WAL clear operation
func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := New(walPath, 10)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Add some data
	wal.Append([]byte("data1"))
	wal.Append([]byte("data2"))

	// Clear
	if err := wal.Clear(); err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Verify empty
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read after clear: %v", err)
	}

	if len(records) != 0 {
		t.Fatalf("Expected 0 records after clear, got %d", len(records))
	}

	t.Logf("✅ Clear operation: PASS")
}

// TestMultipleTruncates tests that multiple truncates work correctly
func TestMultipleTruncates(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := New(walPath, 10)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// First batch: 10 records
	for i := 0; i < 10; i++ {
		wal.Append([]byte("batch1-record" + string(rune('0'+i))))
	}

	// First truncate: 10 → 5
	wal.Truncate()
	records, _ := wal.ReadAll()
	if len(records) != 5 {
		t.Fatalf("After first truncate: expected 5, got %d", len(records))
	}

	// Add 10 more: 5 + 10 = 15
	for i := 0; i < 10; i++ {
		wal.Append([]byte("batch2-record" + string(rune('0'+i))))
	}

	// Second truncate: 15 → 8 (15/2 = 7, keep records[7:] = 8 records)
	wal.Truncate()
	records, _ = wal.ReadAll()
	if len(records) != 8 {
		t.Fatalf("After second truncate: expected 8, got %d", len(records))
	}

	// Add more: 8 + 6 = 14
	for i := 0; i < 6; i++ {
		wal.Append([]byte("batch3-record" + string(rune('0'+i))))
	}

	// Third truncate: 14 → 7 (14/2 = 7, keep records[7:] = 7 records)
	wal.Truncate()
	records, _ = wal.ReadAll()
	if len(records) != 7 {
		t.Fatalf("After third truncate: expected 7, got %d", len(records))
	}

	t.Logf("✅ Multiple truncates: PASS")
	t.Logf("   - Truncate 1: 10 → 5")
	t.Logf("   - Truncate 2: 15 → 8")
	t.Logf("   - Truncate 3: 14 → 7")
}

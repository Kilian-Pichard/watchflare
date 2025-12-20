package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

// WAL is a simple Write-Ahead Log for metrics persistence
// Format: [Length:4][Data:protobuf][CRC32:4] per record
// File: single append-only file with FIFO truncation
type WAL struct {
	file      *os.File
	mu        sync.Mutex
	path      string
	maxSize   int64 // Max size in bytes before FIFO truncate
}

// New creates or opens a WAL file
func New(path string, maxSizeMB int) (*WAL, error) {
	// Convert MB to bytes
	maxSize := int64(maxSizeMB) * 1024 * 1024

	// Open file with O_APPEND | O_CREATE | O_RDWR
	// No O_APPEND for header updates (we use explicit Seek)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL: %w", err)
	}

	// Seek to end for appends
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to seek to end: %w", err)
	}

	return &WAL{
		file:    file,
		path:    path,
		maxSize: maxSize,
	}, nil
}

// Append adds a metric record to the WAL
// Format: [Length:4 bytes][Data:N bytes][CRC32:4 bytes]
func (w *WAL) Append(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Calculate CRC32
	checksum := crc32.ChecksumIEEE(data)

	// Write length (4 bytes, big-endian)
	length := uint32(len(data))
	if err := binary.Write(w.file, binary.BigEndian, length); err != nil {
		return fmt.Errorf("failed to write length: %w", err)
	}

	// Write data
	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Write CRC32 (4 bytes, big-endian)
	if err := binary.Write(w.file, binary.BigEndian, checksum); err != nil {
		return fmt.Errorf("failed to write checksum: %w", err)
	}

	// Sync to disk (durability)
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	return nil
}

// ReadAll reads all records from the WAL
// Memory-safe: WAL is capped at maxSize (10 MB default), so max ~3000 metrics in memory
// V2: Consider streaming if WAL size increases significantly
func (w *WAL) ReadAll() ([][]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Seek to beginning
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to start: %w", err)
	}

	var records [][]byte

	for {
		// Read length (4 bytes)
		var length uint32
		if err := binary.Read(w.file, binary.BigEndian, &length); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read length: %w", err)
		}

		// Sanity check: max 1 MB per record
		if length > 1024*1024 {
			return nil, fmt.Errorf("invalid record length: %d bytes", length)
		}

		// Read data
		data := make([]byte, length)
		if _, err := io.ReadFull(w.file, data); err != nil {
			return nil, fmt.Errorf("failed to read data: %w", err)
		}

		// Read CRC32 (4 bytes)
		var storedChecksum uint32
		if err := binary.Read(w.file, binary.BigEndian, &storedChecksum); err != nil {
			return nil, fmt.Errorf("failed to read checksum: %w", err)
		}

		// Verify CRC32
		computedChecksum := crc32.ChecksumIEEE(data)
		if computedChecksum != storedChecksum {
			return nil, fmt.Errorf("checksum mismatch: expected %d, got %d", storedChecksum, computedChecksum)
		}

		records = append(records, data)
	}

	// Seek back to end for future appends
	if _, err := w.file.Seek(0, io.SeekEnd); err != nil {
		return nil, fmt.Errorf("failed to seek to end: %w", err)
	}

	return records, nil
}

// Clear removes all records from the WAL
func (w *WAL) Clear() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Truncate to 0
	if err := w.file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate: %w", err)
	}

	// Seek to start
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start: %w", err)
	}

	// Sync
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	return nil
}

// Size returns the current WAL file size
func (w *WAL) Size() (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	stat, err := w.file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat: %w", err)
	}

	return stat.Size(), nil
}

// Truncate performs FIFO truncation: keeps 50% most recent records
// Called when WAL exceeds maxSize
func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Read all records
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start: %w", err)
	}

	var records [][]byte
	for {
		var length uint32
		if err := binary.Read(w.file, binary.BigEndian, &length); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read length: %w", err)
		}

		if length > 1024*1024 {
			return fmt.Errorf("invalid record length: %d bytes", length)
		}

		data := make([]byte, length)
		if _, err := io.ReadFull(w.file, data); err != nil {
			return fmt.Errorf("failed to read data: %w", err)
		}

		var checksum uint32
		if err := binary.Read(w.file, binary.BigEndian, &checksum); err != nil {
			return fmt.Errorf("failed to read checksum: %w", err)
		}

		if crc32.ChecksumIEEE(data) != checksum {
			return fmt.Errorf("checksum mismatch during truncate")
		}

		records = append(records, data)
	}

	// Keep 50% most recent records
	if len(records) <= 1 {
		return nil // Nothing to truncate
	}

	keepFrom := len(records) / 2
	recentRecords := records[keepFrom:]

	// Truncate file to 0
	if err := w.file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate: %w", err)
	}

	// Seek to start
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start: %w", err)
	}

	// Rewrite recent records
	for _, data := range recentRecords {
		length := uint32(len(data))
		checksum := crc32.ChecksumIEEE(data)

		if err := binary.Write(w.file, binary.BigEndian, length); err != nil {
			return fmt.Errorf("failed to write length: %w", err)
		}

		if _, err := w.file.Write(data); err != nil {
			return fmt.Errorf("failed to write data: %w", err)
		}

		if err := binary.Write(w.file, binary.BigEndian, checksum); err != nil {
			return fmt.Errorf("failed to write checksum: %w", err)
		}
	}

	// Sync
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	// Seek to end for future appends
	if _, err := w.file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	return nil
}

// Close closes the WAL file
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Sync(); err != nil {
		return err
	}

	return w.file.Close()
}

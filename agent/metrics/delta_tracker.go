package metrics

import (
	"sync"
	"time"
)

// DeltaTracker maintains previous counter values for rate calculations
type DeltaTracker struct {
	mu sync.Mutex

	prevDiskReadBytes  uint64
	prevDiskWriteBytes uint64
	prevDiskTime       time.Time
	diskInitialized    bool

	prevNetRxBytes  uint64
	prevNetTxBytes  uint64
	prevNetTime     time.Time
	netInitialized  bool
}

// NewDeltaTracker creates a new delta tracker
func NewDeltaTracker() *DeltaTracker {
	return &DeltaTracker{}
}

// ComputeDiskIORate calculates disk read/write bytes per second from cumulative counters
func (dt *DeltaTracker) ComputeDiskIORate(readBytes, writeBytes uint64, now time.Time) (uint64, uint64) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if !dt.diskInitialized {
		dt.prevDiskReadBytes = readBytes
		dt.prevDiskWriteBytes = writeBytes
		dt.prevDiskTime = now
		dt.diskInitialized = true
		return 0, 0
	}

	elapsed := now.Sub(dt.prevDiskTime).Seconds()
	if elapsed <= 0 {
		return 0, 0
	}

	var readRate, writeRate uint64

	// Handle counter rollover (current < prev)
	if readBytes >= dt.prevDiskReadBytes {
		readRate = uint64(float64(readBytes-dt.prevDiskReadBytes) / elapsed)
	}
	if writeBytes >= dt.prevDiskWriteBytes {
		writeRate = uint64(float64(writeBytes-dt.prevDiskWriteBytes) / elapsed)
	}

	dt.prevDiskReadBytes = readBytes
	dt.prevDiskWriteBytes = writeBytes
	dt.prevDiskTime = now

	return readRate, writeRate
}

// ComputeNetworkRate calculates network RX/TX bytes per second from cumulative counters
func (dt *DeltaTracker) ComputeNetworkRate(rxBytes, txBytes uint64, now time.Time) (uint64, uint64) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if !dt.netInitialized {
		dt.prevNetRxBytes = rxBytes
		dt.prevNetTxBytes = txBytes
		dt.prevNetTime = now
		dt.netInitialized = true
		return 0, 0
	}

	elapsed := now.Sub(dt.prevNetTime).Seconds()
	if elapsed <= 0 {
		return 0, 0
	}

	var rxRate, txRate uint64

	if rxBytes >= dt.prevNetRxBytes {
		rxRate = uint64(float64(rxBytes-dt.prevNetRxBytes) / elapsed)
	}
	if txBytes >= dt.prevNetTxBytes {
		txRate = uint64(float64(txBytes-dt.prevNetTxBytes) / elapsed)
	}

	dt.prevNetRxBytes = rxBytes
	dt.prevNetTxBytes = txBytes
	dt.prevNetTime = now

	return rxRate, txRate
}

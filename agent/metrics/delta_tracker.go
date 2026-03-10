package metrics

import (
	"sync"
	"time"
)

// containerNetState tracks previous network counters for a single container
type containerNetState struct {
	prevRxBytes   uint64
	prevTxBytes   uint64
	prevTime      time.Time
	initialized   bool
}

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

	// Per-container network tracking
	containerNet map[string]*containerNetState
}

// NewDeltaTracker creates a new delta tracker
func NewDeltaTracker() *DeltaTracker {
	return &DeltaTracker{
		containerNet: make(map[string]*containerNetState),
	}
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

// ComputeContainerNetworkRate calculates per-container network rates
func (dt *DeltaTracker) ComputeContainerNetworkRate(containerID string, rxBytes, txBytes uint64, now time.Time) (uint64, uint64) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	state, exists := dt.containerNet[containerID]
	if !exists {
		dt.containerNet[containerID] = &containerNetState{
			prevRxBytes: rxBytes,
			prevTxBytes: txBytes,
			prevTime:    now,
			initialized: true,
		}
		return 0, 0
	}

	if !state.initialized {
		state.prevRxBytes = rxBytes
		state.prevTxBytes = txBytes
		state.prevTime = now
		state.initialized = true
		return 0, 0
	}

	elapsed := now.Sub(state.prevTime).Seconds()
	if elapsed <= 0 {
		return 0, 0
	}

	var rxRate, txRate uint64
	if rxBytes >= state.prevRxBytes {
		rxRate = uint64(float64(rxBytes-state.prevRxBytes) / elapsed)
	}
	if txBytes >= state.prevTxBytes {
		txRate = uint64(float64(txBytes-state.prevTxBytes) / elapsed)
	}

	state.prevRxBytes = rxBytes
	state.prevTxBytes = txBytes
	state.prevTime = now

	return rxRate, txRate
}

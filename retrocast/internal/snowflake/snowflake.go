package snowflake

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// Custom epoch: January 1, 2025 00:00:00 UTC.
const epoch int64 = 1735689600000

// Bit layout.
const (
	workerIDBits   = 5
	processIDBits  = 5
	sequenceBits   = 12

	maxWorkerID  = (1 << workerIDBits) - 1
	maxProcessID = (1 << processIDBits) - 1
	maxSequence  = (1 << sequenceBits) - 1

	workerIDShift  = sequenceBits + processIDBits
	processIDShift = sequenceBits
	timestampShift = sequenceBits + processIDBits + workerIDBits
)

// ID is a snowflake ID that marshals to/from JSON as a string.
type ID int64

func (id ID) Int64() int64 {
	return int64(id)
}

func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(id), 10))
}

func (id *ID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Try as a number for backwards compat.
		var n int64
		if nerr := json.Unmarshal(data, &n); nerr != nil {
			return fmt.Errorf("snowflake: cannot unmarshal %s: %w", string(data), err)
		}
		*id = ID(n)
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("snowflake: invalid id string %q: %w", s, err)
	}
	*id = ID(n)
	return nil
}

// Generator produces unique snowflake IDs.
type Generator struct {
	mu        sync.Mutex
	workerID  int64
	processID int64
	sequence  int64
	lastTime  int64
}

// NewGenerator creates a generator with the given worker and process IDs.
// Both must be in the range [0, 31].
func NewGenerator(workerID, processID int64) (*Generator, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, fmt.Errorf("snowflake: workerID must be between 0 and %d", maxWorkerID)
	}
	if processID < 0 || processID > maxProcessID {
		return nil, fmt.Errorf("snowflake: processID must be between 0 and %d", maxProcessID)
	}
	return &Generator{
		workerID:  workerID,
		processID: processID,
	}, nil
}

// Generate returns the next unique snowflake ID.
func (g *Generator) Generate() ID {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli() - epoch

	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			// Sequence exhausted; spin until next millisecond.
			for now <= g.lastTime {
				now = time.Now().UnixMilli() - epoch
			}
		}
	} else {
		g.sequence = 0
	}

	g.lastTime = now

	id := (now << timestampShift) |
		(g.workerID << workerIDShift) |
		(g.processID << processIDShift) |
		g.sequence

	return ID(id)
}

// ExtractTimestamp returns the wall-clock time embedded in a snowflake ID.
func ExtractTimestamp(id int64) time.Time {
	ms := (id >> timestampShift) + epoch
	return time.UnixMilli(ms)
}

package uncorsapp

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
)

const (
	historyInitialSize = 1 << 20 // 1 MB
	historyGrowFactor  = 2
)

type lineInfo struct {
	offset int64
	length int32
}

// history stores log lines in a memory-mapped temp file so the backing store
// stays off-heap while string reads are zero-copy.
type history struct {
	mu       sync.RWMutex
	file     *os.File
	data     []byte
	lines    []lineInfo // index: (offset, length) per line
	cache    []string   // string view of each line (for viewport)
	writePos int64
	capacity int64
}

func newHistory() (*history, error) {
	f, err := os.CreateTemp("", "uncors-history-*.log")
	if err != nil {
		return nil, fmt.Errorf("create history file: %w", err)
	}

	capacity := int64(historyInitialSize)
	if err := f.Truncate(capacity); err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("allocate history file: %w", err)
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(capacity),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("mmap history: %w", err)
	}

	return &history{
		file:     f,
		data:     data,
		lines:    make([]lineInfo, 0, 1024),
		cache:    make([]string, 0, 1024),
		capacity: capacity,
	}, nil
}

// AppendLine writes line to the mmap file and appends it to the string cache.
// Multi-line strings (logo, box messages) are split on '\n' so the viewport
// receives one entry per visual row.
// Must be called from the bubbletea Update goroutine.
func (h *history) AppendLine(line string) {
	line = strings.TrimRight(line, "\n")
	for subline := range strings.SplitSeq(line, "\n") {
		h.appendSingleLine(subline)
	}
}

func (h *history) appendSingleLine(line string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	content := line + "\n"

	needed := h.writePos + int64(len(content))
	if needed > h.capacity {
		err := h.grow(needed)
		if err != nil {
			return
		}
	}

	offset := h.writePos
	n := copy(h.data[offset:], content)
	lineLen := int32(n - 1)
	h.lines = append(h.lines, lineInfo{offset: offset, length: lineLen})
	h.cache = append(h.cache, string(h.data[offset:offset+int64(lineLen)]))
	h.writePos += int64(n)
}

// Lines returns the cached slice of all stored lines.
// The slice is valid until the next AppendLine call.
func (h *history) Lines() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.cache
}

// LineCount returns the total number of stored lines.
func (h *history) LineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.lines)
}

// Close unmaps memory, closes and removes the temp file.
func (h *history) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	_ = syscall.Munmap(h.data)
	name := h.file.Name()
	_ = h.file.Close()

	return os.Remove(name)
}

func (h *history) grow(needed int64) error {
	newCap := h.capacity * historyGrowFactor
	for newCap < needed {
		newCap *= historyGrowFactor
	}

	if err := syscall.Munmap(h.data); err != nil {
		return err
	}

	if err := h.file.Truncate(newCap); err != nil {
		return err
	}

	data, err := syscall.Mmap(int(h.file.Fd()), 0, int(newCap),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return err
	}

	h.data = data
	h.capacity = newCap

	return nil
}

// Code generated by http://github.com/gojuno/minimock (v3.3.12). DO NOT EDIT.

package mocks

//go:generate minimock -i io.Closer -o closer_mock.go -n CloserMock -p mocks

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
)

// CloserMock implements io.Closer
type CloserMock struct {
	t          minimock.Tester
	finishOnce sync.Once

	funcClose          func() (err error)
	inspectFuncClose   func()
	afterCloseCounter  uint64
	beforeCloseCounter uint64
	CloseMock          mCloserMockClose
}

// NewCloserMock returns a mock for io.Closer
func NewCloserMock(t minimock.Tester) *CloserMock {
	m := &CloserMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.CloseMock = mCloserMockClose{mock: m}

	t.Cleanup(m.MinimockFinish)

	return m
}

type mCloserMockClose struct {
	optional           bool
	mock               *CloserMock
	defaultExpectation *CloserMockCloseExpectation
	expectations       []*CloserMockCloseExpectation

	expectedInvocations uint64
}

// CloserMockCloseExpectation specifies expectation struct of the Closer.Close
type CloserMockCloseExpectation struct {
	mock *CloserMock

	results *CloserMockCloseResults
	Counter uint64
}

// CloserMockCloseResults contains results of the Closer.Close
type CloserMockCloseResults struct {
	err error
}

// Marks this method to be optional. The default behavior of any method with Return() is '1 or more', meaning
// the test will fail minimock's automatic final call check if the mocked method was not called at least once.
// Optional() makes method check to work in '0 or more' mode.
// It is NOT RECOMMENDED to use this option unless you really need it, as default behaviour helps to
// catch the problems when the expected method call is totally skipped during test run.
func (mmClose *mCloserMockClose) Optional() *mCloserMockClose {
	mmClose.optional = true
	return mmClose
}

// Expect sets up expected params for Closer.Close
func (mmClose *mCloserMockClose) Expect() *mCloserMockClose {
	if mmClose.mock.funcClose != nil {
		mmClose.mock.t.Fatalf("CloserMock.Close mock is already set by Set")
	}

	if mmClose.defaultExpectation == nil {
		mmClose.defaultExpectation = &CloserMockCloseExpectation{}
	}

	return mmClose
}

// Inspect accepts an inspector function that has same arguments as the Closer.Close
func (mmClose *mCloserMockClose) Inspect(f func()) *mCloserMockClose {
	if mmClose.mock.inspectFuncClose != nil {
		mmClose.mock.t.Fatalf("Inspect function is already set for CloserMock.Close")
	}

	mmClose.mock.inspectFuncClose = f

	return mmClose
}

// Return sets up results that will be returned by Closer.Close
func (mmClose *mCloserMockClose) Return(err error) *CloserMock {
	if mmClose.mock.funcClose != nil {
		mmClose.mock.t.Fatalf("CloserMock.Close mock is already set by Set")
	}

	if mmClose.defaultExpectation == nil {
		mmClose.defaultExpectation = &CloserMockCloseExpectation{mock: mmClose.mock}
	}
	mmClose.defaultExpectation.results = &CloserMockCloseResults{err}
	return mmClose.mock
}

// Set uses given function f to mock the Closer.Close method
func (mmClose *mCloserMockClose) Set(f func() (err error)) *CloserMock {
	if mmClose.defaultExpectation != nil {
		mmClose.mock.t.Fatalf("Default expectation is already set for the Closer.Close method")
	}

	if len(mmClose.expectations) > 0 {
		mmClose.mock.t.Fatalf("Some expectations are already set for the Closer.Close method")
	}

	mmClose.mock.funcClose = f
	return mmClose.mock
}

// Times sets number of times Closer.Close should be invoked
func (mmClose *mCloserMockClose) Times(n uint64) *mCloserMockClose {
	if n == 0 {
		mmClose.mock.t.Fatalf("Times of CloserMock.Close mock can not be zero")
	}
	mm_atomic.StoreUint64(&mmClose.expectedInvocations, n)
	return mmClose
}

func (mmClose *mCloserMockClose) invocationsDone() bool {
	if len(mmClose.expectations) == 0 && mmClose.defaultExpectation == nil && mmClose.mock.funcClose == nil {
		return true
	}

	totalInvocations := mm_atomic.LoadUint64(&mmClose.mock.afterCloseCounter)
	expectedInvocations := mm_atomic.LoadUint64(&mmClose.expectedInvocations)

	return totalInvocations > 0 && (expectedInvocations == 0 || expectedInvocations == totalInvocations)
}

// Close implements io.Closer
func (mmClose *CloserMock) Close() (err error) {
	mm_atomic.AddUint64(&mmClose.beforeCloseCounter, 1)
	defer mm_atomic.AddUint64(&mmClose.afterCloseCounter, 1)

	if mmClose.inspectFuncClose != nil {
		mmClose.inspectFuncClose()
	}

	if mmClose.CloseMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmClose.CloseMock.defaultExpectation.Counter, 1)

		mm_results := mmClose.CloseMock.defaultExpectation.results
		if mm_results == nil {
			mmClose.t.Fatal("No results are set for the CloserMock.Close")
		}
		return (*mm_results).err
	}
	if mmClose.funcClose != nil {
		return mmClose.funcClose()
	}
	mmClose.t.Fatalf("Unexpected call to CloserMock.Close.")
	return
}

// CloseAfterCounter returns a count of finished CloserMock.Close invocations
func (mmClose *CloserMock) CloseAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmClose.afterCloseCounter)
}

// CloseBeforeCounter returns a count of CloserMock.Close invocations
func (mmClose *CloserMock) CloseBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmClose.beforeCloseCounter)
}

// MinimockCloseDone returns true if the count of the Close invocations corresponds
// the number of defined expectations
func (m *CloserMock) MinimockCloseDone() bool {
	if m.CloseMock.optional {
		// Optional methods provide '0 or more' call count restriction.
		return true
	}

	for _, e := range m.CloseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	return m.CloseMock.invocationsDone()
}

// MinimockCloseInspect logs each unmet expectation
func (m *CloserMock) MinimockCloseInspect() {
	for _, e := range m.CloseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to CloserMock.Close")
		}
	}

	afterCloseCounter := mm_atomic.LoadUint64(&m.afterCloseCounter)
	// if default expectation was set then invocations count should be greater than zero
	if m.CloseMock.defaultExpectation != nil && afterCloseCounter < 1 {
		m.t.Error("Expected call to CloserMock.Close")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcClose != nil && afterCloseCounter < 1 {
		m.t.Error("Expected call to CloserMock.Close")
	}

	if !m.CloseMock.invocationsDone() && afterCloseCounter > 0 {
		m.t.Errorf("Expected %d calls to CloserMock.Close but found %d calls",
			mm_atomic.LoadUint64(&m.CloseMock.expectedInvocations), afterCloseCounter)
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *CloserMock) MinimockFinish() {
	m.finishOnce.Do(func() {
		if !m.minimockDone() {
			m.MinimockCloseInspect()
		}
	})
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *CloserMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *CloserMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockCloseDone()
}

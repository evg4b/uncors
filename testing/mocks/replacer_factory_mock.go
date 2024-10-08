// Code generated by http://github.com/gojuno/minimock (v3.3.12). DO NOT EDIT.

package mocks

//go:generate minimock -i github.com/evg4b/uncors/internal/urlreplacer.ReplacerFactory -o replacer_factory_mock.go -n ReplacerFactoryMock -p mocks

import (
	"net/url"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	mm_urlreplacer "github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/gojuno/minimock/v3"
)

// ReplacerFactoryMock implements urlreplacer.ReplacerFactory
type ReplacerFactoryMock struct {
	t          minimock.Tester
	finishOnce sync.Once

	funcMake          func(requestURL *url.URL) (rp1 *mm_urlreplacer.Replacer, rp2 *mm_urlreplacer.Replacer, err error)
	inspectFuncMake   func(requestURL *url.URL)
	afterMakeCounter  uint64
	beforeMakeCounter uint64
	MakeMock          mReplacerFactoryMockMake
}

// NewReplacerFactoryMock returns a mock for urlreplacer.ReplacerFactory
func NewReplacerFactoryMock(t minimock.Tester) *ReplacerFactoryMock {
	m := &ReplacerFactoryMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.MakeMock = mReplacerFactoryMockMake{mock: m}
	m.MakeMock.callArgs = []*ReplacerFactoryMockMakeParams{}

	t.Cleanup(m.MinimockFinish)

	return m
}

type mReplacerFactoryMockMake struct {
	optional           bool
	mock               *ReplacerFactoryMock
	defaultExpectation *ReplacerFactoryMockMakeExpectation
	expectations       []*ReplacerFactoryMockMakeExpectation

	callArgs []*ReplacerFactoryMockMakeParams
	mutex    sync.RWMutex

	expectedInvocations uint64
}

// ReplacerFactoryMockMakeExpectation specifies expectation struct of the ReplacerFactory.Make
type ReplacerFactoryMockMakeExpectation struct {
	mock      *ReplacerFactoryMock
	params    *ReplacerFactoryMockMakeParams
	paramPtrs *ReplacerFactoryMockMakeParamPtrs
	results   *ReplacerFactoryMockMakeResults
	Counter   uint64
}

// ReplacerFactoryMockMakeParams contains parameters of the ReplacerFactory.Make
type ReplacerFactoryMockMakeParams struct {
	requestURL *url.URL
}

// ReplacerFactoryMockMakeParamPtrs contains pointers to parameters of the ReplacerFactory.Make
type ReplacerFactoryMockMakeParamPtrs struct {
	requestURL **url.URL
}

// ReplacerFactoryMockMakeResults contains results of the ReplacerFactory.Make
type ReplacerFactoryMockMakeResults struct {
	rp1 *mm_urlreplacer.Replacer
	rp2 *mm_urlreplacer.Replacer
	err error
}

// Marks this method to be optional. The default behavior of any method with Return() is '1 or more', meaning
// the test will fail minimock's automatic final call check if the mocked method was not called at least once.
// Optional() makes method check to work in '0 or more' mode.
// It is NOT RECOMMENDED to use this option unless you really need it, as default behaviour helps to
// catch the problems when the expected method call is totally skipped during test run.
func (mmMake *mReplacerFactoryMockMake) Optional() *mReplacerFactoryMockMake {
	mmMake.optional = true
	return mmMake
}

// Expect sets up expected params for ReplacerFactory.Make
func (mmMake *mReplacerFactoryMockMake) Expect(requestURL *url.URL) *mReplacerFactoryMockMake {
	if mmMake.mock.funcMake != nil {
		mmMake.mock.t.Fatalf("ReplacerFactoryMock.Make mock is already set by Set")
	}

	if mmMake.defaultExpectation == nil {
		mmMake.defaultExpectation = &ReplacerFactoryMockMakeExpectation{}
	}

	if mmMake.defaultExpectation.paramPtrs != nil {
		mmMake.mock.t.Fatalf("ReplacerFactoryMock.Make mock is already set by ExpectParams functions")
	}

	mmMake.defaultExpectation.params = &ReplacerFactoryMockMakeParams{requestURL}
	for _, e := range mmMake.expectations {
		if minimock.Equal(e.params, mmMake.defaultExpectation.params) {
			mmMake.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmMake.defaultExpectation.params)
		}
	}

	return mmMake
}

// ExpectRequestURLParam1 sets up expected param requestURL for ReplacerFactory.Make
func (mmMake *mReplacerFactoryMockMake) ExpectRequestURLParam1(requestURL *url.URL) *mReplacerFactoryMockMake {
	if mmMake.mock.funcMake != nil {
		mmMake.mock.t.Fatalf("ReplacerFactoryMock.Make mock is already set by Set")
	}

	if mmMake.defaultExpectation == nil {
		mmMake.defaultExpectation = &ReplacerFactoryMockMakeExpectation{}
	}

	if mmMake.defaultExpectation.params != nil {
		mmMake.mock.t.Fatalf("ReplacerFactoryMock.Make mock is already set by Expect")
	}

	if mmMake.defaultExpectation.paramPtrs == nil {
		mmMake.defaultExpectation.paramPtrs = &ReplacerFactoryMockMakeParamPtrs{}
	}
	mmMake.defaultExpectation.paramPtrs.requestURL = &requestURL

	return mmMake
}

// Inspect accepts an inspector function that has same arguments as the ReplacerFactory.Make
func (mmMake *mReplacerFactoryMockMake) Inspect(f func(requestURL *url.URL)) *mReplacerFactoryMockMake {
	if mmMake.mock.inspectFuncMake != nil {
		mmMake.mock.t.Fatalf("Inspect function is already set for ReplacerFactoryMock.Make")
	}

	mmMake.mock.inspectFuncMake = f

	return mmMake
}

// Return sets up results that will be returned by ReplacerFactory.Make
func (mmMake *mReplacerFactoryMockMake) Return(rp1 *mm_urlreplacer.Replacer, rp2 *mm_urlreplacer.Replacer, err error) *ReplacerFactoryMock {
	if mmMake.mock.funcMake != nil {
		mmMake.mock.t.Fatalf("ReplacerFactoryMock.Make mock is already set by Set")
	}

	if mmMake.defaultExpectation == nil {
		mmMake.defaultExpectation = &ReplacerFactoryMockMakeExpectation{mock: mmMake.mock}
	}
	mmMake.defaultExpectation.results = &ReplacerFactoryMockMakeResults{rp1, rp2, err}
	return mmMake.mock
}

// Set uses given function f to mock the ReplacerFactory.Make method
func (mmMake *mReplacerFactoryMockMake) Set(f func(requestURL *url.URL) (rp1 *mm_urlreplacer.Replacer, rp2 *mm_urlreplacer.Replacer, err error)) *ReplacerFactoryMock {
	if mmMake.defaultExpectation != nil {
		mmMake.mock.t.Fatalf("Default expectation is already set for the ReplacerFactory.Make method")
	}

	if len(mmMake.expectations) > 0 {
		mmMake.mock.t.Fatalf("Some expectations are already set for the ReplacerFactory.Make method")
	}

	mmMake.mock.funcMake = f
	return mmMake.mock
}

// When sets expectation for the ReplacerFactory.Make which will trigger the result defined by the following
// Then helper
func (mmMake *mReplacerFactoryMockMake) When(requestURL *url.URL) *ReplacerFactoryMockMakeExpectation {
	if mmMake.mock.funcMake != nil {
		mmMake.mock.t.Fatalf("ReplacerFactoryMock.Make mock is already set by Set")
	}

	expectation := &ReplacerFactoryMockMakeExpectation{
		mock:   mmMake.mock,
		params: &ReplacerFactoryMockMakeParams{requestURL},
	}
	mmMake.expectations = append(mmMake.expectations, expectation)
	return expectation
}

// Then sets up ReplacerFactory.Make return parameters for the expectation previously defined by the When method
func (e *ReplacerFactoryMockMakeExpectation) Then(rp1 *mm_urlreplacer.Replacer, rp2 *mm_urlreplacer.Replacer, err error) *ReplacerFactoryMock {
	e.results = &ReplacerFactoryMockMakeResults{rp1, rp2, err}
	return e.mock
}

// Times sets number of times ReplacerFactory.Make should be invoked
func (mmMake *mReplacerFactoryMockMake) Times(n uint64) *mReplacerFactoryMockMake {
	if n == 0 {
		mmMake.mock.t.Fatalf("Times of ReplacerFactoryMock.Make mock can not be zero")
	}
	mm_atomic.StoreUint64(&mmMake.expectedInvocations, n)
	return mmMake
}

func (mmMake *mReplacerFactoryMockMake) invocationsDone() bool {
	if len(mmMake.expectations) == 0 && mmMake.defaultExpectation == nil && mmMake.mock.funcMake == nil {
		return true
	}

	totalInvocations := mm_atomic.LoadUint64(&mmMake.mock.afterMakeCounter)
	expectedInvocations := mm_atomic.LoadUint64(&mmMake.expectedInvocations)

	return totalInvocations > 0 && (expectedInvocations == 0 || expectedInvocations == totalInvocations)
}

// Make implements urlreplacer.ReplacerFactory
func (mmMake *ReplacerFactoryMock) Make(requestURL *url.URL) (rp1 *mm_urlreplacer.Replacer, rp2 *mm_urlreplacer.Replacer, err error) {
	mm_atomic.AddUint64(&mmMake.beforeMakeCounter, 1)
	defer mm_atomic.AddUint64(&mmMake.afterMakeCounter, 1)

	if mmMake.inspectFuncMake != nil {
		mmMake.inspectFuncMake(requestURL)
	}

	mm_params := ReplacerFactoryMockMakeParams{requestURL}

	// Record call args
	mmMake.MakeMock.mutex.Lock()
	mmMake.MakeMock.callArgs = append(mmMake.MakeMock.callArgs, &mm_params)
	mmMake.MakeMock.mutex.Unlock()

	for _, e := range mmMake.MakeMock.expectations {
		if minimock.Equal(*e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.rp1, e.results.rp2, e.results.err
		}
	}

	if mmMake.MakeMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmMake.MakeMock.defaultExpectation.Counter, 1)
		mm_want := mmMake.MakeMock.defaultExpectation.params
		mm_want_ptrs := mmMake.MakeMock.defaultExpectation.paramPtrs

		mm_got := ReplacerFactoryMockMakeParams{requestURL}

		if mm_want_ptrs != nil {

			if mm_want_ptrs.requestURL != nil && !minimock.Equal(*mm_want_ptrs.requestURL, mm_got.requestURL) {
				mmMake.t.Errorf("ReplacerFactoryMock.Make got unexpected parameter requestURL, want: %#v, got: %#v%s\n", *mm_want_ptrs.requestURL, mm_got.requestURL, minimock.Diff(*mm_want_ptrs.requestURL, mm_got.requestURL))
			}

		} else if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmMake.t.Errorf("ReplacerFactoryMock.Make got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmMake.MakeMock.defaultExpectation.results
		if mm_results == nil {
			mmMake.t.Fatal("No results are set for the ReplacerFactoryMock.Make")
		}
		return (*mm_results).rp1, (*mm_results).rp2, (*mm_results).err
	}
	if mmMake.funcMake != nil {
		return mmMake.funcMake(requestURL)
	}
	mmMake.t.Fatalf("Unexpected call to ReplacerFactoryMock.Make. %v", requestURL)
	return
}

// MakeAfterCounter returns a count of finished ReplacerFactoryMock.Make invocations
func (mmMake *ReplacerFactoryMock) MakeAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMake.afterMakeCounter)
}

// MakeBeforeCounter returns a count of ReplacerFactoryMock.Make invocations
func (mmMake *ReplacerFactoryMock) MakeBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMake.beforeMakeCounter)
}

// Calls returns a list of arguments used in each call to ReplacerFactoryMock.Make.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmMake *mReplacerFactoryMockMake) Calls() []*ReplacerFactoryMockMakeParams {
	mmMake.mutex.RLock()

	argCopy := make([]*ReplacerFactoryMockMakeParams, len(mmMake.callArgs))
	copy(argCopy, mmMake.callArgs)

	mmMake.mutex.RUnlock()

	return argCopy
}

// MinimockMakeDone returns true if the count of the Make invocations corresponds
// the number of defined expectations
func (m *ReplacerFactoryMock) MinimockMakeDone() bool {
	if m.MakeMock.optional {
		// Optional methods provide '0 or more' call count restriction.
		return true
	}

	for _, e := range m.MakeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	return m.MakeMock.invocationsDone()
}

// MinimockMakeInspect logs each unmet expectation
func (m *ReplacerFactoryMock) MinimockMakeInspect() {
	for _, e := range m.MakeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ReplacerFactoryMock.Make with params: %#v", *e.params)
		}
	}

	afterMakeCounter := mm_atomic.LoadUint64(&m.afterMakeCounter)
	// if default expectation was set then invocations count should be greater than zero
	if m.MakeMock.defaultExpectation != nil && afterMakeCounter < 1 {
		if m.MakeMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ReplacerFactoryMock.Make")
		} else {
			m.t.Errorf("Expected call to ReplacerFactoryMock.Make with params: %#v", *m.MakeMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcMake != nil && afterMakeCounter < 1 {
		m.t.Error("Expected call to ReplacerFactoryMock.Make")
	}

	if !m.MakeMock.invocationsDone() && afterMakeCounter > 0 {
		m.t.Errorf("Expected %d calls to ReplacerFactoryMock.Make but found %d calls",
			mm_atomic.LoadUint64(&m.MakeMock.expectedInvocations), afterMakeCounter)
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *ReplacerFactoryMock) MinimockFinish() {
	m.finishOnce.Do(func() {
		if !m.minimockDone() {
			m.MinimockMakeInspect()
		}
	})
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *ReplacerFactoryMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *ReplacerFactoryMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockMakeDone()
}

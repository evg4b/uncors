package mocks

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

//go:generate minimock -i github.com/evg4b/uncors/internal/proxy.URLReplacerFactory -o ./urlreplacer_factory_mock.go -n URLReplacerFactoryMock

import (
	"net/url"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/gojuno/minimock/v3"
)

// URLReplacerFactoryMock implements proxy.URLReplacerFactory
type URLReplacerFactoryMock struct {
	t minimock.Tester

	funcMake          func(requestURL *url.URL) (rp1 *urlreplacer.Replacer, err error)
	inspectFuncMake   func(requestURL *url.URL)
	afterMakeCounter  uint64
	beforeMakeCounter uint64
	MakeMock          mURLReplacerFactoryMockMake

	funcMakeV2          func(requestURL *url.URL) (rp1 *urlreplacer.ReplacerV2, rp2 *urlreplacer.ReplacerV2, err error)
	inspectFuncMakeV2   func(requestURL *url.URL)
	afterMakeV2Counter  uint64
	beforeMakeV2Counter uint64
	MakeV2Mock          mURLReplacerFactoryMockMakeV2
}

// NewURLReplacerFactoryMock returns a mock for proxy.URLReplacerFactory
func NewURLReplacerFactoryMock(t minimock.Tester) *URLReplacerFactoryMock {
	m := &URLReplacerFactoryMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.MakeMock = mURLReplacerFactoryMockMake{mock: m}
	m.MakeMock.callArgs = []*URLReplacerFactoryMockMakeParams{}

	m.MakeV2Mock = mURLReplacerFactoryMockMakeV2{mock: m}
	m.MakeV2Mock.callArgs = []*URLReplacerFactoryMockMakeV2Params{}

	return m
}

type mURLReplacerFactoryMockMake struct {
	mock               *URLReplacerFactoryMock
	defaultExpectation *URLReplacerFactoryMockMakeExpectation
	expectations       []*URLReplacerFactoryMockMakeExpectation

	callArgs []*URLReplacerFactoryMockMakeParams
	mutex    sync.RWMutex
}

// URLReplacerFactoryMockMakeExpectation specifies expectation struct of the URLReplacerFactory.Make
type URLReplacerFactoryMockMakeExpectation struct {
	mock    *URLReplacerFactoryMock
	params  *URLReplacerFactoryMockMakeParams
	results *URLReplacerFactoryMockMakeResults
	Counter uint64
}

// URLReplacerFactoryMockMakeParams contains parameters of the URLReplacerFactory.Make
type URLReplacerFactoryMockMakeParams struct {
	requestURL *url.URL
}

// URLReplacerFactoryMockMakeResults contains results of the URLReplacerFactory.Make
type URLReplacerFactoryMockMakeResults struct {
	rp1 *urlreplacer.Replacer
	err error
}

// Expect sets up expected params for URLReplacerFactory.Make
func (mmMake *mURLReplacerFactoryMockMake) Expect(requestURL *url.URL) *mURLReplacerFactoryMockMake {
	if mmMake.mock.funcMake != nil {
		mmMake.mock.t.Fatalf("URLReplacerFactoryMock.Make mock is already set by Set")
	}

	if mmMake.defaultExpectation == nil {
		mmMake.defaultExpectation = &URLReplacerFactoryMockMakeExpectation{}
	}

	mmMake.defaultExpectation.params = &URLReplacerFactoryMockMakeParams{requestURL}
	for _, e := range mmMake.expectations {
		if minimock.Equal(e.params, mmMake.defaultExpectation.params) {
			mmMake.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmMake.defaultExpectation.params)
		}
	}

	return mmMake
}

// Inspect accepts an inspector function that has same arguments as the URLReplacerFactory.Make
func (mmMake *mURLReplacerFactoryMockMake) Inspect(f func(requestURL *url.URL)) *mURLReplacerFactoryMockMake {
	if mmMake.mock.inspectFuncMake != nil {
		mmMake.mock.t.Fatalf("Inspect function is already set for URLReplacerFactoryMock.Make")
	}

	mmMake.mock.inspectFuncMake = f

	return mmMake
}

// Return sets up results that will be returned by URLReplacerFactory.Make
func (mmMake *mURLReplacerFactoryMockMake) Return(rp1 *urlreplacer.Replacer, err error) *URLReplacerFactoryMock {
	if mmMake.mock.funcMake != nil {
		mmMake.mock.t.Fatalf("URLReplacerFactoryMock.Make mock is already set by Set")
	}

	if mmMake.defaultExpectation == nil {
		mmMake.defaultExpectation = &URLReplacerFactoryMockMakeExpectation{mock: mmMake.mock}
	}
	mmMake.defaultExpectation.results = &URLReplacerFactoryMockMakeResults{rp1, err}
	return mmMake.mock
}

//Set uses given function f to mock the URLReplacerFactory.Make method
func (mmMake *mURLReplacerFactoryMockMake) Set(f func(requestURL *url.URL) (rp1 *urlreplacer.Replacer, err error)) *URLReplacerFactoryMock {
	if mmMake.defaultExpectation != nil {
		mmMake.mock.t.Fatalf("Default expectation is already set for the URLReplacerFactory.Make method")
	}

	if len(mmMake.expectations) > 0 {
		mmMake.mock.t.Fatalf("Some expectations are already set for the URLReplacerFactory.Make method")
	}

	mmMake.mock.funcMake = f
	return mmMake.mock
}

// When sets expectation for the URLReplacerFactory.Make which will trigger the result defined by the following
// Then helper
func (mmMake *mURLReplacerFactoryMockMake) When(requestURL *url.URL) *URLReplacerFactoryMockMakeExpectation {
	if mmMake.mock.funcMake != nil {
		mmMake.mock.t.Fatalf("URLReplacerFactoryMock.Make mock is already set by Set")
	}

	expectation := &URLReplacerFactoryMockMakeExpectation{
		mock:   mmMake.mock,
		params: &URLReplacerFactoryMockMakeParams{requestURL},
	}
	mmMake.expectations = append(mmMake.expectations, expectation)
	return expectation
}

// Then sets up URLReplacerFactory.Make return parameters for the expectation previously defined by the When method
func (e *URLReplacerFactoryMockMakeExpectation) Then(rp1 *urlreplacer.Replacer, err error) *URLReplacerFactoryMock {
	e.results = &URLReplacerFactoryMockMakeResults{rp1, err}
	return e.mock
}

// Make implements proxy.URLReplacerFactory
func (mmMake *URLReplacerFactoryMock) Make(requestURL *url.URL) (rp1 *urlreplacer.Replacer, err error) {
	mm_atomic.AddUint64(&mmMake.beforeMakeCounter, 1)
	defer mm_atomic.AddUint64(&mmMake.afterMakeCounter, 1)

	if mmMake.inspectFuncMake != nil {
		mmMake.inspectFuncMake(requestURL)
	}

	mm_params := &URLReplacerFactoryMockMakeParams{requestURL}

	// Record call args
	mmMake.MakeMock.mutex.Lock()
	mmMake.MakeMock.callArgs = append(mmMake.MakeMock.callArgs, mm_params)
	mmMake.MakeMock.mutex.Unlock()

	for _, e := range mmMake.MakeMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.rp1, e.results.err
		}
	}

	if mmMake.MakeMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmMake.MakeMock.defaultExpectation.Counter, 1)
		mm_want := mmMake.MakeMock.defaultExpectation.params
		mm_got := URLReplacerFactoryMockMakeParams{requestURL}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmMake.t.Errorf("URLReplacerFactoryMock.Make got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmMake.MakeMock.defaultExpectation.results
		if mm_results == nil {
			mmMake.t.Fatal("No results are set for the URLReplacerFactoryMock.Make")
		}
		return (*mm_results).rp1, (*mm_results).err
	}
	if mmMake.funcMake != nil {
		return mmMake.funcMake(requestURL)
	}
	mmMake.t.Fatalf("Unexpected call to URLReplacerFactoryMock.Make. %v", requestURL)
	return
}

// MakeAfterCounter returns a count of finished URLReplacerFactoryMock.Make invocations
func (mmMake *URLReplacerFactoryMock) MakeAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMake.afterMakeCounter)
}

// MakeBeforeCounter returns a count of URLReplacerFactoryMock.Make invocations
func (mmMake *URLReplacerFactoryMock) MakeBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMake.beforeMakeCounter)
}

// Calls returns a list of arguments used in each call to URLReplacerFactoryMock.Make.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmMake *mURLReplacerFactoryMockMake) Calls() []*URLReplacerFactoryMockMakeParams {
	mmMake.mutex.RLock()

	argCopy := make([]*URLReplacerFactoryMockMakeParams, len(mmMake.callArgs))
	copy(argCopy, mmMake.callArgs)

	mmMake.mutex.RUnlock()

	return argCopy
}

// MinimockMakeDone returns true if the count of the Make invocations corresponds
// the number of defined expectations
func (m *URLReplacerFactoryMock) MinimockMakeDone() bool {
	for _, e := range m.MakeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.MakeMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterMakeCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcMake != nil && mm_atomic.LoadUint64(&m.afterMakeCounter) < 1 {
		return false
	}
	return true
}

// MinimockMakeInspect logs each unmet expectation
func (m *URLReplacerFactoryMock) MinimockMakeInspect() {
	for _, e := range m.MakeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to URLReplacerFactoryMock.Make with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.MakeMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterMakeCounter) < 1 {
		if m.MakeMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to URLReplacerFactoryMock.Make")
		} else {
			m.t.Errorf("Expected call to URLReplacerFactoryMock.Make with params: %#v", *m.MakeMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcMake != nil && mm_atomic.LoadUint64(&m.afterMakeCounter) < 1 {
		m.t.Error("Expected call to URLReplacerFactoryMock.Make")
	}
}

type mURLReplacerFactoryMockMakeV2 struct {
	mock               *URLReplacerFactoryMock
	defaultExpectation *URLReplacerFactoryMockMakeV2Expectation
	expectations       []*URLReplacerFactoryMockMakeV2Expectation

	callArgs []*URLReplacerFactoryMockMakeV2Params
	mutex    sync.RWMutex
}

// URLReplacerFactoryMockMakeV2Expectation specifies expectation struct of the URLReplacerFactory.MakeV2
type URLReplacerFactoryMockMakeV2Expectation struct {
	mock    *URLReplacerFactoryMock
	params  *URLReplacerFactoryMockMakeV2Params
	results *URLReplacerFactoryMockMakeV2Results
	Counter uint64
}

// URLReplacerFactoryMockMakeV2Params contains parameters of the URLReplacerFactory.MakeV2
type URLReplacerFactoryMockMakeV2Params struct {
	requestURL *url.URL
}

// URLReplacerFactoryMockMakeV2Results contains results of the URLReplacerFactory.MakeV2
type URLReplacerFactoryMockMakeV2Results struct {
	rp1 *urlreplacer.ReplacerV2
	rp2 *urlreplacer.ReplacerV2
	err error
}

// Expect sets up expected params for URLReplacerFactory.MakeV2
func (mmMakeV2 *mURLReplacerFactoryMockMakeV2) Expect(requestURL *url.URL) *mURLReplacerFactoryMockMakeV2 {
	if mmMakeV2.mock.funcMakeV2 != nil {
		mmMakeV2.mock.t.Fatalf("URLReplacerFactoryMock.MakeV2 mock is already set by Set")
	}

	if mmMakeV2.defaultExpectation == nil {
		mmMakeV2.defaultExpectation = &URLReplacerFactoryMockMakeV2Expectation{}
	}

	mmMakeV2.defaultExpectation.params = &URLReplacerFactoryMockMakeV2Params{requestURL}
	for _, e := range mmMakeV2.expectations {
		if minimock.Equal(e.params, mmMakeV2.defaultExpectation.params) {
			mmMakeV2.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmMakeV2.defaultExpectation.params)
		}
	}

	return mmMakeV2
}

// Inspect accepts an inspector function that has same arguments as the URLReplacerFactory.MakeV2
func (mmMakeV2 *mURLReplacerFactoryMockMakeV2) Inspect(f func(requestURL *url.URL)) *mURLReplacerFactoryMockMakeV2 {
	if mmMakeV2.mock.inspectFuncMakeV2 != nil {
		mmMakeV2.mock.t.Fatalf("Inspect function is already set for URLReplacerFactoryMock.MakeV2")
	}

	mmMakeV2.mock.inspectFuncMakeV2 = f

	return mmMakeV2
}

// Return sets up results that will be returned by URLReplacerFactory.MakeV2
func (mmMakeV2 *mURLReplacerFactoryMockMakeV2) Return(rp1 *urlreplacer.ReplacerV2, rp2 *urlreplacer.ReplacerV2, err error) *URLReplacerFactoryMock {
	if mmMakeV2.mock.funcMakeV2 != nil {
		mmMakeV2.mock.t.Fatalf("URLReplacerFactoryMock.MakeV2 mock is already set by Set")
	}

	if mmMakeV2.defaultExpectation == nil {
		mmMakeV2.defaultExpectation = &URLReplacerFactoryMockMakeV2Expectation{mock: mmMakeV2.mock}
	}
	mmMakeV2.defaultExpectation.results = &URLReplacerFactoryMockMakeV2Results{rp1, rp2, err}
	return mmMakeV2.mock
}

//Set uses given function f to mock the URLReplacerFactory.MakeV2 method
func (mmMakeV2 *mURLReplacerFactoryMockMakeV2) Set(f func(requestURL *url.URL) (rp1 *urlreplacer.ReplacerV2, rp2 *urlreplacer.ReplacerV2, err error)) *URLReplacerFactoryMock {
	if mmMakeV2.defaultExpectation != nil {
		mmMakeV2.mock.t.Fatalf("Default expectation is already set for the URLReplacerFactory.MakeV2 method")
	}

	if len(mmMakeV2.expectations) > 0 {
		mmMakeV2.mock.t.Fatalf("Some expectations are already set for the URLReplacerFactory.MakeV2 method")
	}

	mmMakeV2.mock.funcMakeV2 = f
	return mmMakeV2.mock
}

// When sets expectation for the URLReplacerFactory.MakeV2 which will trigger the result defined by the following
// Then helper
func (mmMakeV2 *mURLReplacerFactoryMockMakeV2) When(requestURL *url.URL) *URLReplacerFactoryMockMakeV2Expectation {
	if mmMakeV2.mock.funcMakeV2 != nil {
		mmMakeV2.mock.t.Fatalf("URLReplacerFactoryMock.MakeV2 mock is already set by Set")
	}

	expectation := &URLReplacerFactoryMockMakeV2Expectation{
		mock:   mmMakeV2.mock,
		params: &URLReplacerFactoryMockMakeV2Params{requestURL},
	}
	mmMakeV2.expectations = append(mmMakeV2.expectations, expectation)
	return expectation
}

// Then sets up URLReplacerFactory.MakeV2 return parameters for the expectation previously defined by the When method
func (e *URLReplacerFactoryMockMakeV2Expectation) Then(rp1 *urlreplacer.ReplacerV2, rp2 *urlreplacer.ReplacerV2, err error) *URLReplacerFactoryMock {
	e.results = &URLReplacerFactoryMockMakeV2Results{rp1, rp2, err}
	return e.mock
}

// MakeV2 implements proxy.URLReplacerFactory
func (mmMakeV2 *URLReplacerFactoryMock) MakeV2(requestURL *url.URL) (rp1 *urlreplacer.ReplacerV2, rp2 *urlreplacer.ReplacerV2, err error) {
	mm_atomic.AddUint64(&mmMakeV2.beforeMakeV2Counter, 1)
	defer mm_atomic.AddUint64(&mmMakeV2.afterMakeV2Counter, 1)

	if mmMakeV2.inspectFuncMakeV2 != nil {
		mmMakeV2.inspectFuncMakeV2(requestURL)
	}

	mm_params := &URLReplacerFactoryMockMakeV2Params{requestURL}

	// Record call args
	mmMakeV2.MakeV2Mock.mutex.Lock()
	mmMakeV2.MakeV2Mock.callArgs = append(mmMakeV2.MakeV2Mock.callArgs, mm_params)
	mmMakeV2.MakeV2Mock.mutex.Unlock()

	for _, e := range mmMakeV2.MakeV2Mock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.rp1, e.results.rp2, e.results.err
		}
	}

	if mmMakeV2.MakeV2Mock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmMakeV2.MakeV2Mock.defaultExpectation.Counter, 1)
		mm_want := mmMakeV2.MakeV2Mock.defaultExpectation.params
		mm_got := URLReplacerFactoryMockMakeV2Params{requestURL}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmMakeV2.t.Errorf("URLReplacerFactoryMock.MakeV2 got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmMakeV2.MakeV2Mock.defaultExpectation.results
		if mm_results == nil {
			mmMakeV2.t.Fatal("No results are set for the URLReplacerFactoryMock.MakeV2")
		}
		return (*mm_results).rp1, (*mm_results).rp2, (*mm_results).err
	}
	if mmMakeV2.funcMakeV2 != nil {
		return mmMakeV2.funcMakeV2(requestURL)
	}
	mmMakeV2.t.Fatalf("Unexpected call to URLReplacerFactoryMock.MakeV2. %v", requestURL)
	return
}

// MakeV2AfterCounter returns a count of finished URLReplacerFactoryMock.MakeV2 invocations
func (mmMakeV2 *URLReplacerFactoryMock) MakeV2AfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMakeV2.afterMakeV2Counter)
}

// MakeV2BeforeCounter returns a count of URLReplacerFactoryMock.MakeV2 invocations
func (mmMakeV2 *URLReplacerFactoryMock) MakeV2BeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMakeV2.beforeMakeV2Counter)
}

// Calls returns a list of arguments used in each call to URLReplacerFactoryMock.MakeV2.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmMakeV2 *mURLReplacerFactoryMockMakeV2) Calls() []*URLReplacerFactoryMockMakeV2Params {
	mmMakeV2.mutex.RLock()

	argCopy := make([]*URLReplacerFactoryMockMakeV2Params, len(mmMakeV2.callArgs))
	copy(argCopy, mmMakeV2.callArgs)

	mmMakeV2.mutex.RUnlock()

	return argCopy
}

// MinimockMakeV2Done returns true if the count of the MakeV2 invocations corresponds
// the number of defined expectations
func (m *URLReplacerFactoryMock) MinimockMakeV2Done() bool {
	for _, e := range m.MakeV2Mock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.MakeV2Mock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterMakeV2Counter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcMakeV2 != nil && mm_atomic.LoadUint64(&m.afterMakeV2Counter) < 1 {
		return false
	}
	return true
}

// MinimockMakeV2Inspect logs each unmet expectation
func (m *URLReplacerFactoryMock) MinimockMakeV2Inspect() {
	for _, e := range m.MakeV2Mock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to URLReplacerFactoryMock.MakeV2 with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.MakeV2Mock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterMakeV2Counter) < 1 {
		if m.MakeV2Mock.defaultExpectation.params == nil {
			m.t.Error("Expected call to URLReplacerFactoryMock.MakeV2")
		} else {
			m.t.Errorf("Expected call to URLReplacerFactoryMock.MakeV2 with params: %#v", *m.MakeV2Mock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcMakeV2 != nil && mm_atomic.LoadUint64(&m.afterMakeV2Counter) < 1 {
		m.t.Error("Expected call to URLReplacerFactoryMock.MakeV2")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *URLReplacerFactoryMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockMakeInspect()

		m.MinimockMakeV2Inspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *URLReplacerFactoryMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *URLReplacerFactoryMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockMakeDone() &&
		m.MinimockMakeV2Done()
}

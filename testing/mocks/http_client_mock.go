// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

package mocks

//go:generate minimock -i github.com/evg4b/uncors/internal/contracts.HTTPClient -o http_client_mock.go -n HTTPClientMock -p mocks

import (
	"net/http"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
)

// HTTPClientMock implements contracts.HTTPClient
type HTTPClientMock struct {
	t          minimock.Tester
	finishOnce sync.Once

	funcDo          func(req *http.Request) (rp1 *http.Response, err error)
	inspectFuncDo   func(req *http.Request)
	afterDoCounter  uint64
	beforeDoCounter uint64
	DoMock          mHTTPClientMockDo
}

// NewHTTPClientMock returns a mock for contracts.HTTPClient
func NewHTTPClientMock(t minimock.Tester) *HTTPClientMock {
	m := &HTTPClientMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.DoMock = mHTTPClientMockDo{mock: m}
	m.DoMock.callArgs = []*HTTPClientMockDoParams{}

	t.Cleanup(m.MinimockFinish)

	return m
}

type mHTTPClientMockDo struct {
	mock               *HTTPClientMock
	defaultExpectation *HTTPClientMockDoExpectation
	expectations       []*HTTPClientMockDoExpectation

	callArgs []*HTTPClientMockDoParams
	mutex    sync.RWMutex
}

// HTTPClientMockDoExpectation specifies expectation struct of the HTTPClient.Do
type HTTPClientMockDoExpectation struct {
	mock    *HTTPClientMock
	params  *HTTPClientMockDoParams
	results *HTTPClientMockDoResults
	Counter uint64
}

// HTTPClientMockDoParams contains parameters of the HTTPClient.Do
type HTTPClientMockDoParams struct {
	req *http.Request
}

// HTTPClientMockDoResults contains results of the HTTPClient.Do
type HTTPClientMockDoResults struct {
	rp1 *http.Response
	err error
}

// Expect sets up expected params for HTTPClient.Do
func (mmDo *mHTTPClientMockDo) Expect(req *http.Request) *mHTTPClientMockDo {
	if mmDo.mock.funcDo != nil {
		mmDo.mock.t.Fatalf("HTTPClientMock.Do mock is already set by Set")
	}

	if mmDo.defaultExpectation == nil {
		mmDo.defaultExpectation = &HTTPClientMockDoExpectation{}
	}

	mmDo.defaultExpectation.params = &HTTPClientMockDoParams{req}
	for _, e := range mmDo.expectations {
		if minimock.Equal(e.params, mmDo.defaultExpectation.params) {
			mmDo.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmDo.defaultExpectation.params)
		}
	}

	return mmDo
}

// Inspect accepts an inspector function that has same arguments as the HTTPClient.Do
func (mmDo *mHTTPClientMockDo) Inspect(f func(req *http.Request)) *mHTTPClientMockDo {
	if mmDo.mock.inspectFuncDo != nil {
		mmDo.mock.t.Fatalf("Inspect function is already set for HTTPClientMock.Do")
	}

	mmDo.mock.inspectFuncDo = f

	return mmDo
}

// Return sets up results that will be returned by HTTPClient.Do
func (mmDo *mHTTPClientMockDo) Return(rp1 *http.Response, err error) *HTTPClientMock {
	if mmDo.mock.funcDo != nil {
		mmDo.mock.t.Fatalf("HTTPClientMock.Do mock is already set by Set")
	}

	if mmDo.defaultExpectation == nil {
		mmDo.defaultExpectation = &HTTPClientMockDoExpectation{mock: mmDo.mock}
	}
	mmDo.defaultExpectation.results = &HTTPClientMockDoResults{rp1, err}
	return mmDo.mock
}

// Set uses given function f to mock the HTTPClient.Do method
func (mmDo *mHTTPClientMockDo) Set(f func(req *http.Request) (rp1 *http.Response, err error)) *HTTPClientMock {
	if mmDo.defaultExpectation != nil {
		mmDo.mock.t.Fatalf("Default expectation is already set for the HTTPClient.Do method")
	}

	if len(mmDo.expectations) > 0 {
		mmDo.mock.t.Fatalf("Some expectations are already set for the HTTPClient.Do method")
	}

	mmDo.mock.funcDo = f
	return mmDo.mock
}

// When sets expectation for the HTTPClient.Do which will trigger the result defined by the following
// Then helper
func (mmDo *mHTTPClientMockDo) When(req *http.Request) *HTTPClientMockDoExpectation {
	if mmDo.mock.funcDo != nil {
		mmDo.mock.t.Fatalf("HTTPClientMock.Do mock is already set by Set")
	}

	expectation := &HTTPClientMockDoExpectation{
		mock:   mmDo.mock,
		params: &HTTPClientMockDoParams{req},
	}
	mmDo.expectations = append(mmDo.expectations, expectation)
	return expectation
}

// Then sets up HTTPClient.Do return parameters for the expectation previously defined by the When method
func (e *HTTPClientMockDoExpectation) Then(rp1 *http.Response, err error) *HTTPClientMock {
	e.results = &HTTPClientMockDoResults{rp1, err}
	return e.mock
}

// Do implements contracts.HTTPClient
func (mmDo *HTTPClientMock) Do(req *http.Request) (rp1 *http.Response, err error) {
	mm_atomic.AddUint64(&mmDo.beforeDoCounter, 1)
	defer mm_atomic.AddUint64(&mmDo.afterDoCounter, 1)

	if mmDo.inspectFuncDo != nil {
		mmDo.inspectFuncDo(req)
	}

	mm_params := HTTPClientMockDoParams{req}

	// Record call args
	mmDo.DoMock.mutex.Lock()
	mmDo.DoMock.callArgs = append(mmDo.DoMock.callArgs, &mm_params)
	mmDo.DoMock.mutex.Unlock()

	for _, e := range mmDo.DoMock.expectations {
		if minimock.Equal(*e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.rp1, e.results.err
		}
	}

	if mmDo.DoMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmDo.DoMock.defaultExpectation.Counter, 1)
		mm_want := mmDo.DoMock.defaultExpectation.params
		mm_got := HTTPClientMockDoParams{req}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmDo.t.Errorf("HTTPClientMock.Do got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmDo.DoMock.defaultExpectation.results
		if mm_results == nil {
			mmDo.t.Fatal("No results are set for the HTTPClientMock.Do")
		}
		return (*mm_results).rp1, (*mm_results).err
	}
	if mmDo.funcDo != nil {
		return mmDo.funcDo(req)
	}
	mmDo.t.Fatalf("Unexpected call to HTTPClientMock.Do. %v", req)
	return
}

// DoAfterCounter returns a count of finished HTTPClientMock.Do invocations
func (mmDo *HTTPClientMock) DoAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDo.afterDoCounter)
}

// DoBeforeCounter returns a count of HTTPClientMock.Do invocations
func (mmDo *HTTPClientMock) DoBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDo.beforeDoCounter)
}

// Calls returns a list of arguments used in each call to HTTPClientMock.Do.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmDo *mHTTPClientMockDo) Calls() []*HTTPClientMockDoParams {
	mmDo.mutex.RLock()

	argCopy := make([]*HTTPClientMockDoParams, len(mmDo.callArgs))
	copy(argCopy, mmDo.callArgs)

	mmDo.mutex.RUnlock()

	return argCopy
}

// MinimockDoDone returns true if the count of the Do invocations corresponds
// the number of defined expectations
func (m *HTTPClientMock) MinimockDoDone() bool {
	for _, e := range m.DoMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.DoMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterDoCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcDo != nil && mm_atomic.LoadUint64(&m.afterDoCounter) < 1 {
		return false
	}
	return true
}

// MinimockDoInspect logs each unmet expectation
func (m *HTTPClientMock) MinimockDoInspect() {
	for _, e := range m.DoMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to HTTPClientMock.Do with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.DoMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterDoCounter) < 1 {
		if m.DoMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to HTTPClientMock.Do")
		} else {
			m.t.Errorf("Expected call to HTTPClientMock.Do with params: %#v", *m.DoMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcDo != nil && mm_atomic.LoadUint64(&m.afterDoCounter) < 1 {
		m.t.Error("Expected call to HTTPClientMock.Do")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *HTTPClientMock) MinimockFinish() {
	m.finishOnce.Do(func() {
		if !m.minimockDone() {
			m.MinimockDoInspect()
			m.t.FailNow()
		}
	})
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *HTTPClientMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *HTTPClientMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockDoDone()
}

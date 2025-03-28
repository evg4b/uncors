// Code generated by http://github.com/gojuno/minimock (v3.3.12). DO NOT EDIT.

package mocks

//go:generate minimock -i github.com/evg4b/uncors/pkg/fakedata.Generator -o fakedata_generator_mock.go -n GeneratorMock -p mocks

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	mm_fakedata "github.com/evg4b/uncors/pkg/fakedata"
	"github.com/gojuno/minimock/v3"
)

// GeneratorMock implements fakedata.Generator
type GeneratorMock struct {
	t          minimock.Tester
	finishOnce sync.Once

	funcGenerate          func(node *mm_fakedata.Node, seed uint64) (a1 any, err error)
	inspectFuncGenerate   func(node *mm_fakedata.Node, seed uint64)
	afterGenerateCounter  uint64
	beforeGenerateCounter uint64
	GenerateMock          mGeneratorMockGenerate
}

// NewGeneratorMock returns a mock for fakedata.Generator
func NewGeneratorMock(t minimock.Tester) *GeneratorMock {
	m := &GeneratorMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GenerateMock = mGeneratorMockGenerate{mock: m}
	m.GenerateMock.callArgs = []*GeneratorMockGenerateParams{}

	t.Cleanup(m.MinimockFinish)

	return m
}

type mGeneratorMockGenerate struct {
	optional           bool
	mock               *GeneratorMock
	defaultExpectation *GeneratorMockGenerateExpectation
	expectations       []*GeneratorMockGenerateExpectation

	callArgs []*GeneratorMockGenerateParams
	mutex    sync.RWMutex

	expectedInvocations uint64
}

// GeneratorMockGenerateExpectation specifies expectation struct of the Generator.Generate
type GeneratorMockGenerateExpectation struct {
	mock      *GeneratorMock
	params    *GeneratorMockGenerateParams
	paramPtrs *GeneratorMockGenerateParamPtrs
	results   *GeneratorMockGenerateResults
	Counter   uint64
}

// GeneratorMockGenerateParams contains parameters of the Generator.Generate
type GeneratorMockGenerateParams struct {
	node *mm_fakedata.Node
	seed uint64
}

// GeneratorMockGenerateParamPtrs contains pointers to parameters of the Generator.Generate
type GeneratorMockGenerateParamPtrs struct {
	node **mm_fakedata.Node
	seed *uint64
}

// GeneratorMockGenerateResults contains results of the Generator.Generate
type GeneratorMockGenerateResults struct {
	a1  any
	err error
}

// Marks this method to be optional. The default behavior of any method with Return() is '1 or more', meaning
// the test will fail minimock's automatic final call check if the mocked method was not called at least once.
// Optional() makes method check to work in '0 or more' mode.
// It is NOT RECOMMENDED to use this option unless you really need it, as default behaviour helps to
// catch the problems when the expected method call is totally skipped during test run.
func (mmGenerate *mGeneratorMockGenerate) Optional() *mGeneratorMockGenerate {
	mmGenerate.optional = true
	return mmGenerate
}

// Expect sets up expected params for Generator.Generate
func (mmGenerate *mGeneratorMockGenerate) Expect(node *mm_fakedata.Node, seed uint64) *mGeneratorMockGenerate {
	if mmGenerate.mock.funcGenerate != nil {
		mmGenerate.mock.t.Fatalf("GeneratorMock.Generate mock is already set by Set")
	}

	if mmGenerate.defaultExpectation == nil {
		mmGenerate.defaultExpectation = &GeneratorMockGenerateExpectation{}
	}

	if mmGenerate.defaultExpectation.paramPtrs != nil {
		mmGenerate.mock.t.Fatalf("GeneratorMock.Generate mock is already set by ExpectParams functions")
	}

	mmGenerate.defaultExpectation.params = &GeneratorMockGenerateParams{node, seed}
	for _, e := range mmGenerate.expectations {
		if minimock.Equal(e.params, mmGenerate.defaultExpectation.params) {
			mmGenerate.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmGenerate.defaultExpectation.params)
		}
	}

	return mmGenerate
}

// ExpectNodeParam1 sets up expected param node for Generator.Generate
func (mmGenerate *mGeneratorMockGenerate) ExpectNodeParam1(node *mm_fakedata.Node) *mGeneratorMockGenerate {
	if mmGenerate.mock.funcGenerate != nil {
		mmGenerate.mock.t.Fatalf("GeneratorMock.Generate mock is already set by Set")
	}

	if mmGenerate.defaultExpectation == nil {
		mmGenerate.defaultExpectation = &GeneratorMockGenerateExpectation{}
	}

	if mmGenerate.defaultExpectation.params != nil {
		mmGenerate.mock.t.Fatalf("GeneratorMock.Generate mock is already set by Expect")
	}

	if mmGenerate.defaultExpectation.paramPtrs == nil {
		mmGenerate.defaultExpectation.paramPtrs = &GeneratorMockGenerateParamPtrs{}
	}
	mmGenerate.defaultExpectation.paramPtrs.node = &node

	return mmGenerate
}

// ExpectSeedParam2 sets up expected param seed for Generator.Generate
func (mmGenerate *mGeneratorMockGenerate) ExpectSeedParam2(seed uint64) *mGeneratorMockGenerate {
	if mmGenerate.mock.funcGenerate != nil {
		mmGenerate.mock.t.Fatalf("GeneratorMock.Generate mock is already set by Set")
	}

	if mmGenerate.defaultExpectation == nil {
		mmGenerate.defaultExpectation = &GeneratorMockGenerateExpectation{}
	}

	if mmGenerate.defaultExpectation.params != nil {
		mmGenerate.mock.t.Fatalf("GeneratorMock.Generate mock is already set by Expect")
	}

	if mmGenerate.defaultExpectation.paramPtrs == nil {
		mmGenerate.defaultExpectation.paramPtrs = &GeneratorMockGenerateParamPtrs{}
	}
	mmGenerate.defaultExpectation.paramPtrs.seed = &seed

	return mmGenerate
}

// Inspect accepts an inspector function that has same arguments as the Generator.Generate
func (mmGenerate *mGeneratorMockGenerate) Inspect(f func(node *mm_fakedata.Node, seed uint64)) *mGeneratorMockGenerate {
	if mmGenerate.mock.inspectFuncGenerate != nil {
		mmGenerate.mock.t.Fatalf("Inspect function is already set for GeneratorMock.Generate")
	}

	mmGenerate.mock.inspectFuncGenerate = f

	return mmGenerate
}

// Return sets up results that will be returned by Generator.Generate
func (mmGenerate *mGeneratorMockGenerate) Return(a1 any, err error) *GeneratorMock {
	if mmGenerate.mock.funcGenerate != nil {
		mmGenerate.mock.t.Fatalf("GeneratorMock.Generate mock is already set by Set")
	}

	if mmGenerate.defaultExpectation == nil {
		mmGenerate.defaultExpectation = &GeneratorMockGenerateExpectation{mock: mmGenerate.mock}
	}
	mmGenerate.defaultExpectation.results = &GeneratorMockGenerateResults{a1, err}
	return mmGenerate.mock
}

// Set uses given function f to mock the Generator.Generate method
func (mmGenerate *mGeneratorMockGenerate) Set(f func(node *mm_fakedata.Node, seed uint64) (a1 any, err error)) *GeneratorMock {
	if mmGenerate.defaultExpectation != nil {
		mmGenerate.mock.t.Fatalf("Default expectation is already set for the Generator.Generate method")
	}

	if len(mmGenerate.expectations) > 0 {
		mmGenerate.mock.t.Fatalf("Some expectations are already set for the Generator.Generate method")
	}

	mmGenerate.mock.funcGenerate = f
	return mmGenerate.mock
}

// When sets expectation for the Generator.Generate which will trigger the result defined by the following
// Then helper
func (mmGenerate *mGeneratorMockGenerate) When(node *mm_fakedata.Node, seed uint64) *GeneratorMockGenerateExpectation {
	if mmGenerate.mock.funcGenerate != nil {
		mmGenerate.mock.t.Fatalf("GeneratorMock.Generate mock is already set by Set")
	}

	expectation := &GeneratorMockGenerateExpectation{
		mock:   mmGenerate.mock,
		params: &GeneratorMockGenerateParams{node, seed},
	}
	mmGenerate.expectations = append(mmGenerate.expectations, expectation)
	return expectation
}

// Then sets up Generator.Generate return parameters for the expectation previously defined by the When method
func (e *GeneratorMockGenerateExpectation) Then(a1 any, err error) *GeneratorMock {
	e.results = &GeneratorMockGenerateResults{a1, err}
	return e.mock
}

// Times sets number of times Generator.Generate should be invoked
func (mmGenerate *mGeneratorMockGenerate) Times(n uint64) *mGeneratorMockGenerate {
	if n == 0 {
		mmGenerate.mock.t.Fatalf("Times of GeneratorMock.Generate mock can not be zero")
	}
	mm_atomic.StoreUint64(&mmGenerate.expectedInvocations, n)
	return mmGenerate
}

func (mmGenerate *mGeneratorMockGenerate) invocationsDone() bool {
	if len(mmGenerate.expectations) == 0 && mmGenerate.defaultExpectation == nil && mmGenerate.mock.funcGenerate == nil {
		return true
	}

	totalInvocations := mm_atomic.LoadUint64(&mmGenerate.mock.afterGenerateCounter)
	expectedInvocations := mm_atomic.LoadUint64(&mmGenerate.expectedInvocations)

	return totalInvocations > 0 && (expectedInvocations == 0 || expectedInvocations == totalInvocations)
}

// Generate implements fakedata.Generator
func (mmGenerate *GeneratorMock) Generate(node *mm_fakedata.Node, seed uint64) (a1 any, err error) {
	mm_atomic.AddUint64(&mmGenerate.beforeGenerateCounter, 1)
	defer mm_atomic.AddUint64(&mmGenerate.afterGenerateCounter, 1)

	if mmGenerate.inspectFuncGenerate != nil {
		mmGenerate.inspectFuncGenerate(node, seed)
	}

	mm_params := GeneratorMockGenerateParams{node, seed}

	// Record call args
	mmGenerate.GenerateMock.mutex.Lock()
	mmGenerate.GenerateMock.callArgs = append(mmGenerate.GenerateMock.callArgs, &mm_params)
	mmGenerate.GenerateMock.mutex.Unlock()

	for _, e := range mmGenerate.GenerateMock.expectations {
		if minimock.Equal(*e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.a1, e.results.err
		}
	}

	if mmGenerate.GenerateMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGenerate.GenerateMock.defaultExpectation.Counter, 1)
		mm_want := mmGenerate.GenerateMock.defaultExpectation.params
		mm_want_ptrs := mmGenerate.GenerateMock.defaultExpectation.paramPtrs

		mm_got := GeneratorMockGenerateParams{node, seed}

		if mm_want_ptrs != nil {

			if mm_want_ptrs.node != nil && !minimock.Equal(*mm_want_ptrs.node, mm_got.node) {
				mmGenerate.t.Errorf("GeneratorMock.Generate got unexpected parameter node, want: %#v, got: %#v%s\n", *mm_want_ptrs.node, mm_got.node, minimock.Diff(*mm_want_ptrs.node, mm_got.node))
			}

			if mm_want_ptrs.seed != nil && !minimock.Equal(*mm_want_ptrs.seed, mm_got.seed) {
				mmGenerate.t.Errorf("GeneratorMock.Generate got unexpected parameter seed, want: %#v, got: %#v%s\n", *mm_want_ptrs.seed, mm_got.seed, minimock.Diff(*mm_want_ptrs.seed, mm_got.seed))
			}

		} else if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmGenerate.t.Errorf("GeneratorMock.Generate got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmGenerate.GenerateMock.defaultExpectation.results
		if mm_results == nil {
			mmGenerate.t.Fatal("No results are set for the GeneratorMock.Generate")
		}
		return (*mm_results).a1, (*mm_results).err
	}
	if mmGenerate.funcGenerate != nil {
		return mmGenerate.funcGenerate(node, seed)
	}
	mmGenerate.t.Fatalf("Unexpected call to GeneratorMock.Generate. %v %v", node, seed)
	return
}

// GenerateAfterCounter returns a count of finished GeneratorMock.Generate invocations
func (mmGenerate *GeneratorMock) GenerateAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGenerate.afterGenerateCounter)
}

// GenerateBeforeCounter returns a count of GeneratorMock.Generate invocations
func (mmGenerate *GeneratorMock) GenerateBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGenerate.beforeGenerateCounter)
}

// Calls returns a list of arguments used in each call to GeneratorMock.Generate.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmGenerate *mGeneratorMockGenerate) Calls() []*GeneratorMockGenerateParams {
	mmGenerate.mutex.RLock()

	argCopy := make([]*GeneratorMockGenerateParams, len(mmGenerate.callArgs))
	copy(argCopy, mmGenerate.callArgs)

	mmGenerate.mutex.RUnlock()

	return argCopy
}

// MinimockGenerateDone returns true if the count of the Generate invocations corresponds
// the number of defined expectations
func (m *GeneratorMock) MinimockGenerateDone() bool {
	if m.GenerateMock.optional {
		// Optional methods provide '0 or more' call count restriction.
		return true
	}

	for _, e := range m.GenerateMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	return m.GenerateMock.invocationsDone()
}

// MinimockGenerateInspect logs each unmet expectation
func (m *GeneratorMock) MinimockGenerateInspect() {
	for _, e := range m.GenerateMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to GeneratorMock.Generate with params: %#v", *e.params)
		}
	}

	afterGenerateCounter := mm_atomic.LoadUint64(&m.afterGenerateCounter)
	// if default expectation was set then invocations count should be greater than zero
	if m.GenerateMock.defaultExpectation != nil && afterGenerateCounter < 1 {
		if m.GenerateMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to GeneratorMock.Generate")
		} else {
			m.t.Errorf("Expected call to GeneratorMock.Generate with params: %#v", *m.GenerateMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGenerate != nil && afterGenerateCounter < 1 {
		m.t.Error("Expected call to GeneratorMock.Generate")
	}

	if !m.GenerateMock.invocationsDone() && afterGenerateCounter > 0 {
		m.t.Errorf("Expected %d calls to GeneratorMock.Generate but found %d calls",
			mm_atomic.LoadUint64(&m.GenerateMock.expectedInvocations), afterGenerateCounter)
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *GeneratorMock) MinimockFinish() {
	m.finishOnce.Do(func() {
		if !m.minimockDone() {
			m.MinimockGenerateInspect()
		}
	})
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *GeneratorMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *GeneratorMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGenerateDone()
}

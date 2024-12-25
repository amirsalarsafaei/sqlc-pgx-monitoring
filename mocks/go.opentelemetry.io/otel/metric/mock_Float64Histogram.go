// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	metric "go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel/metric/embedded"
)

// MockFloat64Histogram is an autogenerated mock type for the Float64Histogram type
type MockFloat64Histogram struct {
	mock.Mock
	embedded.Float64Histogram
}

type MockFloat64Histogram_Expecter struct {
	mock *mock.Mock
}

func (_m *MockFloat64Histogram) EXPECT() *MockFloat64Histogram_Expecter {
	return &MockFloat64Histogram_Expecter{mock: &_m.Mock}
}

// Record provides a mock function with given fields: ctx, incr, options
func (_m *MockFloat64Histogram) Record(ctx context.Context, incr float64, options ...metric.RecordOption) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, incr)
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}

// MockFloat64Histogram_Record_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Record'
type MockFloat64Histogram_Record_Call struct {
	*mock.Call
}

// Record is a helper method to define mock.On call
//   - ctx context.Context
//   - incr float64
//   - options ...metric.RecordOption
func (_e *MockFloat64Histogram_Expecter) Record(ctx interface{}, incr interface{}, options ...interface{}) *MockFloat64Histogram_Record_Call {
	return &MockFloat64Histogram_Record_Call{Call: _e.mock.On("Record",
		append([]interface{}{ctx, incr}, options...)...)}
}

func (_c *MockFloat64Histogram_Record_Call) Run(run func(ctx context.Context, incr float64, options ...metric.RecordOption)) *MockFloat64Histogram_Record_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.RecordOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(metric.RecordOption)
			}
		}
		run(args[0].(context.Context), args[1].(float64), variadicArgs...)
	})
	return _c
}

func (_c *MockFloat64Histogram_Record_Call) Return() *MockFloat64Histogram_Record_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockFloat64Histogram_Record_Call) RunAndReturn(run func(context.Context, float64, ...metric.RecordOption)) *MockFloat64Histogram_Record_Call {
	_c.Run(run)
	return _c
}

// float64Histogram provides a mock function with no fields
func (_m *MockFloat64Histogram) float64Histogram() {
	_m.Called()
}

// MockFloat64Histogram_float64Histogram_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'float64Histogram'
type MockFloat64Histogram_float64Histogram_Call struct {
	*mock.Call
}

// float64Histogram is a helper method to define mock.On call
func (_e *MockFloat64Histogram_Expecter) float64Histogram() *MockFloat64Histogram_float64Histogram_Call {
	return &MockFloat64Histogram_float64Histogram_Call{Call: _e.mock.On("float64Histogram")}
}

func (_c *MockFloat64Histogram_float64Histogram_Call) Run(run func()) *MockFloat64Histogram_float64Histogram_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockFloat64Histogram_float64Histogram_Call) Return() *MockFloat64Histogram_float64Histogram_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockFloat64Histogram_float64Histogram_Call) RunAndReturn(run func()) *MockFloat64Histogram_float64Histogram_Call {
	_c.Run(run)
	return _c
}

// NewMockFloat64Histogram creates a new instance of MockFloat64Histogram. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockFloat64Histogram(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockFloat64Histogram {
	mock := &MockFloat64Histogram{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
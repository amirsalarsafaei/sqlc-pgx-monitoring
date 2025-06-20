// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package metric

import (
	mock "github.com/stretchr/testify/mock"

	_metricotelembedded "go.opentelemetry.io/otel/metric/embedded"
	_traceotelembedded "go.opentelemetry.io/otel/trace/embedded"
)

// NewMockInt64ObservableCounter creates a new instance of MockInt64ObservableCounter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockInt64ObservableCounter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockInt64ObservableCounter {
	mock := &MockInt64ObservableCounter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// MockInt64ObservableCounter is an autogenerated mock type for the Int64ObservableCounter type
type MockInt64ObservableCounter struct {
	mock.Mock
	_metricotelembedded.Meter
	_metricotelembedded.MeterProvider
	_traceotelembedded.Span
	_traceotelembedded.Tracer
	_traceotelembedded.TracerProvider
	_metricotelembedded.Float64Histogram
}

type MockInt64ObservableCounter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockInt64ObservableCounter) EXPECT() *MockInt64ObservableCounter_Expecter {
	return &MockInt64ObservableCounter_Expecter{mock: &_m.Mock}
}

// int64Observable provides a mock function for the type MockInt64ObservableCounter
func (_mock *MockInt64ObservableCounter) int64Observable() {
	_mock.Called()
	return
}

// MockInt64ObservableCounter_int64Observable_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'int64Observable'
type MockInt64ObservableCounter_int64Observable_Call struct {
	*mock.Call
}

// int64Observable is a helper method to define mock.On call
func (_e *MockInt64ObservableCounter_Expecter) int64Observable() *MockInt64ObservableCounter_int64Observable_Call {
	return &MockInt64ObservableCounter_int64Observable_Call{Call: _e.mock.On("int64Observable")}
}

func (_c *MockInt64ObservableCounter_int64Observable_Call) Run(run func()) *MockInt64ObservableCounter_int64Observable_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockInt64ObservableCounter_int64Observable_Call) Return() *MockInt64ObservableCounter_int64Observable_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockInt64ObservableCounter_int64Observable_Call) RunAndReturn(run func()) *MockInt64ObservableCounter_int64Observable_Call {
	_c.Run(run)
	return _c
}

// int64ObservableCounter provides a mock function for the type MockInt64ObservableCounter
func (_mock *MockInt64ObservableCounter) int64ObservableCounter() {
	_mock.Called()
	return
}

// MockInt64ObservableCounter_int64ObservableCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'int64ObservableCounter'
type MockInt64ObservableCounter_int64ObservableCounter_Call struct {
	*mock.Call
}

// int64ObservableCounter is a helper method to define mock.On call
func (_e *MockInt64ObservableCounter_Expecter) int64ObservableCounter() *MockInt64ObservableCounter_int64ObservableCounter_Call {
	return &MockInt64ObservableCounter_int64ObservableCounter_Call{Call: _e.mock.On("int64ObservableCounter")}
}

func (_c *MockInt64ObservableCounter_int64ObservableCounter_Call) Run(run func()) *MockInt64ObservableCounter_int64ObservableCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockInt64ObservableCounter_int64ObservableCounter_Call) Return() *MockInt64ObservableCounter_int64ObservableCounter_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockInt64ObservableCounter_int64ObservableCounter_Call) RunAndReturn(run func()) *MockInt64ObservableCounter_int64ObservableCounter_Call {
	_c.Run(run)
	return _c
}

// observable provides a mock function for the type MockInt64ObservableCounter
func (_mock *MockInt64ObservableCounter) observable() {
	_mock.Called()
	return
}

// MockInt64ObservableCounter_observable_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'observable'
type MockInt64ObservableCounter_observable_Call struct {
	*mock.Call
}

// observable is a helper method to define mock.On call
func (_e *MockInt64ObservableCounter_Expecter) observable() *MockInt64ObservableCounter_observable_Call {
	return &MockInt64ObservableCounter_observable_Call{Call: _e.mock.On("observable")}
}

func (_c *MockInt64ObservableCounter_observable_Call) Run(run func()) *MockInt64ObservableCounter_observable_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockInt64ObservableCounter_observable_Call) Return() *MockInt64ObservableCounter_observable_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockInt64ObservableCounter_observable_Call) RunAndReturn(run func()) *MockInt64ObservableCounter_observable_Call {
	_c.Run(run)
	return _c
}

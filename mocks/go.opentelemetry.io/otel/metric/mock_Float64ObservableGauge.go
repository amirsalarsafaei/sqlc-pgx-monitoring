// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import mock "github.com/stretchr/testify/mock"

// MockFloat64ObservableGauge is an autogenerated mock type for the Float64ObservableGauge type
type MockFloat64ObservableGauge struct {
	mock.Mock
}

type MockFloat64ObservableGauge_Expecter struct {
	mock *mock.Mock
}

func (_m *MockFloat64ObservableGauge) EXPECT() *MockFloat64ObservableGauge_Expecter {
	return &MockFloat64ObservableGauge_Expecter{mock: &_m.Mock}
}

// float64Observable provides a mock function with no fields
func (_m *MockFloat64ObservableGauge) float64Observable() {
	_m.Called()
}

// MockFloat64ObservableGauge_float64Observable_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'float64Observable'
type MockFloat64ObservableGauge_float64Observable_Call struct {
	*mock.Call
}

// float64Observable is a helper method to define mock.On call
func (_e *MockFloat64ObservableGauge_Expecter) float64Observable() *MockFloat64ObservableGauge_float64Observable_Call {
	return &MockFloat64ObservableGauge_float64Observable_Call{Call: _e.mock.On("float64Observable")}
}

func (_c *MockFloat64ObservableGauge_float64Observable_Call) Run(run func()) *MockFloat64ObservableGauge_float64Observable_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockFloat64ObservableGauge_float64Observable_Call) Return() *MockFloat64ObservableGauge_float64Observable_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockFloat64ObservableGauge_float64Observable_Call) RunAndReturn(run func()) *MockFloat64ObservableGauge_float64Observable_Call {
	_c.Run(run)
	return _c
}

// float64ObservableGauge provides a mock function with no fields
func (_m *MockFloat64ObservableGauge) float64ObservableGauge() {
	_m.Called()
}

// MockFloat64ObservableGauge_float64ObservableGauge_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'float64ObservableGauge'
type MockFloat64ObservableGauge_float64ObservableGauge_Call struct {
	*mock.Call
}

// float64ObservableGauge is a helper method to define mock.On call
func (_e *MockFloat64ObservableGauge_Expecter) float64ObservableGauge() *MockFloat64ObservableGauge_float64ObservableGauge_Call {
	return &MockFloat64ObservableGauge_float64ObservableGauge_Call{Call: _e.mock.On("float64ObservableGauge")}
}

func (_c *MockFloat64ObservableGauge_float64ObservableGauge_Call) Run(run func()) *MockFloat64ObservableGauge_float64ObservableGauge_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockFloat64ObservableGauge_float64ObservableGauge_Call) Return() *MockFloat64ObservableGauge_float64ObservableGauge_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockFloat64ObservableGauge_float64ObservableGauge_Call) RunAndReturn(run func()) *MockFloat64ObservableGauge_float64ObservableGauge_Call {
	_c.Run(run)
	return _c
}

// observable provides a mock function with no fields
func (_m *MockFloat64ObservableGauge) observable() {
	_m.Called()
}

// MockFloat64ObservableGauge_observable_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'observable'
type MockFloat64ObservableGauge_observable_Call struct {
	*mock.Call
}

// observable is a helper method to define mock.On call
func (_e *MockFloat64ObservableGauge_Expecter) observable() *MockFloat64ObservableGauge_observable_Call {
	return &MockFloat64ObservableGauge_observable_Call{Call: _e.mock.On("observable")}
}

func (_c *MockFloat64ObservableGauge_observable_Call) Run(run func()) *MockFloat64ObservableGauge_observable_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockFloat64ObservableGauge_observable_Call) Return() *MockFloat64ObservableGauge_observable_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockFloat64ObservableGauge_observable_Call) RunAndReturn(run func()) *MockFloat64ObservableGauge_observable_Call {
	_c.Run(run)
	return _c
}

// NewMockFloat64ObservableGauge creates a new instance of MockFloat64ObservableGauge. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockFloat64ObservableGauge(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockFloat64ObservableGauge {
	mock := &MockFloat64ObservableGauge{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

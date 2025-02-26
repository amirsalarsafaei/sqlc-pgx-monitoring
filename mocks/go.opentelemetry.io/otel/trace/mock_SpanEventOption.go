// Code generated by mockery v2.50.1. DO NOT EDIT.

package trace

import (
	mock "github.com/stretchr/testify/mock"
	trace "go.opentelemetry.io/otel/trace"
)

// MockSpanEventOption is an autogenerated mock type for the SpanEventOption type
type MockSpanEventOption struct {
	mock.Mock
}

type MockSpanEventOption_Expecter struct {
	mock *mock.Mock
}

func (_m *MockSpanEventOption) EXPECT() *MockSpanEventOption_Expecter {
	return &MockSpanEventOption_Expecter{mock: &_m.Mock}
}

// applyEvent provides a mock function with given fields: _a0
func (_m *MockSpanEventOption) applyEvent(_a0 trace.EventConfig) trace.EventConfig {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for applyEvent")
	}

	var r0 trace.EventConfig
	if rf, ok := ret.Get(0).(func(trace.EventConfig) trace.EventConfig); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(trace.EventConfig)
	}

	return r0
}

// MockSpanEventOption_applyEvent_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'applyEvent'
type MockSpanEventOption_applyEvent_Call struct {
	*mock.Call
}

// applyEvent is a helper method to define mock.On call
//   - _a0 trace.EventConfig
func (_e *MockSpanEventOption_Expecter) applyEvent(_a0 interface{}) *MockSpanEventOption_applyEvent_Call {
	return &MockSpanEventOption_applyEvent_Call{Call: _e.mock.On("applyEvent", _a0)}
}

func (_c *MockSpanEventOption_applyEvent_Call) Run(run func(_a0 trace.EventConfig)) *MockSpanEventOption_applyEvent_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(trace.EventConfig))
	})
	return _c
}

func (_c *MockSpanEventOption_applyEvent_Call) Return(_a0 trace.EventConfig) *MockSpanEventOption_applyEvent_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockSpanEventOption_applyEvent_Call) RunAndReturn(run func(trace.EventConfig) trace.EventConfig) *MockSpanEventOption_applyEvent_Call {
	_c.Call.Return(run)
	return _c
}

// applySpanEnd provides a mock function with given fields: _a0
func (_m *MockSpanEventOption) applySpanEnd(_a0 trace.SpanConfig) trace.SpanConfig {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for applySpanEnd")
	}

	var r0 trace.SpanConfig
	if rf, ok := ret.Get(0).(func(trace.SpanConfig) trace.SpanConfig); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(trace.SpanConfig)
	}

	return r0
}

// MockSpanEventOption_applySpanEnd_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'applySpanEnd'
type MockSpanEventOption_applySpanEnd_Call struct {
	*mock.Call
}

// applySpanEnd is a helper method to define mock.On call
//   - _a0 trace.SpanConfig
func (_e *MockSpanEventOption_Expecter) applySpanEnd(_a0 interface{}) *MockSpanEventOption_applySpanEnd_Call {
	return &MockSpanEventOption_applySpanEnd_Call{Call: _e.mock.On("applySpanEnd", _a0)}
}

func (_c *MockSpanEventOption_applySpanEnd_Call) Run(run func(_a0 trace.SpanConfig)) *MockSpanEventOption_applySpanEnd_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(trace.SpanConfig))
	})
	return _c
}

func (_c *MockSpanEventOption_applySpanEnd_Call) Return(_a0 trace.SpanConfig) *MockSpanEventOption_applySpanEnd_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockSpanEventOption_applySpanEnd_Call) RunAndReturn(run func(trace.SpanConfig) trace.SpanConfig) *MockSpanEventOption_applySpanEnd_Call {
	_c.Call.Return(run)
	return _c
}

// applySpanStart provides a mock function with given fields: _a0
func (_m *MockSpanEventOption) applySpanStart(_a0 trace.SpanConfig) trace.SpanConfig {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for applySpanStart")
	}

	var r0 trace.SpanConfig
	if rf, ok := ret.Get(0).(func(trace.SpanConfig) trace.SpanConfig); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(trace.SpanConfig)
	}

	return r0
}

// MockSpanEventOption_applySpanStart_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'applySpanStart'
type MockSpanEventOption_applySpanStart_Call struct {
	*mock.Call
}

// applySpanStart is a helper method to define mock.On call
//   - _a0 trace.SpanConfig
func (_e *MockSpanEventOption_Expecter) applySpanStart(_a0 interface{}) *MockSpanEventOption_applySpanStart_Call {
	return &MockSpanEventOption_applySpanStart_Call{Call: _e.mock.On("applySpanStart", _a0)}
}

func (_c *MockSpanEventOption_applySpanStart_Call) Run(run func(_a0 trace.SpanConfig)) *MockSpanEventOption_applySpanStart_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(trace.SpanConfig))
	})
	return _c
}

func (_c *MockSpanEventOption_applySpanStart_Call) Return(_a0 trace.SpanConfig) *MockSpanEventOption_applySpanStart_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockSpanEventOption_applySpanStart_Call) RunAndReturn(run func(trace.SpanConfig) trace.SpanConfig) *MockSpanEventOption_applySpanStart_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockSpanEventOption creates a new instance of MockSpanEventOption. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockSpanEventOption(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockSpanEventOption {
	mock := &MockSpanEventOption{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.50.1. DO NOT EDIT.

package trace

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	trace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

// MockTracer is an autogenerated mock type for the Tracer type
type MockTracer struct {
	mock.Mock
	embedded.Tracer
}

type MockTracer_Expecter struct {
	mock *mock.Mock
}

func (_m *MockTracer) EXPECT() *MockTracer_Expecter {
	return &MockTracer_Expecter{mock: &_m.Mock}
}

// Start provides a mock function with given fields: ctx, spanName, opts
func (_m *MockTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, spanName)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 context.Context
	var r1 trace.Span
	if rf, ok := ret.Get(0).(func(context.Context, string, ...trace.SpanStartOption) (context.Context, trace.Span)); ok {
		return rf(ctx, spanName, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, ...trace.SpanStartOption) context.Context); ok {
		r0 = rf(ctx, spanName, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(context.Context)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, ...trace.SpanStartOption) trace.Span); ok {
		r1 = rf(ctx, spanName, opts...)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(trace.Span)
		}
	}

	return r0, r1
}

// MockTracer_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type MockTracer_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
//   - ctx context.Context
//   - spanName string
//   - opts ...trace.SpanStartOption
func (_e *MockTracer_Expecter) Start(ctx interface{}, spanName interface{}, opts ...interface{}) *MockTracer_Start_Call {
	return &MockTracer_Start_Call{Call: _e.mock.On("Start",
		append([]interface{}{ctx, spanName}, opts...)...)}
}

func (_c *MockTracer_Start_Call) Run(run func(ctx context.Context, spanName string, opts ...trace.SpanStartOption)) *MockTracer_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]trace.SpanStartOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(trace.SpanStartOption)
			}
		}
		run(args[0].(context.Context), args[1].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockTracer_Start_Call) Return(_a0 context.Context, _a1 trace.Span) *MockTracer_Start_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockTracer_Start_Call) RunAndReturn(run func(context.Context, string, ...trace.SpanStartOption) (context.Context, trace.Span)) *MockTracer_Start_Call {
	_c.Call.Return(run)
	return _c
}

// tracer provides a mock function with no fields
func (_m *MockTracer) tracer() {
	_m.Called()
}

// MockTracer_tracer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'tracer'
type MockTracer_tracer_Call struct {
	*mock.Call
}

// tracer is a helper method to define mock.On call
func (_e *MockTracer_Expecter) tracer() *MockTracer_tracer_Call {
	return &MockTracer_tracer_Call{Call: _e.mock.On("tracer")}
}

func (_c *MockTracer_tracer_Call) Run(run func()) *MockTracer_tracer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockTracer_tracer_Call) Return() *MockTracer_tracer_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockTracer_tracer_Call) RunAndReturn(run func()) *MockTracer_tracer_Call {
	_c.Run(run)
	return _c
}

// NewMockTracer creates a new instance of MockTracer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTracer(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockTracer {
	mock := &MockTracer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
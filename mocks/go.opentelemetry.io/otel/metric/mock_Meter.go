// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import (
	mock "github.com/stretchr/testify/mock"
	metric "go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel/metric/embedded"
)

// MockMeter is an autogenerated mock type for the Meter type
type MockMeter struct {
	mock.Mock
	embedded.Meter
}

type MockMeter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockMeter) EXPECT() *MockMeter_Expecter {
	return &MockMeter_Expecter{mock: &_m.Mock}
}

// Float64Counter provides a mock function with given fields: name, options
func (_m *MockMeter) Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Float64Counter")
	}

	var r0 metric.Float64Counter
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64CounterOption) (metric.Float64Counter, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64CounterOption) metric.Float64Counter); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Float64Counter)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Float64CounterOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Float64Counter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Float64Counter'
type MockMeter_Float64Counter_Call struct {
	*mock.Call
}

// Float64Counter is a helper method to define mock.On call
//   - name string
//   - options ...metric.Float64CounterOption
func (_e *MockMeter_Expecter) Float64Counter(name interface{}, options ...interface{}) *MockMeter_Float64Counter_Call {
	return &MockMeter_Float64Counter_Call{Call: _e.mock.On("Float64Counter",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Float64Counter_Call) Run(run func(name string, options ...metric.Float64CounterOption)) *MockMeter_Float64Counter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Float64CounterOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Float64CounterOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Float64Counter_Call) Return(_a0 metric.Float64Counter, _a1 error) *MockMeter_Float64Counter_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Float64Counter_Call) RunAndReturn(run func(string, ...metric.Float64CounterOption) (metric.Float64Counter, error)) *MockMeter_Float64Counter_Call {
	_c.Call.Return(run)
	return _c
}

// Float64Gauge provides a mock function with given fields: name, options
func (_m *MockMeter) Float64Gauge(name string, options ...metric.Float64GaugeOption) (metric.Float64Gauge, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Float64Gauge")
	}

	var r0 metric.Float64Gauge
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64GaugeOption) (metric.Float64Gauge, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64GaugeOption) metric.Float64Gauge); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Float64Gauge)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Float64GaugeOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Float64Gauge_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Float64Gauge'
type MockMeter_Float64Gauge_Call struct {
	*mock.Call
}

// Float64Gauge is a helper method to define mock.On call
//   - name string
//   - options ...metric.Float64GaugeOption
func (_e *MockMeter_Expecter) Float64Gauge(name interface{}, options ...interface{}) *MockMeter_Float64Gauge_Call {
	return &MockMeter_Float64Gauge_Call{Call: _e.mock.On("Float64Gauge",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Float64Gauge_Call) Run(run func(name string, options ...metric.Float64GaugeOption)) *MockMeter_Float64Gauge_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Float64GaugeOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Float64GaugeOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Float64Gauge_Call) Return(_a0 metric.Float64Gauge, _a1 error) *MockMeter_Float64Gauge_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Float64Gauge_Call) RunAndReturn(run func(string, ...metric.Float64GaugeOption) (metric.Float64Gauge, error)) *MockMeter_Float64Gauge_Call {
	_c.Call.Return(run)
	return _c
}

// Float64Histogram provides a mock function with given fields: name, options
func (_m *MockMeter) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Float64Histogram")
	}

	var r0 metric.Float64Histogram
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64HistogramOption) (metric.Float64Histogram, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64HistogramOption) metric.Float64Histogram); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Float64Histogram)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Float64HistogramOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Float64Histogram_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Float64Histogram'
type MockMeter_Float64Histogram_Call struct {
	*mock.Call
}

// Float64Histogram is a helper method to define mock.On call
//   - name string
//   - options ...metric.Float64HistogramOption
func (_e *MockMeter_Expecter) Float64Histogram(name interface{}, options ...interface{}) *MockMeter_Float64Histogram_Call {
	return &MockMeter_Float64Histogram_Call{Call: _e.mock.On("Float64Histogram",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Float64Histogram_Call) Run(run func(name string, options ...metric.Float64HistogramOption)) *MockMeter_Float64Histogram_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Float64HistogramOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Float64HistogramOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Float64Histogram_Call) Return(_a0 metric.Float64Histogram, _a1 error) *MockMeter_Float64Histogram_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Float64Histogram_Call) RunAndReturn(run func(string, ...metric.Float64HistogramOption) (metric.Float64Histogram, error)) *MockMeter_Float64Histogram_Call {
	_c.Call.Return(run)
	return _c
}

// Float64ObservableCounter provides a mock function with given fields: name, options
func (_m *MockMeter) Float64ObservableCounter(name string, options ...metric.Float64ObservableCounterOption) (metric.Float64ObservableCounter, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Float64ObservableCounter")
	}

	var r0 metric.Float64ObservableCounter
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64ObservableCounterOption) (metric.Float64ObservableCounter, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64ObservableCounterOption) metric.Float64ObservableCounter); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Float64ObservableCounter)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Float64ObservableCounterOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Float64ObservableCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Float64ObservableCounter'
type MockMeter_Float64ObservableCounter_Call struct {
	*mock.Call
}

// Float64ObservableCounter is a helper method to define mock.On call
//   - name string
//   - options ...metric.Float64ObservableCounterOption
func (_e *MockMeter_Expecter) Float64ObservableCounter(name interface{}, options ...interface{}) *MockMeter_Float64ObservableCounter_Call {
	return &MockMeter_Float64ObservableCounter_Call{Call: _e.mock.On("Float64ObservableCounter",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Float64ObservableCounter_Call) Run(run func(name string, options ...metric.Float64ObservableCounterOption)) *MockMeter_Float64ObservableCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Float64ObservableCounterOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Float64ObservableCounterOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Float64ObservableCounter_Call) Return(_a0 metric.Float64ObservableCounter, _a1 error) *MockMeter_Float64ObservableCounter_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Float64ObservableCounter_Call) RunAndReturn(run func(string, ...metric.Float64ObservableCounterOption) (metric.Float64ObservableCounter, error)) *MockMeter_Float64ObservableCounter_Call {
	_c.Call.Return(run)
	return _c
}

// Float64ObservableGauge provides a mock function with given fields: name, options
func (_m *MockMeter) Float64ObservableGauge(name string, options ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Float64ObservableGauge")
	}

	var r0 metric.Float64ObservableGauge
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64ObservableGaugeOption) metric.Float64ObservableGauge); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Float64ObservableGauge)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Float64ObservableGaugeOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Float64ObservableGauge_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Float64ObservableGauge'
type MockMeter_Float64ObservableGauge_Call struct {
	*mock.Call
}

// Float64ObservableGauge is a helper method to define mock.On call
//   - name string
//   - options ...metric.Float64ObservableGaugeOption
func (_e *MockMeter_Expecter) Float64ObservableGauge(name interface{}, options ...interface{}) *MockMeter_Float64ObservableGauge_Call {
	return &MockMeter_Float64ObservableGauge_Call{Call: _e.mock.On("Float64ObservableGauge",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Float64ObservableGauge_Call) Run(run func(name string, options ...metric.Float64ObservableGaugeOption)) *MockMeter_Float64ObservableGauge_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Float64ObservableGaugeOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Float64ObservableGaugeOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Float64ObservableGauge_Call) Return(_a0 metric.Float64ObservableGauge, _a1 error) *MockMeter_Float64ObservableGauge_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Float64ObservableGauge_Call) RunAndReturn(run func(string, ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error)) *MockMeter_Float64ObservableGauge_Call {
	_c.Call.Return(run)
	return _c
}

// Float64ObservableUpDownCounter provides a mock function with given fields: name, options
func (_m *MockMeter) Float64ObservableUpDownCounter(name string, options ...metric.Float64ObservableUpDownCounterOption) (metric.Float64ObservableUpDownCounter, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Float64ObservableUpDownCounter")
	}

	var r0 metric.Float64ObservableUpDownCounter
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64ObservableUpDownCounterOption) (metric.Float64ObservableUpDownCounter, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64ObservableUpDownCounterOption) metric.Float64ObservableUpDownCounter); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Float64ObservableUpDownCounter)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Float64ObservableUpDownCounterOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Float64ObservableUpDownCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Float64ObservableUpDownCounter'
type MockMeter_Float64ObservableUpDownCounter_Call struct {
	*mock.Call
}

// Float64ObservableUpDownCounter is a helper method to define mock.On call
//   - name string
//   - options ...metric.Float64ObservableUpDownCounterOption
func (_e *MockMeter_Expecter) Float64ObservableUpDownCounter(name interface{}, options ...interface{}) *MockMeter_Float64ObservableUpDownCounter_Call {
	return &MockMeter_Float64ObservableUpDownCounter_Call{Call: _e.mock.On("Float64ObservableUpDownCounter",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Float64ObservableUpDownCounter_Call) Run(run func(name string, options ...metric.Float64ObservableUpDownCounterOption)) *MockMeter_Float64ObservableUpDownCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Float64ObservableUpDownCounterOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Float64ObservableUpDownCounterOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Float64ObservableUpDownCounter_Call) Return(_a0 metric.Float64ObservableUpDownCounter, _a1 error) *MockMeter_Float64ObservableUpDownCounter_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Float64ObservableUpDownCounter_Call) RunAndReturn(run func(string, ...metric.Float64ObservableUpDownCounterOption) (metric.Float64ObservableUpDownCounter, error)) *MockMeter_Float64ObservableUpDownCounter_Call {
	_c.Call.Return(run)
	return _c
}

// Float64UpDownCounter provides a mock function with given fields: name, options
func (_m *MockMeter) Float64UpDownCounter(name string, options ...metric.Float64UpDownCounterOption) (metric.Float64UpDownCounter, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Float64UpDownCounter")
	}

	var r0 metric.Float64UpDownCounter
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64UpDownCounterOption) (metric.Float64UpDownCounter, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Float64UpDownCounterOption) metric.Float64UpDownCounter); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Float64UpDownCounter)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Float64UpDownCounterOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Float64UpDownCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Float64UpDownCounter'
type MockMeter_Float64UpDownCounter_Call struct {
	*mock.Call
}

// Float64UpDownCounter is a helper method to define mock.On call
//   - name string
//   - options ...metric.Float64UpDownCounterOption
func (_e *MockMeter_Expecter) Float64UpDownCounter(name interface{}, options ...interface{}) *MockMeter_Float64UpDownCounter_Call {
	return &MockMeter_Float64UpDownCounter_Call{Call: _e.mock.On("Float64UpDownCounter",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Float64UpDownCounter_Call) Run(run func(name string, options ...metric.Float64UpDownCounterOption)) *MockMeter_Float64UpDownCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Float64UpDownCounterOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Float64UpDownCounterOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Float64UpDownCounter_Call) Return(_a0 metric.Float64UpDownCounter, _a1 error) *MockMeter_Float64UpDownCounter_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Float64UpDownCounter_Call) RunAndReturn(run func(string, ...metric.Float64UpDownCounterOption) (metric.Float64UpDownCounter, error)) *MockMeter_Float64UpDownCounter_Call {
	_c.Call.Return(run)
	return _c
}

// Int64Counter provides a mock function with given fields: name, options
func (_m *MockMeter) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Int64Counter")
	}

	var r0 metric.Int64Counter
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64CounterOption) (metric.Int64Counter, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64CounterOption) metric.Int64Counter); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Int64Counter)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Int64CounterOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Int64Counter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Int64Counter'
type MockMeter_Int64Counter_Call struct {
	*mock.Call
}

// Int64Counter is a helper method to define mock.On call
//   - name string
//   - options ...metric.Int64CounterOption
func (_e *MockMeter_Expecter) Int64Counter(name interface{}, options ...interface{}) *MockMeter_Int64Counter_Call {
	return &MockMeter_Int64Counter_Call{Call: _e.mock.On("Int64Counter",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Int64Counter_Call) Run(run func(name string, options ...metric.Int64CounterOption)) *MockMeter_Int64Counter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Int64CounterOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Int64CounterOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Int64Counter_Call) Return(_a0 metric.Int64Counter, _a1 error) *MockMeter_Int64Counter_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Int64Counter_Call) RunAndReturn(run func(string, ...metric.Int64CounterOption) (metric.Int64Counter, error)) *MockMeter_Int64Counter_Call {
	_c.Call.Return(run)
	return _c
}

// Int64Gauge provides a mock function with given fields: name, options
func (_m *MockMeter) Int64Gauge(name string, options ...metric.Int64GaugeOption) (metric.Int64Gauge, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Int64Gauge")
	}

	var r0 metric.Int64Gauge
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64GaugeOption) (metric.Int64Gauge, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64GaugeOption) metric.Int64Gauge); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Int64Gauge)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Int64GaugeOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Int64Gauge_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Int64Gauge'
type MockMeter_Int64Gauge_Call struct {
	*mock.Call
}

// Int64Gauge is a helper method to define mock.On call
//   - name string
//   - options ...metric.Int64GaugeOption
func (_e *MockMeter_Expecter) Int64Gauge(name interface{}, options ...interface{}) *MockMeter_Int64Gauge_Call {
	return &MockMeter_Int64Gauge_Call{Call: _e.mock.On("Int64Gauge",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Int64Gauge_Call) Run(run func(name string, options ...metric.Int64GaugeOption)) *MockMeter_Int64Gauge_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Int64GaugeOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Int64GaugeOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Int64Gauge_Call) Return(_a0 metric.Int64Gauge, _a1 error) *MockMeter_Int64Gauge_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Int64Gauge_Call) RunAndReturn(run func(string, ...metric.Int64GaugeOption) (metric.Int64Gauge, error)) *MockMeter_Int64Gauge_Call {
	_c.Call.Return(run)
	return _c
}

// Int64Histogram provides a mock function with given fields: name, options
func (_m *MockMeter) Int64Histogram(name string, options ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Int64Histogram")
	}

	var r0 metric.Int64Histogram
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64HistogramOption) (metric.Int64Histogram, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64HistogramOption) metric.Int64Histogram); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Int64Histogram)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Int64HistogramOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Int64Histogram_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Int64Histogram'
type MockMeter_Int64Histogram_Call struct {
	*mock.Call
}

// Int64Histogram is a helper method to define mock.On call
//   - name string
//   - options ...metric.Int64HistogramOption
func (_e *MockMeter_Expecter) Int64Histogram(name interface{}, options ...interface{}) *MockMeter_Int64Histogram_Call {
	return &MockMeter_Int64Histogram_Call{Call: _e.mock.On("Int64Histogram",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Int64Histogram_Call) Run(run func(name string, options ...metric.Int64HistogramOption)) *MockMeter_Int64Histogram_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Int64HistogramOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Int64HistogramOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Int64Histogram_Call) Return(_a0 metric.Int64Histogram, _a1 error) *MockMeter_Int64Histogram_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Int64Histogram_Call) RunAndReturn(run func(string, ...metric.Int64HistogramOption) (metric.Int64Histogram, error)) *MockMeter_Int64Histogram_Call {
	_c.Call.Return(run)
	return _c
}

// Int64ObservableCounter provides a mock function with given fields: name, options
func (_m *MockMeter) Int64ObservableCounter(name string, options ...metric.Int64ObservableCounterOption) (metric.Int64ObservableCounter, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Int64ObservableCounter")
	}

	var r0 metric.Int64ObservableCounter
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64ObservableCounterOption) (metric.Int64ObservableCounter, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64ObservableCounterOption) metric.Int64ObservableCounter); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Int64ObservableCounter)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Int64ObservableCounterOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Int64ObservableCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Int64ObservableCounter'
type MockMeter_Int64ObservableCounter_Call struct {
	*mock.Call
}

// Int64ObservableCounter is a helper method to define mock.On call
//   - name string
//   - options ...metric.Int64ObservableCounterOption
func (_e *MockMeter_Expecter) Int64ObservableCounter(name interface{}, options ...interface{}) *MockMeter_Int64ObservableCounter_Call {
	return &MockMeter_Int64ObservableCounter_Call{Call: _e.mock.On("Int64ObservableCounter",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Int64ObservableCounter_Call) Run(run func(name string, options ...metric.Int64ObservableCounterOption)) *MockMeter_Int64ObservableCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Int64ObservableCounterOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Int64ObservableCounterOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Int64ObservableCounter_Call) Return(_a0 metric.Int64ObservableCounter, _a1 error) *MockMeter_Int64ObservableCounter_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Int64ObservableCounter_Call) RunAndReturn(run func(string, ...metric.Int64ObservableCounterOption) (metric.Int64ObservableCounter, error)) *MockMeter_Int64ObservableCounter_Call {
	_c.Call.Return(run)
	return _c
}

// Int64ObservableGauge provides a mock function with given fields: name, options
func (_m *MockMeter) Int64ObservableGauge(name string, options ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Int64ObservableGauge")
	}

	var r0 metric.Int64ObservableGauge
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64ObservableGaugeOption) metric.Int64ObservableGauge); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Int64ObservableGauge)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Int64ObservableGaugeOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Int64ObservableGauge_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Int64ObservableGauge'
type MockMeter_Int64ObservableGauge_Call struct {
	*mock.Call
}

// Int64ObservableGauge is a helper method to define mock.On call
//   - name string
//   - options ...metric.Int64ObservableGaugeOption
func (_e *MockMeter_Expecter) Int64ObservableGauge(name interface{}, options ...interface{}) *MockMeter_Int64ObservableGauge_Call {
	return &MockMeter_Int64ObservableGauge_Call{Call: _e.mock.On("Int64ObservableGauge",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Int64ObservableGauge_Call) Run(run func(name string, options ...metric.Int64ObservableGaugeOption)) *MockMeter_Int64ObservableGauge_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Int64ObservableGaugeOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Int64ObservableGaugeOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Int64ObservableGauge_Call) Return(_a0 metric.Int64ObservableGauge, _a1 error) *MockMeter_Int64ObservableGauge_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Int64ObservableGauge_Call) RunAndReturn(run func(string, ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error)) *MockMeter_Int64ObservableGauge_Call {
	_c.Call.Return(run)
	return _c
}

// Int64ObservableUpDownCounter provides a mock function with given fields: name, options
func (_m *MockMeter) Int64ObservableUpDownCounter(name string, options ...metric.Int64ObservableUpDownCounterOption) (metric.Int64ObservableUpDownCounter, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Int64ObservableUpDownCounter")
	}

	var r0 metric.Int64ObservableUpDownCounter
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64ObservableUpDownCounterOption) (metric.Int64ObservableUpDownCounter, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64ObservableUpDownCounterOption) metric.Int64ObservableUpDownCounter); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Int64ObservableUpDownCounter)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Int64ObservableUpDownCounterOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Int64ObservableUpDownCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Int64ObservableUpDownCounter'
type MockMeter_Int64ObservableUpDownCounter_Call struct {
	*mock.Call
}

// Int64ObservableUpDownCounter is a helper method to define mock.On call
//   - name string
//   - options ...metric.Int64ObservableUpDownCounterOption
func (_e *MockMeter_Expecter) Int64ObservableUpDownCounter(name interface{}, options ...interface{}) *MockMeter_Int64ObservableUpDownCounter_Call {
	return &MockMeter_Int64ObservableUpDownCounter_Call{Call: _e.mock.On("Int64ObservableUpDownCounter",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Int64ObservableUpDownCounter_Call) Run(run func(name string, options ...metric.Int64ObservableUpDownCounterOption)) *MockMeter_Int64ObservableUpDownCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Int64ObservableUpDownCounterOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Int64ObservableUpDownCounterOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Int64ObservableUpDownCounter_Call) Return(_a0 metric.Int64ObservableUpDownCounter, _a1 error) *MockMeter_Int64ObservableUpDownCounter_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Int64ObservableUpDownCounter_Call) RunAndReturn(run func(string, ...metric.Int64ObservableUpDownCounterOption) (metric.Int64ObservableUpDownCounter, error)) *MockMeter_Int64ObservableUpDownCounter_Call {
	_c.Call.Return(run)
	return _c
}

// Int64UpDownCounter provides a mock function with given fields: name, options
func (_m *MockMeter) Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Int64UpDownCounter")
	}

	var r0 metric.Int64UpDownCounter
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error)); ok {
		return rf(name, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...metric.Int64UpDownCounterOption) metric.Int64UpDownCounter); ok {
		r0 = rf(name, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Int64UpDownCounter)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...metric.Int64UpDownCounterOption) error); ok {
		r1 = rf(name, options...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_Int64UpDownCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Int64UpDownCounter'
type MockMeter_Int64UpDownCounter_Call struct {
	*mock.Call
}

// Int64UpDownCounter is a helper method to define mock.On call
//   - name string
//   - options ...metric.Int64UpDownCounterOption
func (_e *MockMeter_Expecter) Int64UpDownCounter(name interface{}, options ...interface{}) *MockMeter_Int64UpDownCounter_Call {
	return &MockMeter_Int64UpDownCounter_Call{Call: _e.mock.On("Int64UpDownCounter",
		append([]interface{}{name}, options...)...)}
}

func (_c *MockMeter_Int64UpDownCounter_Call) Run(run func(name string, options ...metric.Int64UpDownCounterOption)) *MockMeter_Int64UpDownCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Int64UpDownCounterOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Int64UpDownCounterOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_Int64UpDownCounter_Call) Return(_a0 metric.Int64UpDownCounter, _a1 error) *MockMeter_Int64UpDownCounter_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_Int64UpDownCounter_Call) RunAndReturn(run func(string, ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error)) *MockMeter_Int64UpDownCounter_Call {
	_c.Call.Return(run)
	return _c
}

// RegisterCallback provides a mock function with given fields: f, instruments
func (_m *MockMeter) RegisterCallback(f metric.Callback, instruments ...metric.Observable) (metric.Registration, error) {
	_va := make([]interface{}, len(instruments))
	for _i := range instruments {
		_va[_i] = instruments[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, f)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for RegisterCallback")
	}

	var r0 metric.Registration
	var r1 error
	if rf, ok := ret.Get(0).(func(metric.Callback, ...metric.Observable) (metric.Registration, error)); ok {
		return rf(f, instruments...)
	}
	if rf, ok := ret.Get(0).(func(metric.Callback, ...metric.Observable) metric.Registration); ok {
		r0 = rf(f, instruments...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Registration)
		}
	}

	if rf, ok := ret.Get(1).(func(metric.Callback, ...metric.Observable) error); ok {
		r1 = rf(f, instruments...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockMeter_RegisterCallback_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RegisterCallback'
type MockMeter_RegisterCallback_Call struct {
	*mock.Call
}

// RegisterCallback is a helper method to define mock.On call
//   - f metric.Callback
//   - instruments ...metric.Observable
func (_e *MockMeter_Expecter) RegisterCallback(f interface{}, instruments ...interface{}) *MockMeter_RegisterCallback_Call {
	return &MockMeter_RegisterCallback_Call{Call: _e.mock.On("RegisterCallback",
		append([]interface{}{f}, instruments...)...)}
}

func (_c *MockMeter_RegisterCallback_Call) Run(run func(f metric.Callback, instruments ...metric.Observable)) *MockMeter_RegisterCallback_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.Observable, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.Observable)
			}
		}
		run(args[0].(metric.Callback), variadicArgs...)
	})
	return _c
}

func (_c *MockMeter_RegisterCallback_Call) Return(_a0 metric.Registration, _a1 error) *MockMeter_RegisterCallback_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockMeter_RegisterCallback_Call) RunAndReturn(run func(metric.Callback, ...metric.Observable) (metric.Registration, error)) *MockMeter_RegisterCallback_Call {
	_c.Call.Return(run)
	return _c
}

// meter provides a mock function with no fields
func (_m *MockMeter) meter() {
	_m.Called()
}

// MockMeter_meter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'meter'
type MockMeter_meter_Call struct {
	*mock.Call
}

// meter is a helper method to define mock.On call
func (_e *MockMeter_Expecter) meter() *MockMeter_meter_Call {
	return &MockMeter_meter_Call{Call: _e.mock.On("meter")}
}

func (_c *MockMeter_meter_Call) Run(run func()) *MockMeter_meter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockMeter_meter_Call) Return() *MockMeter_meter_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockMeter_meter_Call) RunAndReturn(run func()) *MockMeter_meter_Call {
	_c.Run(run)
	return _c
}

// NewMockMeter creates a new instance of MockMeter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockMeter(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockMeter {
	mock := &MockMeter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

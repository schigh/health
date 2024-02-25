// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ntwrk1/health (interfaces: Checker,Reporter)

// Package std_test is a generated GoMock package.
package std_test

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	healthpb "github.com/schigh/health/pkg/v1"
)

// MockChecker is a mock of Checker interface.
type MockChecker struct {
	ctrl     *gomock.Controller
	recorder *MockCheckerMockRecorder
}

// MockCheckerMockRecorder is the mock recorder for MockChecker.
type MockCheckerMockRecorder struct {
	mock *MockChecker
}

// NewMockChecker creates a new mock instance.
func NewMockChecker(ctrl *gomock.Controller) *MockChecker {
	mock := &MockChecker{ctrl: ctrl}
	mock.recorder = &MockCheckerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChecker) EXPECT() *MockCheckerMockRecorder {
	return m.recorder
}

// Check mocks base method.
func (m *MockChecker) Check(arg0 context.Context) *healthpb.Check {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Check", arg0)
	ret0, _ := ret[0].(*healthpb.Check)
	return ret0
}

// Check indicates an expected call of Check.
func (mr *MockCheckerMockRecorder) Check(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Check", reflect.TypeOf((*MockChecker)(nil).Check), arg0)
}

// MockReporter is a mock of Reporter interface.
type MockReporter struct {
	ctrl     *gomock.Controller
	recorder *MockReporterMockRecorder
}

// MockReporterMockRecorder is the mock recorder for MockReporter.
type MockReporterMockRecorder struct {
	mock *MockReporter
}

// NewMockReporter creates a new mock instance.
func NewMockReporter(ctrl *gomock.Controller) *MockReporter {
	mock := &MockReporter{ctrl: ctrl}
	mock.recorder = &MockReporterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockReporter) EXPECT() *MockReporterMockRecorder {
	return m.recorder
}

// Run mocks base method.
func (m *MockReporter) Run(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockReporterMockRecorder) Run(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockReporter)(nil).Run), arg0)
}

// SetLiveness mocks base method.
func (m *MockReporter) SetLiveness(arg0 context.Context, arg1 bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetLiveness", arg0, arg1)
}

// SetLiveness indicates an expected call of SetLiveness.
func (mr *MockReporterMockRecorder) SetLiveness(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetLiveness", reflect.TypeOf((*MockReporter)(nil).SetLiveness), arg0, arg1)
}

// SetReadiness mocks base method.
func (m *MockReporter) SetReadiness(arg0 context.Context, arg1 bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetReadiness", arg0, arg1)
}

// SetReadiness indicates an expected call of SetReadiness.
func (mr *MockReporterMockRecorder) SetReadiness(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetReadiness", reflect.TypeOf((*MockReporter)(nil).SetReadiness), arg0, arg1)
}

// Stop mocks base method.
func (m *MockReporter) Stop(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockReporterMockRecorder) Stop(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockReporter)(nil).Stop), arg0)
}

// UpdateCircuitBreakers mocks base method.
func (m *MockReporter) UpdateCircuitBreakers(arg0 context.Context, arg1 map[string]*healthpb.CircuitBreaker) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateCircuitBreakers", arg0, arg1)
}

// UpdateCircuitBreakers indicates an expected call of UpdateCircuitBreakers.
func (mr *MockReporterMockRecorder) UpdateCircuitBreakers(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCircuitBreakers", reflect.TypeOf((*MockReporter)(nil).UpdateCircuitBreakers), arg0, arg1)
}

// UpdateHealthChecks mocks base method.
func (m *MockReporter) UpdateHealthChecks(arg0 context.Context, arg1 map[string]*healthpb.Check) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateHealthChecks", arg0, arg1)
}

// UpdateHealthChecks indicates an expected call of UpdateHealthChecks.
func (mr *MockReporterMockRecorder) UpdateHealthChecks(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateHealthChecks", reflect.TypeOf((*MockReporter)(nil).UpdateHealthChecks), arg0, arg1)
}

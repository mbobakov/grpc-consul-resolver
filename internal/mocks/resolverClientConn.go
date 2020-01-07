// Code generated by MockGen. DO NOT EDIT.
// Source: google.golang.org/grpc/resolver (interfaces: ClientConn)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	resolver "google.golang.org/grpc/resolver"
	serviceconfig "google.golang.org/grpc/serviceconfig"
	reflect "reflect"
)

// MockClientConn is a mock of ClientConn interface
type MockClientConn struct {
	ctrl     *gomock.Controller
	recorder *MockClientConnMockRecorder
}

// MockClientConnMockRecorder is the mock recorder for MockClientConn
type MockClientConnMockRecorder struct {
	mock *MockClientConn
}

// NewMockClientConn creates a new mock instance
func NewMockClientConn(ctrl *gomock.Controller) *MockClientConn {
	mock := &MockClientConn{ctrl: ctrl}
	mock.recorder = &MockClientConnMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockClientConn) EXPECT() *MockClientConnMockRecorder {
	return m.recorder
}

// NewAddress mocks base method
func (m *MockClientConn) NewAddress(arg0 []resolver.Address) {
	m.ctrl.Call(m, "NewAddress", arg0)
}

// NewAddress indicates an expected call of NewAddress
func (mr *MockClientConnMockRecorder) NewAddress(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewAddress", reflect.TypeOf((*MockClientConn)(nil).NewAddress), arg0)
}

// NewServiceConfig mocks base method
func (m *MockClientConn) NewServiceConfig(arg0 string) {
	m.ctrl.Call(m, "NewServiceConfig", arg0)
}

// NewServiceConfig indicates an expected call of NewServiceConfig
func (mr *MockClientConnMockRecorder) NewServiceConfig(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewServiceConfig", reflect.TypeOf((*MockClientConn)(nil).NewServiceConfig), arg0)
}

// ParseServiceConfig mocks base method
func (m *MockClientConn) ParseServiceConfig(arg0 string) *serviceconfig.ParseResult {
	ret := m.ctrl.Call(m, "ParseServiceConfig", arg0)
	ret0, _ := ret[0].(*serviceconfig.ParseResult)
	return ret0
}

// ParseServiceConfig indicates an expected call of ParseServiceConfig
func (mr *MockClientConnMockRecorder) ParseServiceConfig(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParseServiceConfig", reflect.TypeOf((*MockClientConn)(nil).ParseServiceConfig), arg0)
}

// ReportError mocks base method
func (m *MockClientConn) ReportError(arg0 error) {
	m.ctrl.Call(m, "ReportError", arg0)
}

// ReportError indicates an expected call of ReportError
func (mr *MockClientConnMockRecorder) ReportError(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportError", reflect.TypeOf((*MockClientConn)(nil).ReportError), arg0)
}

// UpdateState mocks base method
func (m *MockClientConn) UpdateState(arg0 resolver.State) {
	m.ctrl.Call(m, "UpdateState", arg0)
}

// UpdateState indicates an expected call of UpdateState
func (mr *MockClientConnMockRecorder) UpdateState(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateState", reflect.TypeOf((*MockClientConn)(nil).UpdateState), arg0)
}

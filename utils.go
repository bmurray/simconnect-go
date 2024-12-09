package simconnect

import (
	"unsafe"

	"github.com/bmurray/simconnect-go/client"
)

// IsReport Convenience function to check if the data is the correct type
func IsReport[T any](s *client.SimConnect, ppData *client.RecvSimobjectDataByType) (*T, bool) {
	var typed *T
	defineId := s.GetDefineID(typed)
	if ppData.DefineID == defineId {
		return (*T)(unsafe.Pointer(ppData)), true
	}
	return nil, false
}

// RequestData Convenience function to request data
func RequestData[T any](s *client.SimConnect) error {
	var report *T
	defineId := s.GetDefineID(report)
	reqId := defineId
	return s.RequestDataOnSimObjectType(reqId, defineId, 0, client.SIMOBJECT_TYPE_USER)
}

package client

import (
	"fmt"
)

func (e RecvException) Error() string {
	return fmt.Sprintf("Exception (%d), ReqID (%d): %#v", e.Exception, e.SendID, e)
}

func (e RecvOpen) Error() string {
	return fmt.Sprintf("Open: %#v", e.ApplicationName)
}

type RecvEventError RecvEvent

func (e RecvEventError) Error() string {
	return fmt.Sprintf("Event: %#v", e.EventID)
}

type RecvExceptionID uint32

const (
	SIMCONNECT_EXCEPTION_NONE                              RecvExceptionID = 0
	SIMCONNECT_EXCEPTION_ERROR                                             = 1
	SIMCONNECT_EXCEPTION_SIZE_MISMATCH                                     = 2
	SIMCONNECT_EXCEPTION_UNRECOGNIZED_ID                                   = 3
	SIMCONNECT_EXCEPTION_UNOPENED                                          = 4
	SIMCONNECT_EXCEPTION_VERSION_MISMATCH                                  = 5
	SIMCONNECT_EXCEPTION_TOO_MANY_GROUPS                                   = 6
	SIMCONNECT_EXCEPTION_NAME_UNRECOGNIZED                                 = 7
	SIMCONNECT_EXCEPTION_TOO_MANY_EVENT_NAMES                              = 8
	SIMCONNECT_EXCEPTION_EVENT_ID_DUPLICATE                                = 9
	SIMCONNECT_EXCEPTION_TOO_MANY_MAPS                                     = 10
	SIMCONNECT_EXCEPTION_TOO_MANY_OBJECTS                                  = 11
	SIMCONNECT_EXCEPTION_TOO_MANY_REQUESTS                                 = 12
	SIMCONNECT_EXCEPTION_WEATHER_INVALID_PORT                              = 13
	SIMCONNECT_EXCEPTION_WEATHER_INVALID_METAR                             = 14
	SIMCONNECT_EXCEPTION_WEATHER_UNABLE_TO_GET_OBSERVATION                 = 15
	SIMCONNECT_EXCEPTION_WEATHER_UNABLE_TO_CREATE_STATION                  = 16
	SIMCONNECT_EXCEPTION_WEATHER_UNABLE_TO_REMOVE_STATION                  = 17
	SIMCONNECT_EXCEPTION_INVALID_DATA_TYPE                                 = 18
	SIMCONNECT_EXCEPTION_INVALID_DATA_SIZE                                 = 19
	SIMCONNECT_EXCEPTION_DATA_ERROR                                        = 20
	SIMCONNECT_EXCEPTION_INVALID_ARRAY                                     = 21
	SIMCONNECT_EXCEPTION_CREATE_OBJECT_FAILED                              = 22
	SIMCONNECT_EXCEPTION_LOAD_FLIGHTPLAN_FAILED                            = 23
	SIMCONNECT_EXCEPTION_OPERATION_INVALID_FOR_OJBECT_TYPE                 = 24
	SIMCONNECT_EXCEPTION_ILLEGAL_OPERATION                                 = 25
	SIMCONNECT_EXCEPTION_ALREADY_SUBSCRIBED                                = 26
	SIMCONNECT_EXCEPTION_INVALID_ENUM                                      = 27
	SIMCONNECT_EXCEPTION_DEFINITION_ERROR                                  = 28
	SIMCONNECT_EXCEPTION_DUPLICATE_ID                                      = 29
	SIMCONNECT_EXCEPTION_DATUM_ID                                          = 30
	SIMCONNECT_EXCEPTION_OUT_OF_BOUNDS                                     = 31
	SIMCONNECT_EXCEPTION_ALREADY_CREATED                                   = 32
	SIMCONNECT_EXCEPTION_OBJECT_OUTSIDE_REALITY_BUBBLE                     = 33
	SIMCONNECT_EXCEPTION_OBJECT_CONTAINER                                  = 34
	SIMCONNECT_EXCEPTION_OBJECT_AI                                         = 35
	SIMCONNECT_EXCEPTION_OBJECT_ATC                                        = 36
	SIMCONNECT_EXCEPTION_OBJECT_SCHEDULE                                   = 37
)

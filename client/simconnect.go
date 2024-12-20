package client

// MSFS-SDK/SimConnect\ SDK/include/SimConnect.h
// MSFS-SDK/SimConnect\ SDK/lib/SimConnect.dll

import (
	"fmt"
	"log/slog"
	"reflect"
	"syscall"
	"unsafe"
)

// SimConnect is the main struct for connecting to SimConnect
type SimConnect struct {
	handle      unsafe.Pointer
	defineMap   map[string]DWORD
	lastEventID DWORD

	dllPath string
	dll     *dll
	log     *slog.Logger
}

// SimConnectOption is a function that sets options on the SimConnect
type SimConnectOption func(*SimConnect)

// WithLogger sets the logger for the SimConnect
func WithLogger(l *slog.Logger) SimConnectOption {
	return func(s *SimConnect) {
		s.log = l.With("module", "simconnect")
	}
}

// WithDLLPath sets the path to the SimConnect DLL
func WithDLLPath(path string) SimConnectOption {
	return func(s *SimConnect) {
		s.dllPath = path
	}
}

// New creates a new SimConnect connection
func New(name string, opts ...SimConnectOption) (*SimConnect, error) {
	s := &SimConnect{
		defineMap:   map[string]DWORD{"_last": 0},
		lastEventID: 0,
		log:         slog.With("name", name, "module", "simconnect"),
	}

	for _, opt := range opts {
		opt(s)
	}
	if s.dllPath != "" {
		d, err := newDLL(s.dllPath)
		if err != nil {
			return nil, err
		}
		s.dll = d
	} else if defaultDll == nil {
		return nil, fmt.Errorf("no default DLL")
	} else {
		s.dll = defaultDll
	}

	// SimConnect_Open(
	//   HANDLE * phSimConnect,
	//   LPCSTR szName,
	//   HWND hWnd,
	//   DWORD UserEventWin32,
	//   HANDLE hEventHandle,
	//   DWORD ConfigIndex
	// );
	args := []uintptr{
		uintptr(unsafe.Pointer(&s.handle)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
		0,
		0,
		0,
		0,
	}

	r1, _, err := s.dll.proc_SimConnect_Open.Call(args...)
	if int32(r1) < 0 {
		return nil, fmt.Errorf("SimConnect_Open error: %s", err)
	}
	return s, nil
}

// GetEventID returns a new event ID
func (s *SimConnect) GetEventID() DWORD {
	id := s.lastEventID
	s.lastEventID += 1
	return id
}

// GetDefineID returns the define ID for a struct
func (s *SimConnect) GetDefineID(a interface{}) DWORD {
	t := reflect.TypeOf(a)
	if t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	structName := t.Name()

	id, ok := s.defineMap[structName]
	if !ok {
		id = s.defineMap["_last"]
		s.defineMap[structName] = id
		s.defineMap["_last"] = id + 1
	}

	return id
}

// RegisterDataDefinition registers a struct for data definition
func (s *SimConnect) RegisterDataDefinition(a interface{}) error {
	defineID := s.GetDefineID(a)
	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	for j := 1; j < v.NumField(); j++ {
		fieldName := v.Type().Field(j).Name
		nameTag, _ := v.Type().Field(j).Tag.Lookup("name")
		unitTag, _ := v.Type().Field(j).Tag.Lookup("unit")

		fieldType := v.Field(j).Kind().String()
		if fieldType == "array" {
			fieldType = fmt.Sprintf("[%d]byte", v.Field(j).Type().Len())
		}

		if nameTag == "" {
			return fmt.Errorf("%s name tag not found", fieldName)
		}

		dataType, err := derefDataType(fieldType)
		if err != nil {
			return err
		}

		s.AddToDataDefinition(defineID, nameTag, unitTag, dataType)
	}

	return nil
}

// Close closes the SimConnect connection
func (s *SimConnect) Close() error {
	// SimConnect_Open(
	//   HANDLE * phSimConnect,
	// );
	r1, _, err := s.dll.proc_SimConnect_Close.Call(uintptr(s.handle))
	if int32(r1) < 0 {
		return fmt.Errorf("SimConnect_Close error: %d %s", int32(r1), err)
	}
	return nil
}

// derefDataType returns the SimConnect data type for a Go type
func (s *SimConnect) AddToDataDefinition(defineID DWORD, name, unit string, dataType DWORD) error {
	// SimConnect_AddToDataDefinition(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_DATA_DEFINITION_ID DefineID,
	//   const char * DatumName,
	//   const char * UnitsName,
	//   SIMCONNECT_DATATYPE DatumType = SIMCONNECT_DATATYPE_FLOAT64,
	//   float fEpsilon = 0,
	//   DWORD DatumID = SIMCONNECT_UNUSED
	// );

	_name := []byte(name + "\x00")
	_unit := []byte(unit + "\x00")

	args := []uintptr{
		uintptr(s.handle),
		uintptr(defineID),
		uintptr(unsafe.Pointer(&_name[0])),
		uintptr(0),
		uintptr(dataType),
		uintptr(float32(0)),
		uintptr(UNUSED),
	}
	if unit != "" {
		args[3] = uintptr(unsafe.Pointer(&_unit[0]))
	}

	r1, _, err := s.dll.proc_SimConnect_AddToDataDefinition.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("SimConnect_AddToDataDefinition for %s error: %d %s", name, r1, err)
	}

	return nil
}

func (s *SimConnect) SubscribeToSystemEvent(eventID DWORD, eventName string) error {
	// SimConnect_SubscribeToSystemEvent(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_CLIENT_EVENT_ID EventID,
	//   const char * SystemEventName
	// );

	_eventName := []byte(eventName + "\x00")

	args := []uintptr{
		uintptr(s.handle),
		uintptr(eventID),
		uintptr(unsafe.Pointer(&_eventName[0])),
	}

	r1, _, err := s.dll.proc_SimConnect_SubscribeToSystemEvent.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("SimConnect_SubscribeToSystemEvent for %s error: %d %s", eventName, r1, err)
	}

	return nil
}

func (s *SimConnect) RequestDataOnSimObjectType(requestID, defineID, radius, simobjectType DWORD) error {
	// SimConnect_RequestDataOnSimObjectType(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_DATA_REQUEST_ID RequestID,
	//   SIMCONNECT_DATA_DEFINITION_ID DefineID,
	//   DWORD dwRadiusMeters,
	//   SIMCONNECT_SIMOBJECT_TYPE type
	// );
	args := []uintptr{
		uintptr(s.handle),
		uintptr(requestID),
		uintptr(defineID),
		uintptr(radius),
		uintptr(simobjectType),
	}

	r1, _, err := s.dll.proc_SimConnect_RequestDataOnSimObjectType.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_RequestDataOnSimObjectType for requestID %d defineID %d error: %d %s",
			requestID, defineID, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) RequestDataOnSimObject(requestID, defineID, objectID, period, flags, origin, interval, limit DWORD) error {
	// SimConnect_RequestDataOnSimObject(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_DATA_REQUEST_ID RequestID,
	//   SIMCONNECT_DATA_DEFINITION_ID DefineID,
	//   SIMCONNECT_OBJECT_ID ObjectID,
	//   SIMCONNECT_PERIOD Period,
	//   SIMCONNECT_DATA_REQUEST_FLAG Flags = 0,
	//   DWORD origin = 0,
	//   DWORD interval = 0,
	//   DWORD limit = 0
	// );

	args := []uintptr{
		uintptr(s.handle),
		uintptr(requestID),
		uintptr(defineID),
		uintptr(objectID),
		uintptr(period),
		uintptr(flags),
		uintptr(origin),
		uintptr(interval),
		uintptr(limit),
	}

	r1, _, err := s.dll.proc_SimConnect_RequestDataOnSimObject.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_RequestDataOnSimObject for requestID %d defineID %d error: %d %s",
			requestID, defineID, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) SetDataOnSimObject(defineID, simobjectType, flags, arrayCount, size DWORD, buf unsafe.Pointer) error {
	//s.SetDataOnSimObject(defineID, simconnect.OBJECT_ID_USER, 0, 0, size, buf)

	// SimConnect_SetDataOnSimObject(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_DATA_DEFINITION_ID DefineID,
	//   SIMCONNECT_OBJECT_ID ObjectID,
	//   SIMCONNECT_DATA_SET_FLAG Flags,
	//   DWORD ArrayCount,
	//   DWORD cbUnitSize,
	//   void * pDataSet
	// );
	args := []uintptr{
		uintptr(s.handle),
		uintptr(defineID),
		uintptr(simobjectType),
		uintptr(flags),
		uintptr(arrayCount),
		uintptr(size),
		uintptr(buf),
	}

	r1, _, err := s.dll.proc_SimConnect_SetDataOnSimObject.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_SetDataOnSimObject for defineID %d error: %d %s",
			defineID, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) SubscribeToFacilities(facilityType, requestID DWORD) error {
	// SimConnect_SubscribeToFacilities(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_FACILITY_LIST_TYPE type,
	//   SIMCONNECT_DATA_REQUEST_ID RequestID
	// );

	args := []uintptr{
		uintptr(s.handle),
		uintptr(facilityType),
		uintptr(requestID),
	}

	r1, _, err := s.dll.proc_SimConnect_SubscribeToFacilities.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_SubscribeToFacilities for type %d error: %d %s",
			facilityType, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) UnsubscribeToFacilities(facilityType DWORD) error {
	// SimConnect_UnsubscribeToFacilities(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_FACILITY_LIST_TYPE type
	// );

	args := []uintptr{
		uintptr(s.handle),
		uintptr(facilityType),
	}

	r1, _, err := s.dll.proc_SimConnect_UnsubscribeToFacilities.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"UnsubscribeToFacilities for type %d error: %d %s",
			facilityType, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) RequestFacilitiesList(facilityType, requestID DWORD) error {
	// SimConnect_RequestFacilitiesList(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_FACILITY_LIST_TYPE type,
	//   SIMCONNECT_DATA_REQUEST_ID RequestID
	// );

	args := []uintptr{
		uintptr(s.handle),
		uintptr(facilityType),
		uintptr(requestID),
	}

	r1, _, err := s.dll.proc_SimConnect_RequestFacilitiesList.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_RequestFacilitiesList for type %d error: %d %s",
			facilityType, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) MapClientEventToSimEvent(eventID DWORD, eventName string) error {
	// SimConnect_MapClientEventToSimEvent(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_CLIENT_EVENT_ID EventID,
	//   const char * EventName = ""
	// );

	_eventName := []byte(eventName + "\x00")

	args := []uintptr{
		uintptr(s.handle),
		uintptr(eventID),
		uintptr(unsafe.Pointer(&_eventName[0])),
	}

	r1, _, err := s.dll.proc_SimConnect_MapClientEventToSimEvent.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_MapClientEventToSimEvent for eventID %d error: %d %s",
			eventID, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) TransmitClientEvent(objectID, eventID, dwData, groupID, flags DWORD) error {

	r1, _, err := s.dll.proc_SimConnect_TransmitClientEvent.Call(
		uintptr(s.handle),
		uintptr(objectID),
		uintptr(eventID),
		uintptr(dwData),
		uintptr(groupID),
		uintptr(flags),
	)
	if int32(r1) < 0 {
		return fmt.Errorf("SimConnect_TransmitClientEvent for eventID %d error: %d %s", eventID, r1, err)
	}

	return nil
}

func (s *SimConnect) MenuAddItem(menuItem string, menuEventID, Data DWORD) error {
	// SimConnect_MenuAddItem(
	//   HANDLE hSimConnect,
	//   const char * szMenuItem,
	//   SIMCONNECT_CLIENT_EVENT_ID MenuEventID,
	//   DWORD dwData
	// );

	_menuItem := []byte(menuItem + "\x00")

	args := []uintptr{
		uintptr(s.handle),
		uintptr(unsafe.Pointer(&_menuItem[0])),
		uintptr(menuEventID),
		uintptr(Data),
	}

	r1, _, err := s.dll.proc_SimConnect_MenuAddItem.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_MenuAddItem for menuEventID %d '%s' error: %d %s",
			menuEventID, menuItem, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) MenuDeleteItem(menuItem string, menuEventID, Data DWORD) error {
	// SimConnect_MenuDeleteItem(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_CLIENT_EVENT_ID MenuEventID
	// );

	args := []uintptr{
		uintptr(s.handle),
		uintptr(menuEventID),
	}

	r1, _, err := s.dll.proc_SimConnect_MenuDeleteItem.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_MenuDeleteItem for menuEventID %d error: %d %s",
			menuEventID, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) AddClientEventToNotificationGroup(groupID, eventID DWORD) error {
	// SimConnect_AddClientEventToNotificationGroup(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_NOTIFICATION_GROUP_ID GroupID,
	//   SIMCONNECT_CLIENT_EVENT_ID EventID,
	//   BOOL bMaskable = FALSE
	// );

	args := []uintptr{
		uintptr(s.handle),
		uintptr(groupID),
		uintptr(eventID),
	}

	r1, _, err := s.dll.proc_SimConnect_AddClientEventToNotificationGroup.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_AddClientEventToNotificationGroup for groupID %d eventID %d error: %d %s",
			groupID, eventID, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) SetNotificationGroupPriority(groupID, priority DWORD) error {
	// SimConnect_SetNotificationGroupPriority(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_NOTIFICATION_GROUP_ID GroupID,
	//   DWORD uPriority
	// );

	args := []uintptr{
		uintptr(s.handle),
		uintptr(groupID),
		uintptr(priority),
	}

	r1, _, err := s.dll.proc_SimConnect_SetNotificationGroupPriority.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_SetNotificationGroupPriority for groupID %d priority %d error: %d %s",
			groupID, priority, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) ShowText(textType DWORD, duration float64, eventID DWORD, text string) error {
	// SimConnect_Text(
	//   HANDLE hSimConnect,
	//   SIMCONNECT_TEXT_TYPE type,
	//   float fTimeSeconds,
	//   SIMCONNECT_CLIENT_EVENT_ID EventID,
	//   DWORD cbUnitSize,
	//   void * pDataSet
	// );

	_text := []byte(text + "\x00")

	args := []uintptr{
		uintptr(s.handle),
		uintptr(textType),
		uintptr(duration),
		uintptr(eventID),
		uintptr(DWORD(len(_text))),
		uintptr(unsafe.Pointer(&_text[0])),
	}

	r1, _, err := s.dll.proc_SimConnect_Text.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf(
			"SimConnect_Text for eventID %d textType %d text '%s' error: %d %s",
			eventID, textType, text, r1, err,
		)
	}

	return nil
}

func (s *SimConnect) GetNextDispatch() (unsafe.Pointer, int32, error) {
	var ppData unsafe.Pointer
	var ppDataLength DWORD

	r1, _, err := s.dll.proc_SimConnect_GetNextDispatch.Call(
		uintptr(s.handle),
		uintptr(unsafe.Pointer(&ppData)),
		uintptr(unsafe.Pointer(&ppDataLength)),
	)

	return ppData, int32(r1), err
}

// SetData currently only supports float64 fields
func (s *SimConnect) SetData(fr any) error {
	defineId := s.GetDefineID(fr)

	cnt := 0

	val := reflect.ValueOf(fr)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("not a struct: %s", typ.Kind().String())
	}
	buf := []float64{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := field.Tag.Get("name")
		if name == "" {
			continue
		}
		if field.Type.Kind() != reflect.Float64 {
			return fmt.Errorf("not a float64: %s -- %s", field.Name, field.Type.Kind().String())
		}
		buf = append(buf, val.Field(i).Float())
		cnt++
	}

	size := DWORD(cnt * 8)
	slog.Debug("Setting data", "defineid", defineId, "count", cnt, "size", size)
	return s.SetDataOnSimObject(defineId, OBJECT_ID_USER, 0, 0, size, unsafe.Pointer(&buf[0]))

}

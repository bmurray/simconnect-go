package client

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	_ "embed"
)

var defaultDll *dll

func init() {
	path, err := getFilePath()
	if err != nil {
		slog.Error("cannot get dll path", "error", err)
		return
	}
	dd, err := newDLL(path)
	if err != nil {
		slog.Error("cannot load dll", "error", err)
		return
	}
	defaultDll = dd
}

// LoadNewDefaultDLL loads a new default DLL to be used with all connections
func LoadNewDefaultDLL(path string) error {
	dd, err := newDLL(path)
	if err != nil {
		return err
	}
	defaultDll = dd
	return nil
}

//go:embed SimConnect.dll
var simconnectDLL []byte

var sysPaths = []string{
	"c:\\MSFS SDK\\SimConnect SDK\\lib\\SimConnect.dll",
	"c:\\MSFS 2024 SDK\\SimConnect SDK\\lib\\SimConnect.dll",
}

func findSysPath() (string, error) {
	for _, sysPath := range sysPaths {
		st, err := os.Stat(sysPath)
		if err == nil && !st.IsDir() {
			return sysPath, nil
		}
	}
	return "", fmt.Errorf("SimConnect.dll not found")
}

func getFilePath() (string, error) {
	sysPath, err := findSysPath()
	if err == nil {
		return sysPath, nil
	}
	slog.Debug("SimConnect.dll not found in default paths; using bundled")
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	dllPath := filepath.Join(filepath.Dir(exePath), "SimConnect.dll")
	st, err := os.Stat(dllPath)
	if err == nil && !st.IsDir() {
		return dllPath, nil
	}
	path, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot get cwd: %w", err)
	}
	dllPath = filepath.Join(path, "SimConnect.dll")
	st, err = os.Stat(dllPath)
	if err == nil && !st.IsDir() {
		return dllPath, nil
	}
	err = os.WriteFile(dllPath, simconnectDLL, 0644)
	if err != nil {
		return "", fmt.Errorf("cannot write file: %w", err)
	}
	return dllPath, nil
}

type dll struct {
	proc_SimConnect_Open                              *syscall.LazyProc
	proc_SimConnect_Close                             *syscall.LazyProc
	proc_SimConnect_AddToDataDefinition               *syscall.LazyProc
	proc_SimConnect_SubscribeToSystemEvent            *syscall.LazyProc
	proc_SimConnect_GetNextDispatch                   *syscall.LazyProc
	proc_SimConnect_RequestDataOnSimObject            *syscall.LazyProc
	proc_SimConnect_RequestDataOnSimObjectType        *syscall.LazyProc
	proc_SimConnect_SetDataOnSimObject                *syscall.LazyProc
	proc_SimConnect_SubscribeToFacilities             *syscall.LazyProc
	proc_SimConnect_UnsubscribeToFacilities           *syscall.LazyProc
	proc_SimConnect_RequestFacilitiesList             *syscall.LazyProc
	proc_SimConnect_MapClientEventToSimEvent          *syscall.LazyProc
	proc_SimConnect_MenuAddItem                       *syscall.LazyProc
	proc_SimConnect_MenuDeleteItem                    *syscall.LazyProc
	proc_SimConnect_AddClientEventToNotificationGroup *syscall.LazyProc
	proc_SimConnect_SetNotificationGroupPriority      *syscall.LazyProc
	proc_SimConnect_Text                              *syscall.LazyProc
	proc_SimConnect_TransmitClientEvent               *syscall.LazyProc
}

func newDLL(path string) (*dll, error) {
	mod := syscall.NewLazyDLL(path)
	if err := mod.Load(); err != nil {
		return nil, err
	}

	return &dll{
		proc_SimConnect_Open:                              mod.NewProc("SimConnect_Open"),
		proc_SimConnect_Close:                             mod.NewProc("SimConnect_Close"),
		proc_SimConnect_AddToDataDefinition:               mod.NewProc("SimConnect_AddToDataDefinition"),
		proc_SimConnect_SubscribeToSystemEvent:            mod.NewProc("SimConnect_SubscribeToSystemEvent"),
		proc_SimConnect_GetNextDispatch:                   mod.NewProc("SimConnect_GetNextDispatch"),
		proc_SimConnect_RequestDataOnSimObject:            mod.NewProc("SimConnect_RequestDataOnSimObject"),
		proc_SimConnect_RequestDataOnSimObjectType:        mod.NewProc("SimConnect_RequestDataOnSimObjectType"),
		proc_SimConnect_SetDataOnSimObject:                mod.NewProc("SimConnect_SetDataOnSimObject"),
		proc_SimConnect_SubscribeToFacilities:             mod.NewProc("SimConnect_SubscribeToFacilities"),
		proc_SimConnect_UnsubscribeToFacilities:           mod.NewProc("SimConnect_UnsubscribeToFacilities"),
		proc_SimConnect_RequestFacilitiesList:             mod.NewProc("SimConnect_RequestFacilitiesList"),
		proc_SimConnect_MapClientEventToSimEvent:          mod.NewProc("SimConnect_MapClientEventToSimEvent"),
		proc_SimConnect_MenuAddItem:                       mod.NewProc("SimConnect_MenuAddItem"),
		proc_SimConnect_MenuDeleteItem:                    mod.NewProc("SimConnect_MenuDeleteItem"),
		proc_SimConnect_AddClientEventToNotificationGroup: mod.NewProc("SimConnect_AddClientEventToNotificationGroup"),
		proc_SimConnect_SetNotificationGroupPriority:      mod.NewProc("SimConnect_SetNotificationGroupPriority"),
		proc_SimConnect_Text:                              mod.NewProc("SimConnect_Text"),
		proc_SimConnect_TransmitClientEvent:               mod.NewProc("SimConnect_TransmitClientEvent"),
	}, nil

}

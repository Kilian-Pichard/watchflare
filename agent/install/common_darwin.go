//go:build darwin

package install

// GetServiceManager returns the macOS service manager
func GetServiceManager() (ServiceManager, error) {
	return NewMacOSService(), nil
}

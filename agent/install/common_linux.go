// +build linux

package install

// GetServiceManager returns the Linux service manager
func GetServiceManager() (ServiceManager, error) {
	return NewLinuxService(), nil
}

//go:build darwin

package install

import "fmt"

// GetServiceManager is not supported on macOS — the agent is managed via Homebrew.
func GetServiceManager() (ServiceManager, error) {
	return nil, fmt.Errorf("on macOS, use Homebrew to manage the agent:\n" +
		"  brew services [start|stop|restart] watchflare-agent")
}

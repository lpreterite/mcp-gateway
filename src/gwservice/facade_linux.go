//go:build linux

package gwservice

func newFacadePlatformAdapter() PlatformAdapter {
	return &linuxAdapter{}
}

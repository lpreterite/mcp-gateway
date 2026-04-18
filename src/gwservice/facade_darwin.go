//go:build darwin

package gwservice

func newFacadePlatformAdapter() PlatformAdapter {
	return &darwinAdapter{}
}

//go:build !windows

package terminal

func OverlayHost(Capabilities) (string, bool) {
	return "", false
}

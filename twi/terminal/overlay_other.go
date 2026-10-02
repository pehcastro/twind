//go:build !windows

package terminal

func OverlayHost() string {
	return "no host: overlay windows are Windows only"
}

//go:build !windows

package fix

import "errors"

func process() ([]string, uint32, error) {
	return nil, 0, Refused("not needed: ConPTY exists only on Windows")
}

func elevate(string) error {
	return errors.New("elevation exists only on Windows")
}

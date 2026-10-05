//go:build windows

package fix

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os/exec"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"

	konst "github.com/pehcastro/twind/internal/konst/fix"
)

func process() ([]string, uint32, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = windows.CloseHandle(snapshot) }()
	parents := map[uint32]uint32{}
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		parents[entry.ProcessID] = entry.ParentProcessID
	}
	var chain []string
	pid := windows.GetCurrentProcessId()
	for range konst.Ancestors {
		if pid = parents[pid]; pid == 0 {
			break
		}
		if exe, err := imagePath(pid); err == nil {
			chain = append(chain, exe)
		}
	}
	return chain, windows.RtlGetVersion().BuildNumber, nil
}

func imagePath(pid uint32) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}

func elevate(script string) error {
	var encoded []byte
	for _, u := range utf16.Encode([]rune(script)) {
		encoded = binary.LittleEndian.AppendUint16(encoded, u)
	}
	outer := "$p = Start-Process -FilePath powershell.exe -Verb RunAs -Wait -PassThru -WindowStyle Hidden -ArgumentList '-NoProfile','-NonInteractive','-EncodedCommand','" + base64.StdEncoding.EncodeToString(encoded) + "'; exit $p.ExitCode"
	if out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", outer).CombinedOutput(); err != nil {
		return fmt.Errorf("the elevated copy failed or was declined: %w\n%s", err, out)
	}
	return nil
}

package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/twind-dev/twind/internal/dev/konst"
)

type Snapshot struct {
	Version int               `json:"version"`
	Focus   string            `json:"focus"`
	Scroll  map[string]Offset `json:"scroll"`
	Values  map[string]string `json:"values"`
}

type Offset struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func Read(path string) (Snapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var s Snapshot
	if err := decoder.Decode(&s); err != nil {
		return Snapshot{}, fmt.Errorf("dev snapshot %s: %w", path, err)
	}
	if s.Version != konst.SnapshotVersion {
		return Snapshot{}, fmt.Errorf("dev snapshot %s: version %d, want %d", path, s.Version, konst.SnapshotVersion)
	}
	return s, nil
}

func Write(path string, s Snapshot) error {
	s.Version = konst.SnapshotVersion
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	partial := path + ".partial"
	if err := os.WriteFile(partial, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(partial, path)
}

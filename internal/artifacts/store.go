package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

type StoredArtifact struct {
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
}

func Store(root, sourcePath string) (StoredArtifact, error) {
	in, err := os.Open(sourcePath)
	if err != nil {
		return StoredArtifact{}, err
	}
	defer in.Close()

	hash := sha256.New()
	tee := io.TeeReader(in, hash)
	data, err := io.ReadAll(tee)
	if err != nil {
		return StoredArtifact{}, err
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	outPath := filepath.Join(root, sum)
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return StoredArtifact{}, err
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return StoredArtifact{}, err
	}

	return StoredArtifact{
		Path:     outPath,
		Checksum: sum,
		Size:     info.Size(),
	}, nil
}

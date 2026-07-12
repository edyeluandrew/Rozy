package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	return &Local{root: root}, nil
}

func (l *Local) Save(operatorID, docType, filename string, r io.Reader) (key string, hash string, err error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}
	key = filepath.ToSlash(filepath.Join("operators", operatorID, docType, uuid.NewString()+ext))
	abs := filepath.Join(l.root, key)
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return "", "", err
	}

	f, err := os.Create(abs)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(f, io.TeeReader(r, hasher)); err != nil {
		return "", "", err
	}
	return key, hex.EncodeToString(hasher.Sum(nil)), nil
}

func (l *Local) AbsPath(key string) string {
	clean := filepath.Clean(strings.ReplaceAll(key, "/", string(os.PathSeparator)))
	return filepath.Join(l.root, clean)
}

func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func ValidateDocType(docType string) error {
	allowed := map[string]bool{
		"nin_front": true, "nin_back": true, "selfie": true, "permit": true,
		"logbook": true, "insurance": true, "vehicle_front": true,
		"vehicle_side": true, "plate_closeup": true, "interior": true,
	}
	if !allowed[docType] {
		return fmt.Errorf("invalid doc_type")
	}
	return nil
}

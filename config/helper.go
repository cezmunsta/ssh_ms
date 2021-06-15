package config

import (
	"os"

	"github.com/cezmunsta/ssh_ms/log"
	"github.com/gabriel-vasile/mimetype"
)

const (
	// FormatJSON will be used to indicate a JSON file
	FormatJSON = uint(2)
	// FormatText will be used to indicate a text file, or when an unknown mimetype is found
	FormatText = uint(1)
	// FormatUnknown will be used to indicate a problem
	FormatUnknown = uint(0)
)

var (
	formatLookup = map[string]uint{
		"application/json": FormatJSON,
		"default":          FormatText,
	}
)

func ensureDirExists(path string) (bool, error) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Errorf("Failed to create directory '%v': %v", path, err)
		return false, err
	}
	return true, nil
}

// GetFileType will return the mimetype of a file
// fh : file handle
func GetFileType(fh *os.File) (uint, error) {
	mt, err := mimetype.DetectFile(fh.Name())
	if err != nil {
		return FormatUnknown, err
	}

	ct, exists := formatLookup[mt.String()]
	if !exists {
		ct = formatLookup["default"]
	}
	return ct, nil
}

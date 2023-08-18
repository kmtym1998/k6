package fs

import (
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/spf13/afero"
)

// registry is a registry of opened files.
type registry struct {
	// openedFiles holds a safe for concurrent use map of opened openedFiles.
	//
	// Keys are expected to be strings holding the openedFiles' path.
	// Values are expected to be byte slices holding the openedFiles' data.
	//
	// That way, we can cache the file's content and avoid opening too many
	// file descriptor, and re-reading its content every time the file is opened.
	//
	// Importantly, this also means that if the
	// file is modified from outside of k6, the changes will not be reflected in the file's data.
	// openedFiles map[string][]byte
	openedFiles sync.Map
}

// open retrieves the content of a given file from the specified filesystem (fromFs) and
// stores it in the registry's internal `openedFiles` map.
//
// The function cleans the provided filename using filepath.Clean before using it.
//
// If the file was previously "opened" (and thus cached) by the registry, it
// returns the cached content. Otherwise, it reads the file from the
// filesystem, caches its content, and then returns it.
//
// The function is designed to minimize redundant file reads by leveraging an internal cache (openedFiles).
// In case the cached value is not a byte slice (which should never occur in regular use), it
// panics with a descriptive error.
//
// Parameters:
//   - filename: The name of the file to be retrieved. This should be a relative or absolute path.
//   - fromFs: The filesystem (from the afero package) from which the file should be read if not already cached.
//
// Returns:
//   - A byte slice containing the content of the specified file.
//   - An error if there's any issue opening or reading the file. If the file content is successfully cached and returned once,
//     subsequent calls will not produce file-related errors for the same file, as the cached value will be used.
func (fr *registry) open(filename string, fromFs afero.Fs) ([]byte, error) {
	filename = filepath.Clean(filename)

	if f, ok := fr.openedFiles.Load(filename); ok {
		data, ok := f.([]byte)
		if !ok {
			panic(fmt.Errorf("registry's file %s is not stored as a byte slice", filename))
		}

		return data, nil
	}

	f, err := fromFs.Open(filename)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	fr.openedFiles.Store(filename, data)

	return data, nil
}

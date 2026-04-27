package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

// Create a directory given the path
func CreateDirectory(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// Delete a directory given the path
func DeleteDirectory(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}
	return nil
}

// DownloadAndSaveFile - downloads file from url to destination
func DownloadAndSaveFile(url, path string) error {
	client := &http.Client{}

	// Send GET request to the URL
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("error making GET request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body")
		}
	}()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create the local file
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing file")
		}
	}()

	// Write the response body to the local file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	return nil
}

const maxDownloadSizeBytes = 5 * 1024 * 1024 // 5 MB limit for profile images and small downloads

func fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error downloading file: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error downloading file: %v", resp.Status)
	}

	// Limit to 5MB to prevent huge memory usage
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSizeBytes))
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	return content, nil
}

func writeIfDifferent(path string, data []byte) (bool, error) {
	if FileExists(path) {
		existingContent, err := os.ReadFile(path)
		if err == nil {
			if bytes.Equal(existingContent, data) {
				// Files are the same, no need to overwrite
				return false, nil
			}
		} else {
			log.Debug().Err(err).Msg("error reading existing file, proceeding to overwrite")
		}
	}

	tmpPath := path + ".tmp"
	err := os.WriteFile(tmpPath, data, 0644)
	if err != nil {
		return false, fmt.Errorf("error writing temporary file: %v", err)
	}

	// Atomically rename to the final path
	err = os.Rename(tmpPath, path)
	if err != nil {
		os.Remove(tmpPath) // Cleanup on failure
		return false, fmt.Errorf("error renaming temporary file: %v", err)
	}

	return true, nil
}

// DownloadFile downloads file from url to the path provided
func DownloadFile(ctx context.Context, url, path string) error {
	log.Debug().Msgf("downloading file: %s", url)
	data, err := fetchURL(ctx, url)
	if err != nil {
		return err
	}
	_, err = writeIfDifferent(path, data)
	return err
}

// DownloadFileIfChanged downloads file from url to the path provided only if the content is different
func DownloadFileIfChanged(ctx context.Context, url, path string) (bool, error) {
	log.Debug().Msgf("downloading file to check for changes: %s", url)
	data, err := fetchURL(ctx, url)
	if err != nil {
		return false, err
	}
	return writeIfDifferent(path, data)
}

func WriteJsonFile(j interface{}, path string) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// MoveFile - moves file from source to destination.
//
// os.Rename is used if possible, and falls back to copy and delete if it fails (e.g. cross-device link)
func MoveFile(ctx context.Context, source, dest string) error {
	// Try to rename the file first
	err := os.Rename(source, dest)
	if err == nil {
		return nil
	}

	// If rename fails (e.g. cross-device link), fall back to copy and delete
	srcFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			log.Debug().Msg("error closing source file")
		}
	}()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing destination file")
		}
	}()

	// Use io.Copy with context to respect cancellation
	_, err = io.Copy(destFile, &contextReader{ctx: ctx, r: srcFile})
	if err != nil {
		err = destFile.Close()
		if err != nil {
			log.Error().Err(err).Msg("error closing destination file after copy")
		}
		err = os.Remove(dest) // Clean up the partially written file
		if err != nil {
			log.Error().Err(err).Msg("error removing destination file after copy failure")
		}
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Close files before attempting to remove the source
	err = srcFile.Close()
	if err != nil {
		log.Debug().Msg("error closing source file after copy")
	}
	err = destFile.Close()
	if err != nil {
		log.Error().Err(err).Msg("error closing destination file after copy")
	}

	// Remove the source file
	err = os.Remove(source)
	if err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
}

// contextReader wraps an io.Reader with a context
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (cr *contextReader) Read(p []byte) (n int, err error) {
	if err := cr.ctx.Err(); err != nil {
		return 0, err
	}
	return cr.r.Read(p)
}

func CopyFile(sourcePath, destPath string) error {
	log.Debug().Msgf("moving file: %s to %s", sourcePath, destPath)
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		cerr := inputFile.Close()
		if cerr != nil {
			log.Error().Err(err).Msg("error closing input file")
		}
		return fmt.Errorf("error creating file: %v", err)
	}
	defer func() {
		if err := outputFile.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing output file")
		}
	}()
	_, err = io.Copy(outputFile, inputFile)
	cerr := inputFile.Close()
	if cerr != nil {
		log.Error().Err(err).Msg("error closing input file")
	}
	if err != nil {
		return fmt.Errorf("writing to output file failed: %v", err)
	}
	return nil
}

// MoveDirectory - moves directory from source to destination.
func MoveDirectory(ctx context.Context, source, dest string) error {
	// Create the destination directory
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Walk through the source directory
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		// Check if the context has been canceled
		if err := ctx.Err(); err != nil {
			return err
		}

		if err != nil {
			return fmt.Errorf("error accessing path %q: %w", path, err)
		}

		// Compute the relative path
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %q: %w", path, err)
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			// Create the directory in the destination
			return os.MkdirAll(destPath, info.Mode())
		}

		// Move the file
		if err := MoveFile(ctx, path, destPath); err != nil {
			return fmt.Errorf("failed to move file %q: %w", path, err)
		}

		return nil
	})
}

func MoveFolder(src string, dst string) error {
	// Check if the source path exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return err
	}

	// Create the destination directory
	err := os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return err
	}

	// Walk through the source directory and copy each file and directory
	// to the destination
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute the relative path of the file/directory
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Create the destination path
		dstPath := filepath.Join(dst, relPath)

		// If the source path is a directory, create it in the destination
		if info.IsDir() {
			return os.MkdirAll(dstPath, os.ModePerm)
		}

		// Otherwise, it's a file. Open the file for reading
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			if err := srcFile.Close(); err != nil {
				log.Debug().Msg("error closing source file")
			}
		}()

		// Open the destination file for writing
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer func() {
			if err := dstFile.Close(); err != nil {
				log.Debug().Msg("error closing destination file")
			}
		}()

		// Copy the contents of the file
		_, err = io.Copy(dstFile, srcFile)
		return err
	})
	if err != nil {
		return err
	}

	// Finally, remove the source directory
	return os.RemoveAll(src)
}

func DeleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}

func ReadLastLines(path string, lines int) ([]byte, error) {
	cmd := fmt.Sprintf("cat %s | tr '\\r' '\\n' | tail -n %d", path, lines)
	c := exec.Command("bash", "-c", cmd)
	out, err := c.Output()
	if err != nil {
		log.Error().Err(err).Msgf("error reading last lines: %v - supplied path: %s", err, path)
		return nil, fmt.Errorf("error reading last lines: %v - supplied path: %s", err, path)
	}
	return out, nil
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func ReadChatFile(path string) ([]byte, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading chat file: %v", err)
	}

	return data, nil
}

func DeleteFolder(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("error deleting folder: %v", err)
	}
	return nil
}

// GetSizeOfDirectory calculates the total size of all files in a directory recursively.
func GetSizeOfDirectory(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error getting size of folder: %v", err)
	}
	return size, nil
}

func GetFreeSpaceOfDirectory(path string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("error getting free space of directory: %v", err)
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}

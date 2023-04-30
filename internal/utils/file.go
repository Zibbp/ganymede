package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// CreateFolder - creates folder if it doesn't exist
// Adds base directory to path - supply with everything after /vods/
func CreateFolder(path string) error {
	log.Debug().Msgf("creating folder: %s", path)
	err := os.MkdirAll(fmt.Sprintf("/vods/%s", path), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// DownloadFile - downloads file from url to destination
// Adds base directory to path - supply with everything after /vods/
// DownloadFile("http://img", "channel", "profile.png")
func DownloadFile(url, path, filename string) error {
	log.Debug().Msgf("downloading file: %s", url)
	// Get response bytes from URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading file: %v", resp)
	}

	// Create file
	file, err := os.Create(fmt.Sprintf("/vods/%s/%s", path, filename))
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Write bytes to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}
	return nil
}

func WriteJson(j interface{}, path string, filename string) error {
	data, err := json.Marshal(j)
	if err != nil {
		log.Error().Msgf("error marshalling json: %v", err)
	}
	err = os.WriteFile(fmt.Sprintf("/vods/%s/%s", path, filename), data, 0644)
	if err != nil {
		log.Error().Msgf("error writing json: %v", err)
	}
	return nil
}

func MoveFile(sourcePath, destPath string) error {
	log.Debug().Msgf("moving file: %s to %s", sourcePath, destPath)
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("error creating file: %v", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("writing to output file failed: %v", err)
	}
	// Copy was successful - delete source file
	err = os.Remove(sourcePath)
	if err != nil {
		log.Info().Msgf("error deleting source file: %v", err)
	}
	return nil
}

func CopyFile(sourcePath, destPath string) error {
	log.Debug().Msgf("moving file: %s to %s", sourcePath, destPath)
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("error creating file: %v", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("writing to output file failed: %v", err)
	}
	return nil
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
		defer srcFile.Close()

		// Open the destination file for writing
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

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
	log.Debug().Msgf("deleting file: %s", path)
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("error deleting file: %v", err)
	}
	return nil
}

func ReadLastLines(path string, lines int) ([]byte, error) {
	cmd := fmt.Sprintf("cat %s | tr '\\r' '\\n' | tail -n %d", path, lines)
	c := exec.Command("bash", "-c", cmd)
	out, err := c.Output()
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("error reading last lines: %v - supplied path: %s", err, path)
	}
	return out, nil
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func ReadChatFile(path string) ([]byte, error) {

	// Check if file is cached
	//cached, found := cache.Cache().Get(path)
	//if found {
	//	log.Debug().Msgf("using cached file: %s", path)
	//	return cached.([]byte), nil
	//}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading chat file: %v", err)
	}

	// Cache file
	//err = cache.Cache().Set(path, data, 5*time.Minute)
	//if err != nil {
	//
	//	return nil, err
	//}
	//log.Debug().Msgf("set cache for file: %s", path)

	return data, nil
}

func DeleteFolder(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("error deleting folder: %v", err)
	}
	return nil
}

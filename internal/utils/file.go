package utils

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
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
	err = ioutil.WriteFile(fmt.Sprintf("/vods/%s/%s", path, filename), data, 0644)
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
		return fmt.Errorf("error removing file: %v", err)
	}
	return nil
}

func DeleteFile(path string) error {
	log.Debug().Msgf("deleting file: %s", path)
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("error deleting file: %v", err)
	}
	return nil
}

func ReadLastLines(path string, lines string) ([]byte, error) {
	c := exec.Command("tail", "-n", lines, path)
	out, err := c.Output()
	if err != nil {
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

package exec

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type GenerateThumbnailsInput struct {
	Video        string
	Interval     int
	ThumbnailDir string
	Width        int
	Height       int
}

type CreateSpritesInput struct {
	SpriteDir    string
	ThumbnailDir string
	Width        int
	Height       int
	TilesX       int
	TilesY       int
}

// GenerateThumbnails extracts frames from the video at specified intervals.
func GenerateThumbnails(config GenerateThumbnailsInput) error {
	// Get video duration using ffprobe
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", config.Video)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get video duration: %w", err)
	}

	// Parse duration and calculate frames
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return fmt.Errorf("failed to parse video duration: %w", err)
	}

	// Extract frames at intervals
	format := fmt.Sprintf("frame%%0%dd.jpg", 8)
	for t := 0; t < int(duration); t += config.Interval {
		outputPath := filepath.Join(config.ThumbnailDir, fmt.Sprintf(format, t))
		ffmpegArgs := []string{
			"-hide_banner", "-an", "-ss", strconv.Itoa(t), "-i", config.Video,
			"-frames:v", "1",
			"-q:v", "10",
			"-vf", fmt.Sprintf("scale=%d:%d", config.Width, config.Height),
			"-y",
			outputPath,
		}
		cmd := exec.Command("ffmpeg", ffmpegArgs...)
		ffmpegOutput, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf(
				"failed to extract frame at %ds: %w\nCommand: ffmpeg %s\nOutput: %s",
				t, err, strings.Join(ffmpegArgs, " "), string(ffmpegOutput),
			)
		}
	}
	return nil
}

// CreateSprites generates multiple sprite images, each containing a maximum of tilesX * tilesY thumbnails.
func CreateSprites(config CreateSpritesInput) ([]string, error) {
	// Read thumbnail files
	thumbFiles, err := filepath.Glob(filepath.Join(config.ThumbnailDir, "frame*.jpg"))
	if err != nil {
		return nil, fmt.Errorf("failed to read thumbnails: %w", err)
	}

	if len(thumbFiles) == 0 {
		return nil, fmt.Errorf("no thumbnails found in %s", config.ThumbnailDir)
	}

	// Calculate max thumbnails per sprite
	maxThumbnails := config.TilesX * config.TilesY
	spriteIndex := 0

	var spritePaths []string

	for i := 0; i < len(thumbFiles); i += maxThumbnails {
		end := i + maxThumbnails
		if end > len(thumbFiles) {
			end = len(thumbFiles)
		}

		// Create a sprite for the current batch
		spriteWidth := config.Width * config.TilesX
		spriteHeight := config.Height * config.TilesY
		sprite := image.NewRGBA(image.Rect(0, 0, spriteWidth, spriteHeight))

		for j, thumbPath := range thumbFiles[i:end] {
			// Open thumbnail
			file, err := os.Open(thumbPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open thumbnail %s: %w", thumbPath, err)
			}
			defer func() {
				if err := file.Close(); err != nil {
					log.Debug().Err(err).Msg("failed to close thumbnail file")
				}
			}()

			img, _, err := image.Decode(file)
			if err != nil {
				return nil, fmt.Errorf("failed to decode thumbnail %s: %w", thumbPath, err)
			}

			// Calculate position in sprite
			x := (j % config.TilesX) * config.Width
			y := (j / config.TilesX) * config.Height
			rect := image.Rect(x, y, x+config.Width, y+config.Height)
			draw.Draw(sprite, rect, img, image.Point{}, draw.Src)
		}

		// Save the sprite image
		spritePath := filepath.Join(config.SpriteDir, fmt.Sprintf("sprite-%03d.jpg", spriteIndex))
		spriteFile, err := os.Create(spritePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create sprite file %s: %w", spritePath, err)
		}
		defer func() {
			if err := spriteFile.Close(); err != nil {
				log.Debug().Err(err).Msg("failed to close sprite file")
			}
		}()

		if err := jpeg.Encode(spriteFile, sprite, &jpeg.Options{Quality: 90}); err != nil {
			return nil, fmt.Errorf("failed to save sprite image %s: %w", spritePath, err)
		}

		spritePaths = append(spritePaths, spritePath)

		spriteIndex++
	}

	return spritePaths, nil
}

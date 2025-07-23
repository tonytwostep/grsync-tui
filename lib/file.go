package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	photos []string
)

func CopyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Ensure the destination directory and file exist
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("Failed to create destination directory during download: %v\n", err)
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func ScanCameraUsb(out *[]string, cfg Config) {
	var photos []string

	root := cfg.UsbSettings.CameraDir
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() {
			name := strings.ToLower(d.Name())
			if strings.HasSuffix(name, ".dng") || strings.HasSuffix(name, ".jpg") {
				rel, relErr := filepath.Rel(root, path)
				if relErr == nil {
					// Use forward slashes for consistency
					rel = strings.ReplaceAll(rel, string(filepath.Separator), "/")
					photos = append(photos, rel)
				}
			}
		}
		return nil
	})

	*out = photos
}

func GetFileInfo(name string, cfg Config, existingFiles map[string]bool) (size int64, modTime time.Time, exists bool) {
	// Switch on connection method

	// Check if the file exists in the download directory first
	if existingFiles[name] {
		path := filepath.Join(cfg.DownloadDir, name)
		info, err := os.Stat(path)
		if err != nil {
			return 0, time.Time{}, false
		}
		return info.Size(), info.ModTime(), true
	}

	switch cfg.ConnectionMethod {
	case ConnectionMethodUSB:
		path := filepath.Join(cfg.UsbSettings.CameraDir, name)
		info, err := os.Stat(path)
		if err != nil {
			return 0, time.Time{}, false
		}
		return info.Size(), info.ModTime(), true

	case ConnectionMethodWiFi:
		photoInfo := wifiGetPhotoInfo(name, cfg.Mock)
		// Example format: "2024-07-01T12:34:56"
		t, err := time.Parse("2006-01-02T15:04:05", photoInfo.Datetime)
		if err != nil {
			t = time.Time{}
		}
		return photoInfo.Size, t, true
	default:
		return 0, time.Time{}, false
	}
}

func ScanDownloadDir(existingFiles map[string]bool, cfg Config) {
	root := cfg.DownloadDir

	found := make(map[string]struct{})

	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			rel, relErr := filepath.Rel(root, path)
			if relErr == nil {
				rel = strings.ReplaceAll(rel, string(filepath.Separator), "/")
				found[rel] = struct{}{}
			}
		}
		return nil
	})

	// Remove files not found
	for k := range existingFiles {
		if _, ok := found[k]; !ok {
			delete(existingFiles, k)
		}
	}

	// Add new found files
	for k := range found {
		existingFiles[k] = true
	}
}

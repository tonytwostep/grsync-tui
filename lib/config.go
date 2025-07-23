package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// Const for mocking bluetooth and wifi responses in an effort to ease development
const mock = false

var configFileRelativePath = filepath.Join(".config", "grsync-tui.json")

type Config struct {
	Mock             bool             `json:"-"` // for testing purposes, not actually in the config file
	ConnectionMethod ConnectionMethod `json:"connection_method"`
	DownloadDir      string           `json:"download_dir"`
	UsbSettings      UsbSettings      `json:"usb"`
}

type UsbSettings struct {
	CameraDir string `json:"camera_dir"`
}

func defaultConfig() Config {
	exeDir, _ := os.Getwd()
	cfg := Config{
		ConnectionMethod: ConnectionMethodWiFi, // default to wifi for now, in the future, we should have a splash screen to choose connection method
		DownloadDir:      filepath.Join(exeDir, "download"),
		Mock:             mock, // for testing purposes, not actually in the config file
		UsbSettings: UsbSettings{
			CameraDir: filepath.Join(exeDir, "camera"),
		},
	}
	err := saveConfig(cfg)
	if err != nil {
		fmt.Errorf("unable to save default config: %v", err)
	}
	return cfg
}

type ConnectionMethod string

const (
	ConnectionMethodUSB       ConnectionMethod = "usb"
	ConnectionMethodWiFi      ConnectionMethod = "wifi"
	ConnectionMethodBluetooth ConnectionMethod = "bluetooth" // todo not implemented
)

func configFilePath() string {
	usr, _ := user.Current()
	return filepath.Join(usr.HomeDir, configFileRelativePath)
}

func LoadConfig() (Config, error) {
	path := configFilePath()
	f, err := os.Open(path)
	if err != nil {
		// File doesn't exist, return default config
		return defaultConfig(), nil
	}
	defer f.Close()
	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return defaultConfig(), nil
	}
	cfg.Mock = mock
	return cfg, nil
}

func saveConfig(cfg Config) error {
	path := configFilePath()
	os.MkdirAll(filepath.Dir(path), 0700)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	jsonEncoder := json.NewEncoder(f)
	jsonEncoder.SetIndent("", "  ")
	return jsonEncoder.Encode(cfg)
}

func EnsureDownloadDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

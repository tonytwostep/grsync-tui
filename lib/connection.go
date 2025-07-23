package lib

import (
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"
)

func WaitForConnection(cfg Config) {
	fmt.Println("Waiting for connection to camera via", cfg.ConnectionMethod, "...")
	switch cfg.ConnectionMethod {
	case ConnectionMethodUSB:
		// ensure the camera directory exists
		fmt.Println("Camera directory is configured as:", cfg.UsbSettings.CameraDir)
		fmt.Println("Repeatedly checking for existing camera directory")

		// wait infinitely until we see the directory exists
		for {
			if _, err := os.Stat(cfg.UsbSettings.CameraDir); err == nil {
				break
			}
			time.Sleep(1 * time.Second) // wait 1 second before checking again
		}

	case ConnectionMethodWiFi:
		// ensure the camera is connected via Wifi
		fmt.Println("Repeatedly checking connection to", GRHost)
		fmt.Println("Ensure your device is connected to the camera's WiFi network.")

		if cfg.Mock {
			fmt.Println("Mock mode enabled, simulating connection to camera.")
			fmt.Println("Mock connection established.")
			return
		}

		// wait infinitely until we can reach the camera
		for {
			resp, err := http.Get(GRPhotoListURL())
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			time.Sleep(1 * time.Second) // wait 1 second before checking again
		}
	default:
		fmt.Println("Unsupported connection method:", cfg.ConnectionMethod)
		syscall.Exit(1)
	}
	fmt.Println("Connection established, starting TUI.")
}

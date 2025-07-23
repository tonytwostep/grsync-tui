package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	GRHost       = "http://192.168.0.1/"
	PhotoListURI = "v1/photos"
	PropsURI     = "v1/props"
)

func GRPhotoListURL() string {
	return GRHost + PhotoListURI
}

type GRPhotoList struct {
	ErrCode int    `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
	Dirs    []struct {
		Name  string   `json:"name"`
		Files []string `json:"files"`
	} `json:"dirs"`
}

type PhotoInfo struct {
	ErrCode     int    `json:"errCode"`
	ErrMsg      string `json:"errMsg"`
	CameraModel string `json:"cameraModel"`
	Dir         string `json:"dir"`
	File        string `json:"file"`
	Size        int64  `json:"size"`
	Datetime    string `json:"datetime"`
	Orientation int    `json:"orientation"`
	AspectRatio string `json:"aspectRatio"`
	Av          string `json:"av"`
	Tv          string `json:"tv"`
	Sv          string `json:"sv"`
	Xv          string `json:"xv"`
	GpsInfo     string `json:"gpsInfo"`
}

func ScanCameraWiFi(out *[]string, mock bool) {

	var data []byte

	if mock {
		currentDir, err := os.Getwd()
		mockFile := filepath.Join(currentDir, "mock", "photos.json")
		data, err = os.ReadFile(mockFile)
		if err != nil {
			fmt.Println("Failed to read mock file:", err)
			return
		}
	} else {
		resp, err := http.Get(GRPhotoListURL())
		if err != nil {
			fmt.Println("Failed to connect to GR camera:", err)
			return
		}
		defer resp.Body.Close()

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Failed to read GR response:", err)
			return
		}
	}

	var photos GRPhotoList
	if err := json.Unmarshal(data, &photos); err != nil {
		fmt.Println("Failed to parse GR photo list:", err)
		return
	}

	var names []string
	for _, dir := range photos.Dirs {
		for _, file := range dir.Files {
			p := dir.Name + "/" + file
			names = append(names, p)
		}
	}

	*out = names
}

func wifiGetPhotoInfo(name string, mock bool) PhotoInfo {
	var info PhotoInfo

	if mock {
		currentDir, _ := os.Getwd()
		mockFile := filepath.Join(currentDir, "mock", "photoInfo.json")
		data, err := os.ReadFile(mockFile)
		// simulate time delay for mock
		time.Sleep(100 * time.Millisecond)
		if err != nil {
			fmt.Println("Failed to read mock file:", err)
			return info
		}
		if err := json.Unmarshal(data, &info); err != nil {
			fmt.Println("Failed to unmarshal mock data:", err)
		}
		return info
	}

	url := fmt.Sprintf("%s/%s/info", GRPhotoListURL(), name)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("HTTP error:", err)
		return info
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		fmt.Println("Failed to decode response:", err)
	}
	return info
}

package main

import (
	"bytes"
	"fmt"
	"git.boj4ck.com/tonytwostep/grsync-tui/assets"
	"git.boj4ck.com/tonytwostep/grsync-tui/lib"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rwcarlsen/goexif/exif"
	"io"
	"net/http"
	"syscall"

	"github.com/nfnt/resize"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	app    *tview.Application
	layout *tview.Flex
	pages  = tview.NewPages()
	cfg    lib.Config

	// Flexes
	logoFlex *tview.Flex

	// Boxes
	photoListBox  *tview.List
	photoCountBox *tview.TextView
	metadataBox   *tview.TextView
	logoBox       *tview.TextView
	logBox        *tview.TextView

	// State
	photos            []string
	selected          = make(map[int]bool)
	currentItem       int
	existingFiles     = make(map[string]bool)
	termWidth         int
	termHeight        int
	lastMetadataIndex = -1
)

func itemIsSelected(index int) bool {
	_, ok := selected[index]
	return ok
}

func updateMetadata(index int, forceReload bool) {
	if index < 0 || index >= len(photos) {
		metadataBox.SetText("No photo selected")
		return
	}
	// if we just loaded this metadata, don't update it again
	if lastMetadataIndex == currentItem && !forceReload {
		return
	}

	name := photos[index]
	metadataBox.SetText("[yellow]Loading metadata...")
	go func(photoName string, idx int) {
		size, modTime, exists := lib.GetFileInfo(photoName, cfg, existingFiles)
		sizeStr := "N/A"
		dateStr := "N/A"
		if exists {
			sizeStr = fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
			dateStr = modTime.Format("2006-01-02 15:04:05")
		}
		var statusMsg string
		if existingFiles[photoName] {
			statusMsg = "[green]Downloaded 💾"
		} else {
			statusMsg = "[red]Not downloaded"
		}
		// Only extract EXIF if local file exists
		var exifInfo interface{}
		if exists {
			exifInfo = lib.ExtractExifInfo(photoName, cfg)
		}
		app.QueueUpdateDraw(func() {
			// Only update if still on the same photo
			if currentItem == idx {
				metadataInfo := fmt.Sprintf(
					"[white]File:[yellow] %s\n[white]Filesize:[yellow] %s\n[white]Date:[yellow] %s\n\n%s\n[white]Status: %s",
					photoName, sizeStr, dateStr, exifInfo, statusMsg)
				metadataBox.SetText(metadataInfo)
			}
		})
		lastMetadataIndex = idx
	}(name, index)
}

func errorPreviewModal(msg string) *tview.Modal {
	return tview.NewModal().
		SetText("[red]" + msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.RemovePage("preview")
		})
}

func setAppFocus(t *tview.TextView, l *tview.List) {
	if t != nil {
		app.SetFocus(t)
		return
	}
	app.SetFocus(l)
}

func renderPreviewModal(photoName string) tview.Primitive {
	var file io.ReadCloser
	var err error

	// Check if the photo exists in the download directory first
	if existingFiles[photoName] {
		path := filepath.Join(cfg.DownloadDir, photoName)
		file, err = os.Open(path)
		if err != nil {
			return errorPreviewModal("Failed to open downloaded file.")
		}
	} else {
		switch cfg.ConnectionMethod {
		case lib.ConnectionMethodUSB:
			path := filepath.Join(cfg.UsbSettings.CameraDir, photoName)
			file, err = os.Open(path)
			if err != nil {
				return errorPreviewModal("Failed to open local file.")
			}
			defer file.Close()
		case lib.ConnectionMethodWiFi:
			url := lib.GRHost + "v1/photos/" + photoName + "?size=view"
			resp, err := http.Get(url)
			if err != nil || resp.StatusCode != http.StatusOK {
				return errorPreviewModal("Failed to fetch image from camera.")
			}
			file = resp.Body
			defer file.Close()
		default:
			fmt.Errorf("Unknown connection method specified, check config: %s", cfg.ConnectionMethod)
			syscall.Exit(1)
		}
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(file)

	if err != nil {
		return errorPreviewModal("Failed to read image.")
	}

	img, err := jpeg.Decode(buf)
	if err != nil {
		return errorPreviewModal("Failed to decode image.")
	}

	// Usb lets resize the image to speed things up (not necessary for wifi)
	const previewMax = 320
	img = resize.Thumbnail(previewMax, previewMax, img, resize.Lanczos3)

	// Exif stuff
	x, err := exif.Decode(bytes.NewReader(buf.Bytes()))

	if err == nil {
		if orientTag, err := x.Get(exif.Orientation); err == nil {
			orient, _ := orientTag.Int(0)
			img = lib.ApplyOrientation(img, orient)
		}
	}

	w, h := termWidth, termHeight
	if w == 0 || h == 0 {
		w, h = 80, 24 // fallback
	}

	// Leave some space for borders/title/button, say 2 rows and 4 columns total
	maxW := w
	maxH := h - 6

	bounds := img.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()

	// Terminal characters are usually roughly twice as tall as wide (height ~ 2x width),
	const terminalAspectRatio = 0.4

	// Adjust maxH by aspect ratio for terminal cell shape
	adjustedMaxH := int(float64(maxH) / terminalAspectRatio)

	// Calculate scale to fit inside maxW and adjustedMaxH
	scaleW := float64(maxW) / float64(imgW)
	scaleH := float64(adjustedMaxH) / float64(imgH)

	scale := scaleW
	if scaleH < scaleW {
		scale = scaleH
	}

	// Compute final display dimensions
	dispW := int(float64(imgW) * scale)
	dispH := int(float64(imgH) * scale * terminalAspectRatio) // scale back height

	// Clamp to max bounds just in case
	if dispW > maxW {
		dispW = maxW
	}
	if dispH > maxH {
		dispH = maxH
	}

	form := tview.NewForm().
		AddImage("", img, dispW, dispH, 0).
		AddButton("Close", func() {
			pages.RemovePage("preview")
		}).SetButtonsAlign(tview.AlignCenter).
		SetButtonBackgroundColor(tcell.ColorWhite).
		SetButtonTextColor(tcell.ColorPurple)

	form.SetBorder(true).SetTitle(photoName)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.RemovePage("preview")
			return nil
		}
		return event
	})

	// Instead of centering with padding, just use a Flex that fills full screen
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true) // 0 height and weight 1 means full available space
	return flex
}

func updatePhotoList() {
	prevItem := photoListBox.GetCurrentItem()
	photoListBox.Clear()
	for i, name := range photos {
		displayName := name
		if existingFiles[name] {
			displayName = "💾 " + displayName
		}
		if selected[i] {
			displayName = fmt.Sprintf("[green]%s", displayName)
		}
		photoListBox.AddItem(displayName, "", 0, nil)
	}
	if prevItem >= photoListBox.GetItemCount() {
		prevItem = photoListBox.GetItemCount() - 1
	}
	if prevItem < 0 {
		prevItem = 0
	}
	photoListBox.SetCurrentItem(prevItem)
	currentItem = photoListBox.GetCurrentItem()
}

func downloadSelected() {
	var files []string
	for i := range selected {
		name := photos[i]
		if !existingFiles[name] {
			files = append(files, name)
		}
	}

	// If there is no selection, download the current item if it exists and is not already downloaded
	if len(selected) == 0 {
		if currentItem >= 0 && currentItem < len(photos) {
			name := photos[currentItem]
			if !existingFiles[name] {
				files = append(files, name)
			}
		}
	}

	total := len(files)

	// No new files to download
	if total == 0 {
		modal := tview.NewModal().
			SetText("[red]Selection is already downloaded.").
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				pages.RemovePage("modal")
			})
		pages.AddPage("modal", modal, true, true)
		return
	}

	// Show progress modal
	progressModal := tview.NewModal().
		AddButtons([]string{"Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.RemovePage("modal")
		})

	// Unicode blocks for smooth progress bar (8 levels)
	barBlocks := []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

	// Returns a smooth unicode progress bar string
	smoothBar := func(current, total, barLen int) string {
		if total == 0 {
			return "[" + strings.Repeat(" ", barLen) + "]"
		}
		progress := float64(current) / float64(total)
		fullBlocks := int(progress * float64(barLen))
		partialBlockFrac := (progress*float64(barLen) - float64(fullBlocks)) * 8
		partialBlock := int(partialBlockFrac + 0.5) // round to nearest

		bar := strings.Repeat(string(barBlocks[8]), fullBlocks)
		if fullBlocks < barLen && partialBlock > 0 {
			bar += string(barBlocks[partialBlock])
			fullBlocks++
		}
		if fullBlocks < barLen {
			bar += strings.Repeat(" ", barLen-fullBlocks)
		}
		return "[" + bar + "]"
	}

	updateModal := func(current int, filename string) {
		percent := int(float64(current) / float64(total) * 100)
		barLen := 30
		bar := smoothBar(current, total, barLen)
		text := fmt.Sprintf(
			"[yellow]Downloading %d photos...\n[white]%s\n[green]%s %3d%% (%d/%d)",
			total, filename, bar, percent, current, total,
		)
		progressModal.SetText(text)
	}

	pages.AddPage("modal", progressModal, true, true)

	queueStart := time.Now()
	go func() {
		for i, file := range files {
			srcPath := filepath.Join(cfg.UsbSettings.CameraDir, file)
			dstPath := filepath.Join(cfg.DownloadDir, file)

			app.QueueUpdateDraw(func() {
				updateModal(i+1, file)
			})

			perFileStart := time.Now()

			switch cfg.ConnectionMethod {
			case lib.ConnectionMethodUSB:
				lib.WriteLog(fmt.Sprintf("[yellow]%s - Local download %s", time.Now().Format("2006-01-02 15:04:05"), srcPath), logBox)
				if err := lib.CopyFile(srcPath, dstPath); err != nil {
					lib.WriteLog(fmt.Sprintf("[red]%s - Failed to download %s: %v", time.Now().Format("2006-01-02 15:04:05"), file, err), logBox)
					continue
				}
			case lib.ConnectionMethodWiFi:
				// Download from camera via WiFi
				lib.WriteLog(fmt.Sprintf("[yellow]%s - WiFi download %s", time.Now().Format("2006-01-02 15:04:05"), srcPath), logBox)
				url := fmt.Sprintf("%s/%s", lib.GRPhotoListURL(), file)
				resp, err := http.Get(url)
				if err != nil {
					lib.WriteLog(fmt.Sprintf("[red]%s - Failed to download %s: %v", time.Now().Format("2006-01-02 15:04:05"), file, err), logBox)
					continue
				}
				defer resp.Body.Close()

				err = os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
				if err != nil {
					lib.WriteLog(fmt.Sprintf("[red]%s - Couldn't mkdir %s: %v", time.Now().Format("2006-01-02 15:04:05"), file, err), logBox)
				}
				out, err := os.Create(dstPath)
				if err != nil {
					lib.WriteLog(fmt.Sprintf("[red]%s - Failed to download  %s: %v", time.Now().Format("2006-01-02 15:04:05"), file, err), logBox)
					continue
				}
				defer out.Close()

				if _, err := io.Copy(out, resp.Body); err != nil {
					lib.WriteLog(fmt.Sprintf("[red]%s - Failed to download %s: %v", time.Now().Format("2006-01-02 15:04:05"), file, err), logBox)
					continue
				}
			default:
				fmt.Errorf("Unknown connection method specified, check config: %s", cfg.ConnectionMethod)
				syscall.Exit(1)
			}

			elapsed := time.Since(perFileStart).Seconds()
			lib.WriteLog(fmt.Sprintf("[purple]%s - Downloaded %s in %.2f seconds", time.Now().Format("2006-01-02 15:04:05"), file, elapsed), logBox)

			existingFiles[file] = true
		}
		app.QueueUpdateDraw(func() {
			progressModal.SetText("[green]Download complete!")
			progressModal.AddButtons([]string{"OK"})
			lib.ScanDownloadDir(existingFiles, cfg)
			updatePhotoList()
			updateMetadata(currentItem, true)
			totalElapsed := time.Since(queueStart).Seconds()

			// If the queue was longer than one
			if total > 1 {
				lib.WriteLog(fmt.Sprintf("[blue]%s - Downloaded %d photos in %.2f seconds", time.Now().Format("2006-01-02 15:04:05"), total, totalElapsed), logBox)
			}
			updateLogBox()
		})

		// Selected items are cleared after download
		selected = make(map[int]bool)
	}()
}

func updateLogBox() {
	// Conditionally hide the log box based on terminal height
	if termHeight < 22 {
		logoFlex.ResizeItem(logBox, 0, 0)
		logoFlex.ResizeItem(logoBox, 0, 2)
	} else {
		logoFlex.ResizeItem(logBox, 0, 1)
		logoFlex.ResizeItem(logoBox, 12, 2)
	}

	lib.SetLogText(logBox)
	if logBox.GetText(true) == "" {
		logBox.SetText("[yellow]No downloads yet.")
	}
}

func updateLogo() {
	if termWidth >= 140 {
		logoBox.SetText(assets.Logo)
	} else {
		logoBox.SetText(assets.StackedLogo)
	}
}

func toggleSelection(index int) {
	if _, ok := selected[index]; ok {
		delete(selected, index)
	} else {
		selected[index] = true
	}
	updatePhotoList()
	updatePhotoCount()
}

func selectAll() {
	for i := 0; i < len(photos); i++ {
		selected[i] = true
	}
	updatePhotoList()
	updatePhotoCount()
}

func deselectAll() {
	selected = make(map[int]bool)
	updatePhotoList()
	updatePhotoCount()
}

func updatePhotoCount() {
	count := len(selected)
	cameraPhotos := len(photos)
	downloadedPhotos := 0
	for _, name := range photos {
		if existingFiles[name] {
			downloadedPhotos++
		}
	}

	photoCountBox.SetText(fmt.Sprintf("[yellow]%d selected\n[purple]%d on camera\n[green]%d downloaded", count, cameraPhotos, downloadedPhotos))
}

func scanCameraPhotos() {
	switch cfg.ConnectionMethod {
	case lib.ConnectionMethodUSB:
		lib.ScanCameraUsb(&photos, cfg)
	case lib.ConnectionMethodWiFi:
		lib.ScanCameraWiFi(&photos, cfg.Mock)
	default:
		fmt.Errorf("Unknown connection method specified, check config: %s", cfg.ConnectionMethod)
		syscall.Exit(1)
	}
}

func main() {
	var err error
	cfg, _ = lib.LoadConfig()
	err = lib.EnsureDownloadDir(cfg.DownloadDir)
	if err != nil {
		return
	}
	lib.ScanDownloadDir(existingFiles, cfg)

	// does this run every frame?
	lib.WaitForConnection(cfg)

	scanCameraPhotos()

	app = tview.NewApplication()
	photoListBox = tview.NewList()
	photoListBox.ShowSecondaryText(false)
	photoListBox.SetBorder(true)
	photoListBox.SetTitle("Camera Photos")
	photoListBox.SetHighlightFullLine(true)
	photoListBox.SetSelectedBackgroundColor(tcell.ColorBlue)
	photoListBox.SetSelectedTextColor(tcell.ColorWhite)

	metadataBox = tview.NewTextView()
	metadataBox.SetDynamicColors(true)
	metadataBox.SetBorder(true)
	metadataBox.SetTitle("Metadata")

	photoCountBox = tview.NewTextView()
	photoCountBox.SetTextAlign(tview.AlignLeft)
	photoCountBox.SetTextColor(tcell.ColorGreen)
	photoCountBox.SetBackgroundColor(tcell.ColorBlack)
	photoCountBox.SetDynamicColors(true)
	photoCountBox.SetBorder(true)
	photoCountBox.SetTitle("Photo Counts")

	logoBox = tview.NewTextView()
	logoBox.SetDynamicColors(true)
	logoBox.SetBorder(true)
	logoBox.SetTextColor(tcell.ColorBlue)
	logoBox.SetWrap(false)

	logBox = tview.NewTextView()
	logBox.SetDynamicColors(true)
	logBox.SetBorder(true)
	logBox.SetTitle("Log")
	logBox.SetScrollable(true)

	updatePhotoCount()

	photoListBox.SetChangedFunc(func(index int, main string, secondary string, shortcut rune) {
		updateMetadata(index, false)
	})

	// Set up global keybindings (moved to lib)
	lib.SetupControls(
		app,
		photoListBox,
		&currentItem,
		photos,
		pages,
		logBox,
		setAppFocus,
		itemIsSelected,
		toggleSelection,
		downloadSelected,
		selectAll,
		deselectAll,
		renderPreviewModal,
	)

	updatePhotoList()
	updateMetadata(currentItem, true)
	updatePhotoCount()
	updateLogo()

	// if terminal height is > 22, 3 boxes can be stacked
	logoFlex = tview.NewFlex()
	logoFlex.
		AddItem(logoBox, 12, 2, false).
		AddItem(logBox, 0, 1, false).
		AddItem(photoCountBox, 5, 1, false).
		SetDirection(tview.FlexRow)
	photosAndInfoFlex := tview.NewFlex().
		AddItem(photoListBox, 0, 1, false).
		AddItem(metadataBox, 0, 1, false).
		SetDirection(tview.FlexColumn)

	layout = tview.NewFlex().
		AddItem(logoFlex, 0, 1, false).
		AddItem(photosAndInfoFlex, 0, 3, false)

	pages.AddPage("main", layout, true, true)

	go func() {
		ticker := time.NewTicker(1 * time.Second) // scan every 2 seconds
		defer ticker.Stop()
		for range ticker.C {
			scanCameraPhotos()
			lib.ScanDownloadDir(existingFiles, cfg)
			app.QueueUpdateDraw(func() {
				updatePhotoList()
				updatePhotoCount()
				updateLogo()
				updateLogBox()
			})
		}
	}()

	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		termWidth, termHeight = screen.Size()
		updateLogBox()
		return false
	})

	// In renderPreviewModal:
	w, h := termWidth, termHeight
	if w == 0 || h == 0 {
		w, h = 80, 24 // fallback
	}

	if err := app.SetRoot(pages, true).EnableMouse(true).SetFocus(photoListBox).Run(); err != nil {
		panic(err)
	}

}

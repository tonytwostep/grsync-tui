package lib

import (
	"fmt"
	"git.boj4ck.com/tonytwostep/grsync-tui/assets"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SetupControls sets up the global keybindings for the app.
func SetupControls(
	app *tview.Application,
	photoListBox *tview.List,
	currentItem *int,
	photos []string,
	pages *tview.Pages,
	logBox *tview.TextView,
	setAppFocus func(t *tview.TextView, l *tview.List),
	itemIsSelected func(int) bool,
	toggleSelection func(int),
	downloadSelected func(),
	selectAll func(),
	deselectAll func(),
	renderPreviewModal func(string) tview.Primitive,
) {
	photoListBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgDn, tcell.KeyPgUp:
			// Block default up/down navigation
			return nil
		}
		return event
	})

	// Setup keymap
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch key := event.Key(); key {

		// up: move up one item
		case tcell.KeyUp:
			if event.Modifiers()&tcell.ModShift != 0 {
				expandSelection(itemIsSelected, toggleSelection, photoListBox, currentItem, Up)
			} else {
				moveSelection(photoListBox, setAppFocus, currentItem, Up)
			}

		// down: move down one item
		case tcell.KeyDown:
			if event.Modifiers()&tcell.ModShift != 0 {
				expandSelection(itemIsSelected, toggleSelection, photoListBox, currentItem, Down)
			} else {
				moveSelection(photoListBox, setAppFocus, currentItem, Down)
			}

		case tcell.KeyRune:
			switch event.Rune() {

			// j: move down one item
			case 'j':
				moveSelection(photoListBox, setAppFocus, currentItem, Down)
			// k: move up one item
			case 'k':
				moveSelection(photoListBox, setAppFocus, currentItem, Up)

			// Space: toggle selection of current item
			case ' ':
				toggleSelection(*currentItem)

			// d: download selected photos
			case 'd':
				downloadSelected()

			// p: show preview modal
			case 'p':
				if len(photos) > 0 && *currentItem >= 0 && *currentItem < len(photos) {
					name := photos[*currentItem]
					pages.AddPage("preview", renderPreviewModal(name), true, true)
				}

			// Shift + j: expand selection down one
			case 'J':
				expandSelection(itemIsSelected, toggleSelection, photoListBox, currentItem, Down)

			// Shift + k: expand selection down one
			case 'K':
				expandSelection(itemIsSelected, toggleSelection, photoListBox, currentItem, Up)

			// h or ? for help
			case 'h', '?':
				pages.AddPage("help", renderHelpModal(pages), true, true)
			}

		// Ctrl + A: select all items
		case tcell.KeyCtrlA:
			selectAll()

		// Ctrl + D: deselect all items
		case tcell.KeyCtrlD:
			deselectAll()

		// Ctrl + Q: quit the application
		case tcell.KeyCtrlQ:
			app.Stop()

		// Home / End: Scroll to beginning or end of log
		case tcell.KeyHome:
			scrollPhotoList(Up, photoListBox, setAppFocus)
		case tcell.KeyEnd:
			scrollPhotoList(Down, photoListBox, setAppFocus)

		// Page Up / Page Down: scroll the log
		case tcell.KeyPgUp:
			scrollLogBox(Up, logBox, setAppFocus)
		case tcell.KeyPgDn:
			scrollLogBox(Down, logBox, setAppFocus)
		}
		return event
	})
}

func expandSelection(itemIsSelected func(int) bool, toggleSelection func(int), photoListBox *tview.List, currentItem *int, direction Direction) {
	if photoListBox.GetItemCount() == 0 {
		return
	}
	// If the current item is not selected, select it first
	if !itemIsSelected(*currentItem) {
		toggleSelection(*currentItem)
	}

	*currentItem = (*currentItem + int(direction)) % photoListBox.GetItemCount()
	photoListBox.SetCurrentItem(*currentItem)
	toggleSelection(*currentItem)

}

func moveSelection(photoListBox *tview.List, setAppFocus func(t *tview.TextView, l *tview.List), currentItem *int, direction Direction) {
	setAppFocus(nil, photoListBox)
	if photoListBox.GetItemCount() == 0 {
		return
	}
	*currentItem = (*currentItem + int(direction)) % photoListBox.GetItemCount()
	photoListBox.SetCurrentItem(*currentItem)
}

type Direction int

const (
	Up   Direction = -1
	Down Direction = 1
)

func renderHelpModal(pages *tview.Pages) tview.Primitive {
	keybinds := [][]string{
		{"Up / k", "Move up one item"},
		{"Down / j", "Move down one item"},
		{"Shift+Up / K", "Expand selection up"},
		{"Shift+Down / J", "Expand selection down"},
		{"Space", "Toggle selection of current item"},
		{"Ctrl+A", "Select all items"},
		{"Ctrl+D", "Deselect all items"},
		{"d", "Download selected photos"},
		{"p", "Show image preview (ascii)"},
		{"PgUp / PgDn", "Scroll log up or down"},
		{"Home / End", "Scroll photo list to beginning or end"},
		{"h / ?", "Show this help"},
		{"Ctrl+Q", "Quit the application"},
	}

	logoBox := tview.NewTextView()
	logoBox.SetDynamicColors(true)
	logoBox.SetBorder(false)
	logoBox.SetTextAlign(tview.AlignCenter)
	logoBox.SetTextColor(tcell.ColorBlue)
	logoBox.SetWrap(false)
	logoBox.SetText(assets.Logo)

	versionBox := tview.NewTextView()
	versionBox.SetDynamicColors(true)
	versionBox.SetBorder(false)
	versionBox.SetTextAlign(tview.AlignCenter)
	versionBox.SetTextColor(tcell.ColorBlue)
	versionBox.SetWrap(false)
	versionBox.SetText(fmt.Sprintf("[yellow:]v%s", assets.Version))

	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false).
		SetWrapSelection(true, true)

	// Header
	table.SetCell(0, 0, tview.NewTableCell("[green::b]Key").
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignLeft).
		SetSelectable(false))
	table.SetCell(0, 1, tview.NewTableCell("[green::b]Action").
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignLeft).
		SetSelectable(false))

	// Keybinds
	for i, row := range keybinds {
		table.SetCell(i+1, 0, tview.NewTableCell("[white]"+row[0]))
		table.SetCell(i+1, 1, tview.NewTableCell("[white]"+row[1]))
	}

	// Add a row that shows where the configuration file is located
	rowIdx := len(keybinds) + 2
	table.SetCell(rowIdx, 0, tview.NewTableCell("[yellow::b]Config file"))
	table.SetCell(rowIdx, 1, tview.NewTableCell(fmt.Sprintf("[white]~/%s", configFileRelativePath)))

	table.SetBorder(true).SetTitle("Help / Keybindings")

	form := tview.NewForm().
		AddButton("Close", func() {
			pages.RemovePage("help")
		})
	form.SetButtonsAlign(tview.AlignCenter).
		SetButtonBackgroundColor(tcell.ColorWhite).
		SetButtonTextColor(tcell.ColorPurple)

	// Opaque spacers using tview.Box with background color
	leftSpacer := tview.NewBox().SetBackgroundColor(tcell.ColorBlack)
	rightSpacer := tview.NewBox().SetBackgroundColor(tcell.ColorBlack)

	centeredTable := tview.NewFlex().
		AddItem(leftSpacer, 0, 1, false).
		AddItem(table, 60, 0, false).
		AddItem(rightSpacer, 0, 1, false)

	// Flex layout: logo at top, horizontally centered table, close button at the bottom
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logoBox, 5, 0, false).
		AddItem(versionBox, 1, 0, false).
		AddItem(centeredTable, 0, 1, false).
		AddItem(form, 3, 0, true)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.RemovePage("help")
			return nil
		}
		return event
	})

	return flex
}

func scrollLogBox(direction Direction, logBox *tview.TextView, setAppFocus func(t *tview.TextView, l *tview.List)) {
	setAppFocus(logBox, nil)
	// Re-emit the key event to scroll the log box
	var key tcell.Key

	switch direction {
	case Up:
		key = tcell.KeyPgUp
	case Down:
		key = tcell.KeyPgDn
	}
	tcell.NewEventKey(key, 0, tcell.ModNone)
}

func scrollPhotoList(direction Direction, photoListBox *tview.List, setAppFocus func(t *tview.TextView, l *tview.List)) {
	setAppFocus(nil, photoListBox)

	// Re-emit the key event to scroll the log box
	var key tcell.Key

	switch direction {
	case Up:
		key = tcell.KeyHome
	case Down:
		key = tcell.KeyEnd
	}
	tcell.NewEventKey(key, 0, tcell.ModNone)
}

package viewer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

import (
	"github.com/hpxro7/bnkutil/bnk"
	"github.com/hpxro7/bnkutil/util"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

const (
	rsrcPath   = ":qml/images"
	errorTitle = "Error encountered"
)

var supportedFileFilters = strings.Join([]string{
	"SoundBank files (*.bnk *.nbnk)",
	"All files (*.*)",
}, ";;")

var saveFileFilters = strings.Join([]string{
	"MHW SoundBank file (*.nbnk)",
	"SoundBank file (*.bnk)",
	"All files (*.*)",
}, ";;")

var wemFileFilters = strings.Join([]string{
	"Wem files (*.wem)",
}, ";;")

type WwiseViewerWindow struct {
	widgets.QMainWindow

	actionOpen    *widgets.QAction
	actionSave    *widgets.QAction
	actionReplace *widgets.QAction
	actionExport  *widgets.QAction

	table          *WemTable
	selectionIndex int
}

func New() *WwiseViewerWindow {
	wv := NewWwiseViewerWindow(nil, 0)
	wv.SetWindowTitle(core.QCoreApplication_ApplicationName())

	tb := wv.AddToolBar3("Main Toolbar")
	tb.SetToolButtonStyle(core.Qt__ToolButtonTextBesideIcon)
	tb.SetAllowedAreas(core.Qt__TopToolBarArea | core.Qt__BottomToolBarArea)

	wv.setupOpen(tb)
	wv.setupSave(tb)
	wv.setupReplace(tb)
	wv.setupExport(tb)

	tb.AddSeparator()
	wv.AddToolBarBreak(core.Qt__TopToolBarArea)

	ltb := wv.setupLoopOptionsToolbar()
	wv.AddToolBar2(ltb)

	wv.table = NewTable()
	wv.selectionIndex = -1
	wv.table.ConnectSelectionChanged(wv.onWemSelected)
	wv.SetCentralWidget(wv.table)

	wv.SetFocus2()
	return wv
}

func (wv *WwiseViewerWindow) setupOpen(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-open", gui.NewQIcon5(rsrcPath+"/open.png"))
	wv.actionOpen = widgets.NewQAction3(icon, "&Open", wv)
	wv.actionOpen.ConnectTriggered(func(checked bool) {
		home := util.UserHome()
		path := widgets.QFileDialog_GetOpenFileName(
			wv, "Open file", home, supportedFileFilters, "", 0)
		if path != "" {
			wv.openBnk(path)
		}
	})
	toolbar.QWidget.AddAction(wv.actionOpen)
}

func (wv *WwiseViewerWindow) openBnk(path string) {
	bnk, err := bnk.Open(path)
	if err != nil {
		wv.showOpenError(path, err)
		return
	}
	wv.table.UpdateWems(bnk)
	wv.actionSave.SetEnabled(true)
	wv.actionExport.SetEnabled(true)
}

func (wv *WwiseViewerWindow) setupSave(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-save", gui.NewQIcon5(rsrcPath+"/save.png"))
	wv.actionSave = widgets.NewQAction3(icon, "&Save", wv)
	wv.actionSave.SetEnabled(false)
	wv.actionSave.ConnectTriggered(func(checked bool) {
		home := util.UserHome()
		path := widgets.QFileDialog_GetSaveFileName(
			wv, "Save file", home, saveFileFilters, "", 0)
		if path != "" {
			wv.saveBnk(path)
		}
	})
	toolbar.QWidget.AddAction(wv.actionSave)
}

func (wv *WwiseViewerWindow) saveBnk(path string) {
	outputFile, err := os.Create(path)
	if err != nil {
		wv.showSaveError(path, err)
	}
	count := wv.table.CommitReplacements()
	bnk := wv.table.GetSoundBank()

	total, err := bnk.WriteTo(outputFile)
	if err != nil {
		wv.showSaveError(path, err)
	}

	msg := fmt.Sprintf("Successfully saved %s.\n"+
		"%d wems have been replaced.\n"+
		"%d bytes have been written.", path, count, total)
	widgets.QMessageBox_Information(wv, "Save successful", msg, 0, 0)
}

func (wv *WwiseViewerWindow) setupReplace(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-replace",
		gui.NewQIcon5(rsrcPath+"/replace.png"))
	wv.actionReplace = widgets.NewQAction3(icon, "&Replace", wv)
	wv.actionReplace.SetEnabled(false)
	wv.actionReplace.ConnectTriggered(func(checked bool) {
		selection := wv.table.SelectionModel()
		indexes := selection.SelectedRows(0)
		if len(indexes) == 0 {
			return
		}
		home := util.UserHome()
		path := widgets.QFileDialog_GetOpenFileName(
			wv, "Open file", home, wemFileFilters, "", 0)
		if path != "" {
			wv.addReplacement(indexes[0].Row(), path)
		}
	})
	toolbar.QWidget.AddAction(wv.actionReplace)
}

func (wv *WwiseViewerWindow) addReplacement(index int, path string) {
	wem, err := os.Open(path)
	if err != nil {
		wv.showOpenError(path, err)
	}
	stat, err := wem.Stat()
	if err != nil {
		wv.showOpenError(path, err)
	}
	r := &bnk.ReplacementWem{wem, index, stat.Size()}
	wv.table.AddWemReplacement(stat.Name(), r)
}

func (wv *WwiseViewerWindow) setupExport(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-export",
		gui.NewQIcon5(rsrcPath+"/export.png"))
	wv.actionExport = widgets.NewQAction3(icon, "&Export Wems", wv)
	wv.actionExport.SetEnabled(false)
	wv.actionExport.ConnectTriggered(func(checked bool) {
		home := util.UserHome()
		opts := widgets.QFileDialog__ShowDirsOnly |
			widgets.QFileDialog__DontResolveSymlinks
		dir := widgets.QFileDialog_GetExistingDirectory(
			wv, "Choose directory to unpack into", home, opts)
		if dir != "" {
			wv.exportBnk(dir)
		}
	})
	toolbar.QWidget.AddAction(wv.actionExport)
}

func (wv *WwiseViewerWindow) setupLoopOptionsToolbar() *widgets.QToolBar {
	ltb := widgets.NewQToolBar("Loop Toolbar", nil)
	ltb.SetToolButtonStyle(core.Qt__ToolButtonTextOnly)

	checkboxLoop := widgets.NewQCheckBox2("&Loop", wv)
	checkboxInfinity := widgets.NewQCheckBox2("&Infinity", wv)
	lineEditValue := widgets.NewQLineEdit(wv)
	lineEditValue.SetPlaceholderText("Times to loop")
	lineEditValue.SetMaximumWidth(90)
	lineEditValue.SetMaxLength(10)

	actionSetLoop := widgets.NewQAction2("Set &Loop", wv)
	actionSetLoop.ConnectTriggered(func(checked bool) {

	})

	ltb.AddWidget(checkboxLoop)
	ltb.AddWidget(checkboxInfinity)
	ltb.AddWidget(lineEditValue)
	ltb.QWidget.AddAction(actionSetLoop)
	ltb.AddSeparator()
	ltb.SetEnabled(false)

	return ltb
}

func (wv *WwiseViewerWindow) exportBnk(dir string) {
	total := int64(0)
	bnk := wv.table.GetSoundBank()
	for i, wem := range bnk.DataSection.Wems {
		filename := util.CanonicalWemName(i, bnk.IndexSection.WemCount)
		f, err := os.Create(filepath.Join(dir, filename))
		if err != nil {
			wv.showExportError(filename, dir, err)
			return
		}
		n, err := io.Copy(f, wem)
		if err != nil {
			wv.showExportError(filename, dir, err)
			return
		}
		total += n
	}

	count := len(bnk.DataSection.Wems)
	msg := fmt.Sprintf("Successfully exported wems to %s.\n"+
		"%d wems have been exported.\n"+
		"%d bytes have been written.", dir, count, total)
	widgets.QMessageBox_Information(wv, "Save successful", msg, 0, 0)
}

func (wv *WwiseViewerWindow) onWemSelected(selected *core.QItemSelection,
	deselected *core.QItemSelection) {
	// The following is an unfortunate hack. Connecting selection on the
	// table causes graphical selection glitches, likely because the original
	// selection logic was overridden. Since we don't have a way to call the super
	// class's SelectionChanged, we disable this one (to prevent recursion), call
	// SelectionChanged, and connect it back.
	wv.table.DisconnectSelectionChanged()
	wv.table.SelectionChanged(selected, deselected)
	wv.table.ConnectSelectionChanged(wv.onWemSelected)

	if len(selected.Indexes()) == 0 {
		wv.actionReplace.SetEnabled(false)
		return
	}
	wv.actionReplace.SetEnabled(true)
}

func (wv *WwiseViewerWindow) showExportError(filename string, path string,
	err error) {
	msg := fmt.Sprintf("Could not write wem file %s to %s:\n%s.\n"+
		"Aborting the export operation.", filename, path, err)
	widgets.QMessageBox_Critical4(wv, errorTitle, msg, 0, 0)
}

func (wv *WwiseViewerWindow) showSaveError(path string, err error) {
	msg := fmt.Sprintf("Could not save file %s:\n%s", path, err)
	widgets.QMessageBox_Critical4(wv, errorTitle, msg, 0, 0)
}

func (wv *WwiseViewerWindow) showOpenError(path string, err error) {
	msg := fmt.Sprintf("Could not open %s:\n%s", path, err)
	widgets.QMessageBox_Critical4(wv, errorTitle, msg, 0, 0)
}

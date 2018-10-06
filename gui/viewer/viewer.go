package viewer

import (
	"log"
	"strings"
)

import (
	"github.com/hpxro7/bnkutil/util"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

const (
	rsrcPath = ":qml/images"
)

var fileDialogFilters = strings.Join([]string{
	"SoundBank files (*.bnk *.nbnk)",
	"All files (*.*)",
}, ";;")

type WwiseViewerWindow struct {
	widgets.QMainWindow

	actionOpen    *widgets.QAction
	actionSave    *widgets.QAction
	actionReplace *widgets.QAction

	tableView *widgets.QTableView
}

func New() *WwiseViewerWindow {
	wv := NewWwiseViewerWindow(nil, 0)
	wv.SetWindowTitle(core.QCoreApplication_ApplicationName())

	toolbar := wv.AddToolBar3("MainToolbar")
	toolbar.SetToolButtonStyle(core.Qt__ToolButtonTextBesideIcon)

	wv.setupOpen(toolbar)
	wv.setupSave(toolbar)
	wv.setupReplace(toolbar)

	wv.tableView = NewWemTable(NewModel())
	wv.SetCentralWidget(wv.tableView)

	wv.SetFocus2()
	return wv
}

func (wv *WwiseViewerWindow) setupOpen(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-open", gui.NewQIcon5(rsrcPath+"/open.png"))
	wv.actionOpen = widgets.NewQAction3(icon, "&Open", wv)
	wv.actionOpen.ConnectTriggered(func(checked bool) {
		home := util.UserHome()
		path := widgets.QFileDialog_GetOpenFileName(
			wv, "Open file", home, fileDialogFilters, "", 0)
		widgets.QMessageBox_About(wv, "Chose path...", path)
		log.Println("Chose file:", path)
	})
	toolbar.QWidget.AddAction(wv.actionOpen)
}

func (wv *WwiseViewerWindow) setupSave(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-save", gui.NewQIcon5(rsrcPath+"/save.png"))
	wv.actionSave = widgets.NewQAction3(icon, "&Save", wv)
	toolbar.QWidget.AddAction(wv.actionSave)
}

func (wv *WwiseViewerWindow) setupReplace(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-replace",
		gui.NewQIcon5(rsrcPath+"/replace.png"))
	wv.actionReplace = widgets.NewQAction3(icon, "&Replace", wv)
	toolbar.QWidget.AddAction(wv.actionReplace)
}

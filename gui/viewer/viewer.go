package viewer

import (
	"strconv"
)

import (
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

const (
	rsrcPath    = ":qml/images"
	columnCount = 4
)

type WwiseViewerWindow struct {
	widgets.QMainWindow

	actionOpen    *widgets.QAction
	actionSave    *widgets.QAction
	actionReplace *widgets.QAction

	tableView *widgets.QTableView
	wems      []string
}

func New() *WwiseViewerWindow {
	wv := NewWwiseViewerWindow(nil, 0)
	wv.SetWindowTitle(core.QCoreApplication_ApplicationName())

	wv.wems = []string{"001.wem", "002.wem", "003.wem"}

	toolbar := wv.AddToolBar3("MainToolbar")
	toolbar.SetToolButtonStyle(core.Qt__ToolButtonTextBesideIcon)

	wv.setupOpen(toolbar)
	wv.setupSave(toolbar)
	wv.setupReplace(toolbar)

	wv.tableView = widgets.NewQTableView(nil)
	wv.tableView.VerticalHeader().Hide()
	wv.tableView.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	wv.tableView.HorizontalHeader().SetSectionResizeMode(
		widgets.QHeaderView__Stretch)
	wv.tableView.HorizontalHeader().SetHighlightSections(false)

	model := core.NewQAbstractTableModel(nil)

	model.ConnectRowCount(wv.rowCount)
	model.ConnectColumnCount(func(p *core.QModelIndex) int { return columnCount })
	model.ConnectData(wv.data)
	model.ConnectHeaderData(wv.headerData)
	wv.tableView.SetModel(model)

	wv.SetCentralWidget(wv.tableView)

	wv.SetFocus2()
	return wv
}

func (wv *WwiseViewerWindow) setupOpen(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-open",
		gui.NewQIcon5(rsrcPath+"/open.png"))
	wv.actionOpen = widgets.NewQAction3(icon, "&Open", wv)
	toolbar.QWidget.AddAction(wv.actionOpen)
}

func (wv *WwiseViewerWindow) setupSave(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-save",
		gui.NewQIcon5(rsrcPath+"/save.png"))
	wv.actionSave = widgets.NewQAction3(icon, "&Save", wv)
	toolbar.QWidget.AddAction(wv.actionSave)
}

func (wv *WwiseViewerWindow) setupReplace(toolbar *widgets.QToolBar) {
	icon := gui.QIcon_FromTheme2("wwise-replace",
		gui.NewQIcon5(rsrcPath+"/replace.png"))
	wv.actionReplace = widgets.NewQAction3(icon, "&Replace", wv)
	toolbar.QWidget.AddAction(wv.actionReplace)
}

func (vw *WwiseViewerWindow) rowCount(parent *core.QModelIndex) int {
	return len(vw.wems)
}

func (vw *WwiseViewerWindow) data(index *core.QModelIndex,
	role int) *core.QVariant {
	if !index.IsValid() || index.Row() >= len(vw.wems) ||
		role != int(core.Qt__DisplayRole) {
		return core.NewQVariant()
	}

	return core.NewQVariant14(vw.wems[index.Row()])
}

func (vw *WwiseViewerWindow) headerData(section int,
	orientation core.Qt__Orientation, role int) *core.QVariant {
	if role != int(core.Qt__DisplayRole) || orientation != core.Qt__Horizontal {
		return core.NewQVariant()
	}

	return core.NewQVariant14("Col" + strconv.Itoa(section+1))
}

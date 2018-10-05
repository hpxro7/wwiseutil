package viewer

import (
	"strconv"
)

import (
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

const columnCount = 4

type WemModel struct {
	core.QAbstractTableModel
	wems []string
}

func NewWemTable(model *WemModel) *widgets.QTableView {
	table := widgets.NewQTableView(nil)

	table.VerticalHeader().Hide()
	table.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	table.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__Stretch)
	table.HorizontalHeader().SetHighlightSections(false)
	table.SetModel(model)

	return table
}

func NewModel() *WemModel {
	model := NewWemModel(nil)

	model.wems = []string{"001.wem", "002.wem", "003.wem"}

	model.ConnectRowCount(model.rowCount)
	model.ConnectColumnCount(model.columnCount)
	model.ConnectData(model.data)
	model.ConnectHeaderData(model.headerData)

	return model
}

func (m *WemModel) rowCount(parent *core.QModelIndex) int {
	return len(m.wems)
}

func (m *WemModel) columnCount(parent *core.QModelIndex) int {
	return columnCount
}

func (m *WemModel) data(index *core.QModelIndex,
	role int) *core.QVariant {
	if !index.IsValid() || index.Row() >= len(m.wems) ||
		role != int(core.Qt__DisplayRole) {
		return core.NewQVariant()
	}

	return core.NewQVariant14(m.wems[index.Row()])
}

func (m *WemModel) headerData(section int,
	orientation core.Qt__Orientation, role int) *core.QVariant {
	if role != int(core.Qt__DisplayRole) || orientation != core.Qt__Horizontal {
		return core.NewQVariant()
	}

	return core.NewQVariant14("Col" + strconv.Itoa(section+1))
}

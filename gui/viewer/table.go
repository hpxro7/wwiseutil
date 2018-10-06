package viewer

import (
	"fmt"
)

import (
	"github.com/hpxro7/bnkutil/bnk"
	"github.com/hpxro7/bnkutil/util"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

type wemAccessor func(index int) string

type columnBinding struct {
	title    string
	accessor wemAccessor
}

type WemTable struct {
	widgets.QTableView
}

type WemModel struct {
	core.QAbstractTableModel
	sec      *bnk.DataSection
	bindings []*columnBinding
}

func NewTable() *WemTable {
	table := NewWemTable(nil)

	table.VerticalHeader().Hide()
	table.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	table.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__Stretch)
	table.HorizontalHeader().SetHighlightSections(false)

	table.UpdateWems(nil)

	return table
}

func (t *WemTable) UpdateWems(section *bnk.DataSection) {
	m := newModel()
	m.sec = section
	m.bindings = []*columnBinding{
		{"Name", m.defaultOr(m.wemName)},
		{"Replacing with", empty},
		{"Size", m.defaultOr(m.wemSize)},
		{"File offset", m.defaultOr(m.wemOffset)},
		{"Padding", m.defaultOr(m.wemPadding)},
	}
	t.SetModel(m)
}

func newModel() *WemModel {
	model := NewWemModel(nil)

	model.ConnectRowCount(model.rowCount)
	model.ConnectColumnCount(model.columnCount)
	model.ConnectData(model.data)
	model.ConnectHeaderData(model.headerData)

	return model
}

func (m *WemModel) defaultOr(accessor wemAccessor) wemAccessor {
	if m.sec == nil {
		return empty
	}
	return accessor
}

func empty(index int) string {
	return ""
}

func (m *WemModel) wemName(index int) string {
	return util.CanonicalWemName(index, len(m.sec.Wems))
}

func (m *WemModel) wemSize(index int) string {
	return fmt.Sprintf("%d bytes", m.sec.Wems[index].Descriptor.Length)
}

func (m *WemModel) wemOffset(index int) string {
	offsetIntoFile := m.sec.Wems[index].Descriptor.Offset + m.sec.DataStart
	return fmt.Sprintf("0x%X", offsetIntoFile)
}

func (m *WemModel) wemPadding(index int) string {
	paddingSize := m.sec.Wems[index].Padding.Size()
	return fmt.Sprintf("%d bytes", paddingSize)
}

func (m *WemModel) rowCount(parent *core.QModelIndex) int {
	if m.sec == nil {
		return 0
	}
	return len(m.sec.Wems)
}

func (m *WemModel) columnCount(parent *core.QModelIndex) int {
	return len(m.bindings)
}

func (m *WemModel) data(index *core.QModelIndex,
	role int) *core.QVariant {
	if !index.IsValid() || m.sec == nil || index.Row() >= len(m.sec.Wems) ||
		role != int(core.Qt__DisplayRole) {
		return core.NewQVariant()
	}

	accessor := m.bindings[index.Column()].accessor
	return core.NewQVariant14(accessor(index.Row()))
}

func (m *WemModel) headerData(section int,
	orientation core.Qt__Orientation, role int) *core.QVariant {
	if role != int(core.Qt__DisplayRole) || orientation != core.Qt__Horizontal {
		return core.NewQVariant()
	}

	return core.NewQVariant14(m.bindings[section].title)
}

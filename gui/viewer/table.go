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

type replacementWrapper struct {
	name        string
	replacement *bnk.ReplacementWem
}

type WemTable struct {
	widgets.QTableView
	model *WemModel
}

type WemModel struct {
	core.QAbstractTableModel
	bindings []*columnBinding

	bnk *bnk.File
	// A mapping from wem index to the replacement wem.
	replacements map[int]*replacementWrapper
}

func NewTable() *WemTable {
	table := NewWemTable(nil)

	table.VerticalHeader().Hide()
	table.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	table.SetSelectionMode(widgets.QAbstractItemView__SingleSelection)
	table.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__Stretch)
	table.HorizontalHeader().SetHighlightSections(false)

	table.UpdateWems(nil)

	return table
}

func (t *WemTable) UpdateWems(file *bnk.File) {
	m := newModel()
	m.bnk = file
	m.bindings = []*columnBinding{
		{"Name", m.defaultOr(m.wemName)},
		{"Replacing with", m.defaultOr(m.wemReplacement)},
		{"Size", m.defaultOr(m.wemSize)},
		{"File offset", m.defaultOr(m.wemOffset)},
		{"Padding", m.defaultOr(m.wemPadding)},
		{"Loops", m.defaultOr(m.wemLoops)},
	}

	t.model = m
	t.SetModel(t.model)
}

func (t *WemTable) AddWemReplacement(name string, r *bnk.ReplacementWem) {
	t.model.replacements[r.WemIndex] = &replacementWrapper{name, r}
	// Modify the entire row for that wem.
	count := t.model.columnCount(nil)
	start := t.IndexAt(core.NewQPoint2(r.WemIndex, 0))
	end := t.IndexAt(core.NewQPoint2(r.WemIndex, count-1))

	var roles []int
	for i := 0; i < count; i++ {
		roles = append(roles, int(core.Qt__DisplayRole))
	}

	t.DataChanged(start, end, roles)
}

// CommitReplacements commits all changes to the current in-memory audio file.
// Pending replacements are removed, and the table is refreshed. The number
// of replacements commited is returned.
func (t *WemTable) CommitReplacements() int {
	var rs []*bnk.ReplacementWem
	for _, w := range t.model.replacements {
		rs = append(rs, w.replacement)
	}
	count := len(rs)
	t.model.bnk.ReplaceWems(rs...)

	// Clear all current replacements after committing them.
	t.model.replacements = make(map[int]*replacementWrapper)

	// Update the viewmodel with new wem information.
	rows := t.model.rowCount(nil)
	cols := t.model.columnCount(nil)

	start := t.IndexAt(core.NewQPoint2(0, 0))
	end := t.IndexAt(core.NewQPoint2(rows-1, cols-1))

	var roles []int
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			roles = append(roles, int(core.Qt__DisplayRole))
		}
	}

	t.DataChanged(start, end, roles)
	return count
}

func (t *WemTable) GetSoundBank() *bnk.File {
	return t.model.bnk
}

func newModel() *WemModel {
	model := NewWemModel(nil)
	model.replacements = make(map[int]*replacementWrapper)

	model.ConnectRowCount(model.rowCount)
	model.ConnectColumnCount(model.columnCount)
	model.ConnectData(model.data)
	model.ConnectHeaderData(model.headerData)

	return model
}

func (m *WemModel) defaultOr(accessor wemAccessor) wemAccessor {
	if m.bnk == nil {
		return empty
	}
	return accessor
}

func empty(index int) string {
	return ""
}

func (m *WemModel) wemName(index int) string {
	return util.CanonicalWemName(index, len(m.bnk.DataSection.Wems))
}

func (m *WemModel) wemReplacement(index int) string {
	r, ok := m.replacements[index]
	if !ok {
		return ""
	}
	return r.name
}

func (m *WemModel) wemSize(index int) string {
	return fmt.Sprintf("%d bytes", m.bnk.DataSection.Wems[index].Descriptor.Length)
}

func (m *WemModel) wemOffset(index int) string {
	sec := m.bnk.DataSection
	offsetIntoFile := sec.Wems[index].Descriptor.Offset + sec.DataStart
	return fmt.Sprintf("0x%X", offsetIntoFile)
}

func (m *WemModel) wemPadding(index int) string {
	paddingSize := m.bnk.DataSection.Wems[index].Padding.Size()
	return fmt.Sprintf("%d bytes", paddingSize)
}

func (m *WemModel) wemLoops(index int) string {
	str := "None"
	if m.bnk.DataSection == nil || m.bnk.ObjectSection == nil {
		return str
	}

	desc := m.bnk.DataSection.Wems[index].Descriptor
	opt := bnk.OptionalWemDescriptor{WemId: desc.WemId, WemLength: desc.Length}
	loopCount, ok := m.bnk.ObjectSection.LoopOf[opt]
	if ok {
		if loopCount == bnk.InfiniteLoops {
			str = "Infinity"
		} else {
			str = fmt.Sprintf("%d times", loopCount)
		}
	}
	return str
}

func (m *WemModel) rowCount(parent *core.QModelIndex) int {
	if m.bnk == nil {
		return 0
	}
	return m.bnk.IndexSection.WemCount
}

func (m *WemModel) columnCount(parent *core.QModelIndex) int {
	return len(m.bindings)
}

func (m *WemModel) data(index *core.QModelIndex,
	role int) *core.QVariant {
	if !index.IsValid() || m.bnk == nil || m.bnk.DataSection == nil ||
		index.Row() >= len(m.bnk.DataSection.Wems) ||
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

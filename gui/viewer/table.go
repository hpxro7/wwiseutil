package viewer

import (
	"fmt"
)

import (
	"github.com/hpxro7/wwiseutil/bnk"
	"github.com/hpxro7/wwiseutil/pck"
	"github.com/hpxro7/wwiseutil/util"
	"github.com/hpxro7/wwiseutil/wwise"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

type wemAccessor func(index int) string

type columnBinding struct {
	title    string
	accessor wemAccessor
}

type replacementWemWrapper struct {
	name        string
	replacement *wwise.ReplacementWem
}

type loopWrapper struct {
	loops    bool
	infinity bool
	value    uint32
}

type WemTable struct {
	widgets.QTableView
	model *WemModel
}

type WemModel struct {
	core.QAbstractTableModel
	bindings []*columnBinding

	ctn wwise.Container
	// A mapping from wem index to the replacement wem.
	replacements map[int]*replacementWemWrapper
}

func NewTable() *WemTable {
	table := NewWemTable(nil)

	table.VerticalHeader().Hide()
	table.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	table.SetSelectionMode(widgets.QAbstractItemView__SingleSelection)
	table.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__Stretch)
	table.HorizontalHeader().SetHighlightSections(false)

	table.LoadDefaultModel()

	return table
}

func (t *WemTable) LoadDefaultModel() {
	m := newModel()
	m.bindings = []*columnBinding{
		{"Name", empty},
		{"Replacing with", empty},
		{"Size", empty},
		{"File offset", empty},
		{"Padding", empty},
		{"Loops", empty},
	}

	t.model = m
	t.SetModel(t.model)
}

func (t *WemTable) LoadSoundBankModel(file *bnk.File) {
	m := newModel()
	m.ctn = file
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

func (t *WemTable) LoadFilePackageModel(file *pck.File) {
	m := newModel()
	m.ctn = file
	m.bindings = []*columnBinding{
		{"Name", m.defaultOr(m.wemName)},
		{"Replacing with", m.defaultOr(m.wemReplacement)},
		{"Size", m.defaultOr(m.wemSize)},
		{"File offset", m.defaultOr(m.wemOffset)},
		{"Padding", m.defaultOr(m.wemPadding)},
	}

	t.model = m
	t.SetModel(t.model)
}

func (t *WemTable) AddWemReplacement(name string, r *wwise.ReplacementWem) {
	t.model.replacements[r.WemIndex] = &replacementWemWrapper{name, r}
	// Modify the entire row for that wem.
	t.refreshRow(r.WemIndex)
}

func (t *WemTable) UpdateLoop(wemIndex int, r *loopWrapper) {
	switch ctn := t.model.ctn.(type) {
	case *bnk.File:
		loop := bnk.LoopValue{}
		if r.loops {
			if r.infinity {
				loop.Loops, loop.Value = true, 0
			} else {
				loop.Loops, loop.Value = true, r.value
			}
		}
		ctn.ReplaceLoopOf(wemIndex, loop)
		t.refreshRow(wemIndex)
	default:
		return
	}
}

// CommitReplacements commits all changes to the current in-memory audio file.
// Pending replacements are removed, and the table is refreshed. The number
// of replacements commited is returned.
func (t *WemTable) CommitReplacements() int {
	var rs []*wwise.ReplacementWem
	for _, w := range t.model.replacements {
		rs = append(rs, w.replacement)
	}
	count := len(rs)
	t.model.ctn.ReplaceWems(rs...)

	// Clear all current replacements after committing them.
	t.model.replacements = make(map[int]*replacementWemWrapper)

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

func (t *WemTable) GetContainer() wwise.Container {
	return t.model.ctn
}

func (t *WemTable) refreshRow(row int) {
	count := t.model.columnCount(nil)
	start := t.IndexAt(core.NewQPoint2(row, 0))
	end := t.IndexAt(core.NewQPoint2(row, count-1))

	var roles []int
	for i := 0; i < count; i++ {
		roles = append(roles, int(core.Qt__DisplayRole))
	}

	t.model.DataChanged(start, end, roles)
	// We have to repaint the table after changing the data, or the table doesn't
	// refresh properly until we refocus on it.
	t.Viewport().Repaint()
}

func newModel() *WemModel {
	model := NewWemModel(nil)
	model.replacements = make(map[int]*replacementWemWrapper)

	model.ConnectRowCount(model.rowCount)
	model.ConnectColumnCount(model.columnCount)
	model.ConnectData(model.data)
	model.ConnectHeaderData(model.headerData)

	return model
}

func (m *WemModel) defaultOr(accessor wemAccessor) wemAccessor {
	if m.ctn == nil {
		return empty
	}
	return accessor
}

func empty(index int) string {
	return ""
}

func (m *WemModel) wemName(index int) string {
	return util.CanonicalWemName(index, len(m.ctn.Wems()))
}

func (m *WemModel) wemReplacement(index int) string {
	r, ok := m.replacements[index]
	if !ok {
		return ""
	}
	return r.name
}

func (m *WemModel) wemSize(index int) string {
	return fmt.Sprintf("%d bytes", m.ctn.Wems()[index].Descriptor.Length)
}

func (m *WemModel) wemOffset(index int) string {
	wems := m.ctn.Wems()
	offsetIntoFile := wems[index].Descriptor.Offset + m.ctn.DataStart()
	return fmt.Sprintf("0x%X", offsetIntoFile)
}

func (m *WemModel) wemPadding(index int) string {
	paddingSize := m.ctn.Wems()[index].Padding.Size()
	return fmt.Sprintf("%d bytes", paddingSize)
}

func (m *WemModel) wemLoops(index int) string {
	str := "None"
	switch ctn := m.ctn.(type) {
	case *bnk.File:
		loop := ctn.LoopOf(index)
		if loop.Loops {
			if loop.Value == bnk.InfiniteLoops {
				str = "Infinity"
			} else {
				str = fmt.Sprintf("%d times", loop.Value)
			}
		}
	}

	return str
}

func (m *WemModel) rowCount(parent *core.QModelIndex) int {
	if m.ctn == nil {
		return 0
	}
	return len(m.ctn.Wems())
}

func (m *WemModel) columnCount(parent *core.QModelIndex) int {
	return len(m.bindings)
}

func (m *WemModel) data(index *core.QModelIndex,
	role int) *core.QVariant {
	if !index.IsValid() || m.ctn == nil || len(m.ctn.Wems()) == 0 ||
		index.Row() >= len(m.ctn.Wems()) ||
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

package viewer

import (
	"github.com/therecipe/qt/widgets"
)

type WwiseViewerWindow struct {
	widgets.QMainWindow
}

func New() *WwiseViewerWindow {
	return NewWwiseViewerWindow(nil, 0)
}

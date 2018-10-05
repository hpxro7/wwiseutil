package main

import (
	"log"
	"os"
)

import (
	"github.com/hpxro7/bnkutil/gui/viewer"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

const (
	windowWidth  = 860
	windowHeight = 480
)

func main() {
	log.Println("Starting bnkutil GUI...")
	app := widgets.NewQApplication(len(os.Args), os.Args)
	core.QCoreApplication_SetApplicationName("Wwise SoundBank Utils")
	core.QCoreApplication_SetApplicationVersion("0.3")

	parser := core.NewQCommandLineParser()
	parser.SetApplicationDescription(core.QCoreApplication_ApplicationName())
	parser.AddHelpOption()
	parser.AddVersionOption()
	parser.Process2(app)

	window := viewer.New()

	availableGeometry := widgets.QApplication_Desktop().AvailableGeometry(window)
	window.Resize2(windowWidth, windowHeight)
	// Move the window to the center of the screen.
	window.Move2((availableGeometry.Width()-window.Width())/2,
		(availableGeometry.Height()-window.Height())/2)

	window.Show()
	app.Exec()
}

package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
)

const lineWidth = 5

func main() {
	a := app.New()
	w := a.NewWindow("HATO Senju")
	nagase1 := canvas.NewLine(color.White)
	nagase1.StrokeWidth = lineWidth
	nagase1.Position1 = fyne.NewPos(0, 0)
	nagase1.Position2 = fyne.NewPos(1, 0)
	w.SetContent(nagase1)
	w.ShowAndRun()
}

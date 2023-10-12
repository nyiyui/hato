package senri

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"nyiyui.ca/hato/sakayukari/tal"
)

func Main(g *tal.Guide) error {
	a := app.New()
	w := a.NewWindow("HATO Senri")

	placeholder := image.NewRGBA(image.Rectangle{image.Point{}, image.Point{100, 100}})
	for x := 0; x < placeholder.Bounds().Max.X; x++ {
		for y := 0; y < placeholder.Bounds().Max.Y; y++ {
			placeholder.Set(x, y, color.White)
		}
	}
	con := container.New(
		layout.NewVBoxLayout(),
		widget.NewLabel("chart"),
		widget.NewLabel("loading"),
		widget.NewLabel("chart"),
		widget.NewLabel("loading"),
	)
	for i := range con.Objects {
		if i%2 == 0 {
			img := canvas.NewImageFromImage(placeholder)
			img.FillMode = canvas.ImageFillOriginal
			con.Objects[i] = img
		} else {
			label := widget.NewLabel("loading")
			con.Objects[i] = label
		}
	}
	log.Printf("obj: %#v", con.Objects)

	//list := widget.NewList(func() int { return len(images) }, func() fyne.CanvasObject {
	//	_ = img
	//	return container.New(
	//		layout.NewHBoxLayout(),
	//		widget.NewLabel("loading"),
	//		//img,
	//	)
	//}, func(i int, obj fyne.CanvasObject) {
	//	log.Printf("%d rect %#v", i, images[i].Bounds())
	//	cont := obj.(*fyne.Container)
	//	label := cont.Objects[0].(*widget.Label)
	//	label.SetText(labels[i])
	//	label.Refresh()
	//	//img := cont.Objects[1].(*canvas.Image)
	//	//img.Image = images[i]
	//	//img.Refresh()
	//})

	c := make(chan tal.GuideSnapshot, 10)
	g.SnapshotMux.Subscribe("senri", c)
	go func() {
		defer g.SnapshotMux.Unsubscribe(c)
		for gs := range c {
			for i, t := range gs.Trains {
				img := con.Objects[i*2].(*canvas.Image)
				chart, err := trainChart(g, &t)
				if err != nil {
					log.Printf("senri: train %d: chart: %s", i, err)
				} else {
					img.Image = chart
					img.Refresh()
				}

				label := con.Objects[i*2+1].(*widget.Label)
				label.SetText(fmt.Sprintf("%s", &t))
				label.Refresh()
			}
		}
	}()

	w.SetContent(con)
	w.ShowAndRun()
	return nil
}

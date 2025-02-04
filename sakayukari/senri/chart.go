package senri

import (
	"errors"
	"fmt"
	"image"

	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
	"nyiyui.ca/hato/sakayukari/tal"
)

func trainChart(g *tal.Guide, t *tal.Train) (image.Image, error) {
	viridisByY := func(xr, yr chart.Range, index int, x, y float64) drawing.Color {
		return chart.Viridis(y, yr.GetMin(), yr.GetMax())
	}
	_ = viridisByY

	if len(t.History.Spans) == 0 {
		return nil, errors.New("no history")
	}

	start := t.History.Spans[0].Time
	xValues := make([]float64, 0, len(t.History.Spans))
	yValues := make([]float64, 0, len(t.History.Spans))
	for _, span := range t.History.Spans {
		xValues = append(xValues, float64(start.Sub(span.Time).Milliseconds()))
		yValues = append(yValues, float64(span.Position))
	}

	var pointsChart, polyChart chart.Series
	var nValues int
	{
		char := t.History.Character()
		xValues := make([]float64, len(char.Points))
		yValues := make([]float64, len(char.Points))
		for i, point := range char.Points {
			xValues[i] = float64(point[0])
			yValues[i] = float64(point[1])
		}
		fd, ok := g.Model2.GetFormData(t.FormI)
		if ok {
			for _, point := range fd.Points {
				xValues = append(xValues, float64(point[0]))
				yValues = append(yValues, float64(point[1]))
			}
		}
		//fit := polyfit.NewFit(xValues, yValues, 2)
		//log.Printf("solve %#v", fit.Solve())
		pointsChart2 := chart.ContinuousSeries{
			Style: chart.Style{
				StrokeWidth: chart.Disabled,
				DotWidth:    5,
				DotColor:    drawing.ColorFromHex("000000"),
			},
			XValues: xValues,
			YValues: yValues,
		}
		pointsChart = pointsChart2
		polyChart = &chart.PolynomialRegressionSeries{
			Degree:      2,
			InnerSeries: pointsChart2,
		}
		nValues = len(xValues)
	}

	graph := chart.Chart{
		Title:  fmt.Sprintf("%d values", nValues),
		Height: 400,
		Series: []chart.Series{
			polyChart,
			//chart.ContinuousSeries{
			//	Style: chart.Style{
			//		StrokeWidth:      chart.Disabled,
			//		DotWidth:         10,
			//		DotColorProvider: viridisByY,
			//	},
			//	XValues: xValues,
			//	YValues: yValues,
			//},
			pointsChart,
		},
	}

	//	f, _ := os.Create("output.png")
	//	defer f.Close()
	//	graph.Render(chart.PNG, f)

	collector := &chart.ImageWriter{}
	graph.Render(chart.PNG, collector)
	image, err := collector.Image()
	if err != nil {
		return nil, err
	}
	return image, nil
}

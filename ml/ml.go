package ml

import (
	"bufio"
	"encoding/csv"
	"image/color"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const dataPath = "data/dimmer_data.csv"

func drawPlot() {
	r, err := openFile(dataPath)
	dataMap := make(map[string]*plotter.XYs)
	// Iterate through the records
	for {
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		time, _ := strconv.ParseFloat(strings.Replace(record[1], ":", "", -1), 64)
		group := record[2]
		val, _ := strconv.Atoi(record[3])
		xy := plotter.XY{
			X: float64(val) / 255 * 100,
			Y: time / 10000,
		}

		if points, ok := dataMap[group]; !ok {
			dataMap[group] = &plotter.XYs{xy}
		} else {
			*points = append(*points, xy)
		}
	}
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Dimmer values"
	p.X.Label.Text = "values, %"
	p.Y.Label.Text = "time of day"
	colors := []color.RGBA{
		{R: 128, B: 255, A: 133},
		{R: 185, B: 10, A: 3},
	}
	i := 0
	p.Add(plotter.NewGrid())
	rand.Seed(time.Now().UnixNano())
	for k, v := range dataMap {
		s, _ := plotter.NewScatter(v)
		s.GlyphStyle.Color = colors[i]
		i++
		p.Legend.Add(k, s)
		p.Add(s)
	}
	if err := p.Save(10*vg.Inch, 10*vg.Inch, "points.png"); err != nil {
		panic(err)
	}
}

func openFile(path string) (*csv.Reader, error) {
	dimmerFile, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer dimmerFile.Close()
	r := csv.NewReader(bufio.NewReader(dimmerFile))
	return r, err
}

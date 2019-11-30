package stat

import (
	"bufio"
	"encoding/csv"
	"image/color"
	"io"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"github.com/vadymc/ikea-gateway-go/m/sql"
)

const (
	dataPath = "data/dimmer_data.csv"
)

type Group struct {
	Name string
	Data *map[int]Value
}

type Value struct {
	Val *[]int
}

type intSlice []int

func (p intSlice) Len() int           { return len(p) }
func (p intSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p intSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func CalcQuantiles(db *sql.DBStorage) {
	startData := time.Now().AddDate(0, 0, -14)
	rawData := db.SelectRawData(startData)

	data := make(map[string]Group)
	for _, rd := range *rawData {
		if _, ok := data[rd.GroupName]; !ok {
			data[rd.GroupName] = Group{
				Name: rd.GroupName,
				Data: &map[int]Value{},
			}
		}

		time, _ := strconv.ParseFloat(normalizeDateString(rd.Date), 64)
		timeKey := int(time / 10000)

		group := data[rd.GroupName]
		groupData := group.Data
		if _, ok := (*groupData)[timeKey]; !ok {
			(*groupData)[timeKey] = Value{Val: &[]int{}}
		}
		groupDataVals := (*groupData)[timeKey]
		*groupDataVals.Val = append(*groupDataVals.Val, rd.Dimmer)
	}
	for _, lg := range data {
		for hour, vals := range *lg.Data {
			val := vals.Val
			dbGroup := sql.QuantileGroup{
				Name:        lg.Name,
				BucketIndex: hour,
				BucketVal:   percentile(*val, 85),
			}
			db.SaveQuantileGroup(&dbGroup)
		}
	}
	log.Info("Recalculated Quantiles")
}

func normalizeDateString(date string) string {
	s := strings.SplitAfter(date, "T")[1]
	s = strings.Replace(s, ":", "", -1)
	s = strings.Replace(s, "Z", "", -1)
	return s
}

func percentile(values intSlice, perc float64) int {
	ps := []float64{perc}

	scores := make([]float64, len(ps))
	size := len(values)
	if size > 0 {
		sort.Sort(values)
		for i, p := range ps {
			pos := p * float64(size+1)
			if pos < 1.0 {
				scores[i] = float64(values[0])
			} else if pos >= float64(size) {
				scores[i] = float64(values[size-1])
			} else {
				lower := float64(values[int(pos)-1])
				upper := float64(values[int(pos)])
				scores[i] = lower + (pos-math.Floor(pos))*(upper-lower)
			}
		}
	}
	return int(scores[0])
}

func drawPlot() {
	dimmerFile, err := os.Open(dataPath)
	if err != nil {
		log.Fatal(err)
	}
	defer dimmerFile.Close()
	r := csv.NewReader(bufio.NewReader(dimmerFile))
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

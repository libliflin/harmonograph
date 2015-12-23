// Harmonograph generates GIF animations of random Harmonograph immitation figures.
// run https://github.com/liudng/dogo to have a watcher restart the app on change.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
)

var palette = []color.Color{color.White, color.Black}

const (
	whiteIndex = 0
	blackIndex = 1
)

func main() {
	http.HandleFunc("/harmonograph", servePage("harmonograph.html"))
	http.HandleFunc("/png", serveHarmonograph)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func servePage(filename string) func(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("servePage: %v\n", err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(data)
	}
}

func getFloat(r *http.Request, arg string) float64 {
	f, err := strconv.ParseFloat(r.URL.Query().Get(arg), 64)
	if err != nil {
		return 0.0
	}
	return f
}

func serveHarmonograph(w http.ResponseWriter, r *http.Request) {
	X1 := dampedPendulum(getFloat(r, "X1f"), getFloat(r, "X1p"), getFloat(r, "X1A"), getFloat(r, "X1d"))
	X2 := dampedPendulum(getFloat(r, "X2f"), getFloat(r, "X2p"), getFloat(r, "X2A"), getFloat(r, "X2d"))
	Y1 := dampedPendulum(getFloat(r, "Y1f"), getFloat(r, "Y1p"), getFloat(r, "Y1A"), getFloat(r, "Y1d"))
	Y2 := dampedPendulum(getFloat(r, "Y2f"), getFloat(r, "Y2p"), getFloat(r, "Y2A"), getFloat(r, "Y2d"))
	tmax := getFloat(r, "tmax")
	w.Header().Set("Content-Type", "image/png")
	harmonograph(X1, X2, Y1, Y2, tmax, w)
}

func harmonograph(X1, X2, Y1, Y2 func(float64) float64, tmax float64, out io.Writer) {
	const (
		res       = 0.001 // angular resolution
		size      = 300   // image canvas covers [-size..+size]
		alphaStep = 50
	)
	rect := image.Rect(0, 0, 2*size+1, 2*size+1)
	img := image.NewRGBA(rect)
	for alx := 0; alx < 2*size+1; alx++ {
		for aly := 0; aly < 2*size+1; aly++ {
			img.SetRGBA(alx, aly, color.RGBA{0, 0, 0, 0})
		}
	}
	printx := 0
	printy := 0
	maxPrint := 10
	xprev := math.Inf(-1)
	yprev := math.Inf(-1)

	for t := 0.0; t < tmax; t += res {
		x := size + X1(t) + X2(t)
		if printx < maxPrint {
			fmt.Printf("X: %#v\n", x)
			printx++
		}
		if x > 2*size+1 {
			xprev = math.Inf(-1)
			yprev = math.Inf(-1)
			continue
		}
		if x < 0 {
			xprev = math.Inf(-1)
			yprev = math.Inf(-1)
			continue
		}
		y := size + Y1(t) + Y2(t)
		if printy < maxPrint {
			fmt.Printf("Y: %#v\n", y)
			printy++
		}
		if y > 2*size+1 {
			xprev = math.Inf(-1)
			yprev = math.Inf(-1)
			continue
		}
		if y < 0 {
			xprev = math.Inf(-1)
			yprev = math.Inf(-1)
			continue
		}
		if xprev != math.Inf(-1) {
			xiolin_wu_draw_line(xprev, yprev, x, y, img)
		}
		xprev = x
		yprev = y
	}
	png.Encode(out, img) // NOTE: ignoring encoding errors
}

func importance(x1, y1, x2, y2 float64) float64 {
	return 1 - math.Sqrt((x1-x2)*(x1-x2)+(y1-y2)*(y1-y2))
}

func dampedPendulum(frequency, phase, amplitude, damping float64) func(time float64) float64 {
	return func(time float64) float64 {
		f := frequency
		p := phase
		A := amplitude
		d := damping
		t := time
		return A * math.Sin(t*f+p) * math.Exp(-1*d*t)
	}
}

//https://en.wikipedia.org/wiki/Xiaolin_Wu%27s_line_algorithm
func xiolin_wu_draw_line(x0, y0, x1, y1 float64, img *image.RGBA) {
	steep := math.Abs(y1-y0) > math.Abs(x1-x0)
	if steep {
		// swap
		x0, y0 = y0, x0
		x1, y1 = y1, x1
	}
	if x0 > x1 {
		x0, x1 = x1, x0
		y0, y1 = y1, y0
	}
	dx := x1 - x0
	dy := y1 - y0
	gradient := dy / dx

	// handle first endpoint
	xend := xiolin_round(x0)
	yend := y0 + gradient*(xend-x0)
	xgap := xiolin_rfpart(x0 + 0.5)
	xpxl1 := xend // this will be used in the main loop
	ypxl1 := xiolin_ipart(yend)
	if steep {
		xiolin_plot(ypxl1, xpxl1, xiolin_rfpart(yend)*xgap, img)
		xiolin_plot(ypxl1+1, xpxl1, xiolin_fpart(yend)*xgap, img)
	} else {
		xiolin_plot(xpxl1, ypxl1, xiolin_rfpart(yend)*xgap, img)
		xiolin_plot(xpxl1, ypxl1+1, xiolin_fpart(yend)*xgap, img)
	}
	intery := yend + gradient // first y-intersection for the main loop

	// handle second endpoint
	xend = xiolin_round(x1)
	yend = y1 + gradient*(xend-x1)
	xgap = xiolin_fpart(x1 + 0.5)
	xpxl2 := xend //this will be used in the main loop
	ypxl2 := xiolin_ipart(yend)
	if steep {
		xiolin_plot(ypxl2, xpxl2, xiolin_rfpart(yend)*xgap, img)
		xiolin_plot(ypxl2+1, xpxl2, xiolin_fpart(yend)*xgap, img)
	} else {
		xiolin_plot(xpxl2, ypxl2, xiolin_rfpart(yend)*xgap, img)
		xiolin_plot(xpxl2, ypxl2+1, xiolin_fpart(yend)*xgap, img)
	}

	// main loop
	for x := xpxl1 + 1.0; x <= xpxl2-1; x++ {
		if steep {
			xiolin_plot(xiolin_ipart(intery), x, xiolin_rfpart(intery), img)
			xiolin_plot(xiolin_ipart(intery)+1, x, xiolin_fpart(intery), img)
		} else {
			xiolin_plot(x, xiolin_ipart(intery), xiolin_rfpart(intery), img)
			xiolin_plot(x, xiolin_ipart(intery)+1, xiolin_fpart(intery), img)
		}
		intery = intery + gradient
	}
}

func xiolin_plot(x, y, c float64, img *image.RGBA) {
	al := saturating_add(float64(img.RGBAAt(int(x), int(y)).A), 255.0*c)
	img.SetRGBA(int(x), int(y), color.RGBA{al, 0, 0, al})
}

func saturating_add(x, y float64) uint8 {
	if x+y > 255 {
		return 255
	}
	return uint8(x + y)
}

func xiolin_ipart(x float64) float64 {
	if x >= 0.0 {
		return math.Floor(x)
	}
	return math.Ceil(x)
}

func xiolin_round(x float64) float64 {
	return xiolin_ipart(x + 0.5)
}

func xiolin_fpart(x float64) float64 {
	if x < 0.0 {
		return 1.0 - (x - math.Floor(x))
	}
	return x - math.Floor(x)
}

func xiolin_rfpart(x float64) float64 {
	return 1 - xiolin_fpart(x)
}

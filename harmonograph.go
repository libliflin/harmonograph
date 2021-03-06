// Harmonograph generates GIF animations of random Harmonograph immitation figures.
// run https://github.com/liudng/dogo to have a watcher restart the app on change.
package main

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
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
	parametricPlot(
		func(t float64) float64 {
			return X1(t) + X2(t)
		},
		func(t float64) float64 {
			return Y1(t) + Y2(t)
		},
		0.0,
		tmax,
		out)
}
func parametricPlot(X, Y func(float64) float64, tmin, tmax float64, out io.Writer) {
	parallel_parametricPlot_30(X, Y, tmin, tmax, out)
}

func makeImage(size int) *image.RGBA {
	rect := image.Rect(0, 0, 2*size+1, 2*size+1)
	img := image.NewRGBA(rect)
	// for alx := 0; alx < 2*size+1; alx++ {
	// 	for aly := 0; aly < 2*size+1; aly++ {
	// 		img.SetRGBA(alx, aly, color.RGBA{0, 0, 0, 0})
	// 	}
	// }
	return img
}

func serial_parametricPlot_static(X, Y func(float64) float64, tmin, tmax float64, out io.Writer) {
	const (
		size      = 300 // image canvas covers [-size..+size]
		alphaStep = 50
	)

	rectSize := 2*size + 1.0
	img := makeImage(size)
	prevOutside := true
	xprev := math.Inf(-1)
	yprev := math.Inf(-1)
	resDefault := 0.01
	res := resDefault
	for t := tmin; t < tmax; t += res {
		x := size + X(t)
		y := size + Y(t)
		if x < 0 || rectSize < x || y < 0 || rectSize < y {
			prevOutside = true
		}
		if !prevOutside {
			xiolin_wu_draw_line(xprev, yprev, x, y, img)
		}
		xprev = x
		yprev = y
		prevOutside = false
	}
	png.Encode(out, img) // NOTE: ignoring encoding errors
}

func parallel_parametricPlot_30(X, Y func(float64) float64, tmin, tmax float64, out io.Writer) {
	const (
		routines  = 30
		size      = 300 // image canvas covers [-size..+size]
		alphaStep = 50
	)
	img := makeImage(size)

	var wg sync.WaitGroup
	tstep := (tmax - tmin) / float64(routines)
	wg.Add(routines)
	for i := 0; i < routines; i++ {
		lm := tmin + float64(i)*tstep
		lx := tmin + float64(i+1)*tstep
		go func() {
			defer wg.Done()
			serial_parametricPlot_write_image(X, Y, lm, lx, img)
		}()
	}
	wg.Wait()
	png.Encode(out, img) // NOTE: ignoring encoding errors
}

func serial_parametricPlot(X, Y func(float64) float64, tmin, tmax float64, out io.Writer) {
	const (
		size      = 300 // image canvas covers [-size..+size]
		alphaStep = 50
	)

	img := makeImage(size)
	serial_parametricPlot_write_image(X, Y, tmin, tmax, img)
	png.Encode(out, img) // NOTE: ignoring encoding errors
}

func serial_parametricPlot_write_image(X, Y func(float64) float64, tmin, tmax float64, img *image.RGBA) {
	const (
		size      = 300 // image canvas covers [-size..+size]
		alphaStep = 50
	)

	rectSize := 2*size + 1.0
	prevOutside := true
	xprev := math.Inf(-1)
	yprev := math.Inf(-1)
	resDefault := 0.01
	allowedDifferenceMax := 2.8        // if two points are farther than allowedDifferenceMax apart, back up with a halved res
	allowedDifferenceMin := 1.4        // if two points sare closer than allowedDifferenceMin apart, back up with a doubled res
	allowedDifferenceMultiplier := 2.0 // if we are in a loop of allowed difference, do the max, then continue.
	allowedDifferenceMaxSet := false
	allowedDifferenceMinSet := false
	res := resDefault
	for t := tmin; t < tmax; t += res {
		x := size + X(t)
		y := size + Y(t)
		if x < 0 || rectSize < x || y < 0 || rectSize < y {
			prevOutside = true
		}
		if !prevOutside {
			// TODO: MAKE THIS DERIVATIVE BASED!!!!
			d := distance(xprev, yprev, x, y)
			if allowedDifferenceMax < d && allowedDifferenceMinSet {
				//allowedDifferenceMinSet = false
				//allowedDifferenceMultiplier += .1
			} else if d < allowedDifferenceMin && allowedDifferenceMaxSet {
				// do nothing..... we are in a loop.
				// maybe we could possibly muck with the multiplier
				//allowedDifferenceMaxSet = false
				//allowedDifferenceMultiplier -= .1
			} else {
				if allowedDifferenceMax < d {
					// back up
					t -= res
					// half res
					res *= (1.0 / allowedDifferenceMultiplier)
					//
					allowedDifferenceMaxSet = true
					continue
				} else {
					allowedDifferenceMaxSet = false
				}
				if d < allowedDifferenceMin {
					// back up
					t -= res
					// double res
					res *= allowedDifferenceMultiplier
					allowedDifferenceMaxSet = true
					continue
				} else {
					allowedDifferenceMinSet = false
				}
			}
			xiolin_wu_draw_line(xprev, yprev, x, y, img)
		}
		xprev = x
		yprev = y
		prevOutside = false
	}
}

func distance(x0, y0, x1, y1 float64) float64 {
	return math.Max(math.Abs(x1-x0), math.Abs(y1-y0))
	//return math.Sqrt((x1-x0)*(x1-x0) + (y1-y0)*(y1-y0))
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

func stack_overflow_parametricPlot(X, Y func(float64) float64, tmin, tmax float64, out io.Writer) {
	const (
		size      = 300 // image canvas covers [-size..+size]
		alphaStep = 50
	)

	rectSize := 2*size + 1.0
	img := makeImage(size)
	var wg sync.WaitGroup
	pp := ParametricPlot{X, Y, tmin, tmax, img, size, rectSize, wg}
	wg.Add(1)
	pp.par_recur_parametric_plot(tmin, tmax, pp.eX(tmin), pp.eY(tmin), pp.eX(tmax), pp.eY(tmax))
	wg.Wait()
	png.Encode(out, img) // NOTE: ignoring encoding errors
}

type ParametricPlot struct {
	X, Y       func(float64) float64
	tmin, tmax float64
	img        *image.RGBA
	size       float64
	rectSize   float64
	wg         sync.WaitGroup
}

func (pp *ParametricPlot) eX(t float64) float64 {
	return pp.size + pp.X(t)
}

func (pp *ParametricPlot) eY(t float64) float64 {
	return pp.size + pp.Y(t)
}

const (
	allowedDifferenceMax     = 3.0
	allowedTimeDifferenceMax = 0.1
)

// lx >= lm
func (pp *ParametricPlot) par_recur_parametric_plot(
	lm,
	lx,
	x0,
	y0,
	x1,
	y1 float64) {
	defer pp.wg.Done()
	recurse := false
	if (lx - lm) > allowedTimeDifferenceMax {
		// too much time, recurse to lower time
		recurse = true
	}
	if math.Abs(x1-x0) > allowedDifferenceMax {
		// too much X diff, recurse
		recurse = true
	}
	if math.Abs(y1-y0) > allowedDifferenceMax {
		// too much Y diff, recurse
		recurse = true
	}
	if recurse {
		bt := lm + ((lx - lm) / 2.0)
		bx := pp.eX(bt)
		by := pp.eY(bt)
		pp.wg.Add(2)
		pp.par_recur_parametric_plot(lm, bt, x0, x1, bx, by)
		pp.par_recur_parametric_plot(bt, lx, bx, by, x1, y1)
		return
	} else {
		// make sure we are in bounds.
		if 0.0 <= x0 && x0 < pp.rectSize &&
			0.0 <= y0 && y0 < pp.rectSize &&
			0.0 <= x1 && x1 < pp.rectSize &&
			0.0 <= y1 && y1 < pp.rectSize {
			xiolin_wu_draw_line(x0, y0, x1, y1, pp.img)

		}
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
	mahAlpha := 1.0
	ex := float64(img.RGBAAt(int(x), int(y)).A)
	ne := mahAlpha * 255.0 * c
	//al := saturating_add(ex, ne)
	//al := uint8(ne)
	//al := add_little_to_max(ex, ne)
	al := get_max(ex, ne)

	img.SetRGBA(int(x), int(y), color.RGBA{al, 0, 0, al})
}

func add_little_to_max(prev, now float64) uint8 {
	return saturating_add(math.Max(prev, now), 0.01*math.Min(prev, now))
}

func get_max(prev, now float64) uint8 {
	return uint8(math.Max(prev, now))
}

func saturating_add(prev, now float64) uint8 {
	if prev+now > 255 {
		return 255
	}
	return uint8(prev + now)
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

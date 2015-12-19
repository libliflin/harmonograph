// Harmonograph generates GIF animations of random Harmonograph immitation figures.
// this file started as https://github.com/adonovan/gopl.io/blob/master/ch1/lissajous/main.go
package main

import (
	"image"
	"image/color"
	"image/gif"
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
	http.HandleFunc("/harmonograph.html", servePage("harmonograph.html"))
	http.HandleFunc("/gif", serveHarmonograph)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func servePage(filename string) func(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("servePage: %v\n", err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
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

	harmonograph(X1, X2, Y1, Y2, tmax, w)
}

func harmonograph(X1, X2, Y1, Y2 func(float64) float64, tmax float64, out io.Writer) {
	const (
		cycles  = 4     // number of complete x oscillator revolutions
		res     = 0.001 // angular resolution
		size    = 100   // image canvas covers [-size..+size]
		nframes = 64    // number of animation frames
		delay   = 10    // delay between frames in 10ms units
	)
	// freq := rand.Float64() * 3.0 // relative frequence of y oscillator
	anim := gif.GIF{LoopCount: nframes}
	// phase := 0.0 // phase difference
	for i := 0; i < nframes; i++ {
		rect := image.Rect(0, 0, 2*size+1, 2*size+1)
		img := image.NewPaletted(rect, palette)

		for t := 0.0; t < tmax; t += res {
			x := X1(t) + X2(t)
			y := Y1(t) + Y2(t)
			img.SetColorIndex(int(x), int(y), blackIndex)
		}
		// phase += 0.1
		anim.Delay = append(anim.Delay, delay)
		anim.Image = append(anim.Image, img)
	}
	gif.EncodeAll(out, &anim) // NOTE: ignoring encoding errors
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

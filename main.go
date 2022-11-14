package main

import (
	"flag"
	"fmt"
	"math"
	"time"
)

const twopi = 2 * math.Pi

type Shape int

const (
	SINE Shape = iota
	SQUARE
	DOWNWARD_SAWTOOTH
	UPWARD_SAWTOOTH
	TRIANGLE
)

var (
	shapeCalcFunc = map[Shape]func(float64) float64{
		SINE:              sineCalc,
		SQUARE:            squareCalc,
		TRIANGLE:          triangleCalc,
		DOWNWARD_SAWTOOTH: downSawtoothCalc,
		UPWARD_SAWTOOTH:   upwSawtoothCalc,
	}
)

type Oscillator struct {
	curfreq  float64
	curphase float64
	incr     float64
	twopiosr float64 // (2*PI) / samplerate
	tickfunc func(float64) float64
}

// NewOscillator set to a given sample rate
func NewOscillator(sr int, shape Shape) (*Oscillator, error) {
	cf, ok := shapeCalcFunc[shape]
	if !ok {
		return nil, fmt.Errorf("Shape type %v not supported", shape)
	}
	return &Oscillator{
		twopiosr: twopi / float64(sr), // (2 * PI) / SampleRate
		tickfunc: cf,
	}, nil
}

func (o *Oscillator) Tick(freq float64) float64 {
	if o.curfreq != freq {
		o.curfreq = freq
		o.incr = o.twopiosr * freq
	}
	val := o.tickfunc(o.curphase)
	o.curphase += o.incr

	// adjust bounds
	if o.curphase >= twopi {
		o.curphase -= twopi
	}
	if o.curphase < 0 {
		o.curphase = twopi
	}
	return val

}

func sineCalc(phase float64) float64 {
	return math.Sin(phase)
}

func squareCalc(phase float64) float64 {
	val := -1.0
	if phase <= math.Pi {
		val = 1.0
	}
	return val
}

func triangleCalc(phase float64) float64 {
	val := 2.0*(phase*(1.0/twopi)) - 1.0
	if val < 0.0 {
		val = -val
	}
	val = 2.0 * (val - 0.5)
	return val
}

func upwSawtoothCalc(phase float64) float64 {
	val := 2.0*(phase*(1.0/twopi)) - 1.0
	return val
}

func downSawtoothCalc(phase float64) float64 {
	val := 1.0 - 2.0*(phase*(1.0/twopi))
	return val
}

var (
	shape = flag.String("s", "sine", "One of: sine, square, triangle, downsaw, upsaw")
)

var stringToShape = map[string]Shape{
	"sine":     0,
	"square":   1,
	"downsaw":  2,
	"upsaw":    3,
	"triangle": 4,
}

func main() {
	flag.Parse()

	// A, D, S, R := 0.1, 0.2, 0.5, 1
	sampleRate := 44100
	bufferSize := 512
	// bitsPerSample := 16

	osc, err := NewOscillator(sampleRate, stringToShape[*shape])
	if err != nil {
		panic(err)
	}

	speaker := newSpeaker(sampleRate, bufferSize)

	var freq float64 = 100

	for {
		value := osc.Tick(freq)
		speaker.Write([2]float64{value, value})
		time.Sleep(time.Duration(1 / sampleRate))
	}
}

// func generate(dur int, shape Shape, ampStream, freqStream *breakpoint.BreakpointStream, sampleRate int) [][2]float64 {
// 	numSamples := dur * sampleRate
// 	osc, err := NewOscillator(sampleRate, shape)

// 	if err != nil {
// 		panic(err)
// 	}

// 	var samples = make([][2]float64, numSamples)

// 	for i := range samples {
// 		amp := ampStream.Tick()
// 		freq := freqStream.Tick()
// 		value := amp * osc.Tick(freq)
// 		samples[i] = [2]float64{value, value}
// 	}

// 	return samples
// }

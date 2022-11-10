package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"math"

	"github.com/DylanMeeus/GoAudio/breakpoint"
	synth "github.com/DylanMeeus/GoAudio/synthesizer"
	"github.com/DylanMeeus/GoAudio/wave"
)

const tau = 2 * math.Pi

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
		twopiosr: tau / float64(sr), // (2 * PI) / SampleRate
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
	if o.curphase >= tau {
		o.curphase -= tau
	}
	if o.curphase < 0 {
		o.curphase = tau
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
	val := 2.0*(phase*(1.0/tau)) - 1.0
	if val < 0.0 {
		val = -val
	}
	val = 2.0 * (val - 0.5)
	return val
}

func upwSawtoothCalc(phase float64) float64 {
	val := 2.0*(phase*(1.0/tau)) - 1.0
	return val
}

func downSawtoothCalc(phase float64) float64 {
	val := 1.0 - 2.0*(phase*(1.0/tau))
	return val
}

var (
	duration   = flag.Int("d", 10, "duration of signal")
	shape      = flag.String("s", "sine", "One of: sine, square, triangle, downsaw, upsaw")
	amppoints  = flag.String("a", "", "amplitude breakpoints file")
	freqpoints = flag.String("f", "", "frequency breakpoints file")
	output     = flag.String("o", "", "output file")
)

var stringToShape = map[string]synth.Shape{
	"sine":     0,
	"square":   1,
	"downsaw":  2,
	"upsaw":    3,
	"triangle": 4,
}

func main() {
	flag.Parse()
	fmt.Println("usage: go run . -d {dur} -s {shape} -a {amps} -f {freqs} -o {output}")
	if output == nil {
		panic("please provide an output file")
	}

	wfmt := wave.NewWaveFmt(1, 1, 44100, 16, nil)
	amps, err := ioutil.ReadFile(*amppoints)
	if err != nil {
		panic(err)
	}
	ampPoints, err := breakpoint.ParseBreakpoints(bytes.NewReader(amps))
	if err != nil {
		panic(err)
	}
	ampStream, err := breakpoint.NewBreakpointStream(ampPoints, wfmt.SampleRate)

	freqs, err := ioutil.ReadFile(*freqpoints)
	if err != nil {
		panic(err)
	}
	freqPoints, err := breakpoint.ParseBreakpoints(bytes.NewReader(freqs))
	if err != nil {
		panic(err)
	}
	freqStream, err := breakpoint.NewBreakpointStream(freqPoints, wfmt.SampleRate)
	if err != nil {
		panic(err)
	}
	// create wave file sampled at 44.1Khz w/ 16-bit frames

	frames := generate(*duration, stringToShape[*shape], ampStream, freqStream, wfmt)
	wave.WriteFrames(frames, wfmt, *output)
	fmt.Println("done")
}

func generate(dur int, shape synth.Shape, ampStream, freqStream *breakpoint.BreakpointStream, wfmt wave.WaveFmt) []wave.Frame {
	reqFrames := dur * wfmt.SampleRate
	frames := make([]wave.Frame, reqFrames)
	osc, err := synth.NewOscillator(wfmt.SampleRate, shape)
	if err != nil {
		panic(err)
	}

	for i := range frames {
		amp := ampStream.Tick()
		freq := freqStream.Tick()
		frames[i] = wave.Frame(amp * osc.Tick(freq))
	}

	return frames
}

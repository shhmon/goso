package main

import (
	"math"
	"sync"

	"github.com/hajimehoshi/oto"
)

var (
	mu      sync.Mutex
	samples [][2]float64
	buffer  []byte
	context *oto.Context
	player  *oto.Player
	done    chan struct{}
)

func Init(sampleRate int, bufferSize int) error {
	mu.Lock()
	defer mu.Unlock()

	Close()

	numBytes := bufferSize * 4
	samples = make([][2]float64, bufferSize)
	buffer = make([]byte, numBytes)

	context, err := oto.NewContext(int(sampleRate), 2, 2, numBytes)

	if err != nil {
		panic(err)
	}

	player = context.NewPlayer()

	done = make(chan struct{})

	go func() {
		for {
			select {
			default:
				update()
			case <-done:
				return
			}
		}
	}()

	return nil
}

func Close() {
	if player != nil {
		if done != nil {
			done <- struct{}{}
			done = nil
		}
		player.Close()
		context.Close()
		player = nil
	}
}

func update() {
	mu.Lock()
	defer mu.Unlock()

	// Write samples to buffer
	for i := range samples {
		for c := range samples[i] {
			val := samples[i][c]
			val = math.Max(val, -1)
			val = math.Min(val, +1)

			valInt16 := int16(val * (1<<15 - 1))
			low := byte(valInt16)
			high := byte(valInt16 >> 8)
			buffer[i*4+c*2+0] = low
			buffer[i*4+c*2+1] = high
		}
	}

	// Clear the samples
	for i := range samples {
		samples[i] = [2]float64{}
	}

	player.Write(buffer)
}

func Write(newSamples [][2]float64) {
	mu.Lock()
	defer mu.Unlock()

	samples = append(samples, newSamples...)
}

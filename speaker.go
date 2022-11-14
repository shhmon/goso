package main

import (
	"math"
	"sync"

	"github.com/hajimehoshi/oto"
)

var (
	mu sync.Mutex
)

type Speaker struct {
	mu         *sync.Mutex
	sampleRate int
	bufferSize int
	samples    [][2]float64
	buffer     []byte
	done       chan struct{}
	context    *oto.Context
	player     *oto.Player
}

func newSpeaker(sampleRate int, bufferSize int) *Speaker {
	numBytes := bufferSize * 4

	speaker := Speaker{
		mu:         &mu,
		sampleRate: sampleRate,
		bufferSize: bufferSize,
		samples:    make([][2]float64, 0),
		buffer:     make([]byte, numBytes, numBytes),
		done:       make(chan struct{}),
	}

	context, err := oto.NewContext(sampleRate, 2, 2, numBytes)
	if err != nil {
		panic(err)
	}

	speaker.context = context
	speaker.player = context.NewPlayer()

	go func() {
		for {
			select {
			default:
				update(&speaker)
			case <-speaker.done:
				return
			}
		}
	}()

	return &speaker
}

func (s *Speaker) Close() {
	if s.player != nil {
		if s.done != nil {
			s.done <- struct{}{}
			s.done = nil
		}
		s.player.Close()
		s.context.Close()
		s.player = nil
	}
}

func (s *Speaker) Write(sample [2]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.samples = append(s.samples, sample)
}

func update(s *Speaker) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var processed int

	// Write samples to buffer
write:
	for i := range s.samples {
		processed++

		for c := range s.samples[i] {
			val := s.samples[i][c]
			val = math.Max(val, -1)
			val = math.Min(val, +1)

			valInt16 := int16(val * (1<<15 - 1))
			low := byte(valInt16)
			high := byte(valInt16 >> 8)

			lowId := i*4 + c*2 + 0
			highId := i*4 + c*2 + 1

			if highId > cap(s.buffer) {
				break write
			}

			s.buffer[lowId] = low
			s.buffer[highId] = high
		}
	}

	if processed > 0 {
		s.samples = s.samples[processed-1:]
	}

	s.player.Write(s.buffer)
}

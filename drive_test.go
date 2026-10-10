package main

import (
	"math"
	"testing"
)

func TestJourneyAcrossCadences(t *testing.T) {
	var baseline Journey
	for _, hz := range []int{60, 120, 144} {
		j := newJourney(42)
		for i := 0; i < hz*20; i++ {
			j.step(1/float64(hz), 0, false, false)
		}
		if hz == 60 {
			baseline = j
		} else if math.Abs(j.Kilometers-baseline.Kilometers) > .000001 {
			t.Fatalf("%d Hz distance drift: %.9f vs %.9f", hz, j.Kilometers, baseline.Kilometers)
		}
	}
}
func TestEndlessBoundaries(t *testing.T) {
	j := newJourney(99)
	j.Speed = 50
	j.Region = 1 << 40
	j.Offset = regionLength - .01
	j.step(.05, 1, false, false)
	if j.Region != (1<<40)+1 || j.Offset < 0 || j.Offset >= regionLength {
		t.Fatalf("region rollover: %+v", j)
	}
	for i := 0; i < 200000; i++ {
		j.step(.05, 1, i%3 == 0, false)
	}
	if !finite(j.X) || math.Abs(j.X) > 1.45 || j.Offset < 0 || j.Offset >= regionLength {
		t.Fatalf("unbounded state: %+v", j)
	}
	if j.Visited == 0 {
		t.Fatal("discoveries not recorded")
	}
}
func TestRouteContinuityAndVariety(t *testing.T) {
	j := newJourney(17)
	for knot := 0; knot < 10000; knot++ {
		a := j.curve(float64(knot)*100 - .0001)
		b := j.curve(float64(knot)*100 + .0001)
		if math.Abs(a-b) > .00001 {
			t.Fatalf("curve discontinuity at %d: %f", knot, a-b)
		}
	}
	seen := map[int]bool{}
	for region := uint64(0); region < uint64(len(biomes)); region++ {
		seen[biomeFor(j.Seed, region)] = true
	}
	if len(seen) != len(biomes) {
		t.Fatalf("only %d biomes", len(seen))
	}
}
func TestInterruptionCannotSkipTrip(t *testing.T) {
	a, b := newJourney(3), newJourney(3)
	a.step(2000, 0, false, false)
	b.step(.05, 0, false, false)
	if a.Kilometers != b.Kilometers {
		t.Fatal("long pause advanced the journey")
	}
}

func TestCornersAffectHandlingAndBrakeStops(t *testing.T) {
	j := newJourney(42)
	j.Speed = 60
	for i := 0; i < 120*8; i++ {
		j.step(1.0/120, 0, false, false)
	}
	if math.Abs(j.X) < .05 {
		t.Fatal("curving road has no effect on the car")
	}
	for i := 0; i < 120*6; i++ {
		j.step(1.0/120, 0, false, true)
	}
	if j.Speed > .1 {
		t.Fatalf("brake cannot stop the car: %.2f m/s", j.Speed)
	}
}

func TestSprintEnergyAndOffRoadCost(t *testing.T) {
	a, b := Sprint{}, Sprint{}
	for i := 0; i < 120*5; i++ {
		a.advance(.3, 1.0/120, false, false)
		b.advance(.3, 1.0/120, true, false)
	}
	if a.Nitro >= .3 || b.Nitro < .3 {
		t.Fatalf("drafting should earn a boost sooner: %.3f %.3f", a.Nitro, b.Nitro)
	}
	for i := 0; i < 10000; i++ {
		b.advance(1, .05, true, true)
	}
	if b.Nitro != 1 {
		t.Fatal("boost capacity not bounded")
	}
	road, verge := newJourney(42), newJourney(42)
	for i := 0; i < 120*5; i++ {
		road.X = 0
		verge.X = 1.3
		road.drive(1.0/120, Controls{Throttle: 1})
		verge.drive(1.0/120, Controls{Throttle: 1})
	}
	if verge.Speed >= road.Speed*.8 {
		t.Fatal("leaving the track must cost meaningful momentum")
	}
}

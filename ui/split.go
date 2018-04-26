package ui

import "math"

// Split simplifies the layout definition.
type Split struct {
	size   int
	left   int
	points []int
	index  int
}

// NewSplit returns a new Split
func NewSplit(size int) *Split {
	return &Split{size: size, left: size, points: []int{0}}
}

// Fixed defines a set of fixed or absolute points
func (s *Split) Fixed(points ...int) *Split {
	for _, point := range points {
		s.points = append(s.points, point+(s.size-s.left))
		s.left = s.left - point
	}

	return s
}

// Relative defines a set of relative points
func (s *Split) Relative(points ...int) *Split {
	for _, point := range points {
		per := float64(point) / 100.0
		rel := math.Floor(0.5 + float64(s.left)*per)
		s.Fixed(int(rel))
	}
	return s
}

// Next returns the next point in the set
func (s *Split) Next() int {
	if s.index+1 == len(s.points) {
		return 0
	}

	s.index = s.index + 1
	next := s.points[s.index]
	return next
}

// Current returns the current point in the set
func (s *Split) Current() int {
	return s.points[s.index]
}

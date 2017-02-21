package main

import "math"

type Split struct {
	size   int
	left   int
	points []int
	index  int
}

func NewSplit(size int) *Split {
	return &Split{size: size, left: size, points: []int{0}}
}

func (s *Split) Fixed(points ...int) *Split {
	for _, point := range points {
		s.points = append(s.points, point+(s.size-s.left))
		s.left = s.left - point
	}

	return s
}

func (s *Split) Relative(points ...int) *Split {
	for _, point := range points {
		per := float64(point) / 100.0
		rel := math.Floor(0.5 + float64(s.left)*per)
		s.Fixed(int(rel))
	}
	return s
}

func (s *Split) Next() int {
	if s.index+1 == len(s.points) {
		return 0
	}

	s.index += 1
	next := s.points[s.index]
	return next
}

func (s *Split) Current() int {
	return s.points[s.index]
}

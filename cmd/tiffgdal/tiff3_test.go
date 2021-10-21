package main

import (
	"testing"
)

func TestAThing(t *testing.T) {
	pb := &PixelBuffer{
		buf:    []float32{1, 1, 1, 1, 1, 1.5, 1.2, 1, 1, 1, 1, 1},
		width:  4,
		height: 3,
	}

	solid := pb.toSTL(0, 0, 3, 2)

	t.Logf("%d triangles", len(solid.Triangles))

	for _, e := range solid.Triangles {
		t.Log(e)
	}
	solid.WriteFile("test.stl")
}

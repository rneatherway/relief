package main

import (
	"testing"
)

func TestAThing(t *testing.T) {
	pb := &PixelBuffer{
		buf:    []float32{1, 1.1, 1.2, 0.9, 1.4, 2.5, 1.9, 1.1, 0.3, 0.4, 0.8, 0.6},
		width:  4,
		height: 3,
	}

	solid := pb.toSTL()

	t.Logf("%d triangles", len(solid.Triangles))

	for _, e := range solid.Triangles {
		t.Log(e)
	}
	solid.WriteFile("test.stl")
}

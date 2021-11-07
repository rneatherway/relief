package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/airbusgeo/godal"
	"github.com/hschendel/stl"
)

type PixelBuffer struct {
	buf    []float32
	width  uint
	height uint

	minMaxComputed bool
	min            float32
	max            float32
}

func (pb *PixelBuffer) minMax() {
	pb.min = pb.buf[0]
	pb.max = pb.buf[0]
	for _, p := range pb.buf {
		if p < pb.min {
			pb.min = p
		}

		if p > pb.max {
			pb.max = p
		}
	}
	pb.minMaxComputed = true
}

func (pb *PixelBuffer) Min() float32 {
	if !pb.minMaxComputed {
		pb.minMax()
	}
	return pb.min
}

func (pb *PixelBuffer) Max() float32 {
	if !pb.minMaxComputed {
		pb.minMax()
	}
	return pb.max
}

// Imagine width is three, height is two and pixel data is:
//
// a b c
// d e f
//
// This will be in the buf as: a b c d e f
// So for each 'y' we need to advance by 'width'
// x + y*width
func (pb *PixelBuffer) get(x uint, y uint) float32 {
	return pb.buf[x+y*pb.width]
}

func (pb *PixelBuffer) getVec3(x uint, y uint) stl.Vec3 {
	return stl.Vec3{
		float32(x),
		float32(y),
		pb.get(x, pb.height-y-1), // STL coordinate system is inverted, so flip y
	}
}

func AppendRectangle(solid *stl.Solid, tl, tr, bl, br stl.Vec3) {
	solid.AppendTriangle(stl.Triangle{Vertices: [3]stl.Vec3{tl, bl, tr}})
	solid.AppendTriangle(stl.Triangle{Vertices: [3]stl.Vec3{tr, bl, br}})
}

func (pb *PixelBuffer) toSTL() *stl.Solid {
	solid := &stl.Solid{}

	// The code below supports only converting a window, but
	// I moved that to the GeoTIFF parsing.
	x1 := uint(0)
	y1 := uint(0)
	x2 := pb.width - 1
	y2 := pb.height - 1

	// Top
	for i := x1; i < x2; i++ {
		for j := y1; j < y2; j++ {
			AppendRectangle(
				solid,
				pb.getVec3(i, j),
				pb.getVec3(i+1, j),
				pb.getVec3(i, j+1),
				pb.getVec3(i+1, j+1))
		}
	}

	for i := x1; i < x2; i++ {
		// Back
		AppendRectangle(
			solid,
			pb.getVec3(i+1, y1),
			pb.getVec3(i, y1),
			stl.Vec3{float32(i + 1), float32(y1), 0},
			stl.Vec3{float32(i), float32(y1), 0},
		)

		// Front
		AppendRectangle(
			solid,
			pb.getVec3(i, y2),
			pb.getVec3(i+1, y2),
			stl.Vec3{float32(i), float32(y2), 0},
			stl.Vec3{float32(i + 1), float32(y2), 0},
		)
	}

	for j := y1; j < y2; j++ {
		// Left
		AppendRectangle(
			solid,
			pb.getVec3(x1, j),
			pb.getVec3(x1, j+1),
			stl.Vec3{float32(x1), float32(j), 0},
			stl.Vec3{float32(x1), float32(j + 1), 0},
		)

		// Right
		AppendRectangle(
			solid,
			pb.getVec3(x2, j+1),
			pb.getVec3(x2, j),
			stl.Vec3{float32(x2), float32(j + 1), 0},
			stl.Vec3{float32(x2), float32(j), 0},
		)
	}

	// Bottom
	AppendRectangle(
		solid,
		stl.Vec3{float32(x1), float32(y2), 0},
		stl.Vec3{float32(x2), float32(y2), 0},
		stl.Vec3{float32(x1), float32(y1), 0},
		stl.Vec3{float32(x2), float32(y1), 0},
	)

	solid.RecalculateNormals()
	solid.Validate()
	return solid
}

func (pb *PixelBuffer) ToImage() image.Image {
	r := image.Rect(0, 0, int(pb.width)-1, int(pb.height)-1)
	img := image.NewNRGBA(r)

	fmt.Printf("Rescaling image with height min %f and max %f\n", pb.Min(), pb.Max())
	for i := uint(0); i < pb.width; i++ {
		for j := uint(0); j < pb.height; j++ {
			z := pb.get(i, j)
			c := 65535 * (z - pb.Min()) / (pb.Max() - pb.Min())
			img.Set(int(i), int(j), color.Gray16{uint16(c)})
		}
	}
	return img
}

func (pb *PixelBuffer) Zero(min float32) {
	if len(pb.buf) == 0 {
		return
	}

	for i := range pb.buf {
		pb.buf[i] = pb.buf[i] - pb.Min() + min
	}
}

func (pb *PixelBuffer) Scale(s float32) {
	for i := range pb.buf {
		pb.buf[i] = pb.buf[i] * s
	}
}

func (pb *PixelBuffer) Diff(pb2 *PixelBuffer) {
	for i := range pb.buf {
		pb.buf[i] = pb.buf[i] - pb2.buf[i]
	}
}

func FromGeoTIFF(path string, x, y, w, h uint) (*PixelBuffer, error) {
	godal.RegisterAll()
	hDataset, err := godal.Open(path)
	if err != nil {
		return nil, err
	}
	defer hDataset.Close()

	structure := hDataset.Structure()

	if uint(structure.SizeX) < x+w {
		return nil, fmt.Errorf("selected window goes outside image bounds (image width=%d, window max x=%d)", structure.SizeX, x+w)
	}
	if uint(structure.SizeY) < y+h {
		return nil, fmt.Errorf("selected window goes outside image bounds (image height=%d, window max y=%d)", structure.SizeY, y+h)
	}

	if w == 0 {
		w = uint(structure.SizeX) - x
	}
	if h == 0 {
		h = uint(structure.SizeY) - y
	}

	band := hDataset.Bands()[0]
	buf := make([]float32, w*h)
	fmt.Printf("Reading GeoTIFF band %d (of %d) window (%dx%d+%dx%d) into buffer size %d...",
		1, len(hDataset.Bands()), x, y, w, h, len(buf))
	err = band.Read(int(x), int(y), buf, int(w), int(h))
	fmt.Println("done")
	if err != nil {
		return nil, err
	}

	// Set undefined to zero, for now
	for i := range buf {
		if buf[i] == -math.MaxFloat32 {
			buf[i] = 0
		}
	}

	return &PixelBuffer{buf: buf, width: w, height: h}, nil
}

func realMain() error {
	x := flag.Uint("x", 0, "x coordinate (default 0)")
	y := flag.Uint("y", 0, "y coordinate (default 0)")
	w := flag.Uint("w", 0, "width (default max)")
	h := flag.Uint("h", 0, "height (default max)")
	s := flag.Float64("s", 1, "scale vertically (default 1)")
	zero := flag.Float64("z", 10, "translate the model down so that this is the lowest height")
	diff := flag.String("d", "", "a second file to compare against")
	visualize := flag.Bool("v", false, "visualize the model")
	output := flag.String("output", "out.stl", "output STL file (default out.stl)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s [OPTIONS] <input geotiff file>:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if path.Ext(*output) != ".stl" && path.Ext(*output) != ".png" {
		return fmt.Errorf("unsupported output format")
	}

	fmt.Println(flag.NArg())
	switch flag.NArg() {
	case 0:
		flag.Usage()
		return fmt.Errorf("no input file given")
	case 1:
		// Great
	default:
		flag.Usage()
		return fmt.Errorf("unrecognised arguments %s", strings.Join(flag.Args()[1:], ", "))
	}
	input := flag.Arg(0)
	pb, err := FromGeoTIFF(input, *x, *y, *w, *h)
	if err != nil {
		return err
	}

	if *diff != "" {
		pb2, err := FromGeoTIFF(*diff, *x, *y, *w, *h)
		if err != nil {
			return err
		}
		pb.Diff(pb2)
	}

	fmt.Printf("Setting minimum height value to %f...", *zero)
	pb.Zero(float32(*zero))
	fmt.Println("done")

	if *s != 1.0 {
		fmt.Printf("Adjusting vertical scale by factor of %f...", *s)
		pb.Scale(float32(*s))
		fmt.Println("done")
	}

	switch path.Ext(*output) {
	case ".stl":
		fmt.Printf("Converting to STL file '%s'...", *output)
		err = pb.toSTL().WriteFile(*output)
		if err != nil {
			return err
		}
		fmt.Println("done")

		if *visualize {
			fmt.Println("Launching visualisation")
			return exec.Command("f3d", *output).Run()
		}
	case ".png":
		fmt.Printf("Converting to PNG file '%s'...", *output)
		img := pb.ToImage()
		f, err := os.Create(*output)
		if err != nil {
			return err
		}
		defer f.Close()
		err = png.Encode(f, img)
		if err != nil {
			return err
		}
		fmt.Println("done")
	}

	return nil
}

func main() {
	err := realMain()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path"
	"strings"
)

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

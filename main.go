package main

import (
	"log"
	"github.com/tjgq/sane"
	"image"
	"image/jpeg"
	"image/png"
	"golang.org/x/image/tiff"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"os"
)

var unitName = map[sane.Unit]string{
    sane.UnitPixel:   "pixels",
    sane.UnitBit:     "bits",
    sane.UnitMm:      "millimetres",
    sane.UnitDpi:     "dots per inch",
    sane.UnitPercent: "percent",
    sane.UnitUsec:    "microseconds",
}

type EncodeFunc func(io.Writer, image.Image) error

func pathToEncoder(path string) (EncodeFunc, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return png.Encode, nil
	case ".jpg", ".jpeg":
		return func(w io.Writer, m image.Image) error {
			return jpeg.Encode(w, m, nil)
		}, nil
	case ".tif", ".tiff":
		return func(w io.Writer, m image.Image) error {
			return tiff.Encode(w, m, nil)
		}, nil
	default:
		return nil, fmt.Errorf("unrecognized extension")
	}
}

func openDevice(name string) (*sane.Conn, error) {
	c, err1 := sane.Open(name)
	if err1 == nil {
		return c, nil
	}
	// Try a substring match over the available devices
	devs, err2 := sane.Devices()
	if err2 != nil {
		return nil, err2
	}
	for _, d := range devs {
		if strings.Contains(d.Name, name) {
			return sane.Open(d.Name)
		}
	}
	return nil, fmt.Errorf("no device named %s", name)
}

func printConstraints(o sane.Option) {
    first := true
    if o.IsAutomatic {
	print(" auto")
	first = false
    }
    if o.ConstrRange != nil {
	if first {
	    print(" %v..%v", o.ConstrRange.Min, o.ConstrRange.Max)
	} else {
	    print("|%v..%v", o.ConstrRange.Min, o.ConstrRange.Max)
	}
	if (o.Type == sane.TypeInt && o.ConstrRange.Quant != 0) ||
	    (o.Type == sane.TypeFloat && o.ConstrRange.Quant != 0.0) {
	    print(" in steps of %v", o.ConstrRange.Quant)
	}
    } else {
	for _, v := range o.ConstrSet {
	    if first {
		print(" %v", v)
		first = false
	    } else {
		print("|%v", v)
	    }
	}
    }
}

func printOption(o sane.Option, v interface{}) {

	log.Printf("-------------------------------------------------")

	// Print option name
	log.Printf("    -%s", o.Name)

	// Print constraints
	printConstraints(o)

	// Print current value
	if v != nil {
		log.Printf(" [%v]", v)
	} else {
		if !o.IsActive {
			log.Printf(" [inactive]")
		} else {
			log.Printf(" [?]")
		}
	}

	// Print unit
	if name, ok := unitName[o.Unit]; ok {
		log.Printf(" %s", name)
	}

	// Print description
	log.Printf("%s", o.Desc)
}

func showOptions(c *sane.Conn) {

	lastGroup := ""
	log.Printf("Options for device %s:\n", c.Device)
	for _, o := range c.Options() {
		if !o.IsSettable {
			continue
		}
		if o.Group != lastGroup {
			log.Printf("  %s:\n", o.Group)
			lastGroup = o.Group
		}
		v, _ := c.GetOption(o.Name)
		printOption(o, v)
	}
}

func listDevices() {
	devs, _ := sane.Devices()
	if len(devs) == 0 {
		log.Printf("No available devices.")
	}
	for _, d := range devs {
		log.Printf("Device %s is a %s %s %s", d.Name, d.Vendor, d.Model, d.Type)
		c, _ := openDevice(d.Name)
		doScan(c, "1.jpg", nil)
		c.Close()
	}

}

func doScan(c *sane.Conn, fileName string, optargs []string) {

	enc, err := pathToEncoder(fileName)
	if err != nil {
		panic(err)
	}

	stream, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			panic(err)
		}
	}()

	showOptions(c)

	var options []Option
	options = append(options, Option{
		Name: "resolution",
		Int: 600,
	})
	options = append(options, Option{
		Name: "mode",
		String: "color",
	})
	options = append(options, Option{
		Name: "preview",
		Bool: false,
	})


	if err := parseOptions(c, options); err != nil {
		panic(err)
	}

	img, err := c.ReadImage()
	if err != nil {
		panic(err)
	}

	if err := enc(stream, img); err != nil {
		panic(err)
	}

}

type Option struct {
	Name string
	Type int
	Bool bool
	Int int
	Float float64
	String string
	Auto bool
}

func findOption(opts []sane.Option, name string) (*sane.Option, error) {
    for _, o := range opts {
	if o.Name == name {
	    return &o, nil
	}
    }
    return nil, fmt.Errorf("no such option")
}

func parseOptions(c *sane.Conn, args []Option) error {

    for _, a := range args {

	o, err := findOption(c.Options(), a.Name)
	if err != nil {
		panic(err)
	}
	var v interface{}
	if o.IsAutomatic && a.Auto {
	    v = sane.Auto // set to auto value
	} else {
	    switch o.Type {
	    case sane.TypeBool:
		v = a.Bool
	    case sane.TypeInt:
		v = a.Int
	    case sane.TypeFloat:
		v = a.Float
	    case sane.TypeString:
		v = a.String
	    }
	}
	if _, err := c.SetOption(o.Name, v); err != nil {
	    return err // can't set option
	}
    }
    return nil
}



func main() {

	log.Printf("ScanServer v1.0.0")

	if err1 := sane.Init(); err1 != nil {
		panic(err1)
	}
	defer sane.Exit()

	listDevices()

}

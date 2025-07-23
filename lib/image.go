package lib

import (
	"fmt"
	"github.com/rwcarlsen/goexif/exif"
	"image"
	"os"
	"path/filepath"
)

func ApplyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 3:
		return rotate180(img)
	case 6:
		return rotate90(img)
	case 8:
		return rotate270(img)
	default:
		return img
	}
}

func rotate90(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(bounds.Max.Y-y-1, x, src.At(x, y))
		}
	}
	return dst
}

func rotate180(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(bounds.Max.X-x-1, bounds.Max.Y-y-1, src.At(x, y))
		}
	}
	return dst
}

func rotate270(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(y, bounds.Max.X-x-1, src.At(x, y))
		}
	}
	return dst
}

func getFNumber(x *exif.Exif) string {
	tag, err := x.Get(exif.FNumber)
	if err != nil {
		return "N/A"
	}
	num, denom, err := tag.Rat2(0)
	if err != nil || denom == 0 {
		return tag.String()
	}
	return fmt.Sprintf("f/%.1f", float64(num)/float64(denom))
}

func ExtractExifInfo(name string, cfg Config) interface{} {
	path := filepath.Join(cfg.DownloadDir, name)
	exifInfo := ""
	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		x, err := exif.Decode(file)
		if err == nil {
			get := func(tag exif.FieldName) string {
				v, err := x.Get(tag)
				if err != nil {
					return "N/A"
				}
				if s, err := v.StringVal(); err == nil {
					return s
				}
				return v.String()
			}
			exifInfo += fmt.Sprintf(
				"[white]Model:[blue] %s\n"+
					"[white]Firmware:[blue] %s\n"+
					"[white]Date Taken:[blue] %s\n"+
					"[white]Resolution:[blue] %sx%s\n\n"+
					"[white]ISO:[purple] %s\n"+
					"[white]Exposure:[purple] %s sec\n"+
					"[white]Aperture:[purple] %s\n"+
					"[white]Focal Length:[purple] %s mm\n"+
					"[white]Exposure Program:[purple] %s\n"+
					"[white]White Balance:[purple] %s\n"+
					"[white]Color Space:[purple] %s\n",
				get(exif.Model),
				get(exif.Software),
				get(exif.DateTimeOriginal),
				get(exif.PixelXDimension), get(exif.PixelYDimension),
				get(exif.ISOSpeedRatings),
				get(exif.ExposureTime),
				getFNumber(x),
				get(exif.FocalLength),
				get(exif.ExposureProgram),
				get(exif.WhiteBalance),
				get(exif.ColorSpace),
			)
		}
	}
	return exifInfo
}

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/d1nch8g/gav1d"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: decode <file.ivf>")
		os.Exit(1)
	}

	path := os.Args[1]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}

	d, err := gav1d.New(gav1d.Settings{Threads: 4})
	if err != nil {
		fmt.Fprintln(os.Stderr, "new decoder:", err)
		os.Exit(1)
	}
	defer d.Close()

	frames, err := d.DecodeIVF(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "decode:", err)
		os.Exit(1)
	}

	fmt.Printf("декодировано фреймов: %d\n", len(frames))

	// Сохраняем первые 3 фрейма как PNG для визуальной проверки
	saveN := 3
	if len(frames) < saveN {
		saveN = len(frames)
	}

	for i := 0; i < saveN; i++ {
		f := frames[i]
		fmt.Printf("фрейм %d: %dx%d\n", i, f.Width(), f.Height())

		outPath := filepath.Join(".", fmt.Sprintf("frame_%03d.png", i))
		if err := saveFramePNG(f, outPath); err != nil {
			fmt.Fprintf(os.Stderr, "сохранение фрейма %d: %v\n", i, err)
		} else {
			fmt.Printf("  сохранён: %s\n", outPath)
		}
	}

	for _, f := range frames {
		f.Free()
	}
}

func saveFramePNG(f *gav1d.Frame, path string) error {
	w := f.Width()
	h := f.Height()
	y, u, v, yStride, uvStride := f.YCbCr()

	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			yVal := float64(y[row*yStride+col])

			// UV в I420 субдискретизированы 2x2
			uvRow := row / 2
			uvCol := col / 2
			uVal := float64(u[uvRow*uvStride+uvCol]) - 128
			vVal := float64(v[uvRow*uvStride+uvCol]) - 128

			// BT.601 YCbCr → RGB
			r := clamp(yVal + 1.402*vVal)
			g := clamp(yVal - 0.344136*uVal - 0.714136*vVal)
			b := clamp(yVal + 1.772*uVal)

			img.SetRGBA(col, row, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func clamp(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

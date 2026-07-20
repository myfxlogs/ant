// genogimage generates the static default OG image (og-image.png) for the frontend.
// Run: go run ./cmd/genogimage -o ../../frontend/public/og-image.png
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func main() {
	out := flag.String("o", "og-image.png", "output path")
	flag.Parse()

	const W, H = 1200, 630
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{R: 0x0d, G: 0x11, B: 0x17, A: 0xff}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, W, 6), &image.Uniform{color.RGBA{R: 0xd4, G: 0xaf, B: 0x37, A: 0xff}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 580, W, H), &image.Uniform{color.RGBA{R: 0x0a, G: 0x10, B: 0x18, A: 0xff}}, image.Point{}, draw.Src)

	gold := color.RGBA{R: 0xd4, G: 0xaf, B: 0x37, A: 0xff}
	white := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	gray := color.RGBA{R: 0x8c, G: 0x8c, B: 0x8c, A: 0xff}

	drawTextScaled(img, 60, 120, "AlphaForge", gold, 5)
	drawTextScaled(img, 60, 180, "AI-Powered MT4/MT5 Strategy Platform", gray, 2)
	drawTextScaled(img, 60, 300, "Backtest. Optimize. Automate.", white, 3)
	drawTextScaled(img, 60, 380, "alfq.org", gold, 2)
	drawTextScaled(img, 60, 605, "Verified trading strategies — alfq.org", color.RGBA{R: 0x5c, G: 0x6b, B: 0x7a, A: 0xff}, 1)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, buf.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", *out, buf.Len())
}

func drawTextScaled(img *image.RGBA, x, y int, text string, c color.Color, scale int) {
	if scale < 1 {
		scale = 1
	}
	face := basicfont.Face7x13
	if scale == 1 {
		d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
		d.DrawString(text)
		return
	}
	tw := len(text)*7 + 1
	th := 14
	tmp := image.NewRGBA(image.Rect(0, 0, tw, th))
	td := font.Drawer{Dst: tmp, Src: image.NewUniform(c), Face: face, Dot: fixed.P(0, 11)}
	td.DrawString(text)
	for ty := 0; ty < th; ty++ {
		for tx := 0; tx < tw; tx++ {
			pixel := tmp.At(tx, ty)
			_, _, _, a := pixel.RGBA()
			if a == 0 {
				continue
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.Set(x+tx*scale+dx, y-11*scale+ty*scale+dy, pixel)
				}
			}
		}
	}
}

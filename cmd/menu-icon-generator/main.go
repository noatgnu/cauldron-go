package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

const (
	canvasSize = 20
	glyphOx    = 2
	glyphOy    = 2
)

var (
	darkCol  = color.RGBA{30, 30, 30, 255}
	lightCol = color.RGBA{245, 245, 245, 235}
)

// drawFunc draws a glyph translated by (dx, dy) in the given color onto img.
type drawFunc func(img *image.RGBA, dx, dy int, col color.Color)

func setPx(img *image.RGBA, x, y int, col color.Color) {
	if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
		return
	}
	img.Set(x, y, col)
}

func line(img *image.RGBA, x0, y0, x1, y1 int, dx, dy int, col color.Color) {
	x0, y0, x1, y1 = x0+dx, y0+dy, x1+dx, y1+dy
	ddx := abs(x1 - x0)
	ddy := -abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := ddx + ddy
	for {
		setPx(img, x0, y0, col)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= ddy {
			err += ddy
			x0 += sx
		}
		if e2 <= ddx {
			err += ddx
			y0 += sy
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func rectOutline(img *image.RGBA, x0, y0, x1, y1 int, dx, dy int, col color.Color) {
	line(img, x0, y0, x1, y0, dx, dy, col)
	line(img, x0, y1, x1, y1, dx, dy, col)
	line(img, x0, y0, x0, y1, dx, dy, col)
	line(img, x1, y0, x1, y1, dx, dy, col)
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, dx, dy int, col color.Color) {
	for y := y0; y <= y1; y++ {
		line(img, x0, y, x1, y, dx, dy, col)
	}
}

func circleOutline(img *image.RGBA, cx, cy, r int, dx, dy int, col color.Color) {
	x, y, d := r, 0, 1-r
	plot := func(x, y int) {
		setPx(img, cx+dx+x, cy+dy+y, col)
		setPx(img, cx+dx-x, cy+dy+y, col)
		setPx(img, cx+dx+x, cy+dy-y, col)
		setPx(img, cx+dx-x, cy+dy-y, col)
		setPx(img, cx+dx+y, cy+dy+x, col)
		setPx(img, cx+dx-y, cy+dy+x, col)
		setPx(img, cx+dx+y, cy+dy-x, col)
		setPx(img, cx+dx-y, cy+dy-x, col)
	}
	for x >= y {
		plot(x, y)
		y++
		if d < 0 {
			d += 2*y + 1
		} else {
			x--
			d += 2*(y-x) + 1
		}
	}
}

func fillTriangle(img *image.RGBA, x0, y0, x1, y1, x2, y2 int, dx, dy int, col color.Color) {
	minY, maxY := min3(y0, y1, y2), max3(y0, y1, y2)
	for y := minY; y <= maxY; y++ {
		var xs []int
		edges := [][4]int{{x0, y0, x1, y1}, {x1, y1, x2, y2}, {x2, y2, x0, y0}}
		for _, e := range edges {
			ex0, ey0, ex1, ey1 := e[0], e[1], e[2], e[3]
			if ey0 == ey1 {
				continue
			}
			if (y >= ey0 && y < ey1) || (y >= ey1 && y < ey0) {
				t := float64(y-ey0) / float64(ey1-ey0)
				xs = append(xs, int(float64(ex0)+t*float64(ex1-ex0)))
			}
		}
		if len(xs) >= 2 {
			lo, hi := xs[0], xs[1]
			if lo > hi {
				lo, hi = hi, lo
			}
			line(img, lo, y, hi, y, dx, dy, col)
		}
	}
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func max3(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

// renderGlyph draws draw with a light halo (via 8-directional offsets) then
// a dark fill on top, so the glyph stays visible on both light and dark
// native menu backgrounds.
func renderGlyph(draw drawFunc) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, canvasSize, canvasSize))
	for oy := -1; oy <= 1; oy++ {
		for ox := -1; ox <= 1; ox++ {
			if ox == 0 && oy == 0 {
				continue
			}
			draw(img, glyphOx+ox, glyphOy+oy, lightCol)
		}
	}
	draw(img, glyphOx, glyphOy, darkCol)
	return img
}

func iconImportData(img *image.RGBA, dx, dy int, col color.Color) {
	line(img, 3, 12, 12, 12, dx, dy, col)
	line(img, 7, 3, 7, 9, dx, dy, col)
	line(img, 4, 6, 7, 9, dx, dy, col)
	line(img, 10, 6, 7, 9, dx, dy, col)
}

func iconSettings(img *image.RGBA, dx, dy int, col color.Color) {
	circleOutline(img, 7, 7, 4, dx, dy, col)
	teeth := [][2][2]int{
		{{7, 1}, {7, 3}}, {{7, 11}, {7, 13}},
		{{1, 7}, {3, 7}}, {{11, 7}, {13, 7}},
		{{3, 3}, {4, 4}}, {{10, 10}, {11, 11}},
		{{3, 11}, {4, 10}}, {{10, 4}, {11, 3}},
	}
	for _, t := range teeth {
		line(img, t[0][0], t[0][1], t[1][0], t[1][1], dx, dy, col)
	}
}

func iconQuit(img *image.RGBA, dx, dy int, col color.Color) {
	line(img, 2, 2, 2, 13, dx, dy, col)
	line(img, 2, 2, 8, 2, dx, dy, col)
	line(img, 2, 13, 8, 13, dx, dy, col)
	line(img, 6, 7, 13, 7, dx, dy, col)
	line(img, 10, 4, 13, 7, dx, dy, col)
	line(img, 10, 10, 13, 7, dx, dy, col)
}

func iconHome(img *image.RGBA, dx, dy int, col color.Color) {
	fillTriangle(img, 7, 2, 2, 8, 12, 8, dx, dy, col)
	rectOutline(img, 3, 8, 11, 13, dx, dy, col)
	fillRect(img, 6, 10, 8, 13, dx, dy, col)
}

func iconJobs(img *image.RGBA, dx, dy int, col color.Color) {
	rectOutline(img, 5, 3, 10, 6, dx, dy, col)
	rectOutline(img, 2, 6, 13, 13, dx, dy, col)
	line(img, 2, 9, 13, 9, dx, dy, col)
}

func iconPlugins(img *image.RGBA, dx, dy int, col color.Color) {
	rectOutline(img, 2, 2, 5, 5, dx, dy, col)
	rectOutline(img, 10, 2, 13, 5, dx, dy, col)
	rectOutline(img, 2, 10, 5, 13, dx, dy, col)
	rectOutline(img, 10, 10, 13, 13, dx, dy, col)
}

func iconMinimize(img *image.RGBA, dx, dy int, col color.Color) {
	fillRect(img, 3, 11, 12, 13, dx, dy, col)
}

func iconZoom(img *image.RGBA, dx, dy int, col color.Color) {
	rectOutline(img, 2, 2, 13, 13, dx, dy, col)
}

func iconAbout(img *image.RGBA, dx, dy int, col color.Color) {
	circleOutline(img, 7, 7, 5, dx, dy, col)
	line(img, 7, 7, 7, 11, dx, dy, col)
	fillRect(img, 6, 3, 8, 5, dx, dy, col)
}

func iconDocumentation(img *image.RGBA, dx, dy int, col color.Color) {
	rectOutline(img, 2, 3, 13, 13, dx, dy, col)
	line(img, 7, 3, 7, 13, dx, dy, col)
	line(img, 4, 6, 6, 6, dx, dy, col)
	line(img, 9, 6, 11, 6, dx, dy, col)
	line(img, 4, 9, 6, 9, dx, dy, col)
	line(img, 9, 9, 11, 9, dx, dy, col)
}

func iconLogFile(img *image.RGBA, dx, dy int, col color.Color) {
	rectOutline(img, 4, 2, 11, 13, dx, dy, col)
	line(img, 6, 5, 9, 5, dx, dy, col)
	line(img, 6, 7, 9, 7, dx, dy, col)
	line(img, 6, 9, 9, 9, dx, dy, col)
}

func iconLogDirectory(img *image.RGBA, dx, dy int, col color.Color) {
	rectOutline(img, 2, 4, 7, 6, dx, dy, col)
	rectOutline(img, 2, 6, 13, 13, dx, dy, col)
}

func main() {
	outDir := "resources/menu-icons"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output dir: %v\n", err)
		os.Exit(1)
	}

	icons := map[string]drawFunc{
		"import-data":   iconImportData,
		"settings":      iconSettings,
		"quit":          iconQuit,
		"home":          iconHome,
		"jobs":          iconJobs,
		"plugins":       iconPlugins,
		"minimize":      iconMinimize,
		"zoom":          iconZoom,
		"about":         iconAbout,
		"documentation": iconDocumentation,
		"log-file":      iconLogFile,
		"log-directory": iconLogDirectory,
	}

	for name, draw := range icons {
		img := renderGlyph(draw)
		outPath := filepath.Join(outDir, name+".png")
		f, err := os.Create(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create %s: %v\n", outPath, err)
			os.Exit(1)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "failed to encode %s: %v\n", outPath, err)
			os.Exit(1)
		}
		f.Close()
		fmt.Printf("wrote %s\n", outPath)
	}
}

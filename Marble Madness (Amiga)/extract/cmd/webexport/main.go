// webexport renders the Marble Madness course tilemaps for the companion
// website. For each course it decodes the .mlb (Marble_Madness.md Part IV §3)
// and writes the assembled course as a 1x PNG (the viewer scales it up), plus a
// meta.json listing the levels in play order.
//
// Usage: webexport [-adf disk.adf] [-o dir]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"marblemad/extract/mlb"
	"stupidcoder.com/tools/amiga/adf"
	"stupidcoder.com/tools/c64/gfx"
)

// courses in play order: key (.mlb basename) and display name.
var courses = []struct{ key, name string }{
	{"practy", "Practice"},
	{"beginr", "Beginner"},
	{"interm", "Intermediate"},
	{"aerial", "Aerial"},
	{"silly", "Silly"},
	{"ultima", "Ultimate"},
}

type metaLevel struct {
	Name string `json:"name"`
	File string `json:"file"`
	W    int    `json:"w"`
	H    int    `json:"h"`
}

func main() {
	adfPath := flag.String("adf", "../Marble_Madness.adf", "disk image")
	outDir := flag.String("o", "../../site/public/marble", "output directory")
	flag.Parse()
	if err := run(*adfPath, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "webexport:", err)
		os.Exit(1)
	}
}

func run(adfPath, outDir string) error {
	raw, err := os.ReadFile(adfPath)
	if err != nil {
		return err
	}
	vol, err := adf.Open(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// case-insensitive filename -> path
	paths := map[string]string{}
	if err := vol.Walk(func(e adf.Entry) error {
		if !e.IsDir {
			paths[strings.ToLower(e.Name)] = e.Path
		}
		return nil
	}); err != nil {
		return err
	}

	var levels []metaLevel
	for _, c := range courses {
		p, ok := paths[c.key+".mlb"]
		if !ok {
			return fmt.Errorf("%s.mlb not found on disk", c.key)
		}
		d, err := vol.ReadFile(p)
		if err != nil {
			return err
		}
		img, h := mlb.RenderCourse(d)
		file := c.key + ".png"
		if err := gfx.WritePNG(filepath.Join(outDir, file), img); err != nil {
			return err
		}
		levels = append(levels, metaLevel{Name: c.name, File: file, W: mlb.CourseW * 8, H: h * 8})
		fmt.Printf("%-12s %s  %dx%d px (%d tile rows)\n", c.name, file, mlb.CourseW*8, h*8, h)
	}

	b, err := json.Marshal(struct {
		Levels []metaLevel `json:"levels"`
	}{levels})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "meta.json"), b, 0o644)
}

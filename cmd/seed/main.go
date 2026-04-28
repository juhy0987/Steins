// Package main generates a sample catalog under ./data/manga so the API
// has something to serve out of the box. Run with `make seed`.
//
// 샘플 catalog를 생성하여 API가 즉시 데이터를 서빙할 수 있게 합니다.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	root := flag.String("data", "./data/manga", "catalog root directory")
	flag.Parse()

	now := time.Now().UTC()

	series := []seriesSpec{
		{
			Slug:        "steins-sample",
			Title:       "Steins Sample",
			Description: "샘플 만화 — API 동작 확인용 더미 데이터입니다.",
			Authors:     []string{"Sample Author"},
			Genres:      []string{"action", "sci-fi"},
			Status:      "ongoing",
			Language:    "ko",
			ReadingDir:  "rtl",
			PublishedAt: now.AddDate(0, -3, 0),
			UpdatedAt:   now,
			Chapters: []chapterSpec{
				{Number: "0001", Title: "프롤로그", Pages: 3, BG: color.RGBA{R: 230, G: 240, B: 255, A: 255}},
				{Number: "0002", Title: "발단", Pages: 4, BG: color.RGBA{R: 255, G: 240, B: 230, A: 255}},
			},
		},
		{
			Slug:        "test-comic",
			Title:       "Test Comic",
			Description: "두 번째 샘플 시리즈.",
			Genres:      []string{"slice-of-life"},
			Status:      "completed",
			Language:    "en",
			ReadingDir:  "ltr",
			PublishedAt: now.AddDate(-1, 0, 0),
			UpdatedAt:   now.AddDate(0, -1, 0),
			Chapters: []chapterSpec{
				{Number: "1", Title: "Hello", Pages: 2, BG: color.RGBA{R: 240, G: 255, B: 240, A: 255}},
			},
		},
	}

	if err := os.MkdirAll(*root, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", *root, err)
	}

	for _, s := range series {
		if err := writeSeries(*root, s); err != nil {
			log.Fatalf("write series %s: %v", s.Slug, err)
		}
		fmt.Printf("seeded %s (%d chapters)\n", s.Slug, len(s.Chapters))
	}
}

type seriesSpec struct {
	Slug        string
	Title       string
	Description string
	Authors     []string
	Genres      []string
	Status      string
	Language    string
	ReadingDir  string
	PublishedAt time.Time
	UpdatedAt   time.Time
	Chapters    []chapterSpec
}

type chapterSpec struct {
	Number string
	Title  string
	Pages  int
	BG     color.RGBA
}

func writeSeries(root string, s seriesSpec) error {
	dir := filepath.Join(root, s.Slug)
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0o755); err != nil {
		return err
	}

	meta := map[string]any{
		"slug":         s.Slug,
		"title":        s.Title,
		"description":  s.Description,
		"authors":      s.Authors,
		"genres":       s.Genres,
		"status":       s.Status,
		"language":     s.Language,
		"reading_dir":  s.ReadingDir,
		"published_at": s.PublishedAt,
		"updated_at":   s.UpdatedAt,
	}
	if err := writeJSON(filepath.Join(dir, "manga.json"), meta); err != nil {
		return err
	}

	for _, c := range s.Chapters {
		chDir := filepath.Join(dir, "chapters", c.Number)
		if err := os.MkdirAll(chDir, 0o755); err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(chDir, "chapter.json"), map[string]any{
			"title":        c.Title,
			"language":     s.Language,
			"published_at": s.PublishedAt,
		}); err != nil {
			return err
		}
		for i := 1; i <= c.Pages; i++ {
			path := filepath.Join(chDir, fmt.Sprintf("%03d.png", i))
			if err := writePagePNG(path, c.BG, c.Number, i); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// writePagePNG produces a tiny solid-color PNG with a small index pattern so
// each generated page is visually distinct and each file's bytes differ.
func writePagePNG(path string, bg color.RGBA, chapterNumber string, index int) error {
	const w, h = 64, 96
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, bg)
		}
	}

	// Stripe the page index onto the image so different pages have different
	// bytes (and therefore different checksums).
	stripe := color.RGBA{R: 30, G: 30, B: 30, A: 255}
	for y := 4; y < 4+index*4; y++ {
		if y >= h {
			break
		}
		for x := 4; x < w-4; x++ {
			img.Set(x, y, stripe)
		}
	}

	// Encode the chapter number as a single-row marker too, so chapters
	// with identical page indices still hash differently.
	for i, b := range []byte(chapterNumber) {
		img.Set(2+i*2, h-4, color.RGBA{R: b, G: b, B: b, A: 255})
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

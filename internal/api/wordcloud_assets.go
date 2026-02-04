package api

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"media-crawler-go/internal/config"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type wordcloudLexicon struct {
	stop       map[string]struct{}
	customKeys []string
}

func buildWordcloudLexicon(cfg config.Config) wordcloudLexicon {
	stop := defaultStopwords()
	if p := strings.TrimSpace(cfg.StopWordsFile); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				w := strings.TrimSpace(line)
				if w == "" {
					continue
				}
				stop[strings.ToLower(w)] = struct{}{}
			}
		}
	}

	custom := make([]string, 0, len(cfg.CustomWords))
	for k := range cfg.CustomWords {
		w := strings.TrimSpace(k)
		if w == "" {
			continue
		}
		custom = append(custom, strings.ToLower(w))
	}
	sort.Slice(custom, func(i, j int) bool {
		return len([]rune(custom[i])) > len([]rune(custom[j]))
	})

	return wordcloudLexicon{stop: stop, customKeys: custom}
}

func countWordsWithLexicon(texts []string, maxWords int, minCount int, lex wordcloudLexicon) []wordCount {
	stop := lex.stop
	custom := lex.customKeys
	m := make(map[string]int, 2048)
	for _, t := range texts {
		for _, tok := range tokenizeChineseAware(t, stop, custom) {
			tok = strings.TrimSpace(strings.ToLower(tok))
			if tok == "" {
				continue
			}
			if _, ok := stop[tok]; ok {
				continue
			}
			if isASCIIWord(tok) && len(tok) < 2 {
				continue
			}
			if len([]rune(tok)) < 2 && !isASCIIWord(tok) {
				continue
			}
			m[tok]++
		}
	}
	out := make([]wordCount, 0, len(m))
	for w, c := range m {
		if c < minCount {
			continue
		}
		out = append(out, wordCount{Word: w, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Word < out[j].Word
	})
	if len(out) > maxWords {
		out = out[:maxWords]
	}
	return out
}

func tokenizeChineseAware(s string, stop map[string]struct{}, custom []string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return nil
	}
	out := make([]string, 0, 64)

	var asciiBuf []rune
	var hanBuf []rune

	flushASCII := func() {
		if len(asciiBuf) == 0 {
			return
		}
		out = append(out, string(asciiBuf))
		asciiBuf = asciiBuf[:0]
	}
	flushHan := func() {
		if len(hanBuf) == 0 {
			return
		}
		seg := string(hanBuf)
		hanBuf = hanBuf[:0]
		out = append(out, splitByCustomWords(seg, custom)...)
	}

	for _, r := range []rune(s) {
		if isASCIIAlphaNum(r) {
			flushHan()
			asciiBuf = append(asciiBuf, r)
			continue
		}
		if isHan(r) {
			flushASCII()
			w := string(r)
			if _, ok := stop[w]; ok {
				flushHan()
				continue
			}
			hanBuf = append(hanBuf, r)
			continue
		}
		flushASCII()
		flushHan()
	}
	flushASCII()
	flushHan()
	return out
}

func splitByCustomWords(seg string, custom []string) []string {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return nil
	}
	if len(custom) == 0 {
		return []string{seg}
	}

	rs := []rune(seg)
	var tokens []string
	var buf []rune
	i := 0
	for i < len(rs) {
		matched := ""
		matchedLen := 0
		for _, cw := range custom {
			cr := []rune(cw)
			if len(cr) == 0 || i+len(cr) > len(rs) {
				continue
			}
			ok := true
			for j := 0; j < len(cr); j++ {
				if rs[i+j] != cr[j] {
					ok = false
					break
				}
			}
			if ok {
				matched = cw
				matchedLen = len(cr)
				break
			}
		}
		if matchedLen > 0 {
			if len(buf) > 0 {
				tokens = append(tokens, string(buf))
				buf = buf[:0]
			}
			tokens = append(tokens, matched)
			i += matchedLen
			continue
		}
		buf = append(buf, rs[i])
		i++
	}
	if len(buf) > 0 {
		tokens = append(tokens, string(buf))
	}
	return tokens
}

func wordFreqJSON(counts []wordCount) []byte {
	m := make(map[string]int, len(counts))
	for _, wc := range counts {
		m[wc.Word] = wc.Count
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return b
}

func renderWordcloudPNG(words []wordCount, width, height int, seed int64, fontPath string) ([]byte, error) {
	if len(words) == 0 {
		return nil, nil
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	type closer interface{ Close() error }
	var otFont *opentype.Font
	closeFns := make([]func(), 0, 8)
	if p := strings.TrimSpace(fontPath); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			if f, err := opentype.Parse(b); err == nil {
				otFont = f
			}
		}
	}
	faces := map[int]font.Face{}
	defer func() {
		for _, fn := range closeFns {
			fn()
		}
	}()

	palette := []color.RGBA{
		{R: 0x1f, G: 0x77, B: 0xb4, A: 0xff},
		{R: 0xff, G: 0x7f, B: 0x0e, A: 0xff},
		{R: 0x2c, G: 0xa0, B: 0x2c, A: 0xff},
		{R: 0xd6, G: 0x27, B: 0x28, A: 0xff},
		{R: 0x94, G: 0x67, B: 0xbd, A: 0xff},
		{R: 0x8c, G: 0x56, B: 0x4b, A: 0xff},
		{R: 0xe3, G: 0x77, B: 0xc2, A: 0xff},
		{R: 0x7f, G: 0x7f, B: 0x7f, A: 0xff},
		{R: 0xbc, G: 0xbd, B: 0x22, A: 0xff},
		{R: 0x17, G: 0xbe, B: 0xcf, A: 0xff},
	}

	maxC := words[0].Count
	minC := words[len(words)-1].Count
	if minC <= 0 {
		minC = 1
	}
	rnd := rand.New(rand.NewSource(seed))

	placed := make([]box, 0, len(words))
	for i, wc := range words {
		fontSize := scaleFont(wc.Count, minC, maxC, 16, 84)
		face := faces[fontSize]
		if face == nil {
			if otFont != nil {
				fc, err := opentype.NewFace(otFont, &opentype.FaceOptions{
					Size:    float64(fontSize),
					DPI:     72,
					Hinting: font.HintingFull,
				})
				if err == nil {
					face = fc
					if c, ok := fc.(closer); ok {
						closeFns = append(closeFns, func() { _ = c.Close() })
					}
				}
			}
			if face == nil {
				face = basicfont.Face7x13
			}
			faces[fontSize] = face
		}

		w, h := measureText(face, fontSize, wc.Word)

		canvasW := float64(width)
		canvasH := float64(height)
		cx := canvasW / 2
		cy := canvasH / 2

		var chosen box
		ok := false
		for step := 0; step < 1600; step++ {
			t := float64(step) / 6
			ang := t * 0.7
			rad := 6 + t*3.2
			x := cx + rad*math.Cos(ang) + (rnd.Float64()-0.5)*6
			y := cy + rad*math.Sin(ang) + (rnd.Float64()-0.5)*6
			b := box{x0: x - w/2, y0: y - h/2, x1: x + w/2, y1: y + h/2}
			if b.x0 < 6 || b.y0 < 6 || b.x1 > canvasW-6 || b.y1 > canvasH-6 {
				continue
			}
			collide := false
			for _, p := range placed {
				if b.overlaps(p) {
					collide = true
					break
				}
			}
			if collide {
				continue
			}
			chosen = b
			ok = true
			break
		}
		if !ok {
			continue
		}
		placed = append(placed, chosen)

		x := (chosen.x0 + chosen.x1) / 2
		y := (chosen.y0 + chosen.y1) / 2
		col := palette[i%len(palette)]

		drawText(canvas, face, fontSize, int(math.Round(x-w/2)), int(math.Round(y+h/2)), col, wc.Word)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func measureText(base font.Face, size int, s string) (float64, float64) {
	if size <= 0 {
		size = 12
	}
	d := &font.Drawer{Face: base}
	adv := d.MeasureString(s)
	w := float64(adv.Round())
	if base == basicfont.Face7x13 {
		w = w * float64(size) / 13
	}
	h := float64(size)
	return w, h
}

func drawText(dst draw.Image, base font.Face, size int, x int, y int, c color.RGBA, s string) {
	if dst == nil {
		return
	}
	if base == basicfont.Face7x13 && size != 13 {
		base = basicfont.Face7x13
	}
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: base,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

func saveWordcloudAssets(dataDir, platform, baseName string, svg string, pngBytes []byte, freqJSON []byte) (map[string]string, error) {
	out := map[string]string{}
	outDir := filepath.Join(strings.TrimSpace(dataDir), strings.TrimSpace(platform))
	if outDir == "" {
		outDir = filepath.Join("data", platform)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}
	if svg != "" {
		svgPath := filepath.Join(outDir, baseName+".svg")
		if err := os.WriteFile(svgPath, []byte(svg), 0644); err != nil {
			return nil, err
		}
		out["svg"] = svgPath
	}
	if len(pngBytes) > 0 {
		pngPath := filepath.Join(outDir, baseName+".png")
		if err := os.WriteFile(pngPath, pngBytes, 0644); err != nil {
			return nil, err
		}
		out["png"] = pngPath
	}
	if len(freqJSON) > 0 {
		jsonPath := filepath.Join(outDir, baseName+"_word_freq.json")
		if err := os.WriteFile(jsonPath, freqJSON, 0644); err != nil {
			return nil, err
		}
		out["freq"] = jsonPath
	}
	return out, nil
}

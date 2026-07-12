package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	guten "github.com/kitsyai/guten/go"
)

// batchOpts is the parsed form of `guten batch` flags. It shares most fields
// with opts (the template selector and the render-data knobs) but replaces
// -o/--out (repeatable output *files*) with a single output *directory*, and
// adds --name (the per-row filename template).
type batchOpts struct {
	template string
	manifest string
	lib      string
	libDir   string
	renderer string
	data     string
	name     string
	outDir   string
	chrome   string
	theme    string
	sets     []string
	css      []string
	header   string
	footer   string
	slots    []string
}

func parseBatchOpts(args []string) (batchOpts, error) {
	o := batchOpts{renderer: guten.RendererLiquid}
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for %s", a)
			}
			i++
			return args[i], nil
		}
		var err error
		switch a {
		case "-t", "--template":
			o.template, err = next()
		case "--manifest":
			o.manifest, err = next()
		case "--lib":
			o.lib, err = next()
		case "--lib-dir":
			o.libDir, err = next()
		case "-r", "--renderer":
			o.renderer, err = next()
		case "-d", "--data":
			o.data, err = next()
		case "--name":
			o.name, err = next()
		case "-o", "--out":
			o.outDir, err = next()
		case "--theme":
			o.theme, err = next()
		case "--set":
			var v string
			if v, err = next(); err == nil {
				o.sets = append(o.sets, v)
			}
		case "--css":
			var v string
			if v, err = next(); err == nil {
				o.css = append(o.css, v)
			}
		case "--header":
			o.header, err = next()
		case "--footer":
			o.footer, err = next()
		case "--slot":
			var v string
			if v, err = next(); err == nil {
				o.slots = append(o.slots, v)
			}
		case "--chrome":
			o.chrome, err = next()
		default:
			return o, fmt.Errorf("unknown flag %q", a)
		}
		if err != nil {
			return o, err
		}
	}
	return o, nil
}

// baseOpts adapts batchOpts into the shared opts shape so engineAndTemplate
// and renderData (the same internals `render`/`export` use) apply unchanged.
func (o batchOpts) baseOpts() opts {
	return opts{
		template: o.template,
		manifest: o.manifest,
		lib:      o.lib,
		libDir:   o.libDir,
		renderer: o.renderer,
		chrome:   o.chrome,
		theme:    o.theme,
		sets:     o.sets,
		css:      o.css,
		header:   o.header,
		footer:   o.footer,
		slots:    o.slots,
		part:     guten.PartHTML,
	}
}

// BatchRowError is a single row's failure. The row number is 1-based and
// counts only non-blank data rows (JSONL blank lines are skipped, CSV header
// doesn't count).
type BatchRowError struct {
	Row int
	Err error
}

func (e BatchRowError) Error() string { return fmt.Sprintf("row %d: %v", e.Row, e.Err) }

// BatchResult is the outcome of a batch run: files written, plus any
// per-row failures. The run always processes every row.
type BatchResult struct {
	Written []string
	Errors  []BatchRowError
}

// parsedRow is one data row plus its 1-based row number (counting only data
// rows: blank JSONL lines and the CSV header don't count). err is set when
// the row itself failed to parse (e.g. malformed JSON on that line) — the
// row still occupies its slot so runBatch can report it and continue past it,
// exactly like a row that fails later at render time.
type parsedRow struct {
	n    int
	data map[string]any
	err  error
}

// loadRows parses -d input as JSONL (one JSON object per line) or CSV (header
// row + records), chosen by the file extension: .csv is CSV, everything else
// is treated as JSONL. A malformed individual row is captured on the
// returned parsedRow (not fatal); loadRows itself only fails for structural
// problems (can't read the file, no header row, etc).
func loadRows(dataArg string) ([]parsedRow, error) {
	if strings.TrimSpace(dataArg) == "" {
		return nil, fmt.Errorf("batch requires -d @rows.jsonl or -d @rows.csv")
	}
	if !strings.HasPrefix(dataArg, "@") {
		return nil, fmt.Errorf("batch -d must be a file: @rows.jsonl or @rows.csv")
	}
	raw, err := loadArg(dataArg)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(filepath.Ext(dataArg[1:]), ".csv") {
		return parseCSVRows(raw)
	}
	return parseJSONLRows(raw)
}

func parseJSONLRows(raw string) ([]parsedRow, error) {
	var rows []parsedRow
	sc := bufio.NewScanner(strings.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		n++
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			rows = append(rows, parsedRow{n: n, err: fmt.Errorf("parse row: %w", err)})
			continue
		}
		rows = append(rows, parsedRow{n: n, data: row})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func parseCSVRows(raw string) ([]parsedRow, error) {
	r := csv.NewReader(strings.NewReader(raw))
	header, err := r.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	var rows []parsedRow
	n := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		n++
		if err != nil {
			rows = append(rows, parsedRow{n: n, err: fmt.Errorf("parse row: %w", err)})
			continue
		}
		row := make(map[string]any, len(header))
		for i, h := range header {
			if i < len(rec) {
				row[h] = rec[i]
			}
		}
		rows = append(rows, parsedRow{n: n, data: row})
	}
	return rows, nil
}

// runBatch renders one output file per row (testable core of `batch`). Row
// failures are collected but do not stop the run; the caller decides how to
// report a non-nil err (which is non-nil iff at least one row failed).
func runBatch(ctx context.Context, o batchOpts) (BatchResult, error) {
	var res BatchResult
	if strings.TrimSpace(o.name) == "" {
		return res, fmt.Errorf("batch requires --name \"<filename template>\"")
	}
	if strings.TrimSpace(o.outDir) == "" {
		return res, fmt.Errorf("batch requires -o <dir>")
	}
	rows, err := loadRows(o.data)
	if err != nil {
		return res, err
	}
	if len(rows) == 0 {
		return res, fmt.Errorf("no rows found in %s", o.data)
	}
	base := o.baseOpts()
	e, name, baseTheme, baseSample, err := engineAndTemplate(base)
	if err != nil {
		return res, err
	}
	nameTpl, err := guten.NewLiquidRenderer().Compile(o.name)
	if err != nil {
		return res, fmt.Errorf("parse --name template: %w", err)
	}
	part := partForExt(o.name)
	if err := os.MkdirAll(o.outDir, 0o755); err != nil {
		return res, err
	}

	for _, pr := range rows {
		rowNum := pr.n
		if pr.err != nil {
			res.Errors = append(res.Errors, BatchRowError{Row: rowNum, Err: pr.err})
			continue
		}
		rowJSON, merr := json.Marshal(pr.data)
		if merr != nil {
			res.Errors = append(res.Errors, BatchRowError{Row: rowNum, Err: merr})
			continue
		}
		rowOpts := base
		rowOpts.data = string(rowJSON)
		rowOpts.part = part
		data, derr := renderData(rowOpts, baseTheme, baseSample)
		if derr != nil {
			res.Errors = append(res.Errors, BatchRowError{Row: rowNum, Err: derr})
			continue
		}
		filename, ferr := nameTpl.Render(data)
		if ferr != nil {
			res.Errors = append(res.Errors, BatchRowError{Row: rowNum, Err: fmt.Errorf("render filename: %w", ferr)})
			continue
		}
		filename = strings.TrimSpace(filename)
		if filename == "" {
			res.Errors = append(res.Errors, BatchRowError{Row: rowNum, Err: fmt.Errorf("rendered filename is empty")})
			continue
		}
		payload, perr := renderPayload(ctx, e, name, part, data, o.css, o.chrome)
		if perr != nil {
			res.Errors = append(res.Errors, BatchRowError{Row: rowNum, Err: perr})
			continue
		}
		outPath := filepath.Join(o.outDir, filepath.FromSlash(filename))
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			res.Errors = append(res.Errors, BatchRowError{Row: rowNum, Err: err})
			continue
		}
		if err := os.WriteFile(outPath, payload, 0o644); err != nil {
			res.Errors = append(res.Errors, BatchRowError{Row: rowNum, Err: err})
			continue
		}
		res.Written = append(res.Written, outPath)
	}

	if len(res.Errors) > 0 {
		return res, fmt.Errorf("%d of %d row(s) failed", len(res.Errors), len(rows))
	}
	return res, nil
}

func cmdBatch(args []string) error {
	o, err := parseBatchOpts(args)
	if err != nil {
		return err
	}
	res, runErr := runBatch(context.Background(), o)
	for _, w := range res.Written {
		fmt.Fprintf(os.Stderr, "wrote %s\n", w)
	}
	for _, e := range res.Errors {
		fmt.Fprintf(os.Stderr, "error: %s\n", e.Error())
	}
	return runErr
}

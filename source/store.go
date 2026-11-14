package main

// المستودعات: تحويل صرف المخازن إلى قالب PCS (Store Issues)
// إعادة بناء المستودع 3.0 جوه اكشوالات — نفس الواجهة ونفس الـ API (/api/load, /api/sheet, /api/prepare, /api/convert, /api/save)

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

//go:embed assets/store_template.xlsx
var embStoreTemplate []byte

type sField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Hint     string `json:"hint,omitempty"`
}

var storeFields = []sField{
	{"date", "Date — التاريخ", true, "تاريخ الصرف"},
	{"code", "Resource Code — كود الصنف", false, "كود الصنف في PCS لو موجود"},
	{"name", "Resource Name — اسم الصنف", true, "الاسم اللي PCS بيعرف بيه الصنف"},
	{"desc", "Description — الوصف", false, "بيتاخد بداله لو اسم الصنف فاضي"},
	{"unit", "Unit — الوحدة", false, ""},
	{"qty", "Qty — الكمية", true, ""},
	{"rate", "Unit Cost — السعر", false, "السعر أو القيمة"},
	{"rate2", "Rate Assumed — سعر تقديري", false, "بيتاخد لو السعر فاضي"},
	{"amount", "Amount — القيمة", false, "السعر أو القيمة"},
	{"task", "Task Code — تاسك كود", false, "لو فاضي البرنامج بيحطه"},
	{"cost", "Cost Code — كوست كود", false, "لو فاضي البرنامج بيحطه"},
	{"section", "القسم / Task Description", false, "بيساعد في اقتراح الـ Task Code"},
}

var storeAliases = map[string][]string{
	"date":    {"date", "issue date", "التاريخ", "تاريخ", "تاريخ الصرف", "تاريخ الاذن", "تاريخ الإذن"},
	"code":    {"resource code", "item code", "erp code", "code", "كود الصنف", "رقم الصنف", "الكود", "كود"},
	"name":    {"resource name", "uniqe code", "unique code", "اسم الصنف", "اسم الماده", "اسم المادة", "name"},
	"desc":    {"description", "item", "البيان", "الوصف", "الصنف", "بيان"},
	"unit":    {"unit", "uom", "الوحده", "الوحدة"},
	"qty":     {"qty", "quantity", "الكميه", "الكمية", "الكميه المنصرفه", "الكمية المنصرفة", "كميه"},
	"rate":    {"unit cost", "unit price", "rate", "price", "سعر الوحده", "سعر الوحدة", "السعر", "الفئه", "الفئة"},
	"rate2":   {"rate assumed", "سعر تقديري", "متوسط السعر"},
	"section": {"task description", "القسم", "البند", "work package description"},
	"amount":  {"amount", "total", "value", "القيمه", "القيمة", "الاجمالي", "الإجمالي"},
	"task":    {"task code", "تاسك كود", "كود البند"},
	"cost":    {"cost code", "كوست كود", "كود التكلفه", "كود التكلفة"},
}

// ------------------------------------------------------------------ memory (profiles + codes)
type storeMemory struct {
	mu       sync.Mutex
	Profiles map[string]sMapping     `json:"profiles"` // header signature -> mapping
	Codes    map[string][2]string    `json:"codes"`    // resource key -> task, cost
	Learned  map[string]learnedEntry `json:"learned"`  // من الشيتات: وصف -> أكواد
	Sections map[string]string       `json:"sections"` // القسم -> Task Code
	Imported bool                    `json:"importedV3"`
	CodesRev int                     `json:"codesRev"`
}

type learnedEntry struct {
	Text string `json:"text"`
	Task string `json:"task"`
	Cost string `json:"cost"`
}

type sMapping struct {
	Cols    map[string]int `json:"cols"`
	Remarks []int          `json:"remarks"`
}

var smem = &storeMemory{}

func smemPath() string { return filepath.Join(appDir(), "store_memory.json") }

func loadStoreMem() {
	smem.mu.Lock()
	defer smem.mu.Unlock()
	if b, err := os.ReadFile(smemPath()); err == nil {
		json.Unmarshal(b, smem)
	}
	if smem.Profiles == nil {
		smem.Profiles = map[string]sMapping{}
	}
	if smem.Codes == nil {
		smem.Codes = map[string][2]string{}
	}
	if smem.Learned == nil {
		smem.Learned = map[string]learnedEntry{}
	}
	if smem.Sections == nil {
		smem.Sections = map[string]string{}
	}
	if !smem.Imported {
		importV3Memory()
	}
	if migrateStoreMemLocked() {
		os.MkdirAll(appDir(), 0o755)
		b, _ := json.Marshal(smem)
		os.WriteFile(smemPath(), b, 0o644)
	}
}

// نقل ذاكرة المستودع 3.0 (%APPDATA%\PCS Store Converter\memory.json) أول مرة
func importV3Memory() int {
	var paths []string
	if d := os.Getenv("APPDATA"); d != "" {
		paths = append(paths, filepath.Join(d, "PCS Store Converter", "memory.json"))
	}
	if d := os.Getenv("LOCALAPPDATA"); d != "" {
		paths = append(paths, filepath.Join(d, "Al-Mustawdaa", "memory.json"))
	}
	if p := os.Getenv("ACTUALAT_V3_MEMORY"); p != "" {
		paths = append([]string{p}, paths...)
	}
	n := 0
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var v3 struct {
			ByCode    map[string]struct{ Task, Cost string } `json:"byCode"`
			ByDesc    map[string]struct{ Task, Cost string } `json:"byDesc"`
			BySection map[string]string                      `json:"bySection"`
			Profiles  map[string]struct {
				Cols    map[string]int `json:"cols"`
				Remarks []int          `json:"remarks"`
			} `json:"profiles"`
		}
		if json.Unmarshal(b, &v3) != nil {
			continue
		}
		for k, c := range v3.ByCode {
			smem.Codes["c:"+normAr(k)] = [2]string{c.Task, c.Cost}
			n++
		}
		for k, c := range v3.ByDesc {
			smem.Codes["n:"+normAr(k)] = [2]string{c.Task, c.Cost}
			if c.Task != "" && c.Cost != "" {
				smem.Learned[normAr(k)] = learnedEntry{k, c.Task, c.Cost}
			}
			n++
		}
		for k, t := range v3.BySection {
			smem.Sections[normAr(k)] = t
		}
		for sig, p := range v3.Profiles {
			m := sMapping{Cols: map[string]int{}, Remarks: p.Remarks}
			for _, f := range storeFields {
				m.Cols[f.Key] = -1
			}
			for k, j := range p.Cols {
				switch k {
				case "desc":
					m.Cols["name"] = j
				default:
					if _, ok := m.Cols[k]; ok {
						m.Cols[k] = j
					}
				}
			}
			smem.Profiles[normAr(sig)] = m
		}
		break
	}
	if n > 0 {
		smem.Imported = true
		b, _ := json.Marshal(smem)
		os.MkdirAll(appDir(), 0o755)
		os.WriteFile(smemPath(), b, 0o644)
	}
	return n
}

func saveStoreMem() {
	smem.mu.Lock()
	defer smem.mu.Unlock()
	os.MkdirAll(appDir(), 0o755)
	b, _ := json.Marshal(smem)
	os.WriteFile(smemPath(), b, 0o644)
}

// ------------------------------------------------------------------ state
type sRow struct {
	Row     int
	Cells   []string
	Date    time.Time
	HasD    bool
	RawD    string
	Code    string
	Name    string
	Desc    string
	Unit    string
	Qty     float64
	Rate    float64
	HasR    bool
	Amount  float64
	HasA    bool
	Task    string
	Cost    string
	Section string
	key     string
}

type storeState struct {
	mu        sync.Mutex
	data      []byte
	name      string
	sheets    []string
	sheet     string
	rows      [][]string
	hdr       int
	headers   []string
	mapping   sMapping
	parsed    []*sRow
	cl        *CodeLists
	clSrc     string
	groups    map[string]*sGroup
	out, rev  []byte
	outTag    string
	outItems  int
	outAmount float64
	outLabel  string
	lastStats map[string]interface{}
}

var sst = &storeState{}

func normH(s string) string { return normAr(strings.Trim(strings.TrimSpace(s), ":")) }

func storeSignature(h []string) string {
	seen := map[string]bool{}
	var p []string
	for _, x := range h {
		if n := normH(x); n != "" && !seen[n] {
			seen[n] = true
			p = append(p, n)
		}
	}
	sort.Strings(p)
	return normAr(strings.Join(p, "|"))
}

func scoreHeader(r []string) int {
	n := 0
	for _, c := range r {
		k := normH(c)
		if k == "" {
			continue
		}
		for _, al := range storeAliases {
			for _, a := range al {
				if k == normAr(a) {
					n++
					goto next
				}
			}
		}
	next:
	}
	return n
}

func detectHeader(rows [][]string) int {
	best, bs := 0, -1
	for i := 0; i < len(rows) && i < 30; i++ {
		if s := scoreHeader(rows[i]); s > bs {
			best, bs = i, s
		}
	}
	return best
}

func autoMap(h []string) sMapping {
	m := sMapping{Cols: map[string]int{}}
	used := map[int]bool{}
	for _, f := range storeFields {
		m.Cols[f.Key] = -1
		for _, a := range storeAliases[f.Key] {
			na := normAr(a)
			for j, x := range h {
				if !used[j] && normH(x) == na {
					m.Cols[f.Key] = j
					used[j] = true
					goto done
				}
			}
		}
	done:
	}
	return m
}

func sheetInfo(w http.ResponseWriter, fromProfile bool) {
	type hdr struct {
		Index int    `json:"index"`
		Name  string `json:"name"`
	}
	var hs []hdr
	for j, h := range sst.headers {
		if strings.TrimSpace(h) != "" {
			hs = append(hs, hdr{j, strings.TrimSpace(h)})
		}
	}
	var preview [][]string
	for i := sst.hdr + 1; i < len(sst.rows) && len(preview) < 5; i++ {
		r := sst.rows[i]
		empty := true
		for _, c := range r {
			if strings.TrimSpace(c) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		var pr []string
		for _, h := range hs {
			v := ""
			if h.Index < len(r) {
				v = strings.TrimSpace(r[h.Index])
			}
			pr = append(pr, v)
		}
		preview = append(preview, pr)
	}
	writeJSON(w, 200, map[string]interface{}{"file": sst.name, "sheets": sst.sheets, "sheet": sst.sheet, "headerRow": sst.hdr,
		"headers": hs, "preview": preview, "fields": storeFields, "mapping": sst.mapping, "fromProfile": fromProfile})
}

func (s *storeState) loadSheet(sheet string, hdrRow int) (bool, error) {
	f, err := excelize.OpenReader(bytes.NewReader(s.data))
	if err != nil {
		return false, fmt.Errorf("مش قادر أفتح الملف: %v", err)
	}
	defer f.Close()
	s.sheets = f.GetSheetList()
	if sheet == "" {
		// الشيت اللي عناوينه أقرب لشيت مخازن
		bs := -1
		for _, sh := range s.sheets {
			rows, err := f.GetRows(sh, excelize.Options{RawCellValue: true})
			if err != nil || len(rows) < 2 {
				continue
			}
			sc := scoreHeader(rows[detectHeader(rows)])
			if sc > bs {
				bs, sheet = sc, sh
			}
		}
	}
	rows, err := f.GetRows(sheet, excelize.Options{RawCellValue: true})
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, fmt.Errorf("الشيت %s فاضي", sheet)
	}
	s.sheet, s.rows = sheet, rows
	if hdrRow < 0 || hdrRow >= len(rows) {
		hdrRow = detectHeader(rows)
	}
	s.hdr, s.headers = hdrRow, rows[hdrRow]
	// قوائم الأكواد من الملف نفسه لو موجودة
	s.cl, s.clSrc = loadCodeLists()
	var tRows, cRows [][]string
	for _, sh := range s.sheets {
		l := strings.ToLower(strings.ReplaceAll(sh, " ", "_"))
		if strings.Contains(l, "task_code") || l == "task_codes" {
			tRows, _ = f.GetRows(sh, excelize.Options{RawCellValue: true})
		}
		if strings.Contains(l, "cost_code") || l == "cost_codes" {
			cRows, _ = f.GetRows(sh, excelize.Options{RawCellValue: true})
		}
	}
	if len(tRows) > 1 && len(cRows) > 1 {
		if t := codesFromRows(tRows); len(t) > 0 {
			if c := codesFromRows(cRows); len(c) > 0 {
				s.cl = &CodeLists{Task: t, Cost: c}
				s.cl.index()
				s.clSrc = "file"
			}
		}
	}
	sig := storeSignature(s.headers)
	smem.mu.Lock()
	p, ok := smem.Profiles[sig]
	smem.mu.Unlock()
	if ok {
		s.mapping = p
		return true, nil
	}
	s.mapping = autoMap(s.headers)
	return false, nil
}

func codesFromRows(rows [][]string) []*Code {
	f := excelize.NewFile()
	defer f.Close()
	for i, r := range rows {
		for j, v := range r {
			c, _ := excelize.CoordinatesToCellName(j+1, i+1)
			f.SetCellStr("Sheet1", c, v)
		}
	}
	b, _ := f.WriteToBuffer()
	out, _ := readCodeSheet(b.Bytes())
	return out
}

func hStoreLoad(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(200 << 20)
	file, hdr, err := r.FormFile("file")
	if err != nil {
		fail(w, err)
		return
	}
	defer file.Close()
	data, _ := io.ReadAll(file)
	sst.data, sst.name = data, hdr.Filename
	fp, err := sst.loadSheet("", -1)
	if err != nil {
		fail(w, err)
		return
	}
	sheetInfo(w, fp)
}

func hStoreSheet(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Sheet     string `json:"sheet"`
		HeaderRow int    `json:"headerRow"`
	}
	json.NewDecoder(r.Body).Decode(&b)
	if sst.data == nil {
		fail(w, fmt.Errorf("اختار الملف الأول"))
		return
	}
	fp, err := sst.loadSheet(b.Sheet, b.HeaderRow)
	if err != nil {
		fail(w, err)
		return
	}
	sheetInfo(w, fp)
}

// ------------------------------------------------------------------ parsing
func parseDateCell(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if f > 20000 && f < 80000 {
			return time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).Add(time.Duration(math.Round(f*86400)) * time.Second), true
		}
		return time.Time{}, false
	}
	s = strings.NewReplacer("٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9").Replace(s)
	for _, l := range []string{"2/1/2006", "02/01/2006", "2-1-2006", "02-01-2006", "2006-01-02", "2006/01/02", "2006-01-02 15:04:05", "2006-01-02T15:04:05Z", "2-Jan-2006", "2-Jan-06", "2 Jan 2006", "January 2, 2006", "Jan 2, 2006", "2/1/06"} {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (s *storeState) parse() {
	m := s.mapping.Cols
	get := func(r []string, k string) string {
		j, ok := m[k]
		if !ok || j < 0 || j >= len(r) {
			return ""
		}
		return strings.TrimSpace(r[j])
	}
	s.parsed = nil
	for i := s.hdr + 1; i < len(s.rows); i++ {
		r := s.rows[i]
		x := &sRow{Row: i + 1, Cells: r, Code: get(r, "code"), Name: cleanS(get(r, "name")), Desc: cleanS(get(r, "desc")),
			Unit: get(r, "unit"), Task: get(r, "task"), Cost: get(r, "cost"), RawD: get(r, "date")}
		if x.Code == "" && x.Name == "" && x.Desc == "" {
			continue
		}
		x.Qty, _ = numv(get(r, "qty"))
		x.Rate, x.HasR = numv(get(r, "rate"))
		if !x.HasR || x.Rate == 0 {
			if r2v, ok := numv(get(r, "rate2")); ok && r2v != 0 {
				x.Rate, x.HasR = r2v, true
			}
		}
		x.Section = cleanS(get(r, "section"))
		x.Amount, x.HasA = numv(get(r, "amount"))
		x.Date, x.HasD = parseDateCell(x.RawD)
		if strings.HasPrefix(x.Task, "#") {
			x.Task = ""
		}
		if strings.HasPrefix(x.Cost, "#") {
			x.Cost = ""
		}
		x.key = resKey(x)
		s.parsed = append(s.parsed, x)
	}
}

func resKey(x *sRow) string {
	if x.Code != "" {
		return "c:" + normAr(x.Code)
	}
	if x.Name != "" {
		return "n:" + normAr(x.Name)
	}
	return "n:" + normAr(x.Desc)
}

func (x *sRow) amount() float64 {
	if x.HasA && x.Amount != 0 {
		return r2(x.Amount)
	}
	return r2(x.Rate * x.Qty)
}

func (x *sRow) unitCost() float64 {
	if x.HasA && x.Amount != 0 && x.Qty != 0 {
		return x.Amount / x.Qty
	}
	return x.Rate
}

func (x *sRow) label() string {
	if x.Name != "" {
		return x.Name
	}
	return x.Desc
}

var arMonths = []string{"", "يناير", "فبراير", "مارس", "أبريل", "مايو", "يونيو", "يوليو", "أغسطس", "سبتمبر", "أكتوبر", "نوفمبر", "ديسمبر"}

func arMonthLabel(k string) string {
	t, err := time.Parse("2006-01", k)
	if err != nil {
		return k
	}
	return fmt.Sprintf("%s %d", arMonths[t.Month()], t.Year())
}

// ------------------------------------------------------------------ codes
type sGroup struct {
	Key      string  `json:"key"`
	Code     string  `json:"code"`
	Desc     string  `json:"desc"`
	Section  string  `json:"section"`
	Amount   float64 `json:"amount"`
	NeedTask bool    `json:"needTask"`
	NeedCost bool    `json:"needCost"`
	Task     Pick    `json:"task"`
	Cost     Pick    `json:"cost"`
}

// اقتراح من الشيت نفسه: أقرب صنف ليه أكواد
func (s *storeState) nearest(text string) (string, string, float64) {
	v := tokens(text)
	best, bs := (*sRow)(nil), 0.0
	seen := map[string]bool{}
	for _, x := range s.parsed {
		if x.Task == "" || x.Cost == "" {
			continue
		}
		k := normAr(x.label())
		if seen[k] {
			continue
		}
		seen[k] = true
		if sc := sim(v, tokens(x.label()+" "+x.Desc)); sc > bs {
			best, bs = x, sc
		}
	}
	smem.mu.Lock()
	for _, l := range smem.Learned {
		if sc := sim(v, tokens(l.Text)); sc > bs {
			best, bs = &sRow{Task: l.Task, Cost: l.Cost, Name: l.Text}, sc
		}
	}
	smem.mu.Unlock()
	if best == nil {
		return "", "", 0
	}
	return best.Task, best.Cost, bs
}

func (s *storeState) sectionTask(x *sRow) (bool, string) {
	if x.Section == "" {
		return false, ""
	}
	smem.mu.Lock()
	t, ok := smem.Sections[normAr(x.Section)]
	smem.mu.Unlock()
	return ok && t != "", t
}

func (s *storeState) suggestCodes(x *sRow) (Pick, Pick) {
	smem.mu.Lock()
	m, ok := smem.Codes[x.key]
	smem.mu.Unlock()
	if ok && (m[0] != "" || m[1] != "") {
		return Pick{m[0], "memory", 1}, Pick{m[1], "memory", 1}
	}
	if ok2, tk := s.sectionTask(x); ok2 {
		text := x.label() + " " + x.Desc
		_, c, sc := s.nearest(text)
		cp := Pick{c, "suggest", sc}
		if c == "" || sc < 0.34 {
			cp = s.cl.suggestCost(tk, "", text)
			if cp.Src == "rule" {
				cp.Src = "suggest"
			}
		}
		return Pick{tk, "memory", 1}, cp
	}
	text := x.label() + " " + x.Desc
	t, c, sc := s.nearest(text)
	if sc >= 0.34 {
		return Pick{t, "suggest", sc}, Pick{c, "suggest", sc}
	}
	tp := s.cl.suggestTask(text, "", "")
	cp := s.cl.suggestCost(tp.Code, "", text)
	if tp.Src == "rule" {
		tp.Src = "suggest"
	}
	if cp.Src == "rule" {
		cp.Src = "suggest"
	}
	if t != "" && sc > tp.Score {
		return Pick{t, "suggest", sc}, Pick{c, "suggest", sc}
	}
	return tp, cp
}

func hStorePrepare(w http.ResponseWriter, r *http.Request) {
	var b struct {
		HeaderRow int            `json:"headerRow"`
		Cols      map[string]int `json:"cols"`
		Remarks   []int          `json:"remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		fail(w, err)
		return
	}
	if sst.rows == nil {
		fail(w, fmt.Errorf("اختار الملف الأول"))
		return
	}
	for _, need := range []string{"date", "qty"} {
		if b.Cols[need] < 0 {
			fail(w, fmt.Errorf("لازم تربط عمود %s", map[string]string{"date": "التاريخ", "qty": "الكمية"}[need]))
			return
		}
	}
	if b.Cols["name"] < 0 && b.Cols["code"] < 0 && b.Cols["desc"] < 0 {
		fail(w, fmt.Errorf("لازم تربط اسم الصنف أو كوده"))
		return
	}
	if b.Cols["rate"] < 0 && b.Cols["amount"] < 0 {
		fail(w, fmt.Errorf("لازم تربط السعر أو القيمة"))
		return
	}
	sst.mapping = sMapping{Cols: b.Cols, Remarks: b.Remarks}
	smem.mu.Lock()
	smem.Profiles[storeSignature(sst.headers)] = sst.mapping
	smem.mu.Unlock()
	saveStoreMem()
	sst.parse()

	type mInfo struct {
		Key    string  `json:"key"`
		Label  string  `json:"label"`
		Rows   int     `json:"rows"`
		Amount float64 `json:"amount"`
	}
	mm := map[string]*mInfo{}
	type nd struct {
		Row   int     `json:"row"`
		Raw   string  `json:"raw"`
		Total float64 `json:"total"`
	}
	var noDate []nd
	var total float64
	for _, x := range sst.parsed {
		total += x.amount()
		if !x.HasD {
			if x.Qty != 0 && x.amount() != 0 {
				noDate = append(noDate, nd{x.Row, x.RawD, r2(x.amount())})
			}
			continue
		}
		k := x.Date.Format("2006-01")
		if mm[k] == nil {
			mm[k] = &mInfo{Key: k, Label: arMonthLabel(k)}
		}
		mm[k].Rows++
		mm[k].Amount += x.amount()
	}
	var months []*mInfo
	for _, v := range mm {
		v.Amount = r2(v.Amount)
		months = append(months, v)
	}
	sort.Slice(months, func(i, j int) bool { return months[i].Key < months[j].Key })

	// المجموعات اللي ناقصها أكواد — البرنامج بيكوّدها لوحده (autocode.go)
	sst.groups = map[string]*sGroup{}
	var groups []*sGroup
	ac := sst.newAutoCoder()
	for _, x := range sst.parsed {
		nt, nc := x.Task == "", x.Cost == ""
		if !nt && !nc {
			continue
		}
		g := sst.groups[x.key]
		if g == nil {
			g = &sGroup{Key: x.key, Code: x.Code, Desc: x.label(), NeedTask: nt, NeedCost: nc}
			g.Section = x.Section
			g.Task, g.Cost = ac.code(x)
			sst.groups[x.key] = g
			groups = append(groups, g)
		}
		g.NeedTask = g.NeedTask || nt
		g.NeedCost = g.NeedCost || nc
		g.Amount += x.amount()
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].Amount > groups[j].Amount })
	for _, g := range groups {
		g.Amount = r2(g.Amount)
	}
	srcCount := map[string]int{}
	for _, g := range groups {
		srcCount[g.Task.Src]++
	}
	writeJSON(w, 200, map[string]interface{}{"rows": len(sst.parsed), "amount": r2(total), "months": months, "noDate": noDate,
		"groups": groups, "codes": sst.cl, "codesFrom": sst.clSrc, "autoSrc": srcCount})
}

// ------------------------------------------------------------------ convert
type sOut struct {
	x          *sRow
	date       time.Time
	task, cost string
	remarks    string
}

type sReview struct {
	status, why string
	x           *sRow
	task, cost  string
}

func hStoreConvert(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Months     []string                     `json:"months"`
		NoDateDate string                       `json:"noDateDate"`
		Codes      map[string]map[string]string `json:"codes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		fail(w, err)
		return
	}
	if sst.parsed == nil {
		fail(w, fmt.Errorf("دوس التالي الأول"))
		return
	}
	sel := map[string]bool{}
	for _, m := range b.Months {
		sel[m] = true
	}
	var ndDate time.Time
	useND := false
	if b.NoDateDate != "" {
		if t, err := time.Parse("2006-01-02", b.NoDateDate); err == nil {
			ndDate, useND = t, true
		}
	}
	if len(sel) == 0 && !useND {
		fail(w, fmt.Errorf("اختار شهر واحد على الأقل"))
		return
	}
	// احفظ الأكواد اللي اتراجعت
	smem.mu.Lock()
	for k, v := range b.Codes {
		if v["task"] != "" || v["cost"] != "" {
			old := smem.Codes[k]
			if v["task"] != "" {
				old[0] = v["task"]
			}
			if v["cost"] != "" {
				old[1] = v["cost"]
			}
			smem.Codes[k] = old
		}
	}
	// اتعلم من الشيت نفسه
	for _, x := range sst.parsed {
		if x.Task != "" && x.Cost != "" {
			smem.Learned[normAr(x.label())] = learnedEntry{x.label() + " " + x.Desc, x.Task, x.Cost}
		}
		t := x.Task
		if t == "" {
			if g := b.Codes[x.key]; g != nil {
				t = g["task"]
			}
		}
		if x.Section != "" && t != "" {
			smem.Sections[normAr(x.Section)] = t
		}
	}
	smem.mu.Unlock()
	saveStoreMem()

	ac := sst.newAutoCoder()
	type codeLine struct {
		x          *sRow
		task, cost string
		tSrc, cSrc string
		rows       int
		amt        float64
	}
	codeLines := map[string]*codeLine{}
	var codeOrder []string
	var outs []sOut
	var revs []sReview
	var shItems int
	var shQty, shAmt float64
	exQty, exCost, exDate, noName, noTask, noCost, noRate := 0, 0, 0, 0, 0, 0, 0
	var exAmt float64
	nameHdr := "اسم صنف"
	if j := sst.mapping.Cols["name"]; j >= 0 && j < len(sst.headers) && strings.TrimSpace(sst.headers[j]) != "" {
		nameHdr = strings.TrimSpace(sst.headers[j])
	}
	type mb struct {
		rows     int
		qty, amt float64
	}
	byM := map[string]*mb{}
	taskTot := map[string]float64{}
	for _, x := range sst.parsed {
		d := x.Date
		if !x.HasD {
			if !useND {
				if x.Qty != 0 || x.amount() != 0 {
					exDate++
					raw := x.RawD
					if raw == "" {
						raw = "فاضي"
					}
					revs = append(revs, sReview{"مستبعد", "مفيش تاريخ صحيح (مكتوب: " + raw + ")", x, x.Task, x.Cost})
				}
				continue
			}
			d = ndDate
		} else if !sel[d.Format("2006-01")] {
			continue
		}
		shItems++
		shQty += x.Qty
		shAmt += x.amount()
		task, cost := x.Task, x.Cost
		tSrc, cSrc := "sheet-col", "sheet-col"
		if g := b.Codes[x.key]; g != nil {
			if task == "" && g["task"] != "" {
				task, tSrc = g["task"], "user"
			}
			if cost == "" && g["cost"] != "" {
				cost, cSrc = g["cost"], "user"
			}
		}
		if task == "" || cost == "" {
			var tp, cp Pick
			if g := sst.groups[x.key]; g != nil {
				tp, cp = g.Task, g.Cost
			} else {
				tp, cp = ac.code(x)
			}
			if task == "" {
				task, tSrc = tp.Code, tp.Src
			}
			if cost == "" {
				cost, cSrc = cp.Code, cp.Src
			}
		}
		if x.Qty == 0 {
			exQty++
			exAmt += x.amount()
			revs = append(revs, sReview{"مستبعد", "الكمية صفر", x, task, cost})
			continue
		}
		if x.amount() == 0 {
			exCost++
			revs = append(revs, sReview{"مستبعد", "التكلفة صفر — مفيش سعر للصنف", x, task, cost})
			continue
		}
		if x.Name == "" && sst.mapping.Cols["name"] >= 0 {
			noName++
			revs = append(revs, sReview{"تنبيه", "مالوش " + nameHdr + " — اتنقل باسم الوصف (ممكن يتعمل Resource جديد في PCS)", x, task, cost})
		}
		if !x.HasR && !x.HasA {
			noRate++
		}
		if task == "" {
			noTask++
		}
		if cost == "" {
			noCost++
		}
		var rem []string
		for _, j := range sst.mapping.Remarks {
			if j < len(x.Cells) && strings.TrimSpace(x.Cells[j]) != "" {
				rem = append(rem, strings.TrimSpace(x.Cells[j]))
			}
		}
		outs = append(outs, sOut{x, d, task, cost, strings.Join(rem, " | ")})
		if tSrc != "sheet-col" || cSrc != "sheet-col" {
			cl := codeLines[x.key]
			if cl == nil {
				cl = &codeLine{x: x, task: task, cost: cost, tSrc: tSrc, cSrc: cSrc}
				codeLines[x.key] = cl
				codeOrder = append(codeOrder, x.key)
			}
			cl.rows++
			cl.amt += x.amount()
		}
		k := d.Format("2006-01")
		if byM[k] == nil {
			byM[k] = &mb{}
		}
		byM[k].rows++
		byM[k].qty += x.Qty
		byM[k].amt += x.amount()
		taskTot[task] += x.amount()
	}
	if len(outs) == 0 {
		fail(w, fmt.Errorf("مفيش ولا بند يتنقل في الشهور اللي اخترتها"))
		return
	}
	var oQty, oAmt float64
	for _, o := range outs {
		oQty += o.x.Qty
		oAmt += o.x.amount()
	}
	// ------------------------------------------------ PCS file
	f, err := excelize.OpenReader(bytes.NewReader(embStoreTemplate))
	if err != nil {
		fail(w, err)
		return
	}
	sh := "Data"
	ds, _ := f.NewStyle(&excelize.Style{NumFmt: 14, CustomNumFmt: strPtr("yyyy-mm-dd")})
	for i, o := range outs {
		row := i + 2
		set := func(col int, v interface{}) {
			c, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sh, c, v)
		}
		c, _ := excelize.CoordinatesToCellName(1, row)
		f.SetCellValue(sh, c, o.date)
		f.SetCellStyle(sh, c, c, ds)
		if o.x.Code != "" {
			set(2, o.x.Code)
		}
		set(3, o.x.label())
		if o.x.Unit != "" {
			set(4, o.x.Unit)
		}
		set(5, o.x.Qty)
		set(6, math.Round(o.x.unitCost()*1e6)/1e6)
		set(7, math.Round(o.x.amount()*100)/100)
		if o.task != "" {
			set(8, o.task)
		}
		if o.cost != "" {
			set(9, o.cost)
		}
		if o.remarks != "" {
			set(10, o.remarks)
		}
	}
	f.SetCellValue(sh, "A1", "Date")
	ob, err := f.WriteToBuffer()
	f.Close()
	if err != nil {
		fail(w, err)
		return
	}
	// tag
	var keys []string
	for k := range byM {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tag := keys[0]
	if len(keys) > 1 {
		tag = keys[0] + "_to_" + keys[len(keys)-1]
	}
	// ------------------------------------------------ review file
	rv := excelize.NewFile()
	hs, _ := rv.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"2F5D5A"}, Pattern: 1}})
	put := func(shn string, rows [][]interface{}, widths []float64, hdrRows ...int) {
		rv.NewSheet(shn)
		t := true
		rv.SetSheetView(shn, 0, &excelize.ViewOptions{RightToLeft: &t})
		hr := map[int]bool{0: true}
		for _, h := range hdrRows {
			hr[h] = true
		}
		for i, r := range rows {
			for j, v := range r {
				c, _ := excelize.CoordinatesToCellName(j+1, i+1)
				rv.SetCellValue(shn, c, v)
				if hr[i] {
					rv.SetCellStyle(shn, c, c, hs)
				}
			}
		}
		for j, wd := range widths {
			cn, _ := excelize.ColumnNumberToName(j + 1)
			rv.SetColWidth(shn, cn, cn, wd)
		}
	}
	sum := [][]interface{}{{"البند", "القيمة"}, {"ملف المصدر", sst.name}, {"تاريخ التحويل", time.Now().Format("2006-01-02 15:04")},
		{"عدد البنود اللي اتنقلت", len(outs)}, {"إجمالي القيمة اللي اتنقلت", r2(oAmt)}, {"بنود مستبعدة — الكمية صفر", exQty},
		{"بنود مستبعدة — التكلفة صفر", exCost}, {"بنود مستبعدة — من غير تاريخ", exDate}, {"بنود اتنقلت من غير " + nameHdr, noName},
		{"بنود اتنقلت من غير Task Code", noTask}, {}, {"الشهر", "عدد البنود", "القيمة"}}
	mh := len(sum) - 1
	for _, k := range keys {
		sum = append(sum, []interface{}{arMonthLabel(k), byM[k].rows, r2(byM[k].amt)})
	}
	sum = append(sum, []interface{}{}, []interface{}{"Task Code", "القيمة"})
	th := len(sum) - 1
	var tks []string
	for k := range taskTot {
		tks = append(tks, k)
	}
	sort.Strings(tks)
	for _, k := range tks {
		label := k
		if label == "" {
			label = "(من غير كود)"
		}
		sum = append(sum, []interface{}{label, r2(taskTot[k])})
	}
	put("ملخص", sum, []float64{36, 22, 18}, mh, th)
	var rr [][]interface{}
	hdrs := []interface{}{"الحالة", "السبب", "رقم الصف في شيت المخازن"}
	shown := []int{}
	for _, k := range []string{"date", "code", "name", "desc", "unit", "qty", "rate", "amount"} {
		if j := sst.mapping.Cols[k]; j >= 0 {
			shown = append(shown, j)
			hdrs = append(hdrs, strings.TrimSpace(sst.headers[j]))
		}
	}
	hdrs = append(hdrs, "Task Code", "Cost Code")
	rr = append(rr, hdrs)
	for _, v := range revs {
		line := []interface{}{v.status, v.why, v.x.Row}
		for _, j := range shown {
			val := ""
			if j < len(v.x.Cells) {
				val = strings.TrimSpace(v.x.Cells[j])
			}
			if n, err := strconv.ParseFloat(val, 64); err == nil && j != sst.mapping.Cols["code"] && j != sst.mapping.Cols["date"] {
				line = append(line, n)
			} else if j == sst.mapping.Cols["date"] && v.x.HasD {
				line = append(line, v.x.Date.Format("02/01/2006"))
			} else {
				line = append(line, val)
			}
		}
		line = append(line, v.task, v.cost)
		rr = append(rr, line)
	}
	put("مراجعة", rr, []float64{10, 50, 12, 12, 16, 40, 30, 8, 10, 12, 14, 12, 12})
	// الأكواد اللي البرنامج حطها لوحده
	if len(codeOrder) > 0 {
		sort.SliceStable(codeOrder, func(i, j int) bool { return codeLines[codeOrder[i]].amt > codeLines[codeOrder[j]].amt })
		desc := func(m map[string]*Code, c string) string {
			if v := m[c]; v != nil {
				if v.AR != "" {
					return v.AR
				}
				return v.EN
			}
			return ""
		}
		var byT, byC map[string]*Code
		if sst.cl != nil {
			byT, byC = sst.cl.byT, sst.cl.byC
		}
		cr := [][]interface{}{{"كود الصنف", "الصنف", "القسم", "عدد السطور", "القيمة", "Task Code", "وصف Task", "Cost Code", "وصف Cost", "مصدر الكود"}}
		for _, k := range codeOrder {
			c := codeLines[k]
			src := autoSrcLabel(c.tSrc)
			if c.cSrc != c.tSrc {
				src += " / " + autoSrcLabel(c.cSrc)
			}
			cr = append(cr, []interface{}{c.x.Code, c.x.label(), c.x.Section, c.rows, r2(c.amt), c.task, desc(byT, c.task), c.cost, desc(byC, c.cost), src})
		}
		put("الأكواد", cr, []float64{14, 44, 14, 10, 14, 12, 30, 12, 34, 24})
	}
	rv.DeleteSheet("Sheet1")
	rb, _ := rv.WriteToBuffer()
	rv.Close()
	sst.out, sst.rev, sst.outTag = ob.Bytes(), rb.Bytes(), tag
	sst.outItems, sst.outAmount = len(outs), r2(oAmt)
	var lbls []string
	for _, k := range keys {
		lbls = append(lbls, arMonthLabel(k))
	}
	sst.outLabel = strings.Join(lbls, " + ")

	var bm []map[string]interface{}
	for _, k := range keys {
		bm = append(bm, map[string]interface{}{"label": arMonthLabel(k), "rows": byM[k].rows, "qty": r2(byM[k].qty), "amount": r2(byM[k].amt)})
	}
	writeJSON(w, 200, map[string]interface{}{
		"match":      math.Abs(oAmt+exAmt-shAmt) < 0.01,
		"sheetItems": shItems, "sheetQty": r2(shQty), "sheetAmount": r2(shAmt),
		"items": len(outs), "qty": r2(oQty), "amount": r2(oAmt), "byMonth": bm,
		"noRate": noRate, "noTask": noTask, "noCost": noCost, "excluded": exQty + exCost, "excludedNoDate": exDate,
		"autoCoded": len(codeOrder),
		"previous":  samePeriod("store", tag),
	})
}

func hStoreSave(w http.ResponseWriter, r *http.Request) {
	if sst.out == nil {
		fail(w, fmt.Errorf("حوّل الأول"))
		return
	}
	run := &Run{Kind: "store", Period: sst.outTag, Label: "صرف مخازن " + sst.outLabel, Source: sst.name,
		Items: sst.outItems, Amount: sst.outAmount,
		OutName: "PCS_Store_Issues_" + sst.outTag + ".xlsx", RevName: "Review_Store_Issues_" + sst.outTag + ".xlsx"}
	saveAndRecord(w, run, sst.out, sst.rev)
}

func hStoreReset(w http.ResponseWriter, r *http.Request) {
	smem.mu.Lock()
	smem.Profiles, smem.Codes, smem.Learned, smem.Sections = map[string]sMapping{}, map[string][2]string{}, map[string]learnedEntry{}, map[string]string{}
	smem.mu.Unlock()
	saveStoreMem()
	writeJSON(w, 200, map[string]string{"ok": "1"})
}

func sSafe(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				writeJSON(w, 500, map[string]string{"error": fmt.Sprint("خطأ غير متوقع: ", e)})
			}
		}()
		sst.mu.Lock()
		defer sst.mu.Unlock()
		h(w, r)
	}
}

func registerStore(mux *http.ServeMux) {
	loadStoreMem()
	mux.HandleFunc("/api/load", sSafe(hStoreLoad))
	mux.HandleFunc("/api/sheet", sSafe(hStoreSheet))
	mux.HandleFunc("/api/prepare", sSafe(hStorePrepare))
	mux.HandleFunc("/api/convert", sSafe(hStoreConvert))
	mux.HandleFunc("/api/save", sSafe(hStoreSave))
	mux.HandleFunc("/api/reset-memory", sSafe(hStoreReset))
}

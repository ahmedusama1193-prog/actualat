package main

// المالية: شيت GL في ملف Project Finance (مجمّع لكل المشاريع والشركتين)
// ← فلتر الشركة + المشروع + الفترة ← ملف استيراد PCS (قيود GL) متكود Task / Cost لكل سطر

import (
	"bytes"
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

type finLine struct {
	Row     int
	Date    time.Time
	Acc     string
	AccName string
	Debit   float64
	Credit  float64
	Voucher string
	Desc    string
	Center  string // المركز كما هو في الشيت
	Company string
	Year    string
	Dep     string // عمود الاهلاك
	cKey    string // المركز بعد التنضيف
}

type finState struct {
	mu      sync.Mutex
	name    string
	sheet   string
	lines   []*finLine
	skipped int // سطور مافيهاش مدين ولا دائن
	cl      *CodeLists
	// آخر تحويل
	opts   finOpts
	sel    []*finLine
	codes  map[string][2]string
	out    []byte
	review []byte
	sum    map[string]interface{}
}

var fst = &finState{}

type finOpts struct {
	Company string                `json:"company"`
	Project string                `json:"project"`
	From    string                `json:"from"` // 2025-01
	To      string                `json:"to"`
	Codes   map[string][2]string  `json:"codes"`   // حساب -> task, cost
	Exclude map[string]bool       `json:"exclude"` // حسابات مش هتطلع
	Groups  map[string]bool       `json:"groups,omitempty"`
}

// ------------------------------------------------------------------ memory
type finMemory struct {
	mu       sync.Mutex
	Accounts map[string][2]string `json:"accounts"`
	Exclude  map[string]bool      `json:"exclude"`
	Company  string               `json:"company"`
	Project  string               `json:"project"`
}

var fmem = &finMemory{Accounts: map[string][2]string{}, Exclude: map[string]bool{}}

func fmemPath() string { return filepath.Join(appDir(), "finance_memory.json") }

func loadFinMem() {
	fmem.mu.Lock()
	defer fmem.mu.Unlock()
	if b, err := os.ReadFile(fmemPath()); err == nil {
		json.Unmarshal(b, fmem)
	}
	if fmem.Accounts == nil {
		fmem.Accounts = map[string][2]string{}
	}
	if fmem.Exclude == nil {
		fmem.Exclude = map[string]bool{}
	}
}

func saveFinMem() {
	fmem.mu.Lock()
	defer fmem.mu.Unlock()
	os.MkdirAll(appDir(), 0o755)
	b, _ := json.MarshalIndent(fmem, "", " ")
	os.WriteFile(fmemPath(), b, 0o644)
}

// ------------------------------------------------------------------ reading
var finAliases = map[string][]string{
	"debit":   {"مدين", "debit", "dr"},
	"credit":  {"دائن", "credit", "cr"},
	"desc":    {"وصف الطرف", "البيان", "الوصف", "description", "narration"},
	"center":  {"المركز", "مركز التكلفه", "cost center", "project"},
	"accname": {"اسم الحساب", "account name"},
	"acc":     {"رقم الحساب", "account", "account no", "كود الحساب"},
	"date":    {"التاريخ", "date"},
	"voucher": {"الرقم", "رقم السند", "رقم القيد", "voucher", "voucher no"},
	"company": {"الشركه", "company"},
	"year":    {"السنه", "year"},
	"dep":     {"الاهلاك"},
	"balance": {"الرصيد", "balance"},
}

func finMapHeader(r []string) map[string]int {
	m := map[string]int{}
	for i, h := range r {
		n := normH(h)
		if n == "" {
			continue
		}
		for k, al := range finAliases {
			if _, done := m[k]; done {
				continue
			}
			for _, a := range al {
				if n == normAr(a) {
					m[k] = i
					break
				}
			}
		}
	}
	return m
}

func finHeaderOK(m map[string]int) bool {
	_, d := m["debit"]
	_, c := m["credit"]
	_, a := m["acc"]
	_, n := m["accname"]
	_, t := m["date"]
	return d && c && (a || n) && t
}

func finCenterKey(s string) string { return normAr(s) }

func accStr(s string) string {
	s = strings.TrimSpace(s)
	if f, err := strconv.ParseFloat(s, 64); err == nil && f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return s
}

// بيدوّر على الشيت اللي فيه أعمدة GL (الأولوية لشيت اسمه GL)
func readFinance(data []byte, want string) (string, []string, []*finLine, int, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{RawCellValue: true})
	if err != nil {
		return "", nil, nil, 0, fmt.Errorf("مقدرتش أفتح الملف: %v", err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	order := []string{}
	if want != "" {
		order = append(order, want)
	} else {
		for _, s := range sheets {
			if strings.EqualFold(strings.TrimSpace(s), "GL") {
				order = append(order, s)
			}
		}
		for _, s := range sheets {
			if !strings.EqualFold(strings.TrimSpace(s), "GL") {
				order = append(order, s)
			}
		}
	}
	for _, sh := range order {
		rows, err := f.Rows(sh)
		if err != nil {
			continue
		}
		var hm map[string]int
		var lines []*finLine
		skipped, i := 0, 0
		for rows.Next() {
			i++
			r, _ := rows.Columns()
			if hm == nil {
				if i > 25 {
					break
				}
				if m := finMapHeader(r); finHeaderOK(m) {
					hm = m
				}
				continue
			}
			g := func(k string) string {
				if c, ok := hm[k]; ok && c < len(r) {
					return strings.TrimSpace(r[c])
				}
				return ""
			}
			d, _ := numv(g("debit"))
			c, _ := numv(g("credit"))
			acc, an := accStr(g("acc")), cleanS(g("accname"))
			if acc == "" && an == "" {
				continue
			}
			if r2(d) == 0 && r2(c) == 0 {
				skipped++
				continue
			}
			dt, _ := parseDateCell(g("date"))
			ln := &finLine{Row: i, Date: dt, Acc: acc, AccName: an, Debit: d, Credit: c, Voucher: accStr(g("voucher")),
				Desc: cleanS(g("desc")), Center: cleanS(g("center")), Company: cleanS(g("company")), Year: accStr(g("year")), Dep: cleanS(g("dep"))}
			ln.cKey = finCenterKey(ln.Center)
			lines = append(lines, ln)
		}
		rows.Close()
		if hm != nil {
			return sh, sheets, lines, skipped, nil
		}
		if want != "" {
			return "", sheets, nil, 0, fmt.Errorf("الشيت «%s» مافيهوش عناوين GL (التاريخ، رقم الحساب، مدين، دائن)", want)
		}
	}
	return "", sheets, nil, 0, fmt.Errorf("مالقيتش شيت GL في الملف — لازم يكون فيه أعمدة: التاريخ، رقم الحساب، مدين، دائن")
}

// ------------------------------------------------------------------ helpers
func monthOf(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01")
}

func (o finOpts) match(l *finLine) bool {
	if o.Company != "" && l.Company != o.Company {
		return false
	}
	if o.Project != "" && l.cKey != finCenterKey(o.Project) {
		return false
	}
	m := monthOf(l.Date)
	if o.From != "" && m != "" && m < o.From {
		return false
	}
	if o.To != "" && m != "" && m > o.To {
		return false
	}
	return true
}

func finAccKey(l *finLine) string {
	if l.Acc != "" {
		return l.Acc
	}
	return "name:" + normAr(l.AccName)
}

// الكود الافتراضي للحساب: ذاكرة ← جدول الحسابات ← أقرب اسم في قوائم PCS
func (cl *CodeLists) finAccount(key, name string) (string, string, string) {
	fmem.mu.Lock()
	m, ok := fmem.Accounts[key]
	fmem.mu.Unlock()
	if ok && (m[0] != "" || m[1] != "") {
		return m[0], m[1], "memory"
	}
	if s, ok := finSeed[key]; ok {
		return s.task, s.cost, "rule"
	}
	// مالوش قاعدة: أقرب Task / Cost باسم الحساب
	t, ts := cl.nearestCode(cl.Task, name)
	c, cs := cl.nearestCode(cl.Cost, name)
	if ts >= 0.34 && cs >= 0.34 {
		return t, c, "suggest"
	}
	return "", "", "none"
}

func (cl *CodeLists) nearestCode(list []*Code, text string) (string, float64) {
	v := tokens(text)
	best, bs := "", 0.0
	for _, c := range list {
		if s := sim(v, c.vec); s > bs {
			best, bs = c.Code, s
		}
	}
	return best, bs
}

// كود السطر: كود الحساب، إلا لو البيان فيه كلمة بتغيّره (والحساب مش متغير من المستخدم)
func finLineCodes(l *finLine, accT, accC string, user bool) (string, string, bool) {
	if user {
		return accT, accC, false
	}
	d := normAr(l.Desc)
	for _, r := range finKw {
		if r.acc != l.Acc {
			continue
		}
		for _, w := range r.words {
			if strings.Contains(d, w) {
				t, c := accT, accC
				if r.task != "" {
					t = r.task
				}
				if r.cost != "" {
					c = r.cost
				}
				return t, c, t != accT || c != accC
			}
		}
	}
	return accT, accC, false
}

func finGroupOf(acc string) string {
	if s, ok := finSeed[acc]; ok {
		return s.group
	}
	return ""
}

// ------------------------------------------------------------------ API
func finInfo() map[string]interface{} {
	type proj struct {
		Name   string  `json:"name"`
		Lines  int     `json:"lines"`
		Amount float64 `json:"amount"`
		From   string  `json:"from"`
		To     string  `json:"to"`
	}
	comp := map[string]map[string]*proj{}
	cLines := map[string]int{}
	minM, maxM := "", ""
	for _, l := range fst.lines {
		c := l.Company
		if comp[c] == nil {
			comp[c] = map[string]*proj{}
		}
		cLines[c]++
		p := comp[c][l.cKey]
		if p == nil {
			p = &proj{Name: l.Center}
			comp[c][l.cKey] = p
		}
		p.Lines++
		p.Amount += l.Debit - l.Credit
		if m := monthOf(l.Date); m != "" {
			if p.From == "" || m < p.From {
				p.From = m
			}
			if m > p.To {
				p.To = m
			}
			if minM == "" || m < minM {
				minM = m
			}
			if m > maxM {
				maxM = m
			}
		}
	}
	type co struct {
		Name     string  `json:"name"`
		Lines    int     `json:"lines"`
		Projects []*proj `json:"projects"`
	}
	var cos []*co
	for c, ps := range comp {
		x := &co{Name: c, Lines: cLines[c]}
		for _, p := range ps {
			p.Amount = r2(p.Amount)
			x.Projects = append(x.Projects, p)
		}
		sort.Slice(x.Projects, func(i, j int) bool { return x.Projects[i].Lines > x.Projects[j].Lines })
		cos = append(cos, x)
	}
	sort.Slice(cos, func(i, j int) bool { return cos[i].Lines > cos[j].Lines })
	months := []map[string]string{}
	if minM != "" {
		t, _ := time.Parse("2006-01", minM)
		for k := minM; k <= maxM; t = t.AddDate(0, 1, 0) {
			k = t.Format("2006-01")
			if k > maxM {
				break
			}
			months = append(months, map[string]string{"key": k, "label": arMonthLabel(k)})
		}
	}
	fmem.mu.Lock()
	lastC, lastP := fmem.Company, fmem.Project
	fmem.mu.Unlock()
	return map[string]interface{}{"file": fst.name, "sheet": fst.sheet, "lines": len(fst.lines), "skipped": fst.skipped,
		"companies": cos, "months": months, "lastCompany": lastC, "lastProject": lastP}
}

func hFinLoad(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(300 << 20)
	file, hdr, err := r.FormFile("file")
	if err != nil {
		fail(w, err)
		return
	}
	defer file.Close()
	data, _ := io.ReadAll(file)
	sheet := r.FormValue("sheet")
	sh, sheets, lines, skipped, err := readFinance(data, sheet)
	if err != nil {
		writeJSON(w, 400, map[string]interface{}{"error": err.Error(), "sheets": sheets})
		return
	}
	fst.name, fst.sheet, fst.lines, fst.skipped = hdr.Filename, sh, lines, skipped
	fst.out, fst.review, fst.sel = nil, nil, nil
	info := finInfo()
	info["sheets"] = sheets
	writeJSON(w, 200, info)
}

type finAcc struct {
	Key     string  `json:"key"`
	Acc     string  `json:"acc"`
	Name    string  `json:"name"`
	Group   string  `json:"group"`
	Lines   int     `json:"lines"`
	Debit   float64 `json:"debit"`
	Credit  float64 `json:"credit"`
	Net     float64 `json:"net"`
	Task    string  `json:"task"`
	Cost    string  `json:"cost"`
	Src     string  `json:"src"`
	Split   int     `json:"split"` // سطور كودها اتغير من البيان
	Sample  string  `json:"sample"`
	Exclude bool    `json:"exclude"`
}

func (o finOpts) selected() []*finLine {
	var out []*finLine
	for _, l := range fst.lines {
		if o.match(l) {
			out = append(out, l)
		}
	}
	return out
}

func hFinPrepare(w http.ResponseWriter, r *http.Request) {
	var o finOpts
	json.NewDecoder(r.Body).Decode(&o)
	if fst.lines == nil {
		fail(w, fmt.Errorf("اختار ملف المالية الأول"))
		return
	}
	if o.Company == "" || o.Project == "" {
		fail(w, fmt.Errorf("اختار الشركة والمشروع"))
		return
	}
	fst.cl, _ = loadCodeLists()
	sel := o.selected()
	accs := map[string]*finAcc{}
	var order []string
	var td, tc float64
	for _, l := range sel {
		k := finAccKey(l)
		a := accs[k]
		if a == nil {
			t, c, src := fst.cl.finAccount(k, l.AccName)
			fmem.mu.Lock()
			ex := fmem.Exclude[k]
			fmem.mu.Unlock()
			a = &finAcc{Key: k, Acc: l.Acc, Name: l.AccName, Group: finGroupOf(l.Acc), Task: t, Cost: c, Src: src, Sample: l.Desc, Exclude: ex}
			accs[k] = a
			order = append(order, k)
		}
		a.Lines++
		a.Debit += l.Debit
		a.Credit += l.Credit
		if _, _, ch := finLineCodes(l, a.Task, a.Cost, a.Src == "memory"); ch {
			a.Split++
		}
		td += l.Debit
		tc += l.Credit
	}
	list := make([]*finAcc, 0, len(order))
	for _, k := range order {
		a := accs[k]
		a.Debit, a.Credit, a.Net = r2(a.Debit), r2(a.Credit), r2(a.Debit-a.Credit)
		list = append(list, a)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Acc < list[j].Acc })
	writeJSON(w, 200, map[string]interface{}{"accounts": list, "lines": len(sel), "debit": r2(td), "credit": r2(tc), "net": r2(td - tc),
		"codes": fst.cl, "previous": samePeriod("fin", finPeriod(o))})
}

func finPeriod(o finOpts) string {
	return "fin:" + o.Company + "|" + strings.TrimSpace(o.Project) + "|" + o.From + ".." + o.To
}

func finPeriodLabel(o finOpts) string {
	if o.From == o.To && o.From != "" {
		return arMonthLabel(o.From)
	}
	a, b := arMonthLabel(o.From), arMonthLabel(o.To)
	if o.From == "" {
		a = "البداية"
	}
	if o.To == "" {
		b = "آخر حركة"
	}
	return a + " ← " + b
}

// نفس أعمدة شيت GL وبنفس الترتيب (قالب PCS اللي بعته أحمد) + عمودين الأكواد في الآخر
var finPcsCols = []string{"الرصيد", "دائن", "مدين", "وصف الطرف", "المركز", "اسم الحساب", "رقم الحساب", "التاريخ", "الرقم", "الشركة", "السنة", "الاهلاك", "Task Code", "Cost Code"}

func hFinConvert(w http.ResponseWriter, r *http.Request) {
	var o finOpts
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		fail(w, err)
		return
	}
	if fst.lines == nil || fst.cl == nil {
		fail(w, fmt.Errorf("اختار الملف والمشروع الأول"))
		return
	}
	// احفظ الأكواد اللي اتغيرت + الحسابات المستبعدة
	fmem.mu.Lock()
	for k, v := range o.Codes {
		if v[0] == "" && v[1] == "" {
			continue
		}
		if s, ok := finSeed[k]; ok && s.task == v[0] && s.cost == v[1] {
			delete(fmem.Accounts, k) // رجع للقاعدة
			continue
		}
		fmem.Accounts[k] = v
	}
	for k := range fmem.Exclude {
		if !o.Exclude[k] {
			delete(fmem.Exclude, k)
		}
	}
	for k, v := range o.Exclude {
		if v {
			fmem.Exclude[k] = true
		}
	}
	fmem.Company, fmem.Project = o.Company, o.Project
	fmem.mu.Unlock()
	saveFinMem()

	sel := o.selected()
	type accSum struct {
		acc, name, task, cost, src string
		lines, split                int
		debit, credit               float64
		ex                          bool
	}
	sums := map[string]*accSum{}
	var keys []string
	var out []*finLine
	codes := map[*finLine][2]string{}
	var srcD, srcC, outD, outC, exD, exC float64
	noT, noC := 0, 0
	for _, l := range sel {
		k := finAccKey(l)
		srcD += l.Debit
		srcC += l.Credit
		s := sums[k]
		if s == nil {
			t, c, src := fst.cl.finAccount(k, l.AccName)
			s = &accSum{acc: l.Acc, name: l.AccName, task: t, cost: c, src: src, ex: o.Exclude[k]}
			sums[k] = s
			keys = append(keys, k)
		}
		s.lines++
		s.debit += l.Debit
		s.credit += l.Credit
		if s.ex {
			exD += l.Debit
			exC += l.Credit
			continue
		}
		t, c, ch := finLineCodes(l, s.task, s.cost, s.src == "memory")
		if ch {
			s.split++
		}
		if t == "" {
			noT++
		}
		if c == "" {
			noC++
		}
		codes[l] = [2]string{t, c}
		out = append(out, l)
		outD += l.Debit
		outC += l.Credit
	}
	if len(out) == 0 {
		fail(w, fmt.Errorf("مفيش سطور تطلع — كل الحسابات مستبعدة أو الفترة فاضية"))
		return
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.Before(out[j].Date)
		}
		return out[i].Row < out[j].Row
	})
	pcs, err := writeFinPCS(out, codes)
	if err != nil {
		fail(w, err)
		return
	}
	// ---- ملف المراجعة
	sort.Strings(keys)
	type tc struct{ t, c string }
	byCode := map[tc]*[3]float64{}
	var codeKeys []tc
	for _, l := range out {
		k := tc{codes[l][0], codes[l][1]}
		if byCode[k] == nil {
			byCode[k] = &[3]float64{}
			codeKeys = append(codeKeys, k)
		}
		byCode[k][0]++
		byCode[k][1] += l.Debit
		byCode[k][2] += l.Credit
	}
	sort.Slice(codeKeys, func(i, j int) bool {
		return codeKeys[i].t+codeKeys[i].c < codeKeys[j].t+codeKeys[j].c
	})
	desc := func(m map[string]*Code, c string) string {
		if x := m[c]; x != nil {
			return x.AR
		}
		return ""
	}
	srcLbl := map[string]string{"memory": "متغير منك", "rule": "قاعدة الحساب", "suggest": "اقتراح", "none": "محتاج كود"}
	rv := excelize.NewFile()
	hs, _ := rv.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"2F5D50"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}})
	nf, _ := rv.NewStyle(&excelize.Style{NumFmt: 4})
	put := func(sh string, rows [][]interface{}, widths []float64, numCols ...int) {
		rv.NewSheet(sh)
		rv.SetSheetView(sh, 0, &excelize.ViewOptions{RightToLeft: boolPtr(true)})
		for i, row := range rows {
			cell, _ := excelize.CoordinatesToCellName(1, i+1)
			rv.SetSheetRow(sh, cell, &row)
		}
		end, _ := excelize.CoordinatesToCellName(len(rows[0]), 1)
		rv.SetCellStyle(sh, "A1", end, hs)
		for i, wd := range widths {
			c, _ := excelize.ColumnNumberToName(i + 1)
			rv.SetColWidth(sh, c, c, wd)
		}
		for _, nc := range numCols {
			c, _ := excelize.ColumnNumberToName(nc)
			rv.SetCellStyle(sh, c+"2", fmt.Sprintf("%s%d", c, len(rows)), nf)
		}
	}
	sumRows := [][]interface{}{{"البند", "القيمة"},
		{"الملف", fst.name}, {"الشيت", fst.sheet}, {"الشركة", o.Company}, {"المشروع", o.Project}, {"الفترة", finPeriodLabel(o)},
		{"سطور الفلتر", len(sel)}, {"مدين الفلتر", r2(srcD)}, {"دائن الفلتر", r2(srcC)}, {"صافي الفلتر", r2(srcD - srcC)},
		{"مستبعد (مدين)", r2(exD)}, {"مستبعد (دائن)", r2(exC)},
		{"سطور ملف PCS", len(out)}, {"مدين ملف PCS", r2(outD)}, {"دائن ملف PCS", r2(outC)}, {"صافي ملف PCS", r2(outD - outC)},
		{"سطور من غير Task Code", noT}, {"سطور من غير Cost Code", noC}}
	put("ملخص", sumRows, []float64{26, 40})
	accRows := [][]interface{}{{"رقم الحساب", "اسم الحساب", "التصنيف", "السطور", "مدين", "دائن", "الصافي", "Task Code", "وصف التاسك", "Cost Code", "وصف الكوست", "المصدر", "سطور كودها من البيان", "الحالة"}}
	for _, k := range keys {
		s := sums[k]
		stt := "طالع"
		if s.ex {
			stt = "مستبعد"
		}
		accRows = append(accRows, []interface{}{s.acc, s.name, finGroupLabel[finGroupOf(s.acc)], s.lines, r2(s.debit), r2(s.credit), r2(s.debit - s.credit),
			s.task, desc(fst.cl.byT, s.task), s.cost, desc(fst.cl.byC, s.cost), srcLbl[s.src], s.split, stt})
	}
	put("الحسابات", accRows, []float64{14, 34, 22, 8, 16, 14, 16, 12, 32, 12, 36, 14, 12, 10}, 5, 6, 7)
	codeRows := [][]interface{}{{"Task Code", "وصف التاسك", "Cost Code", "وصف الكوست", "السطور", "مدين", "دائن", "الصافي"}}
	for _, k := range codeKeys {
		v := byCode[k]
		codeRows = append(codeRows, []interface{}{k.t, desc(fst.cl.byT, k.t), k.c, desc(fst.cl.byC, k.c), int(v[0]), r2(v[1]), r2(v[2]), r2(v[1] - v[2])})
	}
	put("حسب الكود", codeRows, []float64{12, 34, 12, 38, 8, 16, 14, 16}, 6, 7, 8)
	rv.DeleteSheet("Sheet1")
	rb, _ := rv.WriteToBuffer()
	rv.Close()

	fst.opts, fst.sel, fst.out, fst.review = o, out, pcs, rb.Bytes()
	byGroup := map[string]float64{}
	for _, l := range out {
		byGroup[finGroupOf(l.Acc)] += l.Debit - l.Credit
	}
	groups := []map[string]interface{}{}
	for _, g := range []string{"works", "material", "expense", ""} {
		if v, ok := byGroup[g]; ok {
			groups = append(groups, map[string]interface{}{"key": g, "label": finGroupLabel[g], "net": r2(v)})
		}
	}
	fst.sum = map[string]interface{}{"srcLines": len(sel), "srcDebit": r2(srcD), "srcCredit": r2(srcC),
		"outLines": len(out), "outDebit": r2(outD), "outCredit": r2(outC), "exDebit": r2(exD), "exCredit": r2(exC),
		"noTask": noT, "noCost": noC, "groups": groups, "period": finPeriodLabel(o),
		"match": math.Abs((outD+exD)-srcD) < 0.01 && math.Abs((outC+exC)-srcC) < 0.01,
		"previous": samePeriod("fin", finPeriod(o))}
	writeJSON(w, 200, fst.sum)
}

func boolPtr(b bool) *bool { return &b }

func writeFinPCS(lines []*finLine, codes map[*finLine][2]string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sh := "Sheet1"
	hs, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	ds, _ := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr("mm-dd-yy")})
	ns, _ := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr(`_(* #,##0.00_);_(* \(#,##0.00\);_(* "-"??_);_(@_)`)})
	sw, err := f.NewStreamWriter(sh)
	if err != nil {
		return nil, err
	}
	widths := []float64{14, 13, 14, 60, 22, 26, 13, 11, 10, 9, 7, 9, 12, 12}
	for i, wd := range widths {
		sw.SetColWidth(i+1, i+1, wd)
	}
	sw.SetPanes(&excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	hdr := make([]interface{}, len(finPcsCols))
	for i, h := range finPcsCols {
		hdr[i] = excelize.Cell{Value: h, StyleID: hs}
	}
	sw.SetRow("A1", hdr)
	num := func(s string) interface{} {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		return s
	}
	for i, l := range lines {
		c := codes[l]
		var dv interface{} = ""
		if !l.Date.IsZero() {
			dv = excelize.Cell{Value: l.Date, StyleID: ds}
		}
		row := []interface{}{
			excelize.Cell{Value: r2(l.Debit - l.Credit), StyleID: ns},
			excelize.Cell{Value: r2(l.Credit), StyleID: ns},
			excelize.Cell{Value: r2(l.Debit), StyleID: ns},
			l.Desc, l.Center, l.AccName, num(l.Acc), dv, num(l.Voucher), l.Company, num(l.Year), l.Dep, c[0], c[1]}
		for k := range row {
			if v, ok := row[k].(string); ok && v == "" {
				row[k] = nil
			}
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := sw.SetRow(cell, row); err != nil {
			return nil, err
		}
	}
	if err := sw.Flush(); err != nil {
		return nil, err
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func finSafeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "-").Replace(s)
	if len([]rune(s)) > 40 {
		s = string([]rune(s)[:40])
	}
	return s
}

func hFinSave(w http.ResponseWriter, r *http.Request) {
	if fst.out == nil {
		fail(w, fmt.Errorf("دوس تحويل الأول"))
		return
	}
	o := fst.opts
	tag := o.From + "_" + o.To
	if o.From == o.To {
		tag = o.From
	}
	n1 := fmt.Sprintf("PCS_GL_%s_%s_%s.xlsx", finSafeName(o.Project), finSafeName(o.Company), tag)
	n2 := fmt.Sprintf("مراجعة_المالية_%s_%s_%s.xlsx", finSafeName(o.Project), finSafeName(o.Company), tag)
	run := &Run{Kind: "fin", Period: finPeriod(o), Label: "مالية — " + strings.TrimSpace(o.Project) + " · " + o.Company + " · " + finPeriodLabel(o),
		Source: fst.name, Items: len(fst.sel), Amount: fst.sum["outDebit"].(float64) - fst.sum["outCredit"].(float64),
		Extra: fmt.Sprintf("%d سطر", len(fst.sel)), OutName: n1, RevName: n2}
	saveAndRecord(w, run, fst.out, fst.review)
}

func fSafe(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				writeJSON(w, 500, map[string]string{"error": fmt.Sprint("خطأ غير متوقع: ", e)})
			}
		}()
		fst.mu.Lock()
		defer fst.mu.Unlock()
		h(w, r)
	}
}

func registerFinance(mux *http.ServeMux) {
	loadFinMem()
	mux.HandleFunc("/api/fin/load", fSafe(hFinLoad))
	mux.HandleFunc("/api/fin/prepare", fSafe(hFinPrepare))
	mux.HandleFunc("/api/fin/convert", fSafe(hFinConvert))
	mux.HandleFunc("/api/fin/save", fSafe(hFinSave))
}

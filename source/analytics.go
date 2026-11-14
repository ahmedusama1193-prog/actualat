package main

// Dashboard: تحليل الملفات المحفوظة في السجل
// المستودعات: أكتر المواد صرفاً بالفلوس (حسب الصنف / Task Code / الشهر)
// المستخلصات: أكتر الأعمال والبنود والمقاولين صرفاً في الفترة
// البيانات بتتقري من ملفات PCS المتسجلة في قاعدة البيانات — فبتشتغل كمان على الملفات اللي اتحفظت قبل 1.3

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

// سطر واحد من ملف PCS بعد القراية
type anaLine struct {
	Month   string // 2026-10
	Name    string // الصنف أو البند
	Unit    string
	Qty     float64
	Amount  float64
	Task    string
	Cost    string
	Trade   string // المستخلصات: الأعمال
	Contr   string // المستخلصات: المقاول
	Contrno string
}

var (
	anaMu    sync.Mutex
	anaCache = map[uint64][]anaLine{}
	anaCL    *CodeLists
)

func anaForget(id uint64) {
	anaMu.Lock()
	delete(anaCache, id)
	anaMu.Unlock()
}

func anaReset() {
	anaMu.Lock()
	anaCL = nil
	anaMu.Unlock()
}

func anaCodes() *CodeLists {
	anaMu.Lock()
	defer anaMu.Unlock()
	if anaCL == nil {
		anaCL, _ = loadCodeLists()
	}
	return anaCL
}

func cellF(s string) float64 {
	f, _ := numv(s)
	return f
}

func excelMonth(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 20000 && f < 80000 {
		return time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).Add(time.Duration(math.Round(f*86400)) * time.Second).Format("2006-01")
	}
	if t, ok := parseDateCell(s); ok {
		return t.Format("2006-01")
	}
	return ""
}

// قراية ملف PCS المحفوظ (مستودعات أو مستخلصات) لسطور
func anaParse(run *Run, data []byte) ([]anaLine, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sh := "Data"
	if idx, _ := f.GetSheetIndex(sh); idx < 0 {
		sh = f.GetSheetList()[0]
	}
	rows, err := f.GetRows(sh, excelize.Options{RawCellValue: true})
	if err != nil || len(rows) < 2 {
		return nil, err
	}
	col := map[string]int{}
	for j, h := range rows[0] {
		col[strings.ToLower(strings.TrimSpace(h))] = j
	}
	g := func(r []string, k string) string {
		if j, ok := col[k]; ok && j < len(r) {
			return strings.TrimSpace(r[j])
		}
		return ""
	}
	var out []anaLine
	for _, r := range rows[1:] {
		if run.Kind == "store" {
			name := g(r, "resource name")
			amt := cellF(g(r, "amount"))
			if name == "" && amt == 0 {
				continue
			}
			out = append(out, anaLine{Month: excelMonth(g(r, "date")), Name: name, Unit: g(r, "unit"),
				Qty: cellF(g(r, "qty")), Amount: amt, Task: g(r, "task code"), Cost: g(r, "cost code")})
			continue
		}
		desc := g(r, "description")
		if desc == "" {
			continue
		}
		amt := 0.0
		q := 0.0
		if v := g(r, "amount to date"); v != "" {
			amt = cellF(v)
			q = cellF(g(r, "cumulative qty"))
		} else {
			q = cellF(g(r, "qty this ipc"))
			amt = q * cellF(g(r, "rate"))
		}
		out = append(out, anaLine{Month: excelMonth(g(r, "ipc date")), Name: desc, Unit: g(r, "unit"), Qty: q, Amount: amt,
			Task: g(r, "task code"), Cost: g(r, "cost code"), Trade: g(r, "trade"), Contr: g(r, "subcontractor"), Contrno: g(r, "contract no")})
	}
	return out, nil
}

func anaLines(id uint64) (*Run, []anaLine, error) {
	run, o, _, err := getRun(id)
	if err != nil {
		return nil, nil, err
	}
	anaMu.Lock()
	ls, ok := anaCache[id]
	anaMu.Unlock()
	if ok {
		return run, ls, nil
	}
	ls, err = anaParse(run, o)
	if err != nil {
		return run, nil, err
	}
	migrateAnaLines(run, ls)
	anaMu.Lock()
	anaCache[id] = ls
	anaMu.Unlock()
	return run, ls, nil
}

type anaBar struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	Sub    string  `json:"sub,omitempty"`
	Amount float64 `json:"amount"`
	Qty    float64 `json:"qty,omitempty"`
	Unit   string  `json:"unit,omitempty"`
	Lines  int     `json:"lines"`
	Share  float64 `json:"share"`
	EN     string  `json:"en,omitempty"`
	SubEN  string  `json:"subEn,omitempty"`
}

type anaAgg struct {
	m     map[string]*anaBar
	units map[string]map[string]bool
}

func newAgg() *anaAgg { return &anaAgg{m: map[string]*anaBar{}, units: map[string]map[string]bool{}} }

func (a *anaAgg) add(key, label, sub, unit string, amt, qty float64, subEn ...string) {
	k := normAr(key)
	if k == "" {
		k, label = "—", "—"
	}
	b := a.m[k]
	if b == nil {
		b = &anaBar{Key: key, Label: label, Sub: sub}
		if len(subEn) > 0 {
			b.SubEN = subEn[0]
		}
		a.m[k] = b
		a.units[k] = map[string]bool{}
	}
	b.Amount += amt
	b.Qty += qty
	b.Lines++
	if unit != "" {
		a.units[k][unit] = true
	}
}

// top N + الباقي في سطر "باقي"
func (a *anaAgg) top(n int, total float64) ([]anaBar, int) {
	var all []anaBar
	for k, b := range a.m {
		if len(a.units[k]) == 1 {
			for u := range a.units[k] {
				b.Unit = u
			}
		} else {
			b.Qty = 0 // وحدات مختلفة — الكمية مالهاش معنى
		}
		all = append(all, *b)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Amount > all[j].Amount })
	cnt := len(all)
	if n > 0 && len(all) > n {
		var rest anaBar
		rest.Key, rest.Label = "__rest", fmt.Sprintf("باقي %d", len(all)-n)
		for _, b := range all[n:] {
			rest.Amount += b.Amount
			rest.Lines += b.Lines
		}
		all = append(all[:n], rest)
	}
	for i := range all {
		all[i].Amount = r2(all[i].Amount)
		all[i].Qty = r2(all[i].Qty)
		if total != 0 {
			all[i].Share = math.Round(all[i].Amount/total*1000) / 10
		}
	}
	return all, cnt
}

type anaOpt struct {
	ID      uint64  `json:"id"`
	Label   string  `json:"label"`
	Period  string  `json:"period"`
	Mode    string  `json:"mode"` // store | cumulative | current
	Amount  float64 `json:"amount"`
	Created string  `json:"created"`
}

func hAnalytics(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if e := recover(); e != nil {
			writeJSON(w, 500, map[string]string{"error": fmt.Sprint("خطأ غير متوقع: ", e)})
		}
	}()
	q := r.URL.Query()
	kind := q.Get("kind")
	if kind != "ipc" {
		kind = "store"
	}
	topN := 10
	if n, err := strconv.Atoi(q.Get("top")); err == nil && n > 0 && n <= 50 {
		topN = n
	}
	runs, err := listRuns()
	if err != nil {
		fail(w, err)
		return
	}
	// قايمة الفترات المتاحة
	var opts []anaOpt
	for _, rn := range runs {
		if rn.Kind != kind {
			continue
		}
		mode := "store"
		if kind == "ipc" {
			mode = "cumulative"
			if strings.HasPrefix(rn.Period, "current:") {
				mode = "current"
			}
		}
		lbl := strings.TrimPrefix(strings.TrimPrefix(rn.Label, "صرف مخازن "), "مستخلصات — ")
		opts = append(opts, anaOpt{rn.ID, lbl, rn.Period, mode, rn.Amount, rn.Created})
	}
	if opts == nil {
		opts = []anaOpt{}
	}
	resp := map[string]interface{}{"kind": kind, "options": opts}
	if len(opts) == 0 {
		writeJSON(w, 200, resp)
		return
	}
	sel := q.Get("run")
	var lines []anaLine
	label := ""
	if kind == "store" && (sel == "all" || sel == "") {
		// كل الشهور: كل شهر بياخد من أحدث ملف فيه الشهر ده (عشان مايتحسبش مرتين)
		claimed := map[string]uint64{}
		for _, o := range opts {
			_, ls, err := anaLines(o.ID)
			if err != nil {
				continue
			}
			ms := map[string]bool{}
			for _, l := range ls {
				ms[l.Month] = true
			}
			for m := range ms {
				if _, ok := claimed[m]; !ok {
					claimed[m] = o.ID
				}
			}
			for _, l := range ls {
				if claimed[l.Month] == o.ID {
					lines = append(lines, l)
				}
			}
		}
		sel = "all"
		label = "كل الشهور المحفوظة"
	} else {
		id, _ := strconv.ParseUint(sel, 10, 64)
		if id == 0 {
			id = opts[0].ID
		}
		run, ls, err := anaLines(id)
		if err != nil {
			fail(w, err)
			return
		}
		lines = ls
		sel = strconv.FormatUint(id, 10)
		label = strings.TrimPrefix(strings.TrimPrefix(run.Label, "صرف مخازن "), "مستخلصات — ")
	}
	cl := anaCodes()
	taskName := func(c string) (string, string) {
		if t := cl.byT[c]; t != nil {
			return t.AR, t.EN
		}
		return "", ""
	}
	var total float64
	byName, byTask, byMonth, byTrade, byContr := newAgg(), newAgg(), newAgg(), newAgg(), newAgg()
	for _, l := range lines {
		total += l.Amount
		ar, en := taskName(l.Task)
		if kind == "store" {
			byName.add(l.Name, l.Name, ar, l.Unit, l.Amount, l.Qty, en)
		} else {
			byName.add(l.Name+"|"+l.Trade, l.Name, l.Trade, l.Unit, l.Amount, l.Qty)
			byTrade.add(l.Trade, l.Trade, "", "", l.Amount, 0)
			byContr.add(l.Contr, l.Contr, "", "", l.Amount, 0)
		}
		tl := l.Task
		if tl == "" {
			tl = "(من غير كود)"
		}
		byTask.add(l.Task, tl, ar, "", l.Amount, 0)
		if l.Month != "" {
			byMonth.add(l.Month, arMonthLabel(l.Month), "", "", l.Amount, 0)
		}
	}
	tn, tCnt := byName.top(topN, total)
	tt, _ := byTask.top(topN, total)
	for i := range tt {
		_, en := taskName(tt[i].Key)
		tt[i].EN = en
	}
	months, _ := byMonth.top(0, total)
	sort.Slice(months, func(i, j int) bool { return months[i].Key < months[j].Key })
	resp["selected"] = sel
	resp["label"] = label
	resp["total"] = r2(total)
	resp["lines"] = len(lines)
	resp["distinct"] = tCnt
	resp["byName"] = tn
	resp["byTask"] = tt
	resp["byMonth"] = months
	if kind == "ipc" {
		tr, trCnt := byTrade.top(topN, total)
		co, coCnt := byContr.top(topN, total)
		resp["byTrade"] = tr
		resp["byContractor"] = co
		resp["trades"] = trCnt
		resp["contractors"] = coCnt
	}
	writeJSON(w, 200, resp)
}

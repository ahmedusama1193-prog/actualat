package main

// قراءة شيت Sub Contractor من الكوست ريبورت وتحويله لملف استيراد مستخلصات PCS

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

//go:embed assets/ipc_template.xlsx
var embTemplate []byte

var pcsCols = []string{"Subcontractor", "Contract No", "Trade", "Item No", "Description", "Unit", "Rate",
	"Amount To Date", "Contract Qty", "IPC No", "IPC Date", "Cumulative Qty", "Qty This IPC",
	"Task Code", "Cost Code", "Retention %", "VAT %", "Advance Amount", "Advance Recovery %",
	"Retention This IPC", "Penalties This IPC", "Other Deductions This IPC", "Advance Recovered This IPC"}

var srcAliases = map[string][]string{
	"item":       {"item", "البند", "بيان الأعمال", "description"},
	"unit":       {"unit", "الوحدة", "الوحده"},
	"rate":       {"rate", "الفئة", "السعر"},
	"cqty":       {"cont. qty", "contract qty", "كمية العقد"},
	"qty":        {"qty", "الكمية", "الكمية التراكمية"},
	"amount":     {"amount", "القيمة", "الاجمالي"},
	"ret":        {"ret.", "ret", "retention", "نسبة الضمان"},
	"retAmt":     {"ret. amount", "retention amount", "المحتجز"},
	"ipc":        {"invoice no", "ipc no", "رقم المستخلص"},
	"contractor": {"contractor name", "اسم المقاول", "المقاول"},
	"section":    {"project name", "section"},
	"type":       {"type of work"},
	"costCode":   {"cost code"},
	"taskCode":   {"task code"},
	"wp":         {"work_package", "work package"},
	"task":       {"task description"},
}

type Item struct {
	Row      int                `json:"row"`
	Desc     string             `json:"desc"`
	Unit     string             `json:"unit"`
	Rate     float64            `json:"rate"`
	HasRate  bool               `json:"-"`
	CQty     float64            `json:"cqty"`
	Qty      float64            `json:"qty"`
	Amount   float64            `json:"amount"`
	Ret      float64            `json:"ret"`
	HasRet   bool               `json:"-"`
	RetAmt   float64            `json:"retAmt"`
	IPC      int                `json:"ipc"`
	Section  string             `json:"section"`
	Type     string             `json:"type"`
	WP       string             `json:"wp"`
	Task     string             `json:"task"`
	SrcTask  string             `json:"-"`
	SrcCost  string             `json:"-"`
	ByMonth  map[string]float64 `json:"-"`
	groupKey string
}

type Contract struct {
	Row        int     `json:"row"`
	Contractor string  `json:"contractor"`
	Trade      string  `json:"trade"`
	Items      []*Item `json:"-"`
}

type Report struct {
	File      string
	Sheet     string
	Contracts []*Contract
	Months    []string // "2026-10"
	MonthLbl  map[string]string
	Cutoff    string // "2026-10"
	HeaderAmt float64
}

func numv(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

var monAbbr = map[string]int{"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12}
var reMon = regexp.MustCompile(`(?i)^([a-z]{3})[a-z]*[-/ ](\d{2,4})$`)

// "Oct-26" أو رقم تاريخ إكسل -> "2026-10"
func monthKey(s string) string {
	s = strings.TrimSpace(s)
	if m := reMon.FindStringSubmatch(s); m != nil {
		mo, ok := monAbbr[strings.ToLower(m[1])]
		if !ok {
			return ""
		}
		y, _ := strconv.Atoi(m[2])
		if y < 100 {
			y += 2000
		}
		return fmt.Sprintf("%04d-%02d", y, mo)
	}
	if f, ok := numv(s); ok && f > 40000 && f < 60000 {
		t := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).Add(time.Duration(f*24) * time.Hour)
		return t.Format("2006-01")
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Format("2006-01")
	}
	return ""
}

func monthLabel(k string) string {
	t, err := time.Parse("2006-01", k)
	if err != nil {
		return k
	}
	return t.Format("Jan-2006")
}

func cleanS(s string) string {
	return strings.TrimSpace(reSpace.ReplaceAllString(s, " "))
}

func findSubSheet(f *excelize.File) string {
	for _, s := range f.GetSheetList() {
		t := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "-", ""))
		if strings.Contains(t, "subcontractor") || strings.Contains(s, "مقاول") {
			return s
		}
	}
	return ""
}

func ReadReport(name string, data []byte, sheet string) (*Report, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("مش قادر أفتح الملف: %v", err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if sheet == "" {
		sheet = findSubSheet(f)
	}
	if sheet == "" {
		return nil, sheets, fmt.Errorf("مش لاقي شيت Sub Contractor — اختاره من القايمة")
	}
	rows, err := f.GetRows(sheet, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, sheets, err
	}
	hdrI := -1
	for i := 0; i < len(rows) && i < 25; i++ {
		has := map[string]bool{}
		for _, c := range rows[i] {
			has[strings.ToLower(strings.TrimSpace(c))] = true
		}
		if has["item"] && (has["invoice no"] || has["ipc no"]) {
			hdrI = i
			break
		}
	}
	if hdrI < 0 {
		return nil, sheets, fmt.Errorf("مش لاقي صف العناوين (Item / Invoice No) في شيت %s", sheet)
	}
	hdr := rows[hdrI]
	col := map[string]int{}
	for k, names := range srcAliases {
		for _, n := range names {
			for j, h := range hdr {
				if strings.ToLower(strings.TrimSpace(h)) == n {
					col[k] = j
					break
				}
			}
			if _, ok := col[k]; ok {
				break
			}
		}
	}
	for _, need := range []string{"item", "qty", "ipc", "contractor"} {
		if _, ok := col[need]; !ok {
			return nil, sheets, fmt.Errorf("عمود ناقص في شيت المقاولين: %s", need)
		}
	}
	rep := &Report{File: name, Sheet: sheet, MonthLbl: map[string]string{}}
	type mcol struct {
		j int
		k string
	}
	var months []mcol
	minJ := col["task"]
	for j, h := range hdr {
		if j <= minJ {
			continue
		}
		if k := monthKey(h); k != "" {
			months = append(months, mcol{j, k})
			rep.Months = append(rep.Months, k)
			rep.MonthLbl[k] = monthLabel(k)
		}
	}
	// cut-off
	for i := 0; i < hdrI; i++ {
		for j, c := range rows[i] {
			if strings.Contains(strings.ToLower(c), "cut off") {
				for ii := i + 1; ii < hdrI; ii++ {
					if j < len(rows[ii]) {
						if k := monthKey(rows[ii][j]); k != "" {
							rep.Cutoff = k
						}
					}
				}
			}
		}
	}
	if rep.Cutoff == "" && len(rep.Months) > 0 {
		rep.Cutoff = rep.Months[len(rep.Months)-1]
	}
	g := func(r []string, k string) string {
		j, ok := col[k]
		if !ok || j >= len(r) {
			return ""
		}
		return r[j]
	}
	var cur *Contract
	for i := hdrI + 1; i < len(rows); i++ {
		r := rows[i]
		con := cleanS(g(r, "contractor"))
		desc := cleanS(g(r, "item"))
		if con == "" && desc == "" {
			continue
		}
		if strings.HasPrefix(con, "📁") {
			cur = &Contract{Row: i + 1, Contractor: cleanS(strings.TrimPrefix(con, "📁")), Trade: desc}
			rep.Contracts = append(rep.Contracts, cur)
			continue
		}
		if desc == "" {
			continue
		}
		if cur == nil || (con != "" && normAr(con) != normAr(cur.Contractor)) {
			cur = &Contract{Row: i + 1, Contractor: con, Trade: cleanS(g(r, "type"))}
			rep.Contracts = append(rep.Contracts, cur)
		}
		it := &Item{Row: i + 1, Desc: desc, Unit: cleanS(g(r, "unit")), Section: cleanS(g(r, "section")),
			Type: cleanS(g(r, "type")), WP: cleanS(g(r, "wp")), Task: cleanS(g(r, "task")),
			SrcTask: cleanS(g(r, "taskCode")), SrcCost: cleanS(g(r, "costCode")), ByMonth: map[string]float64{}}
		it.Rate, it.HasRate = numv(g(r, "rate"))
		it.CQty, _ = numv(g(r, "cqty"))
		it.Qty, _ = numv(g(r, "qty"))
		it.Amount, _ = numv(g(r, "amount"))
		it.Ret, it.HasRet = numv(g(r, "ret"))
		it.RetAmt, _ = numv(g(r, "retAmt"))
		if v, ok := numv(g(r, "ipc")); ok {
			it.IPC = int(math.Round(v))
		}
		for _, m := range months {
			if m.j < len(r) {
				if v, ok := numv(r[m.j]); ok && v != 0 {
					it.ByMonth[m.k] += v
				}
			}
		}
		cur.Items = append(cur.Items, it)
		rep.HeaderAmt += it.Amount
	}
	return rep, sheets, nil
}

func isDeduction(it *Item) bool {
	d := normAr(it.Desc)
	return d == "خصم" || d == "خصومات" || d == "غرامه" || d == "غرامات" || strings.Contains(normAr(it.Type), "خصومات")
}

// ------------------------------------------------------------------ prepare (codes per group)
type CodeGroup struct {
	Key      string  `json:"key"`
	TaskDesc string  `json:"taskDesc"`
	Trade    string  `json:"trade"`
	Sample   string  `json:"sample"`
	Lines    int     `json:"lines"`
	Amount   float64 `json:"amount"`
	Task     Pick    `json:"task"`
	Cost     Pick    `json:"cost"`
	FromFile bool    `json:"fromFile"`
}

func (rep *Report) Groups(cl *CodeLists) []*CodeGroup {
	idx := map[string]*CodeGroup{}
	var out []*CodeGroup
	for _, c := range rep.Contracts {
		for _, it := range c.Items {
			if isDeduction(it) {
				continue
			}
			td := it.Task
			if td == "" {
				td = c.Trade
			}
			tp := cl.suggestTask(td, it.WP, c.Trade)
			_, bucket := kwBucket(tp.Code, it.Desc+" "+c.Trade)
			key := costKey(td, c.Trade, bucket)
			it.groupKey = key
			gr := idx[key]
			if gr == nil {
				gr = &CodeGroup{Key: key, TaskDesc: td, Trade: c.Trade, Sample: it.Desc, Task: tp}
				gr.Cost = cl.suggestCost(tp.Code, key, it.Desc+" "+c.Trade+" "+td)
				if it.SrcTask != "" && it.SrcCost != "" {
					gr.FromFile = true
				}
				idx[key] = gr
				out = append(out, gr)
			}
			gr.Lines++
			gr.Amount += it.Amount
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Amount > out[j].Amount })
	return out
}

// ------------------------------------------------------------------ convert
type Opts struct {
	Mode  string               `json:"mode"`  // cumulative | current
	Month string               `json:"month"` // for current
	Date  string               `json:"date"`  // IPC date yyyy-mm-dd
	VAT   float64              `json:"vat"`
	Codes map[string][2]string `json:"codes"` // group key -> task,cost
}

type OutRow map[string]interface{}

type ContractSum struct {
	Contractor string  `json:"contractor"`
	ContractNo string  `json:"contractNo"`
	Trade      string  `json:"trade"`
	IPC        int     `json:"ipc"`
	Lines      int     `json:"lines"`
	Works      float64 `json:"works"`
	Deduct     float64 `json:"deduct"`
	Net        float64 `json:"net"`
	RetPct     float64 `json:"retPct"`
	Retention  float64 `json:"retention"`
}

type Skip struct {
	Contractor string  `json:"contractor"`
	Row        int     `json:"row"`
	Desc       string  `json:"desc"`
	Amount     float64 `json:"amount"`
	Why        string  `json:"why"`
}

type Result struct {
	Rows       []OutRow          `json:"-"`
	Contracts  []ContractSum     `json:"contracts"`
	Skips      []Skip            `json:"skips"`
	Warnings   []Skip            `json:"warnings"`
	SheetNet   float64           `json:"sheetNet"` // صافي الأعمال في الشيت (بعد الخصومات) للمقاولين اللي اتنقلوا + اللي اتشالوا
	SheetRet   float64           `json:"sheetRet"`
	OutWorks   float64           `json:"outWorks"`
	OutDeduct  float64           `json:"outDeduct"`
	OutRet     float64           `json:"outRet"`
	SkipAmount float64           `json:"skipAmount"`
	Match      bool              `json:"match"`
	Mode       string            `json:"mode"`
	Label      string            `json:"label"`
	newCnos    map[string]string // أرقام عقود جديدة تتحفظ في الذاكرة لما الملف يتحفظ
}

// rememberContractNos: يثبت أرقام العقود الجديدة بعد الحفظ
func (res *Result) rememberContractNos() {
	if len(res.newCnos) == 0 {
		return
	}
	mem.mu.Lock()
	if mem.Contract == nil {
		mem.Contract = map[string]string{}
	}
	for k, v := range res.newCnos {
		mem.Contract[k] = v
	}
	mem.mu.Unlock()
	saveMem()
}

var reCodeLike = regexp.MustCompile(`^[A-Za-z]{3}-`)

func r2(f float64) float64 { return math.Round(f*100) / 100 }
func r6(f float64) float64 { return math.Round(f*1e6) / 1e6 }

// contractTask: Task Code لأول بند أعمال في العقد
func contractTask(c *Contract, o Opts) string {
	for _, it := range c.Items {
		if !isDeduction(it) {
			if v, ok := o.Codes[it.groupKey]; ok && v[0] != "" {
				return v[0]
			}
			break
		}
	}
	return "GEN"
}

// contractNos: رقم عقد لكل عقد في الشيت (كلهم، حتى اللي مالوش حركة الشهر ده)
// القاعدة: رقم العقد مايتكررش لأكتر من شركة. لو الرقم اتاخد قبل كده (في الملف
// أو في الذاكرة) لشركة تانية، بنزود -2، -3 ...، والرقم بيتحفظ عشان يفضل ثابت كل شهر.
func (rep *Report) contractNos(o Opts) ([]string, map[string]string) {
	mem.mu.Lock()
	defer mem.mu.Unlock()
	fresh := map[string]string{}
	owner := map[string]string{} // cno -> مفتاح العقد صاحبه
	for k, v := range mem.Contract {
		owner[v] = k
	}
	occ := map[string]int{}
	keys := make([]string, len(rep.Contracts))
	out := make([]string, len(rep.Contracts))
	used := map[string]bool{}
	for i, c := range rep.Contracts {
		b := normAr(c.Contractor) + "|" + normAr(c.Trade)
		occ[b]++
		keys[i] = fmt.Sprintf("%s#%d", b, occ[b])
	}
	// 1) الأرقام المحفوظة من قبل
	for i, k := range keys {
		if v, ok := mem.Contract[k]; ok && !used[v] {
			out[i] = v
			used[v] = true
		}
	}
	// 2) عقود جديدة
	for i, c := range rep.Contracts {
		if out[i] != "" {
			continue
		}
		base := "SC-" + contractTask(c, o)
		cno := base
		for n := 2; used[cno] || (owner[cno] != "" && owner[cno] != keys[i]); n++ {
			cno = fmt.Sprintf("%s-%d", base, n)
		}
		out[i] = cno
		used[cno] = true
		owner[cno] = keys[i]
		fresh[keys[i]] = cno
	}
	return out, fresh
}

func (rep *Report) Convert(cl *CodeLists, o Opts) *Result {
	res := &Result{Mode: o.Mode}
	if o.VAT == 0 {
		o.VAT = 15
	}
	cur := o.Mode == "current"
	if cur {
		res.Label = "حالي " + monthLabel(o.Month)
	} else {
		res.Label = "إجمالي حتى " + o.Date
	}
	cnos, fresh := rep.contractNos(o)
	res.newCnos = fresh
	for ci, c := range rep.Contracts {
		// قيم البند حسب الاختيار
		qtyOf := func(it *Item) float64 {
			if cur {
				return it.ByMonth[o.Month]
			}
			return it.Qty
		}
		amtOf := func(it *Item) float64 {
			if cur {
				rate := it.Rate
				if !it.HasRate && it.Qty != 0 {
					rate = it.Amount / it.Qty
				}
				return qtyOf(it) * rate
			}
			return it.Amount
		}
		var gross, ded, ret float64
		ipc := 0
		moved := false
		var rets []float64
		for _, it := range c.Items {
			a := amtOf(it)
			if isDeduction(it) {
				ded += -a
			} else {
				gross += a
				if it.HasRet {
					rets = append(rets, it.Ret)
				}
			}
			if qtyOf(it) != 0 {
				moved = true
			}
			if it.IPC > ipc {
				ipc = it.IPC
			}
			if cur {
				if it.HasRet {
					ret += a * it.Ret
				}
			} else {
				ret += it.RetAmt
			}
		}
		res.SheetNet += gross - ded
		res.SheetRet += ret
		if !moved || ipc == 0 {
			if gross != 0 {
				why := "مفيش رقم مستخلص"
				if !moved {
					why = "مفيش كميات"
				}
				res.Skips = append(res.Skips, Skip{c.Contractor, c.Row, c.Trade, r2(gross - ded), why})
				res.SkipAmount += gross - ded
			}
			continue
		}
		// نسبة المحتجز الأغلب
		retPct := 0.0
		if len(rets) > 0 {
			cnt := map[float64]int{}
			for _, x := range rets {
				cnt[x]++
			}
			bc := -1
			for x, n := range cnt {
				if n > bc {
					retPct, bc = x, n
				}
			}
		}
		if retPct <= 1 {
			retPct *= 100
		}
		cno := cnos[ci]
		// تمييز البنود بنفس الوصف
		dcount := map[string]int{}
		for _, it := range c.Items {
			if !isDeduction(it) {
				dcount[normAr(it.Desc)]++
			}
		}
		dseen := map[string]int{}
		used := map[string]bool{}
		first := true
		lines := 0
		var works float64
		for _, it := range c.Items {
			if isDeduction(it) {
				continue
			}
			q := qtyOf(it)
			a := amtOf(it)
			rate := it.Rate
			if (!it.HasRate || rate == 0) && it.Qty != 0 {
				rate = it.Amount / it.Qty
			}
			if cur && q == 0 {
				continue
			}
			if !cur && rate == 0 && it.Amount == 0 && it.Qty == 0 {
				res.Skips = append(res.Skips, Skip{c.Contractor, it.Row, it.Desc, 0, "بند بدون فئة وبدون كمية"})
				continue
			}
			itemNo := ""
			k := normAr(it.Desc)
			if dcount[k] > 1 {
				dseen[k]++
				tag := it.Section
				if tag == "" || reCodeLike.MatchString(tag) {
					tag = fmt.Sprintf("%d", dseen[k])
				}
				itemNo = tag
				for n := 2; used[k+"|"+itemNo]; n++ {
					itemNo = fmt.Sprintf("%s-%d", tag, n)
				}
				used[k+"|"+itemNo] = true
			}
			codes := o.Codes[it.groupKey]
			tc, cc := codes[0], codes[1]
			if it.SrcTask != "" {
				tc = it.SrcTask
			}
			if it.SrcCost != "" {
				cc = it.SrcCost
			}
			if !cur && math.Abs(it.Amount-it.Qty*rate) > 0.5 {
				res.Warnings = append(res.Warnings, Skip{c.Contractor, it.Row, it.Desc, r2(it.Amount), fmt.Sprintf("القيمة ≠ الكمية × الفئة (فرق %.2f) — اتنقلت القيمة زي ما هي", it.Amount-it.Qty*rate)})
			}
			if q < 0 {
				res.Warnings = append(res.Warnings, Skip{c.Contractor, it.Row, it.Desc, r2(a), "كمية بالسالب — راجعها قبل الاستيراد"})
			}
			row := OutRow{
				"Subcontractor": c.Contractor, "Contract No": cno, "Trade": c.Trade, "Description": it.Desc,
				"Unit": it.Unit, "Rate": r6(rate), "IPC No": ipc, "IPC Date": o.Date,
				"Task Code": tc, "Cost Code": cc, "Retention %": r6(retPct), "VAT %": o.VAT,
			}
			if itemNo != "" {
				row["Item No"] = itemNo
			}
			if it.CQty > 0 {
				row["Contract Qty"] = r6(it.CQty)
			}
			if cur {
				row["Qty This IPC"] = r6(q)
			} else {
				row["Cumulative Qty"] = r6(q)
				row["Amount To Date"] = r2(a)
			}
			if first {
				row["Retention This IPC"] = r2(ret)
				if ded != 0 {
					row["Other Deductions This IPC"] = r2(ded)
				}
				first = false
			}
			res.Rows = append(res.Rows, row)
			lines++
			works += a
		}
		if lines == 0 {
			continue
		}
		res.OutWorks += works
		res.OutDeduct += ded
		res.OutRet += ret
		res.Contracts = append(res.Contracts, ContractSum{c.Contractor, cno, c.Trade, ipc, lines, r2(works), r2(ded), r2(works - ded), retPct, r2(ret)})
	}
	res.Match = math.Abs(res.SheetNet-(res.OutWorks-res.OutDeduct+res.SkipAmount)) < 1
	res.SheetNet, res.SheetRet = r2(res.SheetNet), r2(res.SheetRet)
	res.OutWorks, res.OutDeduct, res.OutRet, res.SkipAmount = r2(res.OutWorks), r2(res.OutDeduct), r2(res.OutRet), r2(res.SkipAmount)
	return res
}

// ------------------------------------------------------------------ write
func WritePCS(res *Result) ([]byte, error) {
	f, err := excelize.OpenReader(bytes.NewReader(embTemplate))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sh := "Data"
	if idx, _ := f.GetSheetIndex(sh); idx < 0 {
		sh = f.GetSheetList()[0]
	}
	hdr, _ := f.GetRows(sh)
	cols := pcsCols
	if len(hdr) > 0 && len(hdr[0]) > 0 {
		cols = hdr[0]
	}
	dateStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 14, CustomNumFmt: strPtr("yyyy-mm-dd")})
	numStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 4})
	for i, row := range res.Rows {
		for j, h := range cols {
			v, ok := row[h]
			if !ok || v == nil {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			if h == "IPC Date" {
				if t, err := time.Parse("2006-01-02", fmt.Sprint(v)); err == nil {
					f.SetCellValue(sh, cell, t)
					f.SetCellStyle(sh, cell, cell, dateStyle)
					continue
				}
			}
			f.SetCellValue(sh, cell, v)
			if h == "Amount To Date" || h == "Rate" || h == "Retention This IPC" || h == "Other Deductions This IPC" {
				f.SetCellStyle(sh, cell, cell, numStyle)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func strPtr(s string) *string { return &s }

// ملف مراجعة صغير: ملخص + المستخلصات + الملاحظات + ربط الأكواد
func WriteReview(res *Result, groups []*CodeGroup, o Opts, cl *CodeLists) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	hs, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E78"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}})
	nf, _ := f.NewStyle(&excelize.Style{NumFmt: 4})
	put := func(sh string, rows [][]interface{}, widths []float64) {
		f.NewSheet(sh)
		t := true
		f.SetSheetView(sh, 0, &excelize.ViewOptions{RightToLeft: &t})
		for i, r := range rows {
			for j, v := range r {
				c, _ := excelize.CoordinatesToCellName(j+1, i+1)
				f.SetCellValue(sh, c, v)
				if i == 0 {
					f.SetCellStyle(sh, c, c, hs)
				} else if _, ok := v.(float64); ok {
					f.SetCellStyle(sh, c, c, nf)
				}
			}
		}
		for j, w := range widths {
			cn, _ := excelize.ColumnNumberToName(j + 1)
			f.SetColWidth(sh, cn, cn, w)
		}
		f.SetPanes(sh, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	}
	put("الملخص", [][]interface{}{
		{"البند", "القيمة"},
		{"نوع التحويل", res.Label},
		{"صافي الأعمال في شيت المقاولين", res.SheetNet},
		{"بنود الأعمال المنقولة لـ PCS", res.OutWorks},
		{"الخصومات المنقولة (Other Deductions)", res.OutDeduct},
		{"صافي PCS = البنود − الخصومات", r2(res.OutWorks - res.OutDeduct)},
		{"مقاولين اتشالوا (مفيش مستخلص)", res.SkipAmount},
		{"الفرق", r2(res.SheetNet - (res.OutWorks - res.OutDeduct + res.SkipAmount))},
		{"المحتجز المنقول", res.OutRet},
		{"عدد المستخلصات", float64(len(res.Contracts))},
		{"عدد السطور", float64(len(res.Rows))},
	}, []float64{42, 22})
	rows := [][]interface{}{{"المقاول", "Contract No", "الأعمال", "رقم المستخلص", "عدد السطور", "قيمة البنود", "الخصومات", "الصافي", "نسبة المحتجز %", "المحتجز"}}
	for _, c := range res.Contracts {
		rows = append(rows, []interface{}{c.Contractor, c.ContractNo, c.Trade, c.IPC, c.Lines, c.Works, c.Deduct, c.Net, c.RetPct, c.Retention})
	}
	put("المستخلصات", rows, []float64{42, 16, 40, 10, 8, 16, 12, 16, 10, 14})
	rows = [][]interface{}{{"وصف التاسك في الشيت", "عنوان العقد", "مثال بند", "Task Code", "وصف التاسك PCS", "Cost Code", "وصف الكوست PCS", "عدد السطور", "القيمة"}}
	for _, g := range groups {
		v := o.Codes[g.Key]
		td, cd := "", ""
		if c := cl.byT[v[0]]; c != nil {
			td = c.AR
		}
		if c := cl.byC[v[1]]; c != nil {
			cd = c.AR
		}
		rows = append(rows, []interface{}{g.TaskDesc, g.Trade, g.Sample, v[0], td, v[1], cd, g.Lines, r2(g.Amount)})
	}
	put("ربط الأكواد", rows, []float64{28, 34, 40, 12, 30, 12, 36, 8, 14})
	rows = [][]interface{}{{"النوع", "المقاول", "صف الشيت", "البند", "القيمة", "الملاحظة"}}
	for _, w := range res.Warnings {
		rows = append(rows, []interface{}{"مراجعة", w.Contractor, w.Row, w.Desc, w.Amount, w.Why})
	}
	for _, s := range res.Skips {
		rows = append(rows, []interface{}{"اتشال", s.Contractor, s.Row, s.Desc, s.Amount, s.Why})
	}
	put("ملاحظات", rows, []float64{10, 40, 10, 50, 14, 50})
	f.DeleteSheet("Sheet1")
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

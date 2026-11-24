package main

// 2.27 — المستخلصات من «دفتر الشركة» (Company Books / GL): كل قيد على حساب مقاولين = مستخلص.
// البرنامج بيطلّع من البيان: اسم المقاول، نوع الأعمال، رقم المستخلص، رقم الفاتورة،
// ويقسّم المقاول لأكتر من عقد لو ترقيم المستخلصات بدأ من 1 تاني، ويطلّع قالب مستخلصات PCS
// بالمبالغ زي ما هي (كل قيد = بند مقطوعية LS بكمية 1).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

type glLine struct {
	I        int     `json:"i"`
	Row      int     `json:"row"`
	Date     string  `json:"date"`
	Voucher  string  `json:"voucher"`
	Account  string  `json:"account"`
	Desc     string  `json:"desc"`
	Task     string  `json:"task"`
	Cost     string  `json:"cost"`
	Amount   float64 `json:"amount"`
	Name     string  `json:"name"`     // اسم المقاول زي ما اتقرا من البيان
	Work     string  `json:"work"`     // «أعمال …» من البيان
	Invoice  string  `json:"invoice"`  // رقم الفاتورة
	ExplIPC  int     `json:"explIpc"`  // رقم المستخلص المكتوب (0 = مش مكتوب)
	IPC      int     `json:"ipc"`      // رقم المستخلص النهائي
	IPCSrc   string  `json:"ipcSrc"`   // مكتوب / محسوب
	Contract int     `json:"contract"` // رقم العقد في القايمة
	Before   string  `json:"before"`   // اتصدّر قبل كده (تاريخ)
	core     map[string]bool
}

type glContract struct {
	ID         int     `json:"id"`
	Contractor string  `json:"contractor"`
	Trade      string  `json:"trade"`
	ContractNo string  `json:"contractNo"`
	Lines      int     `json:"lines"`
	IPCs       int     `json:"ipcs"`
	Amount     float64 `json:"amount"`
	Known      bool    `json:"known"` // رقم العقد من الذاكرة أو مكتوب بإيدك
	user       bool
	key        string
}

type glState struct {
	mu        sync.Mutex
	name      string
	data      []byte
	sheets    []string
	sheet     string
	cols      map[string]int
	lines     []*glLine
	contracts []*glContract
	out, rev  []byte
	newCnos   map[string]string
	skipped   int
	project   string
}

var gls = &glState{}

var glHead = map[string][]string{
	"date":    {"date", "التاريخ", "تاريخ", "تاريخ القيد", "posting date"},
	"voucher": {"voucher", "voucher no", "entry", "entry no", "je", "رقم القيد", "القيد", "قيد", "سند", "رقم السند"},
	"account": {"account", "account name", "الحساب", "اسم الحساب", "حساب"},
	"desc":    {"description", "desc", "narration", "memo", "البيان", "بيان", "الوصف", "الشرح", "شرح"},
	"task":    {"task code", "task", "تاسك", "تاسك كود"},
	"cost":    {"cost code", "cost", "كوست كود", "كود التكلفه"},
	"debit":   {"debit", "dr", "مدين"},
	"credit":  {"credit", "cr", "دائن"},
	"net":     {"debit − credit", "debit - credit", "debit-credit", "net", "الصافي", "صافي", "مدين - دائن", "amount", "المبلغ", "القيمه"},
}

func glHeadKind(s string) string {
	c := cellNorm(s)
	if c == "" {
		return ""
	}
	for _, k := range []string{"net", "date", "voucher", "account", "desc", "task", "cost", "debit", "credit"} {
		for _, h := range glHead[k] {
			if c == cellNorm(h) {
				return k
			}
		}
	}
	return ""
}

func glDetect(rows [][]string) (int, map[string]int) {
	for ri := 0; ri < len(rows) && ri < 30; ri++ {
		c := map[string]int{}
		for ci, v := range rows[ri] {
			if k := glHeadKind(v); k != "" {
				if _, ok := c[k]; !ok {
					c[k] = ci
				}
			}
		}
		_, hasD := c["desc"]
		_, hasN := c["net"]
		_, hasDr := c["debit"]
		if hasD && (hasN || hasDr) {
			for _, k := range []string{"date", "voucher", "account", "desc", "task", "cost", "debit", "credit", "net"} {
				if _, ok := c[k]; !ok {
					c[k] = -1
				}
			}
			return ri, c
		}
	}
	return -1, nil
}

// ---------------------------------------------------------------- قراءة البيان
var (
	reGlIPC1   = regexp.MustCompile(`مستخلص\s*(?:رقم|رقم:|no\.?|#)?\s*0*(\d{1,3})\b`)
	reGlIPC2   = regexp.MustCompile(`(?:^|\s)ف\s*\d+\s*م\s*0*(\d{1,3})\b`)
	reGlInv1   = regexp.MustCompile(`ف\s*رقم\s*(\d+)`)
	reGlInv2   = regexp.MustCompile(`(?:^|\s)ف\s*(\d+)`)
	reGlComp   = regexp.MustCompile(`(مؤسسه|شركه|مكتب|مصنع|ورشه|مجموعه|est\b|co\b|company|contracting)`)
	reGlSplit  = regexp.MustCompile(`\s*[-–—|]\s*`)
	reGlDigits = regexp.MustCompile(`\d`)
)

var glStop = map[string]bool{}

func init() {
	for _, w := range strings.Fields("مؤسسه شركه للمقاولات المقاولات مقاولات العامه العامة للمقاولات والمقاولات للتجاره التجاريه والتجاره المحدوده ذ م مشروع est co company contracting general trading ltd llc ال و") {
		glStop[normAr(w)] = true
	}
}

func glCore(name string) map[string]bool {
	m := map[string]bool{}
	for _, w := range cdWords(name) {
		w = normAr(w)
		if !glStop[w] && utf8.RuneCountInString(w) > 1 && !reGlDigits.MatchString(w) {
			m[w] = true
		}
	}
	return m
}

func subset(a, b map[string]bool) bool {
	if len(a) == 0 {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// اسم المقاول + نوع الأعمال من البيان (normalized text لكن بنرجّع الأصل)
func glParseDesc(desc, project string) (name, work string) {
	segs := reGlSplit.Split(strings.TrimSpace(desc), -1)
	for _, s := range segs {
		n := normAr(s)
		if loc := reGlComp.FindStringIndex(n); loc != nil {
			// النص من أول «مؤسسة/شركة» — على النص الأصلي بنفس عدد الكلمات
			before := strings.Fields(n[:loc[0]])
			ws := strings.Fields(s)
			if len(before) <= len(ws) {
				name = strings.Join(ws[len(before):], " ")
				if len(before) > 0 && work == "" && strings.HasPrefix(normAr(ws[0]), "اعمال") {
					work = strings.Join(ws[:len(before)], " ")
				}
			}
			break
		}
	}
	for _, s := range segs {
		n := normAr(strings.TrimSpace(s))
		if i := strings.Index(n, "اعمال"); i >= 0 && work == "" {
			ws := strings.Fields(s)
			nw := strings.Fields(n)
			for k, x := range nw {
				if strings.HasPrefix(x, "اعمال") {
					work = strings.Join(ws[k:], " ")
					break
				}
			}
			if loc := reGlComp.FindStringIndex(normAr(work)); loc != nil {
				cut := len(strings.Fields(normAr(work)[:loc[0]]))
				work = strings.Join(strings.Fields(work)[:cut], " ")
			}
		}
	}
	if name == "" { // مقطع من غير أرقام، مش أعمال، ومش اسم المشروع
		for i := len(segs) - 1; i >= 1; i-- {
			s := strings.TrimSpace(segs[i])
			n := normAr(s)
			if n == "" || reGlDigits.MatchString(n) || strings.HasPrefix(n, "اعمال") || n == project ||
				strings.Contains(n, "مستخلص") || strings.Contains(n, "مشتريات") || strings.Contains(n, "قيمه") || strings.Contains(n, "مقابل") ||
				strings.TrimPrefix(n, "مشروع ") == project || utf8.RuneCountInString(n) < 3 {
				continue
			}
			name = s
			break
		}
	}
	// «مؤسسة مؤسسة …» ← مرة واحدة
	ws := strings.Fields(name)
	var clean []string
	for i, w := range ws {
		if i > 0 && normAr(w) == normAr(ws[i-1]) {
			continue
		}
		clean = append(clean, w)
	}
	name = strings.TrimSpace(strings.Join(clean, " "))
	work = strings.TrimSpace(strings.Trim(work, "-–— "))
	// «أعمال دفان المرسى» ← من غير اسم المشروع في الآخر
	if project != "" {
		ws := strings.Fields(work)
		for len(ws) > 1 {
			last := normAr(ws[len(ws)-1])
			if last == project || strings.TrimSuffix(last, "ي") == strings.TrimSuffix(project, "ي") || last == "مشروع" {
				ws = ws[:len(ws)-1]
				continue
			}
			break
		}
		work = strings.Join(ws, " ")
	}
	return
}

func glIPC(desc string) int {
	n := normAr(desc)
	for _, re := range []*regexp.Regexp{reGlIPC1, reGlIPC2} {
		if m := re.FindStringSubmatch(n); m != nil {
			v, _ := strconv.Atoi(m[1])
			return v
		}
	}
	return 0
}

func glInvoice(desc string) string {
	n := normAr(desc)
	if m := reGlInv1.FindStringSubmatch(n); m != nil {
		return "ف " + m[1]
	}
	if m := reGlInv2.FindStringSubmatch(n); m != nil {
		return "ف " + m[1]
	}
	return ""
}

func glDate(s string) string {
	s = strings.TrimSpace(s)
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 20000 && f < 80000 {
		t, _ := excelize.ExcelDateToTime(f, false)
		return t.Format("2006-01-02")
	}
	for _, l := range []string{"2006-01-02", "2006-01-02 15:04:05", "02/01/2006", "2/1/2006", "01-02-06", "1/2/06", "02-01-2006", "2006/01/02"} {
		if t, err := time.Parse(l, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return s
}

// ---------------------------------------------------------------- التحميل
func (g *glState) load() error {
	f, err := excelize.OpenReader(bytes.NewReader(g.data))
	if err != nil {
		return fmt.Errorf("مقدرتش أفتح الملف — لازم يكون xlsx")
	}
	defer f.Close()
	g.sheets = f.GetSheetList()
	pick := g.sheet
	var rows [][]string
	var head int
	var cols map[string]int
	try := func(sh string) bool {
		r, err := f.GetRows(sh, excelize.Options{RawCellValue: true})
		if err != nil {
			return false
		}
		h, c := glDetect(r)
		if h < 0 {
			return false
		}
		rows, head, cols, pick = r, h, c, sh
		return true
	}
	ok := false
	if pick != "" {
		ok = try(pick)
	}
	if !ok {
		// الأولوية لشيت اسمه فيه Subcontract / مقاول
		names := append([]string{}, g.sheets...)
		sort.SliceStable(names, func(i, j int) bool {
			a := strings.Contains(strings.ToLower(names[i]), "subcon") || strings.Contains(names[i], "مقاول")
			b := strings.Contains(strings.ToLower(names[j]), "subcon") || strings.Contains(names[j], "مقاول")
			return a && !b
		})
		for _, sh := range names {
			if try(sh) {
				ok = true
				break
			}
		}
	}
	if !ok {
		return fmt.Errorf("مالقيتش شيت فيه عمود البيان (Description) وعمود المبلغ (Debit / Debit − credit)")
	}
	g.sheet, g.cols = pick, cols
	g.lines, g.skipped = nil, 0
	for ri := head + 1; ri < len(rows); ri++ {
		r := rows[ri]
		desc := cell(r, cols["desc"])
		var amt float64
		if v, ok := parseNumCell(cell(r, cols["net"])); ok && cols["net"] >= 0 {
			amt = v
		} else {
			d, _ := parseNumCell(cell(r, cols["debit"]))
			c, _ := parseNumCell(cell(r, cols["credit"]))
			amt = d - c
		}
		if desc == "" && amt == 0 {
			continue
		}
		if amt == 0 || isTotalText(desc) {
			g.skipped++
			continue
		}
		g.lines = append(g.lines, &glLine{Row: ri + 1, Date: glDate(cell(r, cols["date"])), Voucher: cell(r, cols["voucher"]), Account: cell(r, cols["account"]),
			Desc: desc, Task: cell(r, cols["task"]), Cost: cell(r, cols["cost"]), Amount: r2(amt)})
	}
	if len(g.lines) == 0 {
		return fmt.Errorf("الشيت مافيهوش قيود بمبالغ")
	}
	g.analyze()
	return nil
}

func (g *glState) analyze() {
	// اسم المشروع = آخر مقطع الأكتر تكرار (المرسى …)
	cnt := map[string]int{}
	for _, l := range g.lines {
		segs := reGlSplit.Split(strings.TrimSpace(l.Desc), -1)
		last := normAr(strings.TrimSpace(segs[len(segs)-1]))
		last = strings.TrimPrefix(last, "مشروع ")
		if !reGlDigits.MatchString(last) && last != "" {
			cnt[last]++
		}
	}
	best := 0
	g.project = ""
	for k, v := range cnt {
		if v > best && v >= 2 {
			g.project, best = k, v
		}
	}
	// اسم المشروع بالياء/الألف المقصورة
	for _, l := range g.lines {
		l.Name, l.Work = glParseDesc(l.Desc, g.project)
		if p := strings.TrimSuffix(g.project, "ي"); l.Name != "" && (normAr(l.Name) == p+"ي" || normAr(l.Name) == p+"ى") {
			l.Name = ""
		}
		l.ExplIPC = glIPC(l.Desc)
		l.Invoice = glInvoice(l.Desc)
		l.core = glCore(l.Name)
	}
	// تجميع المقاولين: الاسم الأقصر جوه الأطول = نفس المقاول
	type grp struct {
		name string
		core map[string]bool
		ls   []*glLine
	}
	var groups []*grp
	sorted := append([]*glLine{}, g.lines...)
	sort.SliceStable(sorted, func(i, j int) bool { return len(sorted[i].core) > len(sorted[j].core) })
	for _, l := range sorted {
		var hit *grp
		if len(l.core) > 0 {
			for _, gr := range groups {
				if len(gr.core) > 0 && (subset(l.core, gr.core) || subset(gr.core, l.core)) {
					hit = gr
					break
				}
			}
		} else {
			for _, gr := range groups {
				if len(gr.core) == 0 {
					hit = gr
					break
				}
			}
		}
		if hit == nil {
			hit = &grp{name: l.Name, core: l.core}
			groups = append(groups, hit)
		}
		if utf8.RuneCountInString(l.Name) > utf8.RuneCountInString(hit.name) {
			hit.name = l.Name
		}
		hit.ls = append(hit.ls, l)
	}
	// العقود: جوه كل مقاول بترتيب التاريخ — لو المستخلص رقم 1 ظهر تاني يبقى عقد جديد
	g.contracts = nil
	for _, gr := range groups {
		name := gr.name
		if name == "" {
			name = "مقاول غير محدد"
		}
		ls := gr.ls
		sort.SliceStable(ls, func(i, j int) bool {
			if ls[i].Date != ls[j].Date {
				return ls[i].Date < ls[j].Date
			}
			a, b := ls[i].ExplIPC, ls[j].ExplIPC
			if a == 0 {
				a = 999
			}
			if b == 0 {
				b = 999
			}
			return a < b
		})
		var cur *glContract
		maxE := 0
		for _, l := range ls {
			if cur == nil || (l.ExplIPC == 1 && maxE >= 1) {
				cur = &glContract{ID: len(g.contracts), Contractor: name}
				g.contracts = append(g.contracts, cur)
				maxE = 0
			}
			l.Contract = cur.ID
			if l.ExplIPC > maxE {
				maxE = l.ExplIPC
			}
		}
	}
	// الأعمال لكل عقد: أول عقد للمقاول = اسم الحساب، والعقود اللي بعده = «أعمال …» من البيان
	seen := map[string]bool{}
	for _, c := range g.contracts {
		var first *glLine
		for _, l := range g.lines {
			if l.Contract == c.ID && (first == nil || l.Date < first.Date) {
				first = l
			}
		}
		acc := strings.TrimSpace(strings.TrimLeft(first.Account, "0123456789 -"))
		if !seen[c.Contractor] && acc != "" {
			c.Trade = acc
		} else if first.Work != "" {
			c.Trade = first.Work
		} else {
			c.Trade = acc
		}
		if c.Trade == "" {
			c.Trade = "أعمال مقاولات"
		}
		seen[c.Contractor] = true
	}
	g.numberIPCs()
	g.assignCnos()
	g.markBefore()
	g.out, g.rev = nil, nil
}

// المستخلص اللي مالوش رقم ياخد (اللي قبله + 1)، ولو الرقم ده مستخدم يبقى جزء من نفس المستخلص اللي قبله
func (g *glState) numberIPCs() {
	for _, c := range g.contracts {
		var ls []*glLine
		used := map[int]bool{}
		for _, l := range g.lines {
			if l.Contract == c.ID {
				ls = append(ls, l)
				if l.ExplIPC > 0 {
					used[l.ExplIPC] = true
				}
			}
		}
		sort.SliceStable(ls, func(i, j int) bool {
			if ls[i].Date != ls[j].Date {
				return ls[i].Date < ls[j].Date
			}
			return ls[i].ExplIPC < ls[j].ExplIPC && ls[i].ExplIPC > 0
		})
		prev := 0
		for _, l := range ls {
			if l.IPCSrc == "user" {
				prev = l.IPC
				continue
			}
			if l.ExplIPC > 0 {
				l.IPC, l.IPCSrc = l.ExplIPC, "مكتوب"
			} else {
				n := prev + 1
				if used[n] && prev > 0 {
					n = prev
				}
				l.IPC, l.IPCSrc = n, "محسوب"
				used[n] = true
			}
			prev = l.IPC
		}
	}
	g.summarize()
}

func (g *glState) summarize() {
	for _, c := range g.contracts {
		c.Lines, c.IPCs, c.Amount = 0, 0, 0
		ip := map[int]bool{}
		for _, l := range g.lines {
			if l.Contract == c.ID {
				c.Lines++
				c.Amount += l.Amount
				ip[l.IPC] = true
			}
		}
		c.IPCs = len(ip)
		c.Amount = r2(c.Amount)
	}
}

func glTaskOf(c *glContract, lines []*glLine) string {
	for _, l := range lines {
		if l.Contract == c.ID && l.Task != "" {
			return l.Task
		}
	}
	return "GEN"
}

// رقم العقد: من الذاكرة لو العقد ده اتعمل قبل كده، وإلا SC-<Task> من غير ما يتكرر لشركتين
func (g *glState) assignCnos() {
	mem.mu.Lock()
	defer mem.mu.Unlock()
	owner := map[string]string{}
	for k, v := range mem.Contract {
		owner[v] = k
	}
	used := map[string]bool{}
	g.newCnos = map[string]string{}
	occ := map[string]int{}
	for _, c := range g.contracts {
		b := normAr(c.Contractor) + "|" + normAr(c.Trade)
		occ[b]++
		c.key = fmt.Sprintf("%s#%d", b, occ[b])
		if c.Known && c.ContractNo != "" {
			used[c.ContractNo] = true
			if c.user {
				g.newCnos[c.key] = c.ContractNo
			}
			continue
		}
		if v, ok := mem.Contract[c.key]; ok && !used[v] {
			c.ContractNo, c.Known = v, true
			used[v] = true
		} else {
			c.ContractNo, c.Known = "", false
		}
	}
	for _, c := range g.contracts {
		if c.ContractNo != "" {
			continue
		}
		base := "SC-" + glTaskOf(c, g.lines)
		cno := base
		for n := 2; used[cno] || (owner[cno] != "" && owner[cno] != c.key); n++ {
			cno = fmt.Sprintf("%s-%d", base, n)
		}
		c.ContractNo = cno
		used[cno] = true
		owner[cno] = c.key
		g.newCnos[c.key] = cno
	}
}

func (g *glState) markBefore() {
	mem.mu.Lock()
	defer mem.mu.Unlock()
	for _, l := range g.lines {
		l.Before = ""
		if l.Voucher != "" && mem.GLVouchers != nil {
			l.Before = mem.GLVouchers[l.Voucher+"|"+strconv.FormatFloat(l.Amount, 'f', 2, 64)]
		}
	}
}

// ---------------------------------------------------------------- الإخراج
type glOpts struct {
	Ret  float64 `json:"ret"`
	VAT  float64 `json:"vat"`
	Unit string  `json:"unit"`
	Skip []int   `json:"skip"` // قيود متشالة
}

func (g *glState) build(o glOpts) (int, float64, error) {
	skip := map[int]bool{}
	for _, i := range o.Skip {
		skip[i] = true
	}
	if o.Unit == "" {
		o.Unit = "LS"
	}
	f, err := excelize.OpenReader(bytes.NewReader(embTemplate))
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	head, _ := f.GetRows("Data")
	cols := head[0]
	ls := append([]*glLine{}, g.lines...)
	sort.SliceStable(ls, func(i, j int) bool {
		if ls[i].Contract != ls[j].Contract {
			return ls[i].Contract < ls[j].Contract
		}
		if ls[i].IPC != ls[j].IPC {
			return ls[i].IPC < ls[j].IPC
		}
		return ls[i].Date < ls[j].Date
	})
	n, total := 0, 0.0
	r := 2
	for _, l := range ls {
		if skip[l.I] {
			continue
		}
		c := g.contracts[l.Contract]
		desc := l.Work
		if desc == "" {
			desc = c.Trade
		}
		if l.IPCSrc == "مكتوب" {
			desc = fmt.Sprintf("مستخلص رقم %d - %s", l.IPC, desc)
		}
		item := l.Invoice
		if item == "" {
			item = l.Voucher
		}
		var dt interface{} = l.Date
		if t, err := time.Parse("2006-01-02", l.Date); err == nil {
			dt = t
		}
		rec := map[string]interface{}{"Subcontractor": c.Contractor, "Contract No": c.ContractNo, "Trade": c.Trade, "Item No": item,
			"Description": desc, "Unit": o.Unit, "Rate": l.Amount, "Amount To Date": l.Amount, "Contract Qty": 1, "IPC No": l.IPC,
			"IPC Date": dt, "Cumulative Qty": 1, "Qty This IPC": 1, "Task Code": l.Task, "Cost Code": l.Cost,
			"Retention %": o.Ret, "VAT %": o.VAT, "Retention This IPC": r2(l.Amount * o.Ret / 100)}
		row := make([]interface{}, len(cols))
		for i, h := range cols {
			row[i] = rec[h]
		}
		cl, _ := excelize.CoordinatesToCellName(1, r)
		f.SetSheetRow("Data", cl, &row)
		r++
		n++
		total += l.Amount
	}
	if n == 0 {
		return 0, 0, fmt.Errorf("مفيش قيود تطلع")
	}
	ds, _ := f.NewStyle(&excelize.Style{NumFmt: 14})
	for i, h := range cols {
		if h == "IPC Date" {
			a, _ := excelize.CoordinatesToCellName(i+1, 2)
			b, _ := excelize.CoordinatesToCellName(i+1, r-1)
			f.SetCellStyle("Data", a, b, ds)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return 0, 0, err
	}
	g.out = buf.Bytes()
	g.rev = g.review(skip)
	return n, r2(total), nil
}

func (g *glState) review(skip map[int]bool) []byte {
	f := excelize.NewFile()
	hs, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"163F33"}, Pattern: 1}})
	s1 := "العقود"
	f.SetSheetName("Sheet1", s1)
	f.SetSheetRow(s1, "A1", &[]string{"Contract No", "المقاول", "الأعمال", "عدد المستخلصات", "عدد القيود", "الإجمالي", "رقم العقد"})
	f.SetCellStyle(s1, "A1", "G1", hs)
	for i, c := range g.contracts {
		k := "جديد"
		if c.Known {
			k = "من الذاكرة"
		}
		cl, _ := excelize.CoordinatesToCellName(1, i+2)
		f.SetSheetRow(s1, cl, &[]interface{}{c.ContractNo, c.Contractor, c.Trade, c.IPCs, c.Lines, c.Amount, k})
	}
	f.SetColWidth(s1, "A", "A", 18)
	f.SetColWidth(s1, "B", "C", 44)
	s2 := "القيود"
	f.NewSheet(s2)
	f.SetSheetRow(s2, "A1", &[]string{"Row", "Date", "Voucher", "Contract No", "المقاول", "IPC No", "رقم المستخلص", "رقم الفاتورة", "Task Code", "Cost Code", "المبلغ", "البيان في الدفتر", "ملاحظة"})
	f.SetCellStyle(s2, "A1", "M1", hs)
	r := 2
	var total float64
	for _, l := range g.lines {
		c := g.contracts[l.Contract]
		note := ""
		if skip[l.I] {
			note = "اتشال (مش في ملف PCS)"
		} else {
			total += l.Amount
		}
		if l.Before != "" {
			note = strings.TrimSpace(note + " · اتصدّر قبل كده " + l.Before)
		}
		cl, _ := excelize.CoordinatesToCellName(1, r)
		f.SetSheetRow(s2, cl, &[]interface{}{l.Row, l.Date, l.Voucher, c.ContractNo, c.Contractor, l.IPC, l.IPCSrc, l.Invoice, l.Task, l.Cost, l.Amount, l.Desc, note})
		r++
	}
	cl, _ := excelize.CoordinatesToCellName(10, r+1)
	f.SetSheetRow(s2, cl, &[]interface{}{"الإجمالي", r2(total)})
	f.SetColWidth(s2, "E", "E", 36)
	f.SetColWidth(s2, "L", "L", 110)
	f.SetColWidth(s2, "M", "M", 30)
	f.SetPanes(s2, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	var buf bytes.Buffer
	f.Write(&buf)
	return buf.Bytes()
}

// ---------------------------------------------------------------- API
func glSafe(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				writeJSON(w, 500, map[string]string{"error": fmt.Sprint("خطأ غير متوقع: ", e)})
			}
		}()
		gls.mu.Lock()
		defer gls.mu.Unlock()
		h(w, r)
	}
}

func (g *glState) info() map[string]interface{} {
	for i, l := range g.lines {
		l.I = i
	}
	var total float64
	before := 0
	for _, l := range g.lines {
		total += l.Amount
		if l.Before != "" {
			before++
		}
	}
	// أرقام العقود المعروفة (للاختيار لو العقد موجود على PCS)
	mem.mu.Lock()
	var known []map[string]string
	for k, v := range mem.Contract {
		known = append(known, map[string]string{"cno": v, "key": strings.SplitN(k, "#", 2)[0]})
	}
	mem.mu.Unlock()
	sort.Slice(known, func(i, j int) bool { return known[i]["cno"] < known[j]["cno"] })
	return map[string]interface{}{"file": g.name, "sheets": g.sheets, "sheet": g.sheet, "lines": g.lines, "contracts": g.contracts,
		"total": r2(total), "skipped": g.skipped, "before": before, "known": known, "project": g.project}
}

func hGlLoad(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(200 << 20)
	file, hdr, err := r.FormFile("file")
	if err != nil {
		fail(w, err)
		return
	}
	defer file.Close()
	gls.data, _ = io.ReadAll(file)
	gls.name, gls.sheet = hdr.Filename, ""
	if err := gls.load(); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, gls.info())
}

func hGlSheet(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Sheet string `json:"sheet"`
	}
	json.NewDecoder(r.Body).Decode(&b)
	gls.sheet = b.Sheet
	if err := gls.load(); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, gls.info())
}

type glEdit struct {
	Contracts []struct {
		ID         int    `json:"id"`
		Contractor string `json:"contractor"`
		Trade      string `json:"trade"`
		ContractNo string `json:"contractNo"`
	} `json:"contracts"`
	Lines []struct {
		I        int `json:"i"`
		Contract int `json:"contract"` // -1 = عقد جديد لنفس المقاول
		IPC      int `json:"ipc"`
	} `json:"lines"`
}

func (g *glState) applyEdits(e glEdit) {
	for _, x := range e.Contracts {
		if x.ID < 0 || x.ID >= len(g.contracts) {
			continue
		}
		c := g.contracts[x.ID]
		if s := strings.TrimSpace(x.Contractor); s != "" {
			c.Contractor = s
		}
		if s := strings.TrimSpace(x.Trade); s != "" {
			c.Trade = s
		}
		if s := strings.TrimSpace(x.ContractNo); s != "" && s != c.ContractNo {
			c.ContractNo, c.Known, c.user = s, true, true
		}
	}
	for _, x := range e.Lines {
		if x.I < 0 || x.I >= len(g.lines) {
			continue
		}
		l := g.lines[x.I]
		if x.Contract == -1 {
			src := g.contracts[l.Contract]
			nc := &glContract{ID: len(g.contracts), Contractor: src.Contractor, Trade: src.Trade}
			g.contracts = append(g.contracts, nc)
			x.Contract = nc.ID
		}
		if x.Contract >= 0 && x.Contract < len(g.contracts) {
			l.Contract = x.Contract
		}
		if x.IPC > 0 {
			l.IPC, l.IPCSrc = x.IPC, "user"
		}
	}
	// أرقام العقود الجديدة: ثبّت اللي اتكتب، وكمّل الباقي
	g.numberIPCs()
	for _, c := range g.contracts {
		if !c.Known {
			c.ContractNo = ""
		}
	}
	g.assignCnos()
	for _, l := range g.lines {
		if l.IPCSrc == "user" {
			l.IPCSrc = "اتعدّل"
		}
	}
}

func hGlConvert(w http.ResponseWriter, r *http.Request) {
	var b struct {
		glEdit
		glOpts
	}
	json.NewDecoder(r.Body).Decode(&b)
	if len(gls.lines) == 0 {
		fail(w, fmt.Errorf("ارفع الملف الأول"))
		return
	}
	if len(b.Contracts) > 0 || len(b.Lines) > 0 {
		gls.applyEdits(b.glEdit)
	}
	n, total, err := gls.build(b.glOpts)
	if err != nil {
		fail(w, err)
		return
	}
	info := gls.info()
	info["outLines"] = n
	info["outTotal"] = total
	writeJSON(w, 200, info)
}

func hGlSave(w http.ResponseWriter, r *http.Request) {
	if gls.out == nil {
		fail(w, fmt.Errorf("دوس تحويل الأول"))
		return
	}
	base := strings.TrimSuffix(gls.name, filepath.Ext(gls.name))
	var total float64
	for _, l := range gls.lines {
		total += l.Amount
	}
	ipcs := 0
	for _, c := range gls.contracts {
		ipcs += c.IPCs
	}
	run := &Run{Kind: "ipc", Period: "gl:" + time.Now().Format("2006-01-02"), Label: "مستخلصات من دفتر الشركة — " + base, Source: gls.name,
		Items: ipcs, Amount: r2(total), Extra: fmt.Sprintf("%d عقد · %d قيد", len(gls.contracts), len(gls.lines)),
		OutName: "PCS_IPC_" + finSafeName(base) + ".xlsx", RevName: "مراجعة_مستخلصات_الدفتر_" + finSafeName(base) + ".xlsx"}
	run.User = reqUser(r)
	cnos := map[string]string{}
	for k, v := range gls.newCnos {
		cnos[k] = v
	}
	var vs []string
	for _, l := range gls.lines {
		if l.Voucher != "" {
			vs = append(vs, l.Voucher+"|"+strconv.FormatFloat(l.Amount, 'f', 2, 64))
		}
	}
	run.onSaved = func() {
		mem.mu.Lock()
		if mem.Contract == nil {
			mem.Contract = map[string]string{}
		}
		for k, v := range cnos {
			mem.Contract[k] = v
		}
		if mem.GLVouchers == nil {
			mem.GLVouchers = map[string]string{}
		}
		d := time.Now().Format("2006-01-02")
		for _, v := range vs {
			mem.GLVouchers[v] = d
		}
		mem.mu.Unlock()
		saveMem()
	}
	saveAndRecord(w, run, gls.out, gls.rev)
}

func registerGLIPC(mux *http.ServeMux) {
	mux.HandleFunc("/api/ipcgl/load", glSafe(hGlLoad))
	mux.HandleFunc("/api/ipcgl/sheet", glSafe(hGlSheet))
	mux.HandleFunc("/api/ipcgl/convert", glSafe(hGlConvert))
	mux.HandleFunc("/api/ipcgl/save", glSafe(hGlSave))
}

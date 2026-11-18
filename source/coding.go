package main

// 2.25 — Coding: تكويد BOQ و Resource List.
//   BOQ: كل بند بيتقري (الوصف + عنوان القسم اللي فوقه) ويتحدد ليه كود MasterFormat 2018 و Task Code من قوايم PCS،
//        وكود البند نفسه بيتعمل من الـ CSI (033000.001 …) — مش من رقم البند في المقايسة.
//   Resources: كل مورد ← نوعه (مواد / عمالة / معدات / مقاول باطن) + CSI + Cost Code
//        (من ذاكرة البرنامج والأصناف المعروفة في المستودعات، وإلا من القواعد).
// الملف الطالع: من غير عناوين أقسام ولا إجماليات — أول صف هو أول بند (وصف عناوين الأعمدة اختياري).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// ------------------------------------------------------------------ القواعد
type cdTerm struct {
	words []string
	n     int // طول الكلمة (للترجيح)
}

type csiRule struct {
	ID                                int
	Code, EN, AR, Task                string
	Mat, Lab, Sub, Eqp                string
	Div                               string
	groups                            [][]cdTerm
	neg                               []cdTerm
}

var (
	csiOnce  sync.Once
	csiRules []*csiRule
)

var rePunct = regexp.MustCompile(`[()\[\]{}/\\,،؛;:.\-_+*"'“”«»!?|=<>~#%&@$^]+`)

// كلمات النص بعد التطبيع
func cdWords(s string) []string {
	s = normAr(s)
	s = rePunct.ReplaceAllString(s, " ")
	// فصل الأرقام عن الحروف (12مم → 12 مم)
	var b strings.Builder
	var prev rune
	for i, r := range s {
		if i > 0 && ((unicode.IsDigit(r) && unicode.IsLetter(prev)) || (unicode.IsLetter(r) && unicode.IsDigit(prev))) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
		prev = r
	}
	return strings.Fields(b.String())
}

var arPrefixes = []string{"وال", "بال", "فال", "كال", "لل", "ال"}

// صيغ الكلمة: نفسها + من غير «ال/وال/…» + من غير «و» في الأول
func cdAlts(w string) []string {
	out := []string{w}
	for _, p := range arPrefixes {
		if strings.HasPrefix(w, p) && utf8.RuneCountInString(w)-utf8.RuneCountInString(p) >= 2 {
			out = append(out, strings.TrimPrefix(w, p))
			break
		}
	}
	if strings.HasPrefix(w, "و") && utf8.RuneCountInString(w) >= 4 {
		out = append(out, strings.TrimPrefix(w, "و"))
	}
	return out
}

func cdStem(w string) string {
	a := cdAlts(w)
	if len(a) > 1 && !strings.HasPrefix(w, "و") {
		return a[1]
	}
	return w
}

func mkTerm(s string) (cdTerm, bool) {
	ws := cdWords(s)
	if len(ws) == 0 {
		return cdTerm{}, false
	}
	for i := range ws {
		ws[i] = cdStem(ws[i])
	}
	return cdTerm{words: ws, n: utf8.RuneCountInString(strings.Join(ws, " "))}, true
}

func loadCSIRules() {
	csiOnce.Do(func() {
		for _, ln := range strings.Split(csiRuleSrc, "\n") {
			ln = strings.TrimSpace(ln)
			if ln == "" {
				continue
			}
			p := strings.Split(ln, "|")
			if len(p) < 9 {
				continue
			}
			for i := range p {
				p[i] = strings.TrimSpace(p[i])
			}
			r := &csiRule{ID: len(csiRules), Code: p[0], EN: p[1], AR: p[2], Task: p[3], Mat: p[4], Lab: p[5], Sub: p[6], Eqp: p[7], Div: p[0][:2]}
			for _, g := range strings.Split(p[8], ";") {
				g = strings.TrimSpace(g)
				neg := strings.HasPrefix(g, "!")
				g = strings.TrimPrefix(g, "!")
				var ts []cdTerm
				for _, t := range strings.Split(g, ",") {
					if t2, ok := mkTerm(t); ok {
						ts = append(ts, t2)
					}
				}
				if len(ts) == 0 {
					continue
				}
				if neg {
					r.neg = append(r.neg, ts...)
				} else {
					r.groups = append(r.groups, ts)
				}
			}
			csiRules = append(csiRules, r)
		}
	})
}

// ------------------------------------------------------------------ المطابقة
type cdText struct {
	alts [][]string
	head int // أول كام كلمة (الوزن الكامل)
}

func newCdText(s string) *cdText {
	ws := cdWords(s)
	t := &cdText{alts: make([][]string, len(ws)), head: 16}
	for i, w := range ws {
		t.alts[i] = cdAlts(w)
	}
	return t
}

func wordEq(alt, w string) bool {
	if alt == w {
		return true
	}
	if utf8.RuneCountInString(w) >= 4 && strings.HasPrefix(alt, w) && utf8.RuneCountInString(alt)-utf8.RuneCountInString(w) <= 3 {
		return true
	}
	return false
}

// بيرجع مكان أول ظهور للكلمة/العبارة (-1 لو مش موجودة)
func (t *cdText) find(term cdTerm) int {
	n := len(term.words)
	for i := 0; i+n <= len(t.alts); i++ {
		ok := true
		for j, w := range term.words {
			hit := false
			for _, a := range t.alts[i+j] {
				if wordEq(a, w) {
					hit = true
					break
				}
			}
			if !hit {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

type cdMatch struct {
	rule   *csiRule
	score  float64
	ng     int
	words  []string
	second float64 // أحسن درجة لقاعدة بـ Task تاني (للثقة)
}

func termW(t cdTerm) float64 { return 1 + 0.12*math.Min(float64(t.n), 20) }

func (t *cdText) score(r *csiRule) (float64, []string) {
	for _, n := range r.neg {
		if t.find(n) >= 0 {
			return 0, nil
		}
	}
	total := 0.0
	var hit []string
	for _, g := range r.groups {
		best, bw := 0.0, ""
		for _, term := range g {
			pos := t.find(term)
			if pos < 0 {
				continue
			}
			w := termW(term)
			if pos >= t.head {
				w *= 0.5
			}
			if w > best {
				best, bw = w, strings.Join(term.words, " ")
			}
		}
		if best == 0 {
			return 0, nil
		}
		total += best
		hit = append(hit, bw)
	}
	return total, hit
}

// أحسن قاعدة للنص. bonusDiv: Divisions متوقعة من عنوان القسم/الشيت
func bestRule(text string, bonusDiv map[string]float64) *cdMatch {
	loadCSIRules()
	t := newCdText(text)
	if len(t.alts) == 0 {
		return nil
	}
	var all []*cdMatch
	var best *cdMatch
	for _, r := range csiRules {
		s, hit := t.score(r)
		if s == 0 {
			continue
		}
		s += bonusDiv[r.Div] + bonusDiv[r.Code]
		m := &cdMatch{rule: r, score: s, ng: len(r.groups), words: hit}
		all = append(all, m)
		if best == nil || m.score > best.score+1e-9 || (math.Abs(m.score-best.score) < 1e-9 && m.ng > best.ng) {
			best = m
		}
	}
	if best != nil {
		for _, m := range all {
			if m.rule.Task != best.rule.Task && m.score > best.second {
				best.second = m.score
			}
		}
	}
	return best
}

var reDivNum = regexp.MustCompile(`(?i)(?:\bdiv(?:ision)?|قسم|الباب|باب)\s*[-_.:#]?\s*(\d{1,2})\b`)

// Divisions متوقعة من نص (عنوان شيت أو قسم): «Div 15» قديم ← 21/22/23
func divHint(s string) map[string]float64 {
	m := reDivNum.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	n, _ := strconv.Atoi(m[1])
	out := map[string]float64{}
	if n > 16 {
		out[fmt.Sprintf("%02d", n)] = 0.9
		return out
	}
	for _, d := range oldDivMap[n] {
		out[d] = 0.9
	}
	return out
}

// ------------------------------------------------------------------ نوع المورد
var rtLabels = map[string][2]string{
	"M": {"Material", "مواد"}, "L": {"Labor", "عمالة"}, "E": {"Equipment", "معدات"}, "S": {"Subcontractor", "مقاول باطن"},
}

var (
	rtSub = mustTerms("subcontract,subcontractor,sub-contractor,sub contractor,مقاول,مقاولين,مقاوله,مقاولة,مقاول باطن,مصنعيه مقاول")
	rtLab = mustTerms("labor,labour,labourer,laborer,worker,workers,helper,helpers,foreman,mason,masons,carpenter,carpenters,steel fixer,fixer,painter,plumber,electrician,technician,welder,operator,driver,skilled,unskilled,man day,manday,man-hour,manhour,عامل,عمال,عماله,عمالة,فني,فنيين,مساعد,مساعدين,نجار,نجارين,حداد,حدادين,مبيض,مبيضين,محار,نقاش,نقاشين,سباك,سباكين,كهربائي فني,لحام,سائق,مشغل,ملاحظ,مراقب,مصنعيه,مصنعيات,يوميه عامل,اسطى,صنايعي,صنايعيه")
	rtEqp = mustTerms("equipment,plant,rental,hire,crane,cranes,excavator,excavators,loader,loaders,bulldozer,grader,roller,compactor,dump truck,truck,trucks,tipper,mixer truck,concrete pump,compressor,generator,vibrator,jcb,bobcat,forklift,manlift,boom lift,water tanker,scaffold,scaffolding,معدات,معده,ايجار,إيجار,ونش,اوناش,كرين,حفار,حفارات,بوكلين,لودر,لوادر,بلدوزر,جريدر,هراس,هراسات,قلاب,قلابات,تريلا,عربيه,سياره,مضخه خرسانه,طلمبه خرسانه,كمبروسر,مولد,هزاز,فورك,رافعه,سقالات,سقاله,وايت مياه,فنطاس")
	rtMat = mustTerms("material,materials,supply,مواد,خامات,توريد,خامه")
)

func mustTerms(s string) []cdTerm {
	var out []cdTerm
	for _, x := range strings.Split(s, ",") {
		if t, ok := mkTerm(x); ok {
			out = append(out, t)
		}
	}
	return out
}

func anyTerm(t *cdText, ts []cdTerm) (int, string) {
	best, bw := -1, ""
	for _, x := range ts {
		if p := t.find(x); p >= 0 && (best < 0 || p < best) {
			best, bw = p, strings.Join(x.words, " ")
		}
	}
	return best, bw
}

// نوع المورد من عمود «النوع» لو موجود، وإلا من الاسم
func resType(typeCell, desc, sectionHint string) (string, string) {
	if tc := strings.TrimSpace(typeCell); tc != "" {
		t := newCdText(tc)
		if p, _ := anyTerm(t, rtSub); p >= 0 {
			return "S", "عمود النوع"
		}
		if p, _ := anyTerm(t, rtLab); p >= 0 {
			return "L", "عمود النوع"
		}
		if p, _ := anyTerm(t, rtEqp); p >= 0 {
			return "E", "عمود النوع"
		}
		if p, _ := anyTerm(t, rtMat); p >= 0 {
			return "M", "عمود النوع"
		}
		switch strings.ToUpper(tc) {
		case "M", "MAT":
			return "M", "عمود النوع"
		case "L", "LAB":
			return "L", "عمود النوع"
		case "E", "EQ", "EQP", "P":
			return "E", "عمود النوع"
		case "S", "SUB", "SC":
			return "S", "عمود النوع"
		}
	}
	t := newCdText(desc)
	ps, _ := anyTerm(t, rtSub)
	pl, _ := anyTerm(t, rtLab)
	pe, _ := anyTerm(t, rtEqp)
	// أول كلمة تبان في الاسم هي اللي بتحدد (عامل حفار ← عمالة؛ إيجار حفار ← معدات)
	best, k := 1<<30, ""
	for _, c := range []struct {
		p int
		k string
	}{{ps, "S"}, {pl, "L"}, {pe, "E"}} {
		if c.p >= 0 && c.p < best {
			best, k = c.p, c.k
		}
	}
	if k != "" {
		return k, "من الاسم"
	}
	if sectionHint != "" {
		return sectionHint, "من عنوان القسم"
	}
	return "M", "افتراضي"
}

// معدات: الكود حسب نوع المعدة
func eqpCost(desc string) string {
	t := newCdText(desc)
	pick := func(s string) bool { p, _ := anyTerm(t, mustTerms(s)); return p >= 0 }
	switch {
	case pick("crane,cranes,tower crane,mobile crane,ونش,اوناش,كرين,رافعه"):
		return "2-004-001"
	case pick("generator,compressor,مولد,مولدات,كمبروسر"):
		return "2-004-003"
	case pick("scaffold,scaffolding,سقالات,سقاله"):
		return "2-004-007"
	case pick("fuel,diesel,oil,oils,grease,وقود,سولار,ديزل,بنزين,زيت,زيوت,شحم,شحوم"):
		return "2-004-006"
	case pick("tools,tool,small tools,vibrator,drill,grinder,عده,عدد,ادوات,شنيور,صاروخ,هزاز,معدات صغيره"):
		return "2-004-005"
	case pick("concrete pump,mixer truck,طلمبه خرسانه,مضخه خرسانه,خلاطه"):
		return "2-004-002"
	}
	return "2-004-002"
}

// ------------------------------------------------------------------ الذاكرة
type cdMemEntry struct {
	CSI  string `json:"csi"` // كود القاعدة "03 30 00" + العنوان
	Rule int    `json:"rule"`
	Task string `json:"task"`
	Cost string `json:"cost,omitempty"`
	RT   string `json:"rt,omitempty"`
}

var cdMem = struct {
	sync.Mutex
	M map[string]cdMemEntry `json:"m"`
}{M: map[string]cdMemEntry{}}

func cdMemPath() string { return filepath.Join(appDir(), "coding_memory.json") }

func loadCdMem() {
	cdMem.Lock()
	defer cdMem.Unlock()
	if b, err := os.ReadFile(cdMemPath()); err == nil {
		var x struct {
			M map[string]cdMemEntry `json:"m"`
		}
		if json.Unmarshal(b, &x) == nil && x.M != nil {
			cdMem.M = x.M
		}
	}
}

func saveCdMem() {
	cdMem.Lock()
	defer cdMem.Unlock()
	b, _ := json.MarshalIndent(map[string]interface{}{"m": cdMem.M}, "", " ")
	os.MkdirAll(appDir(), 0o755)
	os.WriteFile(cdMemPath(), b, 0o644)
}

func cdKey(mode, desc string) string {
	return mode + ":" + strings.Join(cdWords(desc), " ")
}

// ------------------------------------------------------------------ قراءة الشيت
type cdItem struct {
	I       int     `json:"i"`
	Sheet   string  `json:"sheet"`
	Row     int     `json:"row"`
	No      string  `json:"no"`
	Code    string  `json:"code"` // كود المورد الأصلي
	Desc    string  `json:"desc"`
	Unit    string  `json:"unit"`
	Qty     float64 `json:"qty"`
	HasQty  bool    `json:"hasQty"`
	Rate    float64 `json:"rate"`
	HasRate bool    `json:"hasRate"`
	Amount  float64 `json:"amount"`
	HasAmt  bool    `json:"hasAmt"`
	TypeTxt string  `json:"-"`
	Section string  `json:"section"`
	ctx     string
	hint    map[string]float64
	secRT   string
	// النتيجة
	Rule  int    `json:"rule"`
	CSI   string `json:"csi"`
	Title string `json:"title"`
	Div   string `json:"div"`
	Task  string `json:"task"`
	Cost  string `json:"cost"`
	RT    string `json:"rt"`
	RTSrc string `json:"rtSrc"`
	Conf  string `json:"conf"` // high / mid / low / none / user / memory
	Src   string `json:"src"`
	Words string `json:"words"`
	Auto  [3]string `json:"-"` // rule/task/cost الأوتوماتيك (عشان نعرف المستخدم غيّر إيه)
	BOQ   string `json:"boq"`
}

type cdSheet struct {
	Name     string         `json:"name"`
	HeadRow  int            `json:"headRow"` // 1-based، 0 = مالقيناش
	Cols     map[string]int `json:"cols"`    // -1 = مش موجود
	Headers  []string       `json:"headers"` // عناوين/عينة كل عمود
	Items    int            `json:"items"`
	Headings int            `json:"headings"`
	Footers  int            `json:"footers"`
	Repeats  int            `json:"repeats"`
	Sum      float64        `json:"sum"`
	Total    float64        `json:"total"` // أكبر رقم في سطور الإجمالي
	Skip     bool           `json:"skip"`
	Note     string         `json:"note"`
	rows     [][]string
	removed  [][3]string // [row, kind, text]
}

var cdColKinds = []string{"no", "code", "desc", "unit", "qty", "rate", "amount", "type"}

var cdHeadTerms = map[string][]string{
	"no":     {"item", "item no", "no", "s/n", "sn", "ref", "م", "رقم", "رقم البند", "مسلسل", "ر.م", "البند رقم", "بند رقم"},
	"code":   {"code", "resource code", "item code", "الكود", "كود", "كود المورد", "كود الصنف", "رقم الصنف"},
	"desc":   {"description", "desc", "item description", "description of work", "resource", "resource name", "name", "البيان", "بيان", "بيان الاعمال", "الوصف", "وصف", "وصف البند", "البند", "بنود الاعمال", "اسم المورد", "اسم الصنف", "الصنف", "المورد", "الاسم"},
	"unit":   {"unit", "uom", "u.o.m", "units", "الوحده", "وحده", "وحده القياس", "الوحدة"},
	"qty":    {"qty", "quantity", "quantities", "q'ty", "الكميه", "كميه", "الكمية", "الكميات"},
	"rate":   {"rate", "unit rate", "unit price", "price", "u.price", "سعر", "السعر", "سعر الوحده", "الفئه", "فئه", "فئة", "سعر الوحدة"},
	"amount": {"amount", "total", "total amount", "total price", "value", "الاجمالي", "اجمالي", "القيمه", "قيمه", "الجمله", "جمله", "المبلغ", "الاجمالى"},
	"type":   {"type", "resource type", "category", "class", "النوع", "نوع المورد", "التصنيف", "الفئه الرئيسيه"},
}

func cellNorm(s string) string { return strings.Join(cdWords(s), " ") }

func headKind(cell string) string {
	c := cellNorm(cell)
	if c == "" || utf8.RuneCountInString(c) > 40 {
		return ""
	}
	best, bl := "", 0
	for _, k := range cdColKinds {
		for _, h := range cdHeadTerms[k] {
			hn := cellNorm(h)
			if hn == "" {
				continue
			}
			if c == hn || (utf8.RuneCountInString(hn) >= 3 && (strings.HasPrefix(c, hn+" ") || strings.HasSuffix(c, " "+hn))) {
				if l := utf8.RuneCountInString(hn); l > bl {
					best, bl = k, l
				}
			}
		}
	}
	return best
}

var cdUnits = map[string]bool{}

func init() {
	for _, u := range strings.Split("m3,m2,m,lm,rm,ml,no,nos,nr,each,ea,pcs,pc,set,sets,ls,l s,lump sum,item,ton,tons,t,kg,kgs,lit,ltr,litre,liter,day,days,month,months,hr,hour,hrs,trip,trips,point,points,pt,sqm,cum,m.l,م3,م٣,م2,م٢,م,م ط,مط,م طولي,متر,متر مربع,متر مكعب,عدد,طن,كجم,كيلو,لتر,مقطوعيه,مقطوعية,مقطوع,طقم,اطقم,نقطه,نقطة,يوم,شهر,ساعه,نقله,بند,قطعه,وحده,لفه,شيكاره,شكاره,برميل,جالون,علبه,كرتونه", ",") {
		cdUnits[cellNorm(u)] = true
	}
}

func isUnit(s string) bool {
	c := cellNorm(s)
	return c != "" && (cdUnits[c] || cdUnits[strings.ReplaceAll(c, " ", "")])
}

var reNumClean = strings.NewReplacer(",", "", " ", "", "٬", "", "SAR", "", "EGP", "", "LE", "", "ج.م", "", "ر.س", "", "$", "")

func parseNumCell(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "—" {
		return 0, false
	}
	s = reNumClean.Replace(s)
	// أرقام عربي
	s = strings.Map(func(r rune) rune {
		if r >= '٠' && r <= '٩' {
			return '0' + (r - '٠')
		}
		if r == '٫' {
			return '.'
		}
		return r
	}, s)
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg, s = true, strings.Trim(s, "()")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if neg {
		f = -f
	}
	return f, true
}

var totalTerms = mustTerms("total,sub total,subtotal,grand total,carried forward,carried to summary,brought forward,to collection,collection,page total,اجمالي,الاجمالي,الإجمالي,اجمالى,المجموع,مجموع,اجمالي الصفحه,مرحل,المرحل,منقول,ما قبله,جمله,الجمله,ملخص")

func isTotalText(s string) bool {
	t := newCdText(s)
	t.head = 6
	p, _ := anyTerm(t, totalTerms)
	return p >= 0 && p < 4
}

func cell(r []string, i int) string {
	if i < 0 || i >= len(r) {
		return ""
	}
	return strings.TrimSpace(r[i])
}

// تحديد صف العناوين وأعمدته
func detectCols(rows [][]string, mode string) (int, map[string]int) {
	cols := map[string]int{}
	for _, k := range cdColKinds {
		cols[k] = -1
	}
	bestRow, bestN := -1, 0
	var bestCols map[string]int
	lim := len(rows)
	if lim > 40 {
		lim = 40
	}
	for ri := 0; ri < lim; ri++ {
		c := map[string]int{}
		for k := range cols {
			c[k] = -1
		}
		n := 0
		for ci, v := range rows[ri] {
			k := headKind(v)
			if k == "" || c[k] >= 0 {
				continue
			}
			c[k] = ci
			n++
		}
		// «البند» ممكن تبقى الوصف أو الرقم: لو فيه وصف تاني، يبقى رقم
		if c["desc"] < 0 {
			continue
		}
		key := 0
		for _, k := range []string{"unit", "qty", "rate", "amount", "code", "no"} {
			if c[k] >= 0 {
				key++
			}
		}
		if key >= 1 && n > bestN {
			bestRow, bestN, bestCols = ri, n, c
		}
	}
	if bestRow >= 0 {
		return bestRow, bestCols
	}
	// من غير صف عناوين: الوصف = أطول نص في المتوسط، الوحدة = عمود قيمه وحدات
	ncol := 0
	for _, r := range rows {
		if len(r) > ncol {
			ncol = len(r)
		}
	}
	txtLen, unitHits, numHits := make([]float64, ncol), make([]int, ncol), make([]int, ncol)
	for _, r := range rows {
		for ci, v := range r {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if _, ok := parseNumCell(v); ok {
				numHits[ci]++
				continue
			}
			if isUnit(v) {
				unitHits[ci]++
				continue
			}
			txtLen[ci] += float64(utf8.RuneCountInString(v))
		}
	}
	best := -1
	for ci := range txtLen {
		if best < 0 || txtLen[ci] > txtLen[best] {
			best = ci
		}
	}
	cols["desc"] = best
	bu := -1
	for ci := range unitHits {
		if unitHits[ci] > 0 && (bu < 0 || unitHits[ci] > unitHits[bu]) {
			bu = ci
		}
	}
	cols["unit"] = bu
	// الأعمدة الرقمية بعد الوحدة: كمية، سعر، قيمة (بترتيب الشيت)
	var nums []int
	for ci := range numHits {
		if numHits[ci] > 0 && ci != best && ci != bu && (bu < 0 || ci > bu) {
			nums = append(nums, ci)
		}
	}
	if mode == "res" {
		if len(nums) > 0 {
			cols["rate"] = nums[0]
		}
	} else {
		for i, k := range []string{"qty", "rate", "amount"} {
			if i < len(nums) {
				cols[k] = nums[i]
			}
		}
	}
	if best > 0 {
		cols["no"] = best - 1
		if mode == "res" {
			cols["code"], cols["no"] = best-1, -1
		}
	}
	return -1, cols
}

func sheetHeaders(rows [][]string, head int) []string {
	ncol := 0
	for _, r := range rows {
		if len(r) > ncol {
			ncol = len(r)
		}
	}
	out := make([]string, ncol)
	for ci := 0; ci < ncol; ci++ {
		h := ""
		if head >= 0 {
			h = cell(rows[head], ci)
		}
		if h == "" { // عينة
			for ri := head + 1; ri < len(rows) && ri < head+30; ri++ {
				if v := cell(rows[ri], ci); v != "" {
					h = "«" + trimTo(v, 24) + "»"
					break
				}
			}
		}
		col, _ := excelize.ColumnNumberToName(ci + 1)
		out[ci] = col + (map[bool]string{true: " · " + trimTo(h, 32), false: ""})[h != ""]
	}
	return out
}

func trimTo(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

var skipSheetTerms = mustTerms("summary,collection,cover,index,notes,instructions,ملخص,الملخص,غلاف,فهرس,تعليمات,ملاحظات,اجمالي المشروع")

// قراءة شيت واحد ← البنود
func parseCdSheet(sh *cdSheet, mode string) []*cdItem {
	rows := sh.rows
	head := sh.HeadRow - 1
	c := sh.Cols
	sh.Items, sh.Headings, sh.Footers, sh.Repeats, sh.Sum, sh.Total = 0, 0, 0, 0, 0, 0
	sh.removed = nil
	var out []*cdItem
	var headings []string
	sheetHint := divHint(sh.Name)
	secRT := ""
	headDesc := ""
	if head >= 0 {
		headDesc = cellNorm(cell(rows[head], c["desc"]))
	}
	for ri := head + 1; ri < len(rows); ri++ {
		r := rows[ri]
		desc := cell(r, c["desc"])
		unit := cell(r, c["unit"])
		qty, hq := parseNumCell(cell(r, c["qty"]))
		rate, hr := parseNumCell(cell(r, c["rate"]))
		amt, ha := parseNumCell(cell(r, c["amount"]))
		no := cell(r, c["no"])
		code := cell(r, c["code"])
		empty := true
		for _, v := range r {
			if strings.TrimSpace(v) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		// صف عناوين متكرر (كل صفحة)
		if headDesc != "" && cellNorm(desc) == headDesc {
			sh.Repeats++
			sh.removed = append(sh.removed, [3]string{strconv.Itoa(ri + 1), "عناوين متكررة", desc})
			continue
		}
		// إجمالي / مرحل
		allTxt := strings.Join(r, " ")
		if !isUnit(unit) && (isTotalText(desc) || (desc == "" && isTotalText(allTxt))) {
			sh.Footers++
			for _, v := range []float64{amt, rate, qty} {
				if math.Abs(v) > sh.Total {
					sh.Total = math.Abs(v)
				}
			}
			sh.removed = append(sh.removed, [3]string{strconv.Itoa(ri + 1), "إجمالي/مرحل", trimTo(allTxt, 120)})
			continue
		}
		if desc == "" {
			if hq || ha || hr {
				sh.removed = append(sh.removed, [3]string{strconv.Itoa(ri + 1), "أرقام من غير وصف", trimTo(allTxt, 120)})
			}
			continue
		}
		isItem := false
		if mode == "res" {
			isItem = unit != "" || hr || code != "" || hq
		} else {
			isItem = (hq && qty != 0) || isUnit(unit) || (unit != "" && (hr || ha)) || ha && hr
		}
		if !isItem {
			// عنوان قسم
			sh.Headings++
			headings = append(headings, desc)
			if len(headings) > 3 {
				headings = headings[len(headings)-3:]
			}
			if mode == "res" {
				if k, src := resType("", desc, ""); src == "من الاسم" {
					secRT = k
				} else if p, _ := anyTerm(newCdText(desc), rtMat); p >= 0 {
					secRT = "M"
				}
			}
			sh.removed = append(sh.removed, [3]string{strconv.Itoa(ri + 1), "عنوان قسم", trimTo(desc, 120)})
			continue
		}
		it := &cdItem{Sheet: sh.Name, Row: ri + 1, No: no, Code: code, Desc: desc, Unit: unit, Qty: qty, HasQty: hq,
			Rate: rate, HasRate: hr, Amount: amt, HasAmt: ha, TypeTxt: cell(r, c["type"]), secRT: secRT}
		if !ha && hq && hr {
			it.Amount, it.HasAmt = r2(qty*rate), true
		}
		if len(headings) > 0 {
			it.Section = headings[len(headings)-1]
			it.ctx = strings.Join(headings, " | ")
		}
		it.hint = map[string]float64{}
		for k, v := range sheetHint {
			it.hint[k] = v
		}
		for k, v := range divHint(it.ctx) {
			it.hint[k] = v
		}
		sh.Items++
		sh.Sum += it.Amount
		out = append(out, it)
	}
	sh.Sum = r2(sh.Sum)
	return out
}

// ------------------------------------------------------------------ التكويد
var reDitto = regexp.MustCompile(`(?i)^\s*(ditto|do\.?|same as|as item|as above|ditto but|كالبند|كما في البند|كما بالبند|نفس البند|مثل البند|كالسابق|نفس السابق)`)

func (cs *codingState) classify() {
	loadCSIRules()
	cl := cs.cl
	var prev *cdItem
	prevSheet := ""
	for _, it := range cs.items {
		if it.Sheet != prevSheet {
			prev, prevSheet = nil, it.Sheet
		}
		cs.classifyOne(it, prev, cl)
		it.Auto = [3]string{strconv.Itoa(it.Rule), it.Task, it.Cost}
		prev = it
	}
	cs.number()
}

func ruleByID(id int) *csiRule {
	if id < 0 || id >= len(csiRules) {
		return nil
	}
	return csiRules[id]
}

func (it *cdItem) setRule(r *csiRule) {
	if r == nil {
		it.Rule, it.CSI, it.Title, it.Div = -1, "", "", ""
		return
	}
	it.Rule, it.CSI, it.Title, it.Div = r.ID, r.Code, r.EN, r.Div
}

func ruleCost(r *csiRule, rt, desc string) string {
	if r == nil {
		return ""
	}
	switch rt {
	case "L":
		return r.Lab
	case "S":
		return r.Sub
	case "E":
		if r.Eqp != "" && strings.HasPrefix(r.Eqp, "2-004") {
			return eqpCost(desc)
		}
		if r.Eqp != "" {
			return r.Eqp
		}
		return eqpCost(desc)
	}
	return r.Mat
}

// قاعدة ليها نفس الـ Task (لو الكود جه من مكان تاني)
func ruleForTask(task string) *csiRule {
	for _, r := range csiRules {
		if r.Task == task {
			return r
		}
	}
	return nil
}

func (cs *codingState) classifyOne(it *cdItem, prev *cdItem, cl *CodeLists) {
	mode := cs.mode
	if mode == "res" {
		it.RT, it.RTSrc = resType(it.TypeTxt, it.Desc, it.secRT)
	}
	// 1) ذاكرة (تعديلات أحمد قبل كده)
	cdMem.Lock()
	m, ok := cdMem.M[cdKey(mode, it.Desc)]
	cdMem.Unlock()
	if ok {
		it.setRule(ruleByID(m.Rule))
		if r := ruleByID(m.Rule); r == nil || r.Code != strings.SplitN(m.CSI, " |", 2)[0] {
			// القواعد اتغيرت: دوّر بالكود
			it.setRule(nil)
			for _, r := range csiRules {
				if strings.HasPrefix(m.CSI, r.Code) {
					it.setRule(r)
					break
				}
			}
		}
		it.Task, it.Cost = m.Task, m.Cost
		if m.RT != "" && mode == "res" {
			it.RT, it.RTSrc = m.RT, "ذاكرة"
		}
		it.Conf, it.Src = "memory", "ذاكرة (اتعدّل قبل كده)"
		return
	}
	// 2) «كالبند السابق»
	if prev != nil && reDitto.MatchString(it.Desc) && prev.Rule >= 0 {
		it.setRule(ruleByID(prev.Rule))
		it.Task, it.Cost = prev.Task, ruleCost(ruleByID(prev.Rule), it.RT, it.Desc)
		it.Conf, it.Src = "high", "زي البند اللي قبله"
		return
	}
	// 3) القواعد: الوصف + ترجيح بعنوان القسم
	hint := map[string]float64{}
	for k, v := range it.hint {
		hint[k] = v
	}
	var ctxM *cdMatch
	if it.ctx != "" {
		ctxM = bestRule(it.ctx, it.hint)
		if ctxM != nil {
			hint[ctxM.rule.Div] += 0.7
			hint[ctxM.rule.Code] += 0.3
		}
	}
	dm := bestRule(it.Desc, hint)
	// الموارد: الاسم المعروف في المستودعات (Cost) أولاً
	if mode == "res" {
		key := normAr(it.Desc)
		var kt, kc string
		smem.mu.Lock()
		if v, ok := smem.Codes["n:"+key]; ok {
			kt, kc = v[0], v[1]
		}
		if it.Code != "" {
			if v, ok := smem.Codes["c:"+normAr(it.Code)]; ok && kc == "" {
				kt, kc = v[0], v[1]
			}
		}
		smem.mu.Unlock()
		if kc == "" {
			loadKnown()
			if k, ok := knownByName[key]; ok {
				kt, kc = k.T, k.K
			}
		}
		if kc != "" && cl.byC[kc] != nil {
			r := (*csiRule)(nil)
			if dm != nil {
				r = dm.rule
			} else if kt != "" {
				r = ruleForTask(kt)
			}
			it.setRule(r)
			it.Task, it.Cost = kt, kc
			if it.Task == "" && r != nil {
				it.Task = r.Task
			}
			if it.RT == "M" && it.RTSrc == "افتراضي" {
				if c := cl.byC[kc]; c != nil {
					it.RT = typeFromCost(c.Type)
				}
			}
			it.Conf, it.Src = "high", "صنف معروف في المستودعات"
			if dm != nil {
				it.Words = strings.Join(dm.words, " + ")
			}
			return
		}
	}
	if dm != nil {
		it.setRule(dm.rule)
		it.Task = dm.rule.Task
		it.Cost = ruleCost(dm.rule, it.RT, it.Desc)
		it.Words = strings.Join(dm.words, " + ")
		margin := dm.score - dm.second
		switch {
		case (dm.ng >= 2 || dm.score >= 2.2) && margin >= 0.4:
			it.Conf = "high"
		case margin >= 0.4 || dm.score >= 1.6:
			it.Conf = "mid"
		default:
			it.Conf = "low"
		}
		it.Src = "من وصف البند"
		if ctxM != nil && ctxM.rule.Div == dm.rule.Div {
			it.Src = "من وصف البند + عنوان القسم"
			if it.Conf == "low" {
				it.Conf = "mid"
			}
		}
	} else if ctxM != nil {
		it.setRule(ctxM.rule)
		it.Task = ctxM.rule.Task
		it.Cost = ruleCost(ctxM.rule, it.RT, it.Desc)
		it.Words = strings.Join(ctxM.words, " + ")
		it.Conf, it.Src = "low", "من عنوان القسم بس"
	} else {
		// 4) أقرب بند في قايمة PCS
		code, s := cl.nearestCode(cl.Task, it.Desc+" "+it.ctx)
		if code != "" && s >= 0.18 {
			it.Task = code
			it.setRule(ruleForTask(code))
			it.Cost = ruleCost(ruleByID(it.Rule), it.RT, it.Desc)
			it.Conf, it.Src = "low", "أقرب بند في قايمة PCS"
		} else {
			it.setRule(nil)
			it.Conf, it.Src = "none", "مالقيتش — اختار بإيدك"
		}
	}
	// Cost من غير قاعدة للنوع ده: أقرب Cost من نفس النوع
	if mode == "res" && it.Cost == "" {
		var pool []*Code
		want := rtLabels[it.RT][0]
		for _, c := range cl.Cost {
			if strings.HasPrefix(strings.ToLower(c.Type), strings.ToLower(want[:3])) {
				pool = append(pool, c)
			}
		}
		if code, s := cl.nearestCode(pool, it.Desc); code != "" && s > 0.1 {
			it.Cost = code
		}
	}
}

func typeFromCost(t string) string {
	t = strings.ToLower(t)
	switch {
	case strings.HasPrefix(t, "lab"):
		return "L"
	case strings.HasPrefix(t, "sub"):
		return "S"
	case strings.HasPrefix(t, "equ"):
		return "E"
	}
	return "M"
}

// كود البند من الـ CSI: 033000.001 بالترتيب جوه كل Section
func (cs *codingState) number() {
	seq := map[string]int{}
	for _, it := range cs.items {
		if cs.mode != "boq" {
			it.BOQ = ""
			continue
		}
		base := "000000"
		if it.CSI != "" {
			base = strings.ReplaceAll(it.CSI, " ", "")
		}
		seq[base]++
		it.BOQ = fmt.Sprintf("%s.%03d", base, seq[base])
	}
}

// ------------------------------------------------------------------ الحالة
type codingState struct {
	mu     sync.Mutex
	mode   string // boq | res
	name   string
	data   []byte
	sheets []*cdSheet
	items  []*cdItem
	cl     *CodeLists
	out    []byte
	review []byte
	titles bool
}

var cds = &codingState{}

func (cs *codingState) load(data []byte) error {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("مقدرتش أفتح الملف — لازم يكون xlsx / xlsm")
	}
	defer f.Close()
	cs.sheets = nil
	for _, name := range f.GetSheetList() {
		if v, _ := f.GetSheetVisible(name); !v {
			continue
		}
		rows, err := f.GetRows(name)
		if err != nil || len(rows) == 0 {
			continue
		}
		sh := &cdSheet{Name: name, rows: rows}
		h, cols := detectCols(rows, cs.mode)
		sh.HeadRow, sh.Cols = h+1, cols
		sh.Headers = sheetHeaders(rows, h)
		if p, _ := anyTerm(newCdText(name), skipSheetTerms); p >= 0 {
			sh.Skip, sh.Note = true, "شكله ملخص — اتشال (تقدر ترجّعه)"
		}
		cs.sheets = append(cs.sheets, sh)
	}
	if len(cs.sheets) == 0 {
		return fmt.Errorf("الملف فاضي")
	}
	cs.cl, _ = loadCodeLists()
	cs.rebuild()
	return nil
}

func (cs *codingState) rebuild() {
	cs.items = nil
	for _, sh := range cs.sheets {
		its := parseCdSheet(sh, cs.mode)
		if sh.Cols["desc"] < 0 {
			sh.Note = "مالقيتش عمود الوصف — اختاره"
		}
		if sh.Skip {
			continue
		}
		cs.items = append(cs.items, its...)
	}
	for i, it := range cs.items {
		it.I = i
	}
	cs.classify()
	cs.out, cs.review = nil, nil
}

func (cs *codingState) info() map[string]interface{} {
	conf := map[string]int{}
	type divSum struct {
		Div    string  `json:"div"`
		Title  string  `json:"title"`
		N      int     `json:"n"`
		Amount float64 `json:"amount"`
	}
	dm := map[string]*divSum{}
	for _, it := range cs.items {
		conf[it.Conf]++
		d := it.Div
		if d == "" {
			d = "??"
		}
		if dm[d] == nil {
			t := csiDivTitle[d]
			dm[d] = &divSum{Div: d, Title: t[0]}
		}
		dm[d].N++
		dm[d].Amount += it.Amount
	}
	var divs []*divSum
	for _, v := range dm {
		v.Amount = r2(v.Amount)
		divs = append(divs, v)
	}
	sort.Slice(divs, func(i, j int) bool { return divs[i].Div < divs[j].Div })
	return map[string]interface{}{"mode": cs.mode, "file": cs.name, "sheets": cs.sheets, "items": cs.items, "conf": conf, "divs": divs}
}

// ------------------------------------------------------------------ الإخراج
var boqOutHead = []string{"BOQ Code", "Description", "Unit", "Qty", "Rate", "Amount", "CSI Code", "CSI Title", "CSI Division", "Task Code", "Task Description", "Original Item No", "Section"}
var resOutHead = []string{"Resource Code", "Description", "Unit", "Rate", "Resource Type", "CSI Code", "CSI Title", "CSI Division", "Cost Code", "Cost Description", "Task Code"}

func divLabel(d string) string {
	if t, ok := csiDivTitle[d]; ok {
		return d + " - " + t[0]
	}
	return d
}

func sheetSafe(s string, used map[string]bool) string {
	s = strings.NewReplacer("/", "-", "\\", "-", "?", "", "*", "", "[", "(", "]", ")", ":", "-").Replace(strings.TrimSpace(s))
	if s == "" {
		s = "Sheet"
	}
	if utf8.RuneCountInString(s) > 31 {
		s = string([]rune(s)[:31])
	}
	b := s
	for i := 2; used[strings.ToLower(s)]; i++ {
		suf := fmt.Sprintf(" (%d)", i)
		r := []rune(b)
		if len(r)+len(suf) > 31 {
			r = r[:31-len(suf)]
		}
		s = string(r) + suf
	}
	used[strings.ToLower(s)] = true
	return s
}

func numOrBlank(has bool, v float64) interface{} {
	if !has {
		return nil
	}
	return v
}

func (cs *codingState) outRow(it *cdItem) []interface{} {
	cl := cs.cl
	if cs.mode == "boq" {
		tdesc := ""
		if c := cl.byT[it.Task]; c != nil {
			tdesc = c.EN
		}
		return []interface{}{it.BOQ, it.Desc, it.Unit, numOrBlank(it.HasQty, it.Qty), numOrBlank(it.HasRate, it.Rate), numOrBlank(it.HasAmt, it.Amount),
			it.CSI, it.Title, divLabel(it.Div), it.Task, tdesc, it.No, it.Section}
	}
	cdesc := ""
	if c := cl.byC[it.Cost]; c != nil {
		cdesc = c.EN
	}
	return []interface{}{it.Code, it.Desc, it.Unit, numOrBlank(it.HasRate, it.Rate), rtLabels[it.RT][0], it.CSI, it.Title, divLabel(it.Div), it.Cost, cdesc, it.Task}
}

func (cs *codingState) build() error {
	head := boqOutHead
	if cs.mode == "res" {
		head = resOutHead
	}
	// ملف PCS: شيت لكل شيت في الأصل (PCS بيعمل تاب لكل شيت)، من غير عناوين أقسام ولا إجماليات
	f := excelize.NewFile()
	used := map[string]bool{}
	first := true
	bySheet := map[string][]*cdItem{}
	var order []string
	for _, it := range cs.items {
		if _, ok := bySheet[it.Sheet]; !ok {
			order = append(order, it.Sheet)
		}
		bySheet[it.Sheet] = append(bySheet[it.Sheet], it)
	}
	if len(order) == 0 {
		return fmt.Errorf("مفيش بنود تطلع")
	}
	numStyle, _ := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr("#,##0.00")})
	for _, sn := range order {
		name := sheetSafe(sn, used)
		if first {
			f.SetSheetName("Sheet1", name)
			first = false
		} else {
			f.NewSheet(name)
		}
		r := 1
		if cs.titles {
			for ci, h := range head {
				c, _ := excelize.CoordinatesToCellName(ci+1, 1)
				f.SetCellValue(name, c, h)
			}
			r = 2
		}
		for _, it := range bySheet[sn] {
			c, _ := excelize.CoordinatesToCellName(1, r)
			row := cs.outRow(it)
			f.SetSheetRow(name, c, &row)
			r++
		}
		if cs.mode == "boq" && r > 1 {
			a, _ := excelize.CoordinatesToCellName(4, 1)
			b, _ := excelize.CoordinatesToCellName(6, r-1)
			f.SetCellStyle(name, a, b, numStyle)
		}
		widths := map[string]float64{"A": 14, "B": 70, "C": 8, "D": 12, "E": 12, "F": 14, "G": 11, "H": 34, "I": 26, "J": 11, "K": 34, "L": 10, "M": 40}
		if cs.mode == "res" {
			widths = map[string]float64{"A": 14, "B": 60, "C": 8, "D": 12, "E": 14, "F": 11, "G": 34, "H": 26, "I": 11, "J": 40, "K": 11}
		}
		for k, w := range widths {
			f.SetColWidth(name, k, k, w)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return err
	}
	cs.out = buf.Bytes()
	cs.review = cs.buildReview()
	return nil
}


var confAr = map[string]string{"high": "عالي", "mid": "متوسط", "low": "ضعيف — راجعه", "none": "مالقيتش", "memory": "ذاكرة", "user": "اتعدّل بإيدك"}

func (cs *codingState) buildReview() []byte {
	f := excelize.NewFile()
	cl := cs.cl
	sh := "البنود"
	f.SetSheetName("Sheet1", sh)
	f.SetSheetView(sh, 0, &excelize.ViewOptions{RightToLeft: boolPtr(false)})
	hs, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"163F33"}, Pattern: 1}})
	low, _ := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Color: []string{"FDECE3"}, Pattern: 1}})
	var head []string
	if cs.mode == "boq" {
		head = append([]string{"Sheet", "Row"}, boqOutHead...)
	} else {
		head = append([]string{"Sheet", "Row"}, resOutHead...)
	}
	head = append(head, "الثقة", "المصدر", "الكلمات اللي اتعرف بيها")
	if cs.mode == "res" {
		head = append(head, "النوع جه منين")
	}
	f.SetSheetRow(sh, "A1", &head)
	f.SetCellStyle(sh, "A1", colName(len(head))+"1", hs)
	for i, it := range cs.items {
		row := append([]interface{}{it.Sheet, it.Row}, cs.outRow(it)...)
		row = append(row, confAr[it.Conf], it.Src, it.Words)
		if cs.mode == "res" {
			row = append(row, it.RTSrc)
		}
		c, _ := excelize.CoordinatesToCellName(1, i+2)
		f.SetSheetRow(sh, c, &row)
		if it.Conf == "low" || it.Conf == "none" {
			f.SetCellStyle(sh, c, colName(len(head))+strconv.Itoa(i+2), low)
		}
	}
	f.AutoFilter(sh, "A1:"+colName(len(head))+strconv.Itoa(len(cs.items)+1), nil)
	f.SetPanes(sh, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	f.SetColWidth(sh, "D", "D", 60)
	// ملخص CSI
	s2 := "ملخص CSI"
	f.NewSheet(s2)
	type agg struct {
		n   int
		amt float64
		t   string
	}
	m := map[string]*agg{}
	var keys []string
	for _, it := range cs.items {
		k := it.CSI + "\t" + it.Title + "\t" + it.Task
		if m[k] == nil {
			m[k] = &agg{}
			keys = append(keys, k)
		}
		m[k].n++
		m[k].amt += it.Amount
	}
	sort.Strings(keys)
	h2 := []string{"CSI Division", "CSI Code", "CSI Title", "Task Code", "Task Description", "عدد البنود"}
	if cs.mode == "boq" {
		h2 = append(h2, "Amount")
	}
	f.SetSheetRow(s2, "A1", &h2)
	f.SetCellStyle(s2, "A1", colName(len(h2))+"1", hs)
	for i, k := range keys {
		p := strings.Split(k, "\t")
		td := ""
		if c := cl.byT[p[2]]; c != nil {
			td = c.EN
		}
		div := ""
		if len(p[0]) >= 2 {
			div = divLabel(p[0][:2])
		}
		row := []interface{}{div, p[0], p[1], p[2], td, m[k].n}
		if cs.mode == "boq" {
			row = append(row, r2(m[k].amt))
		}
		c, _ := excelize.CoordinatesToCellName(1, i+2)
		f.SetSheetRow(s2, c, &row)
	}
	f.SetColWidth(s2, "A", "A", 30)
	f.SetColWidth(s2, "C", "C", 40)
	f.SetColWidth(s2, "E", "E", 36)
	// المحذوف
	s3 := "اللي اتشال"
	f.NewSheet(s3)
	f.SetSheetRow(s3, "A1", &[]string{"Sheet", "Row", "النوع", "النص"})
	f.SetCellStyle(s3, "A1", "D1", hs)
	r := 2
	for _, s := range cs.sheets {
		for _, x := range s.removed {
			c, _ := excelize.CoordinatesToCellName(1, r)
			n, _ := strconv.Atoi(x[0])
			f.SetSheetRow(s3, c, &[]interface{}{s.Name, n, x[1], x[2]})
			r++
		}
		if s.Skip {
			c, _ := excelize.CoordinatesToCellName(1, r)
			f.SetSheetRow(s3, c, &[]interface{}{s.Name, "", "شيت كامل اتشال", s.Note})
			r++
		}
	}
	f.SetColWidth(s3, "D", "D", 90)
	// مطابقة
	if cs.mode == "boq" {
		s4 := "المطابقة"
		f.NewSheet(s4)
		f.SetSheetRow(s4, "A1", &[]string{"Sheet", "عدد البنود", "مجموع البنود", "أكبر إجمالي مكتوب في الشيت", "الفرق"})
		f.SetCellStyle(s4, "A1", "E1", hs)
		i := 2
		for _, s := range cs.sheets {
			if s.Skip {
				continue
			}
			var diff interface{}
			if s.Total > 0 {
				diff = r2(s.Sum - s.Total)
			}
			c, _ := excelize.CoordinatesToCellName(1, i)
			f.SetSheetRow(s4, c, &[]interface{}{s.Name, s.Items, s.Sum, numOrBlank(s.Total > 0, s.Total), diff})
			i++
		}
		f.SetColWidth(s4, "A", "E", 22)
	}
	var buf bytes.Buffer
	f.Write(&buf)
	return buf.Bytes()
}

func colName(n int) string { c, _ := excelize.ColumnNumberToName(n); return c }

// ------------------------------------------------------------------ API
func cSafe(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				writeJSON(w, 500, map[string]string{"error": fmt.Sprint("خطأ غير متوقع: ", e)})
			}
		}()
		cds.mu.Lock()
		defer cds.mu.Unlock()
		h(w, r)
	}
}

func hCdMeta(w http.ResponseWriter, r *http.Request) {
	loadCSIRules()
	cl, _ := loadCodeLists()
	type rr struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
		EN   string `json:"en"`
		AR   string `json:"ar"`
		Div  string `json:"div"`
		Task string `json:"task"`
		Mat  string `json:"mat"`
		Lab  string `json:"lab"`
		Sub  string `json:"sub"`
		Eqp  string `json:"eqp"`
	}
	var rules []rr
	for _, x := range csiRules {
		rules = append(rules, rr{x.ID, x.Code, x.EN, x.AR, x.Div, x.Task, x.Mat, x.Lab, x.Sub, x.Eqp})
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Code < rules[j].Code })
	divs := map[string]string{}
	for k, v := range csiDivTitle {
		divs[k] = v[0]
	}
	writeJSON(w, 200, map[string]interface{}{"rules": rules, "task": cl.Task, "cost": cl.Cost, "divs": divs, "types": rtLabels})
}

func hCdLoad(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(300 << 20)
	file, hdr, err := r.FormFile("file")
	if err != nil {
		fail(w, err)
		return
	}
	defer file.Close()
	data, _ := io.ReadAll(file)
	mode := r.FormValue("mode")
	if mode != "res" {
		mode = "boq"
	}
	cds.mode, cds.name, cds.data = mode, hdr.Filename, data
	if err := cds.load(data); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, cds.info())
}

// تغيير أعمدة شيت أو استبعاده
func hCdRemap(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Sheet   string         `json:"sheet"`
		Cols    map[string]int `json:"cols"`
		HeadRow *int           `json:"headRow"`
		Skip    *bool          `json:"skip"`
	}
	json.NewDecoder(r.Body).Decode(&b)
	for _, sh := range cds.sheets {
		if sh.Name != b.Sheet {
			continue
		}
		for k, v := range b.Cols {
			if _, ok := sh.Cols[k]; ok {
				sh.Cols[k] = v
			}
		}
		if b.HeadRow != nil && *b.HeadRow >= 0 && *b.HeadRow <= len(sh.rows) {
			sh.HeadRow = *b.HeadRow
			sh.Headers = sheetHeaders(sh.rows, sh.HeadRow-1)
		}
		if b.Skip != nil {
			sh.Skip = *b.Skip
			if !sh.Skip {
				sh.Note = ""
			}
		}
	}
	cds.rebuild()
	writeJSON(w, 200, cds.info())
}

type cdEdit struct {
	I    int    `json:"i"`
	Rule *int   `json:"rule"`
	Task string `json:"task"`
	Cost string `json:"cost"`
	RT   string `json:"rt"`
}

func applyCdEdits(edits []cdEdit) int {
	n := 0
	for _, e := range edits {
		if e.I < 0 || e.I >= len(cds.items) {
			continue
		}
		it := cds.items[e.I]
		changed := false
		if e.Rule != nil && *e.Rule != it.Rule {
			it.setRule(ruleByID(*e.Rule))
			changed = true
		}
		if e.Task != "" && e.Task != it.Task {
			it.Task, changed = e.Task, true
		}
		if e.Cost != "" && e.Cost != it.Cost {
			it.Cost, changed = e.Cost, true
		}
		if e.RT != "" && e.RT != it.RT {
			it.RT, changed = e.RT, true
		}
		if !changed {
			continue
		}
		n++
		it.Conf, it.Src = "user", "اتعدّل بإيدك"
		cdMem.Lock()
		cdMem.M[cdKey(cds.mode, it.Desc)] = cdMemEntry{CSI: it.CSI + " | " + it.Title, Rule: it.Rule, Task: it.Task, Cost: it.Cost, RT: it.RT}
		cdMem.Unlock()
	}
	if n > 0 {
		saveCdMem()
	}
	cds.number()
	return n
}

func hCdConvert(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Edits  []cdEdit `json:"edits"`
		Titles bool     `json:"titles"`
	}
	json.NewDecoder(r.Body).Decode(&b)
	if len(cds.items) == 0 {
		fail(w, fmt.Errorf("ارفع الملف الأول"))
		return
	}
	n := applyCdEdits(b.Edits)
	cds.titles = b.Titles
	if err := cds.build(); err != nil {
		fail(w, err)
		return
	}
	var amt float64
	none := 0
	for _, it := range cds.items {
		amt += it.Amount
		if it.CSI == "" || (cds.mode == "boq" && it.Task == "") || (cds.mode == "res" && it.Cost == "") {
			none++
		}
	}
	info := cds.info()
	info["edited"] = n
	info["outItems"] = len(cds.items)
	info["outAmount"] = r2(amt)
	info["uncoded"] = none
	writeJSON(w, 200, info)
}

func hCdSave(w http.ResponseWriter, r *http.Request) {
	if cds.out == nil {
		fail(w, fmt.Errorf("دوس تحويل الأول"))
		return
	}
	base := strings.TrimSuffix(cds.name, filepath.Ext(cds.name))
	tag := time.Now().Format("2006-01-02")
	kind, label, n1, n2 := "boq", "تكويد BOQ — ", "PCS_BOQ_CSI_"+finSafeName(base)+".xlsx", "مراجعة_تكويد_BOQ_"+finSafeName(base)+".xlsx"
	if cds.mode == "res" {
		kind, label, n1, n2 = "res", "تكويد Resources — ", "PCS_Resources_Coded_"+finSafeName(base)+".xlsx", "مراجعة_تكويد_Resources_"+finSafeName(base)+".xlsx"
	}
	var amt float64
	for _, it := range cds.items {
		amt += it.Amount
	}
	run := &Run{Kind: kind, Period: "coding:" + tag, Label: label + base, Source: cds.name, Items: len(cds.items), Amount: r2(amt),
		Extra: fmt.Sprintf("%d بند", len(cds.items)), OutName: n1, RevName: n2}
	run.User = reqUser(r)
	saveAndRecord(w, run, cds.out, cds.review)
}

func registerCoding(mux *http.ServeMux) {
	loadCdMem()
	mux.HandleFunc("/api/coding/meta", cSafe(hCdMeta))
	mux.HandleFunc("/api/coding/load", cSafe(hCdLoad))
	mux.HandleFunc("/api/coding/remap", cSafe(hCdRemap))
	mux.HandleFunc("/api/coding/convert", cSafe(hCdConvert))
	mux.HandleFunc("/api/coding/save", cSafe(hCdSave))
}

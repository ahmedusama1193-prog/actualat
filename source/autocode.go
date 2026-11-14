package main

// المستودعات — تكويد تلقائي: لو شيت المخازن مافيهوش Task Code أو Cost Code
// البرنامج بيحط كود لكل صنف لوحده ويطلّع الملف على طول، من غير ما يسيب صنف فاضي.
//
// الترتيب (أول واحد يلاقي كود بيكسب):
//  1. الذاكرة   — أكواد الصنف ده من مرات قبل كده (أو من المستودع 3.0)
//  2. الشيت     — نفس الصنف في نفس الشيت ليه أكواد في صف تاني
//  3. معروف     — قايمة أصناف متكوّدة جوه البرنامج (من ملفات PCS اللي اتراجعت)
//  4. نفس العيلة — أقرب كود صنف (نفس أول أرقام الكود 3.1.1.1…) ليه أكواد
//  5. شبيه      — أقرب اسم صنف شبهه (نسبة تشابه عالية)
//  6. كلمات     — قواعد بالكلمات (حديد، اسمنت، سيراميك، PVC، سلك…)
//  7. القسم     — Task Code القسم (من الذاكرة أو القواعد) + أقرب Cost مواد من قايمة PCS
//  8. قايمة PCS — أقرب Task لاسم الصنف من قايمة PCS، وأقرب Cost مواد ليه
//  9. افتراضي   — بس لو الاسم مالوش أي علاقة بالقايمة (مستلزمات موقع عامة)

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

//go:embed assets/store_known.json
var embStoreKnown []byte

type knownItem struct {
	C string `json:"c"` // كود الصنف
	N string `json:"n"` // اسم الصنف
	T string `json:"t"` // Task Code
	K string `json:"k"` // Cost Code
}

var (
	knownOnce   sync.Once
	knownByCode map[string]knownItem
	knownByName map[string]knownItem
	knownList   []knownItem
)

func loadKnown() {
	knownOnce.Do(func() {
		knownByCode, knownByName = map[string]knownItem{}, map[string]knownItem{}
		json.Unmarshal(embStoreKnown, &knownList)
		for _, k := range knownList {
			if k.C != "" {
				knownByCode[normAr(k.C)] = k
			}
			if k.N != "" {
				knownByName[normAr(k.N)] = k
			}
		}
	})
}

// ------------------------------------------------------------------ قواعد بالكلمات
type stKW struct {
	words []string // أي كلمة منهم
	need  []string // ولازم كلمة من دول كمان (اختياري)
	not   []string // ومش فيها ولا كلمة من دول
	task  string   // فاضي = خد Task القسم
	cost  string
}

// الترتيب مهم: الأخص الأول
var storeKW = []stKW{
	// كهرباء — قبل السباكة عشان "ماسورة كهرباء" و"كوع كهرباء"
	{words: []string{"ارضي", "ايرث", "earth", "صاعقه", "اسياخ نحاس"}, not: []string{"ارضيات", "ارضيه"}, task: "1-005-006", cost: "1-006-001"},
	{words: []string{"لوحه كهرباء", "لوحه توزيع", "قاطع", "breaker", "مفتاح اوتوماتيك"}, task: "1-005-003", cost: "1-006-003"},
	{words: []string{"لوحه"}, need: []string{"امبير", "فاز", " خط", "ريتاج"}, task: "1-005-003", cost: "1-006-003"},
	{words: []string{"كشاف", "لمبه", "سبوت", "اضاءه", "led", "نجفه", "ابليك"}, task: "1-005-001", cost: "1-006-004"},
	{words: []string{"بريزه", "مفتاح اناره", "مفتاح كهرباء", "فيشه"}, task: "1-005-002", cost: "1-006-004"},
	{words: []string{"كاميرا", "انذار", "داتا", "تليفون", "ريسيفر", "انتركم", "cat6", "cat 6"}, task: "1-005-004", cost: "1-006-009"},
	{words: []string{"كابل", "سلك", "فليكسل", "cable", "wire"}, not: []string{"سلك رباط", "سلك تربيط", "شبك", "فرشه", "صنفره", "سنفره"}, task: "1-005-002", cost: "1-006-001"},
	{words: []string{"ماسوره", "مواسير", "كوع", "جلبه", "علبه", "بوكس", "دكت", "خرطوم", "تي "}, need: []string{"كهرباء", "كهربا"}, task: "1-005-002", cost: "1-006-002"},
	{words: []string{"علبه مفتاح", "بوكس حديد", "دكو دكت", "حبل شداد"}, task: "1-005-002", cost: "1-006-002"},
	// حريق وتكييف
	{words: []string{"حريق", "رشاش", "sprinkler"}, not: []string{"طفايه", "باب", "ابواب"}, task: "1-006-004", cost: "1-006-006"},
	{words: []string{"مكيف", "سبليت", "split"}, task: "1-006-003", cost: "1-006-005"},
	{words: []string{"تكييف", "فريون", "تهويه", "جريل", "دفيوزر", "نحاس تكييف", "ارماflex", "ارمافليكس"}, task: "1-006-003", cost: "1-007-003"},
	// أجهزة صحية
	{words: []string{"خلاط", "حوض", "مرحاض", "قاعده افرنجي", "شطاف", "بانيو", "مغسله", "دش شاور", "اكسسوار حمام"}, task: "1-003-013", cost: "1-004-013"},
	{words: []string{"سخان", "محبس", "صفايه", "بلف", "valve", "عوامه"}, task: "1-006-001", cost: "1-007-002"},
	{words: []string{"مصعد", "اسانسير", "elevator"}, task: "1-003-015", cost: "1-006-008"},
	{words: []string{"طلمبه", "مضخه", "خزان"}, task: "1-006-005", cost: "1-006-007"},
	// سباكة
	{words: []string{"pvc", "ppr", "upvc", "حراري", "ماسوره", "مواسير", "كوع", "جلبه", " تي ", "نبل", "مسلوب", "قسام", "سباكه", "صرف", "تغذيه", "غسول", "كلين اوت", " طبه "}, task: "1-006-001", cost: "1-007-001"},
	// خرسانة وحديد
	{words: []string{"حديد تسليح", "سلك رباط", "سلك تربيط", "كانات"}, task: "1-002-002", cost: "1-002-003"},
	{words: []string{"حديد"}, need: []string{"مم", "مللي", "ملي"}, not: []string{"بوكس", "علبه", "زاويه", "مسمار"}, task: "1-002-002", cost: "1-002-003"},
	{words: []string{"حواجز خرسانيه", "نيوجيرسي", "سابق الصب", "سابقه الصب"}, task: "1-002-003", cost: "1-002-008"},
	{words: []string{"خرسانه جاهزه", "خرسانه مسلحه"}, task: "1-002-002", cost: "1-002-001"},
	{words: []string{"خرسانه عاديه", "خرسانه نظافه"}, task: "1-002-001", cost: "1-002-001"},
	{words: []string{"بسكوت", "زيت تانل", "زيت شده", "شده", "بلتيه", "بلته", "بونتي", "خشب", "لتزانه", "موسكي"}, not: []string{"باب", "ابواب"}, task: "1-002-006", cost: "1-002-007"},
	{words: []string{"مونه", "ايبوكسي حقن", "ويبر", "تزريع", "اضافات خرسانه", "سيكا"}, task: "1-002-002", cost: "1-002-006"},
	// مباني وبياض
	{words: []string{"بلوك", "طوب", "بلك"}, task: "1-003-001", cost: "1-004-001"},
	{words: []string{"اسمنت", "رمل", "زلط", " جير "}, task: "1-003-001", cost: "1-002-006"},
	{words: []string{"بياض", "محاره", "لياسه", "زوايه بياض", "شبك"}, task: "1-003-002", cost: "1-004-002"},
	// تشطيبات
	{words: []string{"سيراميك", "بورسلين", "porcelain", "ceramic", "ترويبه", "بلاط", "غراء بلاط", "لاصق بلاط", "سكرتنج", "skirting"}, task: "1-003-004", cost: "1-004-004"},
	{words: []string{"رخام", "جرانيت", "marble", "granite"}, task: "1-003-005", cost: "1-004-005"},
	{words: []string{"جبس", "جبسم", "gypsum", "بورد", "قطاعات معلق"}, task: "1-003-006", cost: "1-004-006"},
	{words: []string{"ايبوكسي", "epoxy"}, need: []string{"ارضي"}, task: "1-003-019", cost: "1-004-003"},
	{words: []string{"ورق حائط", "ورق جدران", "wallpaper"}, task: "1-003-003", cost: "1-004-003"},
	{words: []string{"دهان", "طلاء", "بويه", "معجون", "سيلر", "برايمر", "ورنيش", "تنر", "جوتن", "سايبس"}, task: "1-003-003", cost: "1-004-003"},
	{words: []string{"عزل", "جيوتكستايل", "geotextile", "ممبرين", "بيتومين", "اسفلت سائل", "بولي يوريثان", "membrane"}, task: "1-003-012", cost: "1-004-012"},
	{words: []string{"فوم"}, not: []string{"كهرباء"}, task: "1-003-012", cost: "1-004-012"},
	{words: []string{"المونيوم", "الومنيوم", "زجاج", "شباك", "aluminum"}, task: "1-003-010", cost: "1-004-010"},
	{words: []string{"باب", "ابواب"}, need: []string{"خشب"}, task: "1-003-008", cost: "1-004-008"},
	{words: []string{"باب", "ابواب"}, task: "1-003-009", cost: "1-004-009"},
	{words: []string{"درابزين", "هاندريل", "استانلس", "ستانلس"}, task: "1-003-014", cost: "1-004-014"},
	{words: []string{"wpc"}, task: "1-003-007", cost: "1-004-007"},
	{words: []string{"grc"}, task: "1-003-011", cost: "1-004-011"},
	{words: []string{"spc", "فينيل", "باركيه", "ارضيات"}, task: "1-003-019", cost: "1-004-004"},
	// موقع
	{words: []string{"انترلوك", "interlock"}, task: "1-004-002", cost: "1-002-010"},
	{words: []string{"بردوره", "curb"}, task: "1-004-002", cost: "1-002-009"},
	{words: []string{"زرع", "نجيل", "اشجار", "تربه زراعيه"}, task: "1-004-001", cost: "1-005-001"},
	// معدات ووقود وسلامة
	{words: []string{"سولار", "ديزل", "بنزين", "زيت موتور", "زيت محرك", "زيت هيدروليك", "شحم", "فلتر زيت", "فلتر سولار", "كاوتش"}, task: "2-004-004", cost: "2-004-006"},
	{words: []string{"طفايه", "خوذه", "خوزه", "فيست", "سيفتي", "جوانتي", "قفاز", "كمامه", "نظاره", "حزام امان", "بوت", "حذاء"}, task: "2-002-010", cost: "2-002-012"},
	{words: []string{"حبر", "طابعه", "ورق تصوير", "ورق a4", "ادوات مكتبيه", "كرسي مكتب", "دباسه", "ملفات"}, task: "2-002-007", cost: "2-002-008"},
	{words: []string{"مياه شرب", "وايت مياه", "نقل مياه", "مياه للموقع"}, task: "2-002-009", cost: "2-002-011"},
	{words: []string{"متر قياس", "ميزان", "شاكوش", "مطرقه", "زرديه", "بنسه", "مفك", "منشار", "صاروخ", "شنيور", "دريل", "عربيه يد", "جاروف", "كوريك", "مسطرين", "سقالات", "سقاله", "حجر قص", "حجر جلخ", "قلم", "ازميل", "بنطه", "فرشه", "ماكينه", "مكنه"}, task: "2-004-003", cost: "2-004-005"},
	{words: []string{"مسمار", "اكمون", "شريط", "سبراي", "اسبراي", "اسبراى", "ديسك", "سنفره", "سليكون", "خيط", "شكاره", "مشمع", "نايلون", "كيس"}, task: "", cost: "2-002-014"},
}

// normAr بيشيل المسافات اللي على الأطراف — هنا بنحافظ عليها عشان " تي " تبقى كلمة لوحدها
func normKeep(w string) string {
	n := normAr(w)
	if strings.HasPrefix(w, " ") {
		n = " " + n
	}
	if strings.HasSuffix(w, " ") {
		n += " "
	}
	return n
}

func init() {
	for i := range storeKW {
		for j, w := range storeKW[i].words {
			storeKW[i].words[j] = normKeep(w)
		}
		for j, w := range storeKW[i].need {
			storeKW[i].need[j] = normKeep(w)
		}
		for j, w := range storeKW[i].not {
			storeKW[i].not[j] = normKeep(w)
		}
	}
	for k, v := range sectionTaskRules {
		delete(sectionTaskRules, k)
		sectionTaskRules[normAr(k)] = v
	}
}

func containsAny(s string, ws []string) bool {
	for _, w := range ws {
		if w != "" && strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func stKWMatch(text string) (stKW, bool) {
	s := " " + strings.NewReplacer("(", " ", ")", " ", "-", " ", "/", " ", ",", " ", "،", " ", "*", " ").Replace(normAr(text)) + " "
	for _, r := range storeKW {
		if !containsAny(s, r.words) {
			continue
		}
		if len(r.need) > 0 && !containsAny(s, r.need) {
			continue
		}
		if len(r.not) > 0 && containsAny(s, r.not) {
			continue
		}
		return r, true
	}
	return stKW{}, false
}

// القسم في شيت المخازن -> Task Code (لو مفيش ذاكرة)
var sectionTaskRules = map[string]string{
	"انشاءات": "1-002-002", "خرسانه": "1-002-002", "هيكل": "1-002-002", "نجاره": "1-002-006", "حداده": "1-002-002",
	"مباني": "1-003-001", "مبانى": "1-003-001", "بياض": "1-003-002", "محاره": "1-003-002", "لياسه": "1-003-002",
	"دهانات": "1-003-003", "نقاشه": "1-003-003", "سيراميك": "1-003-004", "بلاط": "1-003-004", "رخام": "1-003-005",
	"جبس": "1-003-006", "عزل": "1-003-012", "المونيوم": "1-003-010", "الومنيوم": "1-003-010",
	"صحي": "1-003-013", "كهرباء": "1-005-002", "كهربا": "1-005-002", "تيار خفيف": "1-005-004",
	"ميكانيكا": "1-006-001", "سباكه": "1-006-001", "صرف": "1-006-001", "تكييف": "1-006-003", "حريق": "1-006-004",
	"لاندسكيب": "1-004-001", "زراعه": "1-004-001", "طرق": "1-004-003",
	"jcp": "2-004-004", "معدات": "2-004-004", "صيانه": "2-004-004", "مساحه": "2-004-003", "سيارات": "2-002-004", "امن": "2-002-010", "سلامه": "2-002-010",
	"عموميه": "2-004-003", "اداره": "2-002-007", "ضيافه": "2-002-007", "مكتب": "2-002-007",
}

// Cost Code مناسب لـ Task Code (لما مفيش حاجة أدق)
var taskDefaultCost = map[string]string{
	"1-002-001": "1-002-001", "1-002-002": "1-002-001", "1-002-006": "1-002-007", "1-002-007": "1-002-001",
	"1-003-001": "1-004-001", "1-003-002": "1-004-002", "1-003-003": "1-004-003", "1-003-004": "1-004-004",
	"1-003-005": "1-004-005", "1-003-006": "1-004-006", "1-003-007": "1-004-007", "1-003-008": "1-004-008",
	"1-003-009": "1-004-009", "1-003-010": "1-004-010", "1-003-011": "1-004-011", "1-003-012": "1-004-012",
	"1-003-013": "1-004-013", "1-003-014": "1-004-014", "1-003-019": "1-004-004",
	"1-004-001": "1-005-001", "1-004-002": "1-002-010", "1-005-001": "1-006-004", "1-005-002": "1-006-002",
	"1-005-003": "1-006-003", "1-005-004": "1-006-009", "1-005-005": "1-006-001", "1-005-006": "1-006-001",
	"1-006-001": "1-007-001", "1-006-002": "1-007-001", "1-006-003": "1-007-003", "1-006-004": "1-006-006",
	"1-006-005": "1-006-007", "2-002-004": "2-002-004", "2-002-007": "2-002-008", "2-002-009": "2-002-011",
	"2-002-010": "2-002-012", "2-004-003": "2-004-005", "2-004-004": "2-004-006",
}

const (
	fallbackTask = "2-004-003" // معدات صغيرة وأدوات موقع
	fallbackCost = "2-002-014" // مستلزمات وأدوات موقع عامة
)

// ------------------------------------------------------------------ عيلة كود الصنف
func codeSegs(c string) []string {
	c = strings.TrimSpace(c)
	if c == "" {
		return nil
	}
	return strings.FieldsFunc(c, func(r rune) bool { return r == '.' || r == '-' || r == '/' || r == ' ' })
}

type famSrc struct {
	segs       []string
	task, cost string
	w          float64
}

// أقرب صنف في نفس العيلة: أطول بداية كود مشتركة، ولازم تبقى قريبة كفاية
func familyCodes(code string, pool []famSrc) (string, string, int) {
	s := codeSegs(code)
	if len(s) < 3 {
		return "", "", 0
	}
	need := len(s) - 2
	if need < 3 {
		need = 3
	}
	best := 0
	votes := map[[2]string]float64{}
	for _, p := range pool {
		n := 0
		for n < len(s) && n < len(p.segs) && s[n] == p.segs[n] {
			n++
		}
		if n < need || n == 0 {
			continue
		}
		if n > best {
			best = n
			votes = map[[2]string]float64{}
		}
		if n == best {
			votes[[2]string{p.task, p.cost}] += p.w
		}
	}
	if best == 0 {
		return "", "", 0
	}
	var tot, top float64
	var pick [2]string
	for k, v := range votes {
		tot += v
		if v > top || (v == top && k[0]+k[1] < pick[0]+pick[1]) {
			top, pick = v, k
		}
	}
	if top/tot < 0.6 {
		return "", "", 0
	}
	return pick[0], pick[1], best
}

// ------------------------------------------------------------------ المحرك
type autoCoder struct {
	s        *storeState
	pool     []famSrc
	sheetBy  map[string][2]string // مفتاح الصنف -> أكواد من الشيت نفسه
	secMajor map[string]string    // القسم -> أكتر Task في الشيت/الذاكرة
}

func (s *storeState) newAutoCoder() *autoCoder {
	loadKnown()
	a := &autoCoder{s: s, sheetBy: map[string][2]string{}, secMajor: map[string]string{}}
	secVotes := map[string]map[string]float64{}
	addSec := func(sec, task string, w float64) {
		if sec == "" || task == "" {
			return
		}
		k := normAr(sec)
		if secVotes[k] == nil {
			secVotes[k] = map[string]float64{}
		}
		secVotes[k][task] += w
	}
	for _, x := range s.parsed {
		if x.Task != "" && x.Cost != "" {
			a.sheetBy[x.key] = [2]string{x.Task, x.Cost}
			if x.Code != "" {
				a.pool = append(a.pool, famSrc{codeSegs(x.Code), x.Task, x.Cost, 2})
			}
		}
		if x.Task != "" {
			addSec(x.Section, x.Task, 1)
		}
	}
	smem.mu.Lock()
	for k, v := range smem.Codes {
		if strings.HasPrefix(k, "c:") && v[0] != "" && v[1] != "" {
			a.pool = append(a.pool, famSrc{codeSegs(strings.TrimPrefix(k, "c:")), v[0], v[1], 1.5})
		}
	}
	smem.mu.Unlock()
	for _, k := range knownList {
		if k.C != "" {
			a.pool = append(a.pool, famSrc{codeSegs(normAr(k.C)), k.T, k.K, 1})
		}
	}
	for sec, m := range secVotes {
		var bt string
		var bv float64
		for t, v := range m {
			if v > bv || (v == bv && t < bt) {
				bt, bv = t, v
			}
		}
		a.secMajor[sec] = bt
	}
	return a
}

func (a *autoCoder) sectionTask(sec string) (string, string) {
	if sec == "" {
		return "", ""
	}
	n := normAr(sec)
	smem.mu.Lock()
	t, ok := smem.Sections[n]
	smem.mu.Unlock()
	if ok && t != "" {
		return t, "section"
	}
	if t := a.secMajor[n]; t != "" {
		return t, "section"
	}
	// أطول كلمة من القواعد موجودة في اسم القسم
	var keys []string
	for k := range sectionTaskRules {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		if strings.Contains(n, k) {
			return sectionTaskRules[k], "section"
		}
	}
	return "", ""
}

func (a *autoCoder) valid(task, cost string) bool {
	cl := a.s.cl
	if cl == nil || cl.byT == nil {
		return task != "" && cost != ""
	}
	return task != "" && cost != "" && cl.byT[task] != nil && cl.byC[cost] != nil
}

// code بيرجّع Task و Cost ومصدرهم — ودايماً بيرجّع كود (مفيش صنف يفضل فاضي)
func (a *autoCoder) code(x *sRow) (Pick, Pick) {
	text := x.label() + " " + x.Desc
	ret := func(t, c, src string, sc float64) (Pick, Pick) {
		return Pick{t, src, sc}, Pick{c, src, sc}
	}
	// 1. الذاكرة
	smem.mu.Lock()
	m, ok := smem.Codes[x.key]
	if !ok || m[0] == "" || m[1] == "" {
		// جرّب الكود والاسم الاتنين
		if x.Code != "" {
			if v, ok2 := smem.Codes["c:"+normAr(x.Code)]; ok2 && v[0] != "" && v[1] != "" {
				m, ok = v, true
			}
		}
		if (!ok || m[0] == "" || m[1] == "") && x.label() != "" {
			if v, ok2 := smem.Codes["n:"+normAr(x.label())]; ok2 && v[0] != "" && v[1] != "" {
				m, ok = v, true
			}
		}
	}
	smem.mu.Unlock()
	if ok && m[0] != "" && m[1] != "" {
		return ret(m[0], m[1], "memory", 1)
	}
	// 2. نفس الصنف في الشيت
	if v, ok := a.sheetBy[x.key]; ok {
		return ret(v[0], v[1], "sheet", 1)
	}
	// 3. معروف
	if x.Code != "" {
		if k, ok := knownByCode[normAr(x.Code)]; ok && a.valid(k.T, k.K) {
			return ret(k.T, k.K, "known", 1)
		}
	}
	if k, ok := knownByName[normAr(x.label())]; ok && a.valid(k.T, k.K) {
		return ret(k.T, k.K, "known", 1)
	}
	// 4. نفس العيلة
	if t, c, n := familyCodes(normAr(x.Code), a.pool); t != "" && a.valid(t, c) {
		return ret(t, c, "family", 0.6+0.08*float64(n))
	}
	// 5. اسم شبيه جداً
	if t, c, sc := a.s.nearest(text); sc >= 0.55 && a.valid(t, c) {
		return ret(t, c, "similar", sc)
	}
	// 6. كلمات
	secT, _ := a.sectionTask(x.Section)
	if r, ok := stKWMatch(text); ok {
		t := r.task
		if t == "" {
			t = secT
		}
		if t == "" {
			t = fallbackTask
		}
		if a.valid(t, r.cost) {
			return ret(t, r.cost, "rule", 0.7)
		}
	}
	// 5ب. اسم شبيه بنسبة أقل
	if t, c, sc := a.s.nearest(text); sc >= 0.34 && a.valid(t, c) {
		return ret(t, c, "similar", sc)
	}
	// 7. القسم (Task) + أقرب Cost مواد من قايمة PCS
	if secT != "" {
		c := taskDefaultCost[secT]
		if pc, sc := a.pcsCost(secT, text); pc != "" && sc >= 0.2 {
			c = pc
		}
		if c == "" {
			c = fallbackCost
		}
		if a.valid(secT, c) {
			return ret(secT, c, "section", 0.5)
		}
	}
	// 8. من قايمة PCS: أقرب Task لاسم الصنف، وأقرب Cost مواد ليه
	if t, ts := a.pcsTask(text); t != "" && ts > 0 {
		c := taskDefaultCost[t]
		if pc, cs := a.pcsCost(t, text); pc != "" && (cs > 0 || c == "") {
			c = pc
		}
		if c == "" {
			c = fallbackCost
		}
		if a.valid(t, c) {
			return ret(t, c, "pcs", ts)
		}
	}
	// 9. افتراضي — لو اسم الصنف مالوش أي علاقة بقايمة PCS
	return ret(fallbackTask, fallbackCost, "default", 0.2)
}

// ------------------------------------------------------------------ من قايمة أكواد PCS نفسها
// لو مفيش أي حاجة اتعلمناها تنفع: البرنامج بيدوّر في قايمة PCS على أقرب كود لاسم الصنف

// أقرب Task Code لاسم الصنف (من الأعمال المباشرة الأول)
func (a *autoCoder) pcsTask(text string) (string, float64) {
	cl := a.s.cl
	if cl == nil {
		return "", 0
	}
	v := tokens(text)
	best, bs := "", 0.0
	for _, c := range cl.Task {
		sc := sim(v, c.vec)
		if sc < 0.08 {
			continue // مفيش علاقة حقيقية بين الاسم والكود
		}
		if strings.HasPrefix(c.Code, "1-") {
			sc += 0.02 // المواد غالباً أعمال مباشرة
		}
		if sc > bs {
			best, bs = c.Code, sc
		}
	}
	return best, bs
}

// أقرب Cost Code من نوع مواد (Material) — في نفس الـ Work Package لو ينفع
func (a *autoCoder) pcsCost(task, text string) (string, float64) {
	cl := a.s.cl
	if cl == nil {
		return "", 0
	}
	wp := ""
	if t := cl.byT[task]; t != nil {
		wp = t.WP
	}
	v := tokens(text)
	best, bs := "", 0.0
	for _, c := range cl.Cost {
		ty := strings.ToLower(c.Type)
		if ty != "" && !strings.Contains(ty, "material") && !strings.Contains(ty, "equipment") && !strings.Contains(ty, "overhead") {
			continue // صرف مخازن = مواد، مش مقاول باطن ولا عمالة
		}
		sc := sim(v, c.vec)
		if sc < 0.1 {
			continue
		}
		if wp != "" && c.WP == wp {
			sc += 0.1
		}
		if sc > bs {
			best, bs = c.Code, sc
		}
	}
	return best, bs
}

// اسم المصدر بالعربي (لملف المراجعة)
func autoSrcLabel(src string) string {
	switch src {
	case "sheet-col":
		return "من الشيت"
	case "memory":
		return "ذاكرة البرنامج"
	case "sheet":
		return "نفس الصنف في الشيت"
	case "known":
		return "صنف معروف"
	case "family":
		return "نفس عيلة كود الصنف"
	case "similar":
		return "صنف شبيه"
	case "rule":
		return "من اسم الصنف"
	case "section":
		return "من القسم"
	case "pcs":
		return "من قايمة PCS"
	case "default":
		return "افتراضي — راجعه"
	case "user":
		return "اخترته انت"
	}
	return src
}

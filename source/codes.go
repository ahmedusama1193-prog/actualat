package main

// قوائم أكواد PCS (Task / Cost) + الاقتراح + الذاكرة

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

//go:embed assets/task_codes.xlsx
var embTask []byte

//go:embed assets/cost_codes.xlsx
var embCost []byte

type Code struct {
	Code string `json:"code"`
	WP   string `json:"wp"`
	WPN  string `json:"wpName"`
	EN   string `json:"en"`
	AR   string `json:"ar"`
	Unit string `json:"unit"`
	Type string `json:"type"`
	vec  map[string]bool
}

type CodeLists struct {
	Task []*Code `json:"task"`
	Cost []*Code `json:"cost"`
	byT  map[string]*Code
	byC  map[string]*Code
}

var reDiac = regexp.MustCompile("[ً-ْـ]")
var reSpace = regexp.MustCompile(`\s+`)

func normAr(s string) string {
	s = strings.TrimSpace(s)
	s = reDiac.ReplaceAllString(s, "")
	r := strings.NewReplacer("أ", "ا", "إ", "ا", "آ", "ا", "ة", "ه", "ى", "ي", "ی", "ي", "ك", "ك", "‏", "", "‎", "")
	s = r.Replace(s)
	s = reSpace.ReplaceAllString(s, " ")
	return strings.ToLower(strings.TrimSpace(s))
}

var stop = map[string]bool{"اعمال": true, "عمال": true, "و": true, "من": true, "في": true, "الي": true, "على": true, "علي": true,
	"شامل": true, "توريد": false, "مقاول": true, "باطن": true, "the": true, "and": true, "&": true, "work": true, "works": true, "-": true, "/": true}

func tokens(s string) map[string]bool {
	s = normAr(s)
	s = strings.NewReplacer("(", " ", ")", " ", "/", " ", "-", " ", ",", " ", "،", " ", ".", " ").Replace(s)
	out := map[string]bool{}
	for _, w := range strings.Fields(s) {
		if strings.HasPrefix(w, "ال") && len([]rune(w)) > 4 {
			w = string([]rune(w)[2:])
		}
		if strings.HasPrefix(w, "وال") && len([]rune(w)) > 5 {
			w = string([]rune(w)[3:])
		}
		if stop[w] || len([]rune(w)) < 2 {
			continue
		}
		out[w] = true
	}
	return out
}

func sim(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := 0
	for w := range a {
		if b[w] {
			n++
		} else {
			// تطابق جزئي (المباني / مباني / البناء)
			for v := range b {
				if len([]rune(w)) >= 4 && len([]rune(v)) >= 4 && (strings.Contains(v, w) || strings.Contains(w, v)) {
					n++
					break
				}
			}
		}
	}
	return float64(n) / float64(len(a)+len(b)-n)
}

func readCodeSheet(data []byte) ([]*Code, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := f.GetRows(f.GetSheetList()[0], excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, nil
	}
	idx := map[string]int{}
	for i, h := range rows[0] {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	get := func(r []string, k string) string {
		if i, ok := idx[k]; ok && i < len(r) {
			return strings.TrimSpace(r[i])
		}
		return ""
	}
	var out []*Code
	for _, r := range rows[1:] {
		code := get(r, "task code")
		if code == "" {
			code = get(r, "cost code")
		}
		if code == "" {
			continue
		}
		c := &Code{Code: code, WP: get(r, "work package id"), WPN: get(r, "work package"), EN: get(r, "description (en)"),
			AR: get(r, "description (ar)"), Unit: get(r, "unit"), Type: get(r, "resource type")}
		c.vec = tokens(c.AR + " " + c.EN + " " + c.WPN)
		out = append(out, c)
	}
	return out, nil
}

func (cl *CodeLists) index() {
	cl.byT, cl.byC = map[string]*Code{}, map[string]*Code{}
	for _, c := range cl.Task {
		cl.byT[c.Code] = c
	}
	for _, c := range cl.Cost {
		cl.byC[c.Code] = c
	}
}

func loadCodeLists() (*CodeLists, string) {
	cl := &CodeLists{}
	src := "built-in"
	td, cd := embTask, embCost
	if b, err := os.ReadFile(filepath.Join(appDir(), "Task_Codes.xlsx")); err == nil {
		td, src = b, "file"
	}
	if b, err := os.ReadFile(filepath.Join(appDir(), "Cost_Codes.xlsx")); err == nil {
		cd, src = b, "file"
	}
	cl.Task, _ = readCodeSheet(td)
	cl.Cost, _ = readCodeSheet(cd)
	cl.index()
	return cl, src
}

// ------------------------------------------------------------------ memory
type Memory struct {
	mu   sync.Mutex
	Task map[string]string `json:"task"` // norm(task desc from sheet) -> task code
	Cost map[string]string `json:"cost"` // group key -> cost code
	// مقاول|أعمال#ترتيب -> Contract No (عشان الرقم يفضل ثابت ومايتكررش لشركتين)
	Contract map[string]string `json:"contract"`
}

var mem = &Memory{Task: map[string]string{}, Cost: map[string]string{}, Contract: map[string]string{}}

func memPath() string { return filepath.Join(appDir(), "ipc_memory.json") }

func loadMem() {
	mem.mu.Lock()
	defer mem.mu.Unlock()
	if b, err := os.ReadFile(memPath()); err == nil {
		json.Unmarshal(b, mem)
	}
	if mem.Task == nil {
		mem.Task = map[string]string{}
	}
	if mem.Cost == nil {
		mem.Cost = map[string]string{}
	}
	if mem.Contract == nil {
		mem.Contract = map[string]string{}
	}
}

func saveMem() {
	mem.mu.Lock()
	defer mem.mu.Unlock()
	os.MkdirAll(appDir(), 0o755)
	b, _ := json.MarshalIndent(mem, "", " ")
	os.WriteFile(memPath(), b, 0o644)
}

// ------------------------------------------------------------------ seed rules (مراجَعة يدوياً على قوائم PCS)
// وصف التاسك في شيت المقاولين -> Task Code في PCS
var seedTask = map[string]string{
	"اعمال المبانى":                "1-004-001",
	"اعمال الخرسانة المسلحة":       "1-003-002",
	"اعمال الخرسانة العادية":       "1-003-001",
	"اعمال البياض":                 "1-004-002",
	"اعمال الدهانات":               "1-004-003",
	"اعمال السيراميك والبورسلين":   "1-004-004",
	"اعمال الرخام":                 "1-004-005",
	"اعمال الاسقف المعلقة":         "1-004-006",
	"اعمال الجيبسوم بورد":          "1-004-006",
	"اعمال العزل":                  "1-004-012",
	"أعمال ارضيات SPC":             "1-004-019",
	"اعمال ارضيات خشبية":           "1-004-019",
	"اعمال تكسيات WPC":             "1-004-007",
	"اعمال GRC":                    "1-004-011",
	"اعمال الحفر":                  "1-002-001",
	"اعمال الردم والاحلال":         "1-002-002",
	"أعمال مكافحه الحشرات":         "1-002-003",
	"اعمال الجسات":                 "1-001-003",
	"اعمال مصنعيات كهرباء":         "1-006-012",
	"اعمال السباكة مياه وصرف":      "1-007-001",
	"اعمال مصنعيات سباكة":          "1-007-001",
	"اعمال نظام التكيفات":          "1-007-003",
	"اعمال التكيفات والتهويات":     "1-007-003",
	"اعمال خزان المياه":            "1-007-005",
	"اعمال المضخات":                "1-007-005",
	"اعمال شبكه الحريق":            "1-007-004",
	"اعمال اللاند سكيب":            "1-005-001",
	"اعمال الطرق والاسفلت":         "1-005-003",
	"المعدات المؤجره":              "2-004-001",
	"المعدات المملوكه للشركه":      "2-004-002",
	"تكلفه صيانه المعدات":          "2-004-004",
	"امن وحراسات":                  "2-002-010",
	"سكن العماله":                  "2-002-009",
	"اتعاب الاستشاري":              "2-003-002",
	"اعمال الهدم و التكسير والنزع": "2-001-004",
}

// كلمات في عنوان العقد بتغلب على وصف التاسك في الشيت
var seedTradeTask = map[string]string{
	"مخلفات": "2-001-004",
}

// Task Code -> Cost Code الافتراضي لمقاولي الباطن
var seedCost = map[string]string{
	"1-001-003": "1-001-004",
	"1-002-001": "1-002-201", "1-002-002": "1-002-201", "1-002-003": "1-002-203",
	"1-003-001": "1-003-202", "1-003-002": "1-003-202", "1-003-006": "1-003-202", "1-003-007": "1-003-202",
	"1-004-001": "1-004-206", "1-004-002": "1-004-206", "1-004-003": "1-004-206", "1-004-004": "1-004-206",
	"1-004-005": "1-004-206", "1-004-006": "1-004-201", "1-004-007": "1-004-206", "1-004-008": "1-004-207",
	"1-004-009": "1-004-207", "1-004-010": "1-004-202", "1-004-011": "1-004-205", "1-004-012": "1-004-203",
	"1-004-014": "1-004-207", "1-004-017": "1-004-207", "1-004-019": "1-004-206", "1-004-025": "1-004-205",
	"1-005-001": "1-005-001", "1-005-002": "1-005-001",
	"1-006-001": "1-006-202", "1-006-002": "1-006-202", "1-006-004": "1-006-201", "1-006-012": "1-006-202",
	"1-007-001": "1-007-202", "1-007-002": "1-007-202", "1-007-003": "1-007-201", "1-007-005": "1-007-202",
	"1-007-010": "1-007-202",
	"2-001-004": "2-002-201",
	"2-002-009": "2-002-011", "2-002-010": "2-002-002",
	"2-003-002": "1-001-006",
	"2-004-001": "2-004-001", "2-004-004": "2-004-006",
}

// كلمات في وصف البند / عنوان العقد بتغيّر الـ Cost Code
type kwRule struct {
	task  string
	words []string
	code  string
}

var seedKw = []kwRule{
	{"2-004-001", []string{"بوبكات", "بوب كات", "بوكلين", "حفار", "لودر", "قلاب", "سطحه", "سطحة", "تريلة", "جريدر"}, "2-004-002"},
	{"2-004-001", []string{"مولد", "كمبروسر", "كومبريسور"}, "2-004-003"},
	{"2-004-001", []string{"كرين", "ونش", "رافعة", "رافعه"}, "2-004-001"},
	{"1-007-005", []string{"توريد مضخات", "مضخة", "مضخه"}, "1-006-007"},
	{"1-004-005", []string{"توريد رخام", "تصنيع و توريد رخام"}, "1-004-005"},
}

type Pick struct {
	Code  string  `json:"code"`
	Src   string  `json:"src"` // memory | rule | suggest | none
	Score float64 `json:"score"`
}

func taskKey(taskDesc, trade string) string { return normAr(taskDesc) + "|" + normAr(trade) }

func (cl *CodeLists) suggestTask(taskDesc, wpName, trade string) Pick {
	k := normAr(taskDesc)
	if k == "" {
		k = normAr(trade)
	}
	mem.mu.Lock()
	m, ok := mem.Task[taskKey(taskDesc, trade)]
	mem.mu.Unlock()
	if ok && cl.byT[m] != nil {
		return Pick{m, "memory", 1}
	}
	nt := normAr(trade)
	for w, c := range seedTradeTask {
		if strings.Contains(nt, normAr(w)) && cl.byT[c] != nil {
			return Pick{c, "rule", 1}
		}
	}
	for d, c := range seedTask {
		if normAr(d) == k && cl.byT[c] != nil {
			return Pick{c, "rule", 1}
		}
	}
	v := tokens(taskDesc + " " + trade)
	best, bs := "", 0.0
	for _, c := range cl.Task {
		s := sim(v, c.vec)
		if wpName != "" && strings.Contains(normAr(c.WPN), strings.Fields(normAr(wpName) + " x")[0]) {
			s += 0.05
		}
		if s > bs {
			best, bs = c.Code, s
		}
	}
	if bs < 0.08 {
		return Pick{"", "none", 0}
	}
	return Pick{best, "suggest", bs}
}

func costKey(taskDesc, trade, bucket string) string {
	return normAr(taskDesc) + "|" + normAr(trade) + "|" + bucket
}

// bucket = الكلمة المفتاحية اللي بتفرق البنود جوه نفس العقد (معدات مثلاً)
func kwBucket(task, text string) (string, string) {
	t := normAr(text)
	for _, r := range seedKw {
		if r.task != task {
			continue
		}
		for _, w := range r.words {
			if strings.Contains(t, normAr(w)) {
				return r.code, normAr(w)
			}
		}
	}
	return "", ""
}

func (cl *CodeLists) suggestCost(task, key, text string) Pick {
	mem.mu.Lock()
	m, ok := mem.Cost[key]
	mem.mu.Unlock()
	if ok && cl.byC[m] != nil {
		return Pick{m, "memory", 1}
	}
	if c, _ := kwBucket(task, text); c != "" && cl.byC[c] != nil {
		return Pick{c, "rule", 1}
	}
	if c, ok := seedCost[task]; ok && cl.byC[c] != nil {
		return Pick{c, "rule", 1}
	}
	// أقرب Cost Code من نوع Subcontractor في نفس الـ Work Package
	wp := ""
	if t := cl.byT[task]; t != nil {
		wp = t.WP
	}
	v := tokens(text)
	type sc struct {
		c *Code
		s float64
	}
	var all []sc
	for _, c := range cl.Cost {
		s := sim(v, c.vec)
		if wp != "" && c.WP == wp {
			s += 0.15
		}
		if strings.Contains(strings.ToLower(c.Type), "sub") {
			s += 0.1
		}
		all = append(all, sc{c, s})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].s > all[j].s })
	if len(all) == 0 || all[0].s < 0.12 {
		return Pick{"", "none", 0}
	}
	sc0 := all[0].s
	if sc0 > 1 {
		sc0 = 1
	}
	return Pick{all[0].c.Code, "suggest", sc0}
}

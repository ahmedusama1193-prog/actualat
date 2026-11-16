package main

// 2.15: أصناف من غير سعر في شيت المخازن
// 1) البرنامج بيدور على سعر نفس الصنف في أقرب شهر (قبل أو بعد) في نفس الشيت/الملفات
// 2) اللي مالقاش ليه سعر بيظهر في جدول قبل التحويل وانت بتكتب سعره (وبيتحفظ للمرة الجاية كاقتراح)

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

func monthIdx(k string) int {
	var y, m int
	if len(k) >= 7 {
		for _, c := range k[:4] {
			y = y*10 + int(c-'0')
		}
		m = int(k[5]-'0')*10 + int(k[6]-'0')
	}
	return y*12 + m
}

type noPriceItem struct {
	Key    string   `json:"key"`
	Code   string   `json:"code"`
	Name   string   `json:"name"`
	Unit   string   `json:"unit"`
	Rows   int      `json:"rows"`
	Qty    float64  `json:"qty"`
	Months []string `json:"months"`
	Saved  float64  `json:"saved,omitempty"` // آخر سعر دخلته للصنف ده
}

type fillLine struct {
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Rows  int     `json:"rows"`
	Qty   float64 `json:"qty"`
	From  string  `json:"from"`
	Price float64 `json:"price"`
	Amt   float64 `json:"amount"`
}

func nameKey(x *sRow) string { return "n:" + normAr(x.label()) }

// fillPrices: بعد parse — بيحط سعر أقرب شهر للأصناف اللي مالهاش سعر، ويرجّع اللي مالقاش ليها سعر
func (s *storeState) fillPrices() []*noPriceItem {
	type acc struct{ q, a float64 }
	byKey := map[string]map[string]*acc{} // key -> month -> sums
	add := func(k, m string, q, a float64) {
		if byKey[k] == nil {
			byKey[k] = map[string]*acc{}
		}
		if byKey[k][m] == nil {
			byKey[k][m] = &acc{}
		}
		byKey[k][m].q += q
		byKey[k][m].a += a
	}
	for _, x := range s.parsed {
		x.NoPrice, x.Fill, x.FillM, x.PriceBy, x.FillName = false, 0, "", "", ""
		if x.Qty == 0 {
			continue
		}
		if x.amount() == 0 {
			x.NoPrice = true
			continue
		}
		m := "0000-00"
		if x.HasD {
			m = x.Date.Format("2006-01")
		}
		add(x.key, m, x.Qty, x.amount())
		if nk := nameKey(x); nk != x.key {
			add(nk, m, x.Qty, x.amount())
		}
	}
	nearest := func(k string, x *sRow) (float64, string) {
		ms := byKey[k]
		if len(ms) == 0 {
			return 0, ""
		}
		var keys []string
		for m := range ms {
			if ms[m].q != 0 && ms[m].a != 0 {
				keys = append(keys, m)
			}
		}
		if len(keys) == 0 {
			return 0, ""
		}
		sort.Strings(keys)
		best, bd := "", math.MaxInt32
		if !x.HasD {
			best = keys[len(keys)-1] // من غير تاريخ: آخر شهر
		} else {
			t := monthIdx(x.Date.Format("2006-01"))
			for _, m := range keys {
				if m == "0000-00" {
					continue
				}
				d := monthIdx(m) - t
				if d < 0 {
					d = -d
				}
				if d < bd { // التعادل: الشهر اللي قبل (الأقدم) لأنه اتشاف الأول
					best, bd = m, d
				}
			}
			if best == "" {
				best = keys[0]
			}
		}
		a := ms[best]
		return a.a / a.q, best
	}
	// أسماء الأصناف اللي ليها سعر (للبحث بأقرب اسم)
	var plist []*pfName
	seenN := map[string]bool{}
	for _, x := range s.parsed {
		if x.NoPrice || x.Qty == 0 {
			continue
		}
		if k := nameKey(x); !seenN[k] && byKey[k] != nil {
			seenN[k] = true
			plist = append(plist, &pfName{name: x.label(), key: k, tk: pfTokens(x.label())})
		}
	}
	items := map[string]*noPriceItem{}
	var order []string
	for _, x := range s.parsed {
		if !x.NoPrice {
			continue
		}
		p, m := nearest(x.key, x)
		if p == 0 {
			p, m = nearest(nameKey(x), x)
		}
		like := ""
		if p == 0 { // نفس الصنف مكتوب بشكل مختلف شوية (مسافة/حرف/ترتيب كلام) — بنفس المقاسات
			if c := pfBest(x.label(), plist); c != nil {
				p, m = nearest(c.key, x)
				like = c.name
			}
		}
		if p != 0 {
			x.Fill, x.FillM, x.PriceBy, x.FillName = p, m, "month", like
			x.Rate, x.HasR = p, true
			continue
		}
		it := items[x.key]
		if it == nil {
			it = &noPriceItem{Key: x.key, Code: x.Code, Name: x.label(), Unit: x.Unit}
			smem.mu.Lock()
			it.Saved = smem.Prices[x.key]
			smem.mu.Unlock()
			items[x.key] = it
			order = append(order, x.key)
		}
		it.Rows++
		it.Qty += x.Qty
		if x.HasD {
			mk := x.Date.Format("2006-01")
			if len(it.Months) == 0 || it.Months[len(it.Months)-1] != mk {
				dup := false
				for _, v := range it.Months {
					dup = dup || v == mk
				}
				if !dup {
					it.Months = append(it.Months, mk)
				}
			}
		}
	}
	out := make([]*noPriceItem, 0, len(order))
	for _, k := range order {
		it := items[k]
		it.Qty = r2(it.Qty)
		sort.Strings(it.Months)
		for i, m := range it.Months {
			it.Months[i] = arMonthLabel(m)
		}
		out = append(out, it)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Qty > out[j].Qty })
	return out
}

// applyPrices: قبل التحويل — الأسعار اللي كتبتها للأصناف اللي مالقاش ليها سعر
func (s *storeState) applyPrices(pr map[string]float64) {
	for _, x := range s.parsed {
		if !x.NoPrice || x.PriceBy == "month" {
			continue
		}
		x.Rate, x.HasR, x.PriceBy = 0, false, ""
		if p := pr[x.key]; p > 0 {
			x.Rate, x.HasR, x.PriceBy = p, true, "user"
		}
	}
	if len(pr) > 0 {
		smem.mu.Lock()
		for k, p := range pr {
			if p > 0 {
				smem.Prices[k] = p
			}
		}
		smem.mu.Unlock()
		saveStoreMem()
	}
}

// fillSummary: الأصناف اللي خدت سعر من شهر تاني (للعرض وشيت «الأسعار»)
func (s *storeState) fillSummary() map[string]interface{} {
	lines := s.priceLines("month", nil)
	var amt float64
	rows := 0
	for _, l := range lines {
		amt += l.Amt
		rows += l.Rows
	}
	return map[string]interface{}{"items": len(lines), "rows": rows, "amount": r2(amt), "lines": lines}
}

func (s *storeState) priceLines(by string, sel map[string]bool) []*fillLine {
	m := map[string]*fillLine{}
	var order []string
	for _, x := range s.parsed {
		if !x.NoPrice || x.PriceBy != by {
			continue
		}
		if sel != nil && x.HasD && !sel[x.Date.Format("2006-01")] {
			continue
		}
		k := x.key + "|" + x.FillM + "|" + x.FillName
		l := m[k]
		if l == nil {
			from := "سعر دخلته"
			if by == "month" {
				from = arMonthLabel(x.FillM)
				if x.FillM == "0000-00" {
					from = "صفوف من غير تاريخ"
				}
				if x.FillName != "" {
					from += " — صنف بنفس الاسم تقريبًا: " + x.FillName
				}
			}
			l = &fillLine{Code: x.Code, Name: x.label(), Unit: x.Unit, From: from, Price: math.Round(x.Rate*1e4) / 1e4}
			m[k] = l
			order = append(order, k)
		}
		l.Rows++
		l.Qty += x.Qty
		l.Amt += x.amount()
	}
	out := make([]*fillLine, 0, len(order))
	for _, k := range order {
		l := m[k]
		l.Qty, l.Amt = r2(l.Qty), r2(l.Amt)
		out = append(out, l)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Amt > out[j].Amt })
	return out
}

// fillUnits (2.15): صفوف من غير وحدة (مثلاً ملف مافيهوش عمود الوحدة) ← وحدة نفس الصنف من أقرب شهر فيه وحدة
func (s *storeState) fillUnits() map[string]interface{} {
	byKey := map[string]map[string]string{} // key -> month -> unit
	add := func(k, m, u string) {
		if byKey[k] == nil {
			byKey[k] = map[string]string{}
		}
		if _, ok := byKey[k][m]; !ok {
			byKey[k][m] = u
		}
	}
	for _, x := range s.parsed {
		if x.Unit == "" {
			continue
		}
		m := "0000-00"
		if x.HasD {
			m = x.Date.Format("2006-01")
		}
		add(x.key, m, x.Unit)
		add(nameKey(x), m, x.Unit)
	}
	pick := func(k string, x *sRow) string {
		ms := byKey[k]
		if len(ms) == 0 {
			return ""
		}
		best, bd := "", math.MaxInt32
		for m, u := range ms {
			d := 0
			if x.HasD && m != "0000-00" {
				d = monthIdx(m) - monthIdx(x.Date.Format("2006-01"))
				if d < 0 {
					d = -d
				}
			} else {
				d = 1 << 20
			}
			if d < bd || (d == bd && m < best) {
				best, bd = m, d
				_ = u
			}
		}
		return ms[best]
	}
	rows, items, miss := 0, map[string]bool{}, map[string]bool{}
	for _, x := range s.parsed {
		if x.Unit != "" {
			continue
		}
		u := pick(x.key, x)
		if u == "" {
			u = pick(nameKey(x), x)
		}
		if u == "" {
			miss[x.key] = true
			continue
		}
		x.Unit = u
		rows++
		items[x.key] = true
	}
	return map[string]interface{}{"rows": rows, "items": len(items), "missing": len(miss)}
}

// 2.16: أسعار من ملف إكسيل — بس لما تختار «أيوه» في شاشة الأصناف اللي من غير سعر
// الملف: أي شيت فيه كود الصنف و/أو اسم الصنف + السعر (أو القيمة والكمية). البرنامج بيلاقي الشيت والعناوين لوحده.
func hStorePriceFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		fail(w, fmt.Errorf("اختار ملف"))
		return
	}
	fh, hdr, err := r.FormFile("file")
	if err != nil {
		fail(w, fmt.Errorf("اختار ملف"))
		return
	}
	defer fh.Close()
	data, _ := io.ReadAll(fh)
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		fail(w, fmt.Errorf("مش قادر أفتح الملف: %v", err))
		return
	}
	defer f.Close()
	type cand struct {
		sheet                       string
		rows                        [][]string
		h                           int
		code, name, price, amt, qty int
		n                           int
	}
	var best *cand
	for _, sh := range f.GetSheetList() {
		rows, err := f.GetRows(sh, excelize.Options{RawCellValue: true})
		if err != nil || len(rows) < 2 {
			continue
		}
		h, cols := pfHeader(rows)
		if h < 0 {
			continue
		}
		c := &cand{sheet: sh, rows: rows, h: h, code: cols[0], name: cols[1], price: cols[2], amt: cols[3], qty: cols[4]}
		// الشيت اللي فيه أكتر صفوف بسعر هو اللي بيتاخد
		for i := h + 1; i < len(rows); i++ {
			r := rows[i]
			j := c.price
			if j < 0 {
				j = c.amt
			}
			if j >= 0 && j < len(r) {
				if v, ok := numv(strings.TrimSpace(r[j])); ok && v != 0 {
					c.n++
				}
			}
		}
		if c.n > 0 && (best == nil || c.n > best.n) {
			best = c
		}
	}
	if best == nil {
		fail(w, fmt.Errorf("مالقيتش في الملف شيت فيه عمود كود الصنف أو اسمه + عمود السعر"))
		return
	}
	type acc struct {
		q, a, p float64
		n       int
	}
	byCode, byName := map[string]*acc{}, map[string]*acc{}
	var plist []*pfName
	codeName := map[string]string{}
	get := func(r []string, j int) string {
		if j < 0 || j >= len(r) {
			return ""
		}
		return strings.TrimSpace(r[j])
	}
	put := func(m map[string]*acc, k string, q, a, p float64) {
		if k == "" {
			return
		}
		x := m[k]
		if x == nil {
			x = &acc{}
			m[k] = x
		}
		if q != 0 && a != 0 {
			x.q += q
			x.a += a
		}
		if p != 0 {
			x.p += p
			x.n++
		}
	}
	lines := 0
	for i := best.h + 1; i < len(best.rows); i++ {
		r := best.rows[i]
		p, _ := numv(get(r, best.price))
		q, _ := numv(get(r, best.qty))
		a, _ := numv(get(r, best.amt))
		if p == 0 && (q == 0 || a == 0) {
			continue
		}
		lines++
		code, name := get(r, best.code), cleanS(get(r, best.name))
		if code != "" {
			put(byCode, "c:"+normAr(code), q, a, p)
			if _, ok := codeName[normAr(code)]; !ok {
				codeName[normAr(code)] = name
			}
		}
		if name != "" {
			k := "n:" + normAr(name)
			if byName[k] == nil {
				plist = append(plist, &pfName{name: name, key: k, tk: pfTokens(name)})
			}
			put(byName, k, q, a, p)
		}
	}
	val := func(x *acc) float64 {
		if x == nil {
			return 0
		}
		if x.q != 0 && x.a != 0 {
			return x.a / x.q
		}
		if x.n > 0 {
			return x.p / float64(x.n)
		}
		return 0
	}
	// طابق على الأصناف اللي لسه من غير سعر — باسم الصنف بس (مش بالكود): نفس الاسم، وإلا أقرب اسم
	matched := map[string]float64{}
	names := map[string]string{}
	seen := map[string]bool{}
	for _, x := range sst.parsed {
		if !x.NoPrice || x.PriceBy == "month" || seen[x.key] {
			continue
		}
		seen[x.key] = true
		p, nm := 0.0, ""
		if x.Code != "" { // بالكود الأول
			if a := byCode["c:"+normAr(x.Code)]; a != nil {
				p, nm = val(a), codeName[normAr(x.Code)]
			}
		}
		if p == 0 { // بنفس الاسم
			p = val(byName[nameKey(x)])
		}
		if p == 0 { // بأقرب اسم — عربي أو إنجليزي
			if cs := pfMatches(x.label(), plist); len(cs) > 0 {
				sum := 0.0
				for _, c := range cs {
					sum += val(byName[c.key])
				}
				p, nm = sum/float64(len(cs)), cs[0].name
				if len(cs) > 1 {
					nm = fmt.Sprintf("%s (+%d صنف شبهه — متوسط سعرهم)", nm, len(cs)-1)
				}
			}
		}
		if p > 0 {
			matched[x.key] = math.Round(p*1e4) / 1e4
			if nm != "" {
				names[x.key] = nm
			}
		}
	}
	writeJSON(w, 200, map[string]interface{}{"file": hdr.Filename, "sheet": best.sheet, "lines": lines,
		"matched": matched, "names": names, "count": len(matched), "total": len(seen)})
}

// pfHeader: عناوين ملف الأسعار بمرونة (أي عنوان «فيه» الكلمة): صف العناوين و [كود، اسم، سعر، قيمة، كمية]
func pfHeader(rows [][]string) (int, [5]int) {
	has := func(n string, ws ...string) bool {
		for _, w := range ws {
			if strings.Contains(n, normAr(w)) {
				return true
			}
		}
		return false
	}
	bestH, bestS := -1, 0
	var bestC [5]int
	for i := 0; i < len(rows) && i < 40; i++ {
		c := [5]int{-1, -1, -1, -1, -1}
		cs := [5]int{}
		set := func(k, j, pr int) {
			if c[k] < 0 || pr > cs[k] {
				c[k], cs[k] = j, pr
			}
		}
		for j, x := range rows[i] {
			n := normH(x)
			if n == "" || len([]rune(n)) > 40 {
				continue
			}
			skip := has(n, "مستوي", "تصنيف", "تاسك", "task", "cost", "تكلفه", "level", "نسبه", "%")
			switch {
			case !skip && has(n, "كود", "code", "رقم الصنف", "رقم البند"):
				pr := 1
				if has(n, "صنف", "بند", "item", "ماده", "resource") {
					pr = 3
				} else if n == "كود" || n == "code" || n == "الكود" {
					pr = 2
				}
				set(0, j, pr)
			case has(n, "سعر", "price", "rate", "unit cost", "الفئه"):
				pr := 1
				if has(n, "وحده", "unit") {
					pr = 2
				}
				set(2, j, pr)
			case has(n, "كميه", "qty", "quantity"):
				set(4, j, 1)
			case has(n, "قيمه", "اجمالي", "amount", "total", "value"):
				set(3, j, 1)
			case !skip && has(n, "اسم", "الصنف", "البند", "الوصف", "بيان", "item", "description", "name", "material", "ماده"):
				pr := 1
				if has(n, "اسم", "name") {
					pr = 2
				}
				set(1, j, pr)
			}
		}
		if c[0] < 0 && c[1] < 0 {
			continue
		}
		if c[2] < 0 && (c[3] < 0 || c[4] < 0) {
			continue
		}
		sc := 0
		for _, v := range c {
			if v >= 0 {
				sc++
			}
		}
		if sc > bestS {
			bestH, bestS, bestC = i, sc, c
		}
	}
	return bestH, bestC
}

type pfName struct {
	name, key string
	tk        map[string]bool
}

var reNumUnit = regexp.MustCompile(`(\d+(?:\.\d+)?)`)

// pfTokens: كلمات الاسم (عربي/إنجليزي) + الأرقام لوحدها + توحيد الوحدات (مم=mm، سم=cm)
func pfTokens(s string) map[string]bool {
	s = normAr(s)
	s = strings.NewReplacer("×", " ", "*", " x ", "\\", " ", "\"", " ", "'", " ").Replace(s)
	out := map[string]bool{}
	// كلمة فيها شرطة (ppr-uv) بتتحسب كمان ملزوقة (ppruv)
	for _, w := range strings.Fields(s) {
		if strings.Contains(w, "-") || strings.Contains(w, "/") {
			j := strings.NewReplacer("-", "", "/", "").Replace(w)
			if j != "" && !isNum(j) {
				out[j] = true
			}
		}
	}
	s = reNumUnit.ReplaceAllString(s, " $1 ")
	for _, w := range strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r == '.' || r >= 'a' && r <= 'z' || r >= 0x0621 && r <= 0x064A)
	}) {
		w = strings.Trim(w, ".")
		switch w {
		case "مم", "ملي", "ملم":
			w = "mm"
		case "سم":
			w = "cm"
		case "م", "متر":
			w = "m"
		case "وات", "watt":
			w = "w"
		case "انش", "بوصه", "inches", "in":
			w = "inch"
		case "كجم", "كيلو", "kgs":
			w = "kg"
		case "x", "", "و", "of", "with", "and", "the", "for", "size", "مقاس", "نوع", "ref":
			continue
		}
		if strings.HasPrefix(w, "ال") && len([]rune(w)) > 4 {
			w = string([]rune(w)[2:])
		}
		if w == "" {
			continue
		}
		out[canonWord(w)] = true
	}
	if out["pipe"] && out["electrical"] { // ماسورة كهرباء = conduit
		out["conduit"] = true
		delete(out, "pipe")
		delete(out, "electrical")
	}
	return out
}

func isNum(w string) bool { return w != "" && strings.Trim(w, "0123456789.") == "" }

// pfBest / pfMatches: أقرب اسم — لازم معظم كلمات اسم الصنف (بعد الترجمة عربي↔إنجليزي) تكون موجودة، والمقاسات (الأرقام) كلها موجودة.
// لو فيه كذا اسم بنفس الدرجة (زي خوذة بكذا لون) بيرجعهم كلهم والسعر بيبقى متوسطهم.
func pfMatches(name string, list []*pfName) []*pfName {
	a := pfTokens(name)
	if len(a) == 0 {
		return nil
	}
	wa := 0
	for w := range a {
		if !isNum(w) && !unitTok[w] {
			wa++
		}
	}
	if wa == 0 {
		return nil
	}
	var out []*pfName
	bc, bj := 0.0, 0.0
	for _, c := range list {
		n, wh, numA, numHit := 0, 0, 0, 0
		for w := range a {
			hit := c.tk[w]
			if hit {
				n++
			}
			if isNum(w) {
				numA++
				if hit {
					numHit++
				}
			} else if !unitTok[w] && hit {
				wh++
			}
		}
		if numHit < numA || wh == 0 || (wa >= 2 && wh < 2) {
			continue
		}
		cov := float64(wh) / float64(wa)
		if cov < 0.6 {
			continue
		}
		j := float64(n) / float64(len(a)+len(c.tk)-n)
		if j < 0.12 { // اسم طويل فيه الكلمة بالصدفة (زي باب كامل فيه «Lock») — مش نفس الصنف
			continue
		}
		switch {
		case cov > bc+1e-9 || (cov > bc-1e-9 && j > bj+1e-9):
			out, bc, bj = []*pfName{c}, cov, j
		case cov > bc-1e-9 && j > bj-1e-9:
			out = append(out, c)
		}
	}
	return out
}

func pfBest(name string, list []*pfName) *pfName {
	if m := pfMatches(name, list); len(m) > 0 {
		return m[0]
	}
	return nil
}

var unitTok = map[string]bool{"mm": true, "cm": true, "m": true, "inch": true, "w": true, "kg": true, "l": true, "v": true, "pcs": true}

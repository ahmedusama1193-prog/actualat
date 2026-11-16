package main

// 2.15: أصناف من غير سعر في شيت المخازن
// 1) البرنامج بيدور على سعر نفس الصنف في أقرب شهر (قبل أو بعد) في نفس الشيت/الملفات
// 2) اللي مالقاش ليه سعر بيظهر في جدول قبل التحويل وانت بتكتب سعره (وبيتحفظ للمرة الجاية كاقتراح)

import (
	"math"
	"sort"
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
		x.NoPrice, x.Fill, x.FillM, x.PriceBy = false, 0, "", ""
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
		if p != 0 {
			x.Fill, x.FillM, x.PriceBy = p, m, "month"
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
		k := x.key + "|" + x.FillM
		l := m[k]
		if l == nil {
			from := "سعر دخلته"
			if by == "month" {
				from = arMonthLabel(x.FillM)
				if x.FillM == "0000-00" {
					from = "صفوف من غير تاريخ"
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

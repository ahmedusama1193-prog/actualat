package main

// المستخلصات — شهر بشهر (تراكمي): ملف PCS لكل شهر، تراكمي لحد آخر الشهر،
// + ملف مراجعة واحد مجمّع فيه ملخص الشهور وشيت لكل شهر. كل شهر بيتسجل في السجل لوحده.

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type monthOut struct {
	Key    string
	Date   string
	Res    *Result
	Out    []byte
	Review []byte
}

// تاريخ المستخلص لآخر الشهر (زي اللي الواجهة بتحطه: يوم 30، أو آخر يوم لو الشهر أقصر)
func monthEndDate(k string) string {
	t, err := time.Parse("2006-01", k)
	if err != nil {
		return ""
	}
	last := t.AddDate(0, 1, -1).Day()
	d := 30
	if last < d {
		d = last
	}
	return fmt.Sprintf("%s-%02d", k, d)
}

func convertMonthly(w http.ResponseWriter, o Opts) {
	months := append([]string(nil), o.Months...)
	sort.Strings(months)
	// شيل التكرار والشهور اللي بعد الـ cut-off
	var ms []string
	for i, m := range months {
		if i > 0 && m == months[i-1] {
			continue
		}
		if st.rep.Cutoff != "" && m > st.rep.Cutoff {
			continue
		}
		ms = append(ms, m)
	}
	if len(ms) == 0 {
		fail(w, fmt.Errorf("اختار شهر واحد على الأقل"))
		return
	}
	var outs []*monthOut
	type mSum struct {
		Key       string  `json:"key"`
		Label     string  `json:"label"`
		Date      string  `json:"date"`
		Contracts int     `json:"contracts"`
		Lines     int     `json:"lines"`
		Works     float64 `json:"works"`
		Deduct    float64 `json:"deduct"`
		Net       float64 `json:"net"`
		Ret       float64 `json:"ret"`
		Move      float64 `json:"move"`
		Match     bool    `json:"match"`
	}
	var sums []mSum
	prevNet := 0.0
	allMatch := true
	for _, m := range ms {
		mo := o
		mo.Mode, mo.UpTo, mo.Date = "cumulative", m, monthEndDate(m)
		res := st.rep.Convert(st.cl, mo)
		res.Mode = "monthly"
		out, err := WritePCS(res)
		if err != nil {
			fail(w, err)
			return
		}
		rv, _ := WriteReview(res, st.groups, mo, st.cl)
		outs = append(outs, &monthOut{Key: m, Date: mo.Date, Res: res, Out: out, Review: rv})
		net := r2(res.OutWorks - res.OutDeduct)
		sums = append(sums, mSum{m, monthLabel(m), mo.Date, len(res.Contracts), len(res.Rows), res.OutWorks, res.OutDeduct, net, res.OutRet, r2(net - prevNet), res.Match})
		prevNet = net
		allMatch = allMatch && res.Match
	}
	// آخر شهر لازم يساوي "مرة واحدة" لحد نفس الشهر
	last := outs[len(outs)-1]
	full := o
	full.Mode, full.UpTo, full.Date = "cumulative", "", last.Date
	ref := st.rep.Convert(st.cl, full)
	lastMatchesSheet := last.Key != st.rep.Cutoff ||
		math.Abs((last.Res.OutWorks-last.Res.OutDeduct)-(ref.OutWorks-ref.OutDeduct)) < 0.01 && math.Abs(last.Res.OutRet-ref.OutRet) < 0.01
	st.monthly = outs
	st.res, st.opts = last.Res, o
	st.out, st.review = nil, buildMonthlyReview(outs)
	writeJSON(w, 200, map[string]interface{}{
		"monthly": sums, "match": allMatch && lastMatchesSheet, "lastMatchesSheet": lastMatchesSheet,
		"sheetNet": ref.SheetNet, "sheetRet": ref.SheetRet, "cutoff": st.rep.Cutoff,
		"result": last.Res, "lines": len(last.Res.Rows),
	})
}

// ملف المراجعة المجمّع: ملخص الشهور + شيت لكل شهر + ربط الأكواد + ملاحظات آخر شهر
func buildMonthlyReview(outs []*monthOut) []byte {
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
	rows := [][]interface{}{{"الشهر", "تاريخ المستخلص", "عدد المستخلصات", "عدد السطور", "قيمة البنود (تراكمي)", "الخصومات (تراكمي)", "الصافي (تراكمي)", "المحتجز (تراكمي)", "حركة الشهر (الصافي)", "اسم ملف PCS"}}
	prev := 0.0
	for _, m := range outs {
		net := r2(m.Res.OutWorks - m.Res.OutDeduct)
		rows = append(rows, []interface{}{monthLabel(m.Key), m.Date, float64(len(m.Res.Contracts)), float64(len(m.Res.Rows)),
			m.Res.OutWorks, m.Res.OutDeduct, net, m.Res.OutRet, r2(net - prev), monthlyFileName(m)})
		prev = net
	}
	put("ملخص الشهور", rows, []float64{12, 14, 12, 10, 18, 16, 18, 16, 18, 46})
	cols := []string{"Subcontractor", "Contract No", "Trade", "Item No", "Description", "Unit", "Rate", "Cumulative Qty", "Amount To Date",
		"IPC No", "IPC Date", "Task Code", "Cost Code", "Retention %", "Retention This IPC", "Other Deductions This IPC"}
	for _, m := range outs {
		rr := [][]interface{}{}
		h := make([]interface{}, len(cols))
		for i, c := range cols {
			h[i] = c
		}
		rr = append(rr, h)
		for _, row := range m.Res.Rows {
			line := make([]interface{}, len(cols))
			for i, c := range cols {
				v := row[c]
				if n, ok := v.(int); ok {
					v = float64(n)
				}
				if v == nil {
					v = ""
				}
				line[i] = v
			}
			rr = append(rr, line)
		}
		put(monthLabel(m.Key), rr, []float64{36, 14, 30, 8, 44, 8, 12, 12, 16, 8, 12, 12, 12, 10, 14, 14})
	}
	last := outs[len(outs)-1]
	rows = [][]interface{}{{"الشهر", "النوع", "المقاول", "صف الشيت", "البند", "القيمة", "الملاحظة"}}
	for _, m := range outs {
		for _, wr := range m.Res.Warnings {
			rows = append(rows, []interface{}{monthLabel(m.Key), "مراجعة", wr.Contractor, wr.Row, wr.Desc, wr.Amount, wr.Why})
		}
	}
	for _, s := range last.Res.Skips {
		rows = append(rows, []interface{}{monthLabel(last.Key), "اتشال", s.Contractor, s.Row, s.Desc, s.Amount, s.Why})
	}
	put("ملاحظات", rows, []float64{12, 10, 40, 10, 50, 14, 50})
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(0)
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil
	}
	return buf.Bytes()
}

func monthlyFileName(m *monthOut) string {
	return fmt.Sprintf("PCS_Subcontractor_IPC_Cumulative_%s.xlsx", m.Date)
}

// الحفظ: بتختار مكان واسم فولدر، والبرنامج بيعمل الفولدر ويحط فيه ملف لكل شهر + ملف المراجعة المجمّع
func saveMonthly(w http.ResponseWriter) {
	outs := st.monthly
	first, last := outs[0], outs[len(outs)-1]
	folder := fmt.Sprintf("مستخلصات_شهري_%s_to_%s", first.Key, last.Key)
	p, err := saveDialog(folder)
	if err != nil {
		fail(w, err)
		return
	}
	if p == "" {
		writeJSON(w, 200, map[string]string{})
		return
	}
	dir := p
	if strings.EqualFold(filepath.Ext(p), ".xlsx") {
		dir = strings.TrimSuffix(p, filepath.Ext(p))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(w, fmt.Errorf("مقدرتش أعمل الفولدر: %v", err))
		return
	}
	revName := fmt.Sprintf("مراجعة_المستخلصات_شهري_%s_to_%s.xlsx", first.Key, last.Key)
	if err := os.WriteFile(filepath.Join(dir, revName), st.review, 0o644); err != nil {
		fail(w, fmt.Errorf("مقدرتش أحفظ الملفات — لو في ملف منهم مفتوح في Excel اقفله وجرّب تاني"))
		return
	}
	var saved []string
	dbNote := ""
	for _, m := range outs {
		fp := filepath.Join(dir, monthlyFileName(m))
		if err := os.WriteFile(fp, m.Out, 0o644); err != nil {
			fail(w, fmt.Errorf("مقدرتش أحفظ %s — لو مفتوح في Excel اقفله وجرّب تاني", monthlyFileName(m)))
			return
		}
		saved = append(saved, fp)
		mo := st.opts
		mo.Mode, mo.Date, mo.UpTo = "cumulative", m.Date, m.Key
		n1 := monthlyFileName(m)
		n2 := fmt.Sprintf("مراجعة_المستخلصات_Cumulative_%s.xlsx", m.Date)
		run := &Run{Kind: "ipc", Period: ipcPeriod(mo), Label: "مستخلصات — " + m.Res.Label, Source: st.name,
			Items: len(m.Res.Contracts), Amount: r2(m.Res.OutWorks - m.Res.OutDeduct),
			Extra: fmt.Sprintf("%d سطر · شهر بشهر", len(m.Res.Rows)), OutName: n1, RevName: n2, SavedPath: fp}
		if err := saveRun(run, m.Out, m.Review); err != nil && dbNote == "" {
			dbNote = "اتحفظت الملفات بس مااتسجلتش في السجل: " + err.Error()
		}
	}
	last.Res.rememberContractNos()
	setSetting("lastDir", filepath.Dir(dir))
	writeJSON(w, 200, map[string]interface{}{"path": filepath.Join(dir, revName), "folder": dir, "files": len(saved), "dbNote": dbNote})
}

//go:build !windows

package main

// نسخة سطر أوامر للاختبار على لينكس:
//   actualat serve [addr]                       — يشغّل الواجهة على المتصفح
//   actualat report.xlsm out_dir [mode] [month] [date]

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func saveDialog(name string) (string, error) {
	d := filepath.Join(os.TempDir(), "actualat-out")
	os.MkdirAll(d, 0o755)
	return filepath.Join(d, name), nil
}
func replaceSelfAndRestart(b []byte) error { return fmt.Errorf("متاح على ويندوز بس") }
func openFolder(p string) error            { return nil }

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		addr := "127.0.0.1:0"
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		u, err := startServerOn(addr)
		if err != nil {
			panic(err)
		}
		fmt.Println(u)
		select {}
	}
	if len(os.Args) < 3 {
		fmt.Println("usage: report.xlsm outdir [mode] [month] [date]")
		return
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	rep, _, err := ReadReport(filepath.Base(os.Args[1]), data, "")
	if err != nil {
		panic(err)
	}
	cl, src := loadCodeLists()
	groups := rep.Groups(cl)
	o := Opts{Mode: "cumulative", Date: "2026-10-30", Codes: map[string][2]string{}}
	if len(os.Args) > 3 {
		o.Mode = os.Args[3]
	}
	if len(os.Args) > 4 {
		o.Month = os.Args[4]
	}
	if len(os.Args) > 5 {
		o.Date = os.Args[5]
	}
	for _, g := range groups {
		o.Codes[g.Key] = [2]string{g.Task.Code, g.Cost.Code}
	}
	res := rep.Convert(cl, o)
	out, err := WritePCS(res)
	if err != nil {
		panic(err)
	}
	rv, _ := WriteReview(res, groups, o, cl)
	st.opts, st.name = o, filepath.Base(os.Args[1])
	n1, n2 := outName()
	os.WriteFile(filepath.Join(os.Args[2], n1), out, 0o644)
	os.WriteFile(filepath.Join(os.Args[2], n2), rv, 0o644)
	gj, _ := json.MarshalIndent(groups, "", " ")
	os.WriteFile(filepath.Join(os.Args[2], "groups.json"), gj, 0o644)
	fmt.Println("codes:", src, len(cl.Task), len(cl.Cost), "months:", rep.Months, "cutoff:", rep.Cutoff)
	fmt.Printf("contracts=%d rows=%d sheetNet=%.2f outWorks=%.2f deduct=%.2f skip=%.2f ret=%.2f/%.2f match=%v\n",
		len(res.Contracts), len(res.Rows), res.SheetNet, res.OutWorks, res.OutDeduct, res.SkipAmount, res.OutRet, res.SheetRet, res.Match)
	fmt.Println("warnings", len(res.Warnings), "skips", len(res.Skips))
}

package main

// HTTP API للواجهة (نفس فكرة برنامج المخازن: واجهة HTML جوه WebView2)

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed ui.html
var uiHTML []byte

//go:embed web_store.html
var storeHTML []byte

//go:embed web_ipc.html
var ipcHTML []byte

//go:embed web_history.html
var historyHTML []byte

//go:embed web_dashboard.html
var dashboardHTML []byte

//go:embed web_analytics.html
var analyticsHTML []byte

//go:embed web_settings.html
var settingsHTML []byte

//go:embed web_login.html
var loginHTML []byte

//go:embed assets/web
var webFS embed.FS

const appName = "Actualat"

func appDir() string {
	if d := os.Getenv("ACTUALAT_DIR"); d != "" {
		return d
	}
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, appName)
		}
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".actualat")
}

type state struct {
	mu     sync.Mutex
	data   []byte
	name   string
	rep    *Report
	sheets []string
	cl     *CodeLists
	clSrc  string
	groups []*CodeGroup
	res    *Result
	opts   Opts
	out    []byte
	review []byte
}

var st = &state{}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, err error) {
	writeJSON(w, 400, map[string]string{"error": err.Error()})
}

func safe(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				writeJSON(w, 500, map[string]string{"error": fmt.Sprint("خطأ غير متوقع: ", e)})
			}
		}()
		st.mu.Lock()
		defer st.mu.Unlock()
		h(w, r)
	}
}

func infoJSON() map[string]interface{} {
	rep := st.rep
	months := []map[string]interface{}{}
	for _, k := range rep.Months {
		var amt float64
		n := 0
		for _, c := range rep.Contracts {
			for _, it := range c.Items {
				if q := it.ByMonth[k]; q != 0 {
					rate := it.Rate
					if !it.HasRate && it.Qty != 0 {
						rate = it.Amount / it.Qty
					}
					amt += q * rate
					n++
				}
			}
		}
		if n > 0 {
			months = append(months, map[string]interface{}{"key": k, "label": rep.MonthLbl[k], "rows": n, "amount": r2(amt)})
		}
	}
	nItems, nCon := 0, 0
	for _, c := range rep.Contracts {
		if len(c.Items) > 0 {
			nCon++
		}
		nItems += len(c.Items)
	}
	return map[string]interface{}{"file": rep.File, "sheet": rep.Sheet, "sheets": st.sheets, "cutoff": rep.Cutoff,
		"cutoffLabel": monthLabel(rep.Cutoff), "months": months, "contracts": nCon, "items": nItems, "amount": r2(rep.HeaderAmt)}
}

func hLoad(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(200 << 20)
	file, hdr, err := r.FormFile("file")
	if err != nil {
		fail(w, err)
		return
	}
	defer file.Close()
	data, _ := io.ReadAll(file)
	st.data, st.name = data, hdr.Filename
	rep, sheets, err := ReadReport(hdr.Filename, data, "")
	st.sheets = sheets
	if err != nil {
		writeJSON(w, 400, map[string]interface{}{"error": err.Error(), "sheets": sheets})
		return
	}
	st.rep = rep
	writeJSON(w, 200, infoJSON())
}

func hSheet(w http.ResponseWriter, r *http.Request) {
	var b struct{ Sheet string }
	json.NewDecoder(r.Body).Decode(&b)
	rep, sheets, err := ReadReport(st.name, st.data, b.Sheet)
	st.sheets = sheets
	if err != nil {
		fail(w, err)
		return
	}
	st.rep = rep
	writeJSON(w, 200, infoJSON())
}

func hCodes(w http.ResponseWriter, r *http.Request) {
	if st.rep == nil {
		fail(w, fmt.Errorf("اختار الكوست ريبورت الأول"))
		return
	}
	st.cl, st.clSrc = loadCodeLists()
	st.groups = st.rep.Groups(st.cl)
	writeJSON(w, 200, map[string]interface{}{"groups": st.groups, "codes": st.cl, "codesFrom": st.clSrc})
}

func hConvert(w http.ResponseWriter, r *http.Request) {
	var o Opts
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		fail(w, err)
		return
	}
	if st.rep == nil || st.cl == nil {
		fail(w, fmt.Errorf("اختار الملف الأول"))
		return
	}
	if o.Mode == "current" && o.Month == "" {
		fail(w, fmt.Errorf("اختار الشهر"))
		return
	}
	if _, err := time.Parse("2006-01-02", o.Date); err != nil {
		fail(w, fmt.Errorf("تاريخ المستخلص مش مظبوط"))
		return
	}
	// احفظ اختيارات الأكواد في الذاكرة
	mem.mu.Lock()
	for _, g := range st.groups {
		v, ok := o.Codes[g.Key]
		if !ok {
			continue
		}
		if v[0] != "" {
			mem.Task[taskKey(g.TaskDesc, g.Trade)] = v[0]
		}
		if v[1] != "" {
			mem.Cost[g.Key] = v[1]
		}
	}
	mem.mu.Unlock()
	saveMem()
	res := st.rep.Convert(st.cl, o)
	out, err := WritePCS(res)
	if err != nil {
		fail(w, err)
		return
	}
	rv, _ := WriteReview(res, st.groups, o, st.cl)
	st.res, st.opts, st.out, st.review = res, o, out, rv
	noT, noC := 0, 0
	for _, row := range res.Rows {
		if row["Task Code"] == "" {
			noT++
		}
		if row["Cost Code"] == "" {
			noC++
		}
	}
	writeJSON(w, 200, map[string]interface{}{"result": res, "lines": len(res.Rows), "noTask": noT, "noCost": noC,
		"previous": samePeriod("ipc", ipcPeriod(o))})
}

func outName() (string, string) {
	tag := st.opts.Date
	kind := "Cumulative"
	if st.opts.Mode == "current" {
		kind = "Current_" + monthLabel(st.opts.Month)
	}
	base := strings.TrimSuffix(st.name, filepath.Ext(st.name))
	if len([]rune(base)) > 30 {
		base = string([]rune(base)[:30])
	}
	return fmt.Sprintf("PCS_Subcontractor_IPC_%s_%s.xlsx", kind, tag), fmt.Sprintf("مراجعة_المستخلصات_%s_%s.xlsx", kind, tag)
}

func ipcPeriod(o Opts) string {
	if o.Mode == "current" {
		return "current:" + o.Month
	}
	return "cumulative:" + o.Date
}

func hSave(w http.ResponseWriter, r *http.Request) {
	if st.out == nil {
		fail(w, fmt.Errorf("دوس تحويل الأول"))
		return
	}
	n1, n2 := outName()
	run := &Run{Kind: "ipc", Period: ipcPeriod(st.opts), Label: "مستخلصات — " + st.res.Label, Source: st.name,
		Items: len(st.res.Contracts), Amount: r2(st.res.OutWorks - st.res.OutDeduct),
		Extra: fmt.Sprintf("%d سطر", len(st.res.Rows)), OutName: n1, RevName: n2}
	if st.res != nil {
		run.onSaved = st.res.rememberContractNos
	}
	saveAndRecord(w, run, st.out, st.review)
}

func hCodeFile(w http.ResponseWriter, r *http.Request) {
	// تحديث قوائم الأكواد: Task_Codes / Cost_Codes
	r.ParseMultipartForm(50 << 20)
	saved := []string{}
	for _, k := range []string{"task", "cost"} {
		file, _, err := r.FormFile(k)
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(file)
		file.Close()
		if _, err := readCodeSheet(data); err != nil {
			fail(w, err)
			return
		}
		name := map[string]string{"task": "Task_Codes.xlsx", "cost": "Cost_Codes.xlsx"}[k]
		os.MkdirAll(appDir(), 0o755)
		os.WriteFile(filepath.Join(appDir(), name), data, 0o644)
		saved = append(saved, name)
	}
	anaReset()
	writeJSON(w, 200, map[string]interface{}{"saved": saved})
}

func startServer() (string, error) {
	return startServerOn("127.0.0.1:0")
}

func startServerOn(addr string) (string, error) {
	loadMem()
	mux := http.NewServeMux()
	page := func(b []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(b)
		}
	}
	mux.HandleFunc("/", page(uiHTML))
	mux.HandleFunc("/store.html", page(storeHTML))
	mux.HandleFunc("/ipc.html", page(ipcHTML))
	mux.HandleFunc("/history.html", page(historyHTML))
	mux.HandleFunc("/dashboard.html", page(dashboardHTML))
	mux.HandleFunc("/analytics.html", page(analyticsHTML))
	mux.HandleFunc("/settings.html", page(settingsHTML))
	mux.HandleFunc("/login.html", page(loginHTML))
	registerAuth(mux)
	registerStore(mux)
	registerHistory(mux)
	sub, _ := fs.Sub(webFS, "assets/web")
	files := http.StripPrefix("/assets/", http.FileServer(http.FS(sub)))
	mux.Handle("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") || strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "max-age=31536000")
		}
		files.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"version": appVersion})
	})
	mux.HandleFunc("/api/update/check", func(w http.ResponseWriter, r *http.Request) {
		u, err := checkUpdate()
		if err != nil {
			writeJSON(w, 200, map[string]interface{}{"available": false, "version": appVersion, "current": appVersion, "error": err.Error()})
			return
		}
		writeJSON(w, 200, u)
	})
	mux.HandleFunc("/api/update/apply", func(w http.ResponseWriter, r *http.Request) {
		if err := applyUpdate(); err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, 200, map[string]string{"ok": "1"})
	})
	mux.HandleFunc("/api/ipc/load", safe(hLoad))
	mux.HandleFunc("/api/ipc/sheet", safe(hSheet))
	mux.HandleFunc("/api/ipc/codes", safe(hCodes))
	mux.HandleFunc("/api/ipc/convert", safe(hConvert))
	mux.HandleFunc("/api/ipc/save", safe(hSave))
	mux.HandleFunc("/api/ipc/codefiles", safe(hCodeFile))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}
	go http.Serve(ln, requireAuth(mux))
	return "http://" + ln.Addr().String() + "/", nil
}

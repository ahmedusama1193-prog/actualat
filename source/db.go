package main

// قاعدة بيانات على الجهاز: كل ملف بيتصدّر بيتسجل (البيانات + ملف PCS + ملف المراجعة)
// الملف: %LOCALAPPDATA%\Actualat\actualat.db

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	db     *bolt.DB
	dbOnce sync.Once
	dbErr  error
)

var (
	bRuns     = []byte("runs")
	bFiles    = []byte("files")
	bSettings = []byte("settings")
)

type Run struct {
	ID        uint64  `json:"id"`
	Kind      string  `json:"kind"`   // store | ipc
	Period    string  `json:"period"` // 2026-10 / cumulative:2026-10-30 ...
	Label     string  `json:"label"`
	Source    string  `json:"source"`
	Items     int     `json:"items"`
	Amount    float64 `json:"amount"`
	Extra     string  `json:"extra"`
	OutName   string  `json:"outName"`
	RevName   string  `json:"revName"`
	SavedPath string  `json:"savedPath"`
	Created   string  `json:"created"`
	Exists    bool    `json:"exists"` // الملف لسه في مكانه (بيتحسب وقت العرض)
	onSaved   func()  // بيتنفذ بعد ما الملف يتحفظ
}

func dbPath() string { return filepath.Join(appDir(), "actualat.db") }

func openDB() error {
	dbOnce.Do(func() {
		os.MkdirAll(appDir(), 0o755)
		db, dbErr = bolt.Open(dbPath(), 0o600, &bolt.Options{Timeout: 3 * time.Second})
		if dbErr != nil {
			return
		}
		dbErr = db.Update(func(tx *bolt.Tx) error {
			for _, b := range [][]byte{bRuns, bFiles, bSettings} {
				if _, err := tx.CreateBucketIfNotExists(b); err != nil {
					return err
				}
			}
			return nil
		})
	})
	return dbErr
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func saveRun(r *Run, out, rev []byte) error {
	if err := openDB(); err != nil {
		return err
	}
	return db.Update(func(tx *bolt.Tx) error {
		runs := tx.Bucket(bRuns)
		id, _ := runs.NextSequence()
		r.ID = id
		r.Created = time.Now().Format("2006-01-02 15:04")
		r.Exists = false
		js, _ := json.Marshal(r)
		if err := runs.Put(itob(id), js); err != nil {
			return err
		}
		f := tx.Bucket(bFiles)
		f.Put(append(itob(id), 'o'), out)
		return f.Put(append(itob(id), 'r'), rev)
	})
}

func listRuns() ([]Run, error) {
	if err := openDB(); err != nil {
		return nil, err
	}
	var out []Run
	err := db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bRuns).ForEach(func(k, v []byte) error {
			var r Run
			if json.Unmarshal(v, &r) == nil {
				out = append(out, r)
			}
			return nil
		})
	})
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, err
}

func getRun(id uint64) (*Run, []byte, []byte, error) {
	if err := openDB(); err != nil {
		return nil, nil, nil, err
	}
	var r Run
	var o, rv []byte
	err := db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bRuns).Get(itob(id))
		if v == nil {
			return fmt.Errorf("السجل ده مش موجود")
		}
		json.Unmarshal(v, &r)
		f := tx.Bucket(bFiles)
		o = append([]byte(nil), f.Get(append(itob(id), 'o'))...)
		rv = append([]byte(nil), f.Get(append(itob(id), 'r'))...)
		return nil
	})
	return &r, o, rv, err
}

func deleteRun(id uint64) error {
	if err := openDB(); err != nil {
		return err
	}
	anaForget(id)
	return db.Update(func(tx *bolt.Tx) error {
		tx.Bucket(bRuns).Delete(itob(id))
		f := tx.Bucket(bFiles)
		f.Delete(append(itob(id), 'o'))
		return f.Delete(append(itob(id), 'r'))
	})
}

// السجلات السابقة لنفس الفترة (عشان التنبيه)
func samePeriod(kind, period string) []Run {
	runs, _ := listRuns()
	var out []Run
	for _, r := range runs {
		if r.Kind == kind && r.Period == period {
			out = append(out, r)
		}
	}
	return out
}

func getSetting(k string) string {
	if openDB() != nil {
		return ""
	}
	var v string
	db.View(func(tx *bolt.Tx) error {
		v = string(tx.Bucket(bSettings).Get([]byte(k)))
		return nil
	})
	return v
}

func setSetting(k, v string) {
	if openDB() != nil {
		return
	}
	db.Update(func(tx *bolt.Tx) error { return tx.Bucket(bSettings).Put([]byte(k), []byte(v)) })
}

// حفظ ملف PCS + المراجعة في المكان اللي يختاره المستخدم، وتسجيله في الداتابيز
func saveAndRecord(w http.ResponseWriter, run *Run, out, rev []byte) {
	p, err := saveDialog(run.OutName)
	if err != nil {
		fail(w, err)
		return
	}
	if p == "" {
		writeJSON(w, 200, map[string]string{})
		return
	}
	if err := os.WriteFile(p, out, 0o644); err != nil {
		fail(w, fmt.Errorf("مقدرتش أحفظ الملف — لو مفتوح في Excel اقفله وجرّب تاني"))
		return
	}
	rp := ""
	if len(rev) > 0 {
		rp = filepath.Join(filepath.Dir(p), run.RevName)
		os.WriteFile(rp, rev, 0o644)
	}
	setSetting("lastDir", filepath.Dir(p))
	run.SavedPath = p
	if run.onSaved != nil {
		run.onSaved()
	}
	dbNote := ""
	if err := saveRun(run, out, rev); err != nil {
		dbNote = "اتحفظ الملف بس مااتسجلش في السجل: " + err.Error()
	}
	writeJSON(w, 200, map[string]string{"path": p, "review": rp, "dbNote": dbNote})
}

// ------------------------------------------------------------------ API: السجل
func hHistList(w http.ResponseWriter, r *http.Request) {
	runs, err := listRuns()
	if err != nil {
		fail(w, err)
		return
	}
	if runs == nil {
		runs = []Run{}
	}
	for i := range runs {
		if runs[i].SavedPath != "" {
			_, err := os.Stat(runs[i].SavedPath)
			runs[i].Exists = err == nil
		}
	}
	writeJSON(w, 200, map[string]interface{}{"runs": runs, "path": dbPath()})
}

func hHistSave(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ID    uint64 `json:"id"`
		Which string `json:"which"`
	}
	json.NewDecoder(r.Body).Decode(&b)
	run, o, rv, err := getRun(b.ID)
	if err != nil {
		fail(w, err)
		return
	}
	data, name := o, run.OutName
	if b.Which == "review" {
		data, name = rv, run.RevName
	}
	if len(data) == 0 {
		fail(w, fmt.Errorf("الملف ده مش متسجل"))
		return
	}
	p, err := saveDialog(name)
	if err != nil {
		fail(w, err)
		return
	}
	if p == "" {
		writeJSON(w, 200, map[string]string{})
		return
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		fail(w, fmt.Errorf("مقدرتش أحفظ الملف — لو مفتوح في Excel اقفله وجرّب تاني"))
		return
	}
	setSetting("lastDir", filepath.Dir(p))
	writeJSON(w, 200, map[string]string{"path": p})
}

func hHistDelete(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ID uint64 `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&b)
	if err := deleteRun(b.ID); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "1"})
}

func hOpenFolder(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Path string `json:"path"`
	}
	json.NewDecoder(r.Body).Decode(&b)
	if b.Path == "" {
		b.Path = dbPath()
	}
	if err := openFolder(b.Path); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "1"})
}

func registerHistory(mux *http.ServeMux) {
	openDB()
	mux.HandleFunc("/api/history/list", hHistList)
	mux.HandleFunc("/api/history/save", hHistSave)
	mux.HandleFunc("/api/history/delete", hHistDelete)
	mux.HandleFunc("/api/open-folder", hOpenFolder)
	mux.HandleFunc("/api/settings", hSettings)
	mux.HandleFunc("/api/analytics", hAnalytics)
}

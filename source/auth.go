package main

// الحسابات (1.7): كل جهاز ليه حساباته. أول مرة البرنامج يفتح بيطلب إنشاء حساب،
// وبعد كده لازم تسجيل دخول. الباسورد مابيتخزنش — بيتخزن PBKDF2-SHA256 بملح عشوائي.
// الجلسة في الذاكرة بس، فكل مرة البرنامج يفتح لازم دخول من جديد.

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	bolt "go.etcd.io/bbolt"
)

var bUsers = []byte("users")

const pwIter = 210000

type account struct {
	Name    string `json:"name"`
	Salt    string `json:"salt"`
	Hash    string `json:"hash"`
	Created string `json:"created"`
}

var (
	sessMu   sync.Mutex
	sessions = map[string]string{} // token -> username key
)

const sessCookie = "act_s"

func userKey(n string) string { return strings.ToLower(strings.TrimSpace(n)) }

func hashPw(pw string, salt []byte) []byte {
	k, _ := pbkdf2.Key(sha256.New, pw, salt, pwIter, 32)
	return k
}

func ensureUsersBucket() error {
	if err := openDB(); err != nil {
		return err
	}
	return db.Update(func(tx *bolt.Tx) error { _, err := tx.CreateBucketIfNotExists(bUsers); return err })
}

func getAccount(key string) *account {
	var a *account
	db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bUsers); b != nil {
			if v := b.Get([]byte(key)); v != nil {
				a = &account{}
				json.Unmarshal(v, a)
			}
		}
		return nil
	})
	return a
}

func putAccount(a *account) error {
	v, _ := json.Marshal(a)
	return db.Update(func(tx *bolt.Tx) error { return tx.Bucket(bUsers).Put([]byte(userKey(a.Name)), v) })
}

func listAccounts() []account {
	var out []account
	db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bUsers); b != nil {
			b.ForEach(func(k, v []byte) error {
				var a account
				if json.Unmarshal(v, &a) == nil {
					out = append(out, a)
				}
				return nil
			})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Created < out[j].Created })
	return out
}

func newAccount(name, pw string) (*account, string) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 3 || n > 32 || strings.ContainsAny(name, " \t\"'<>&/\\") {
		return nil, "اسم المستخدم لازم يكون من 3 لـ 32 حرف، من غير مسافات"
	}
	if utf8.RuneCountInString(pw) < 6 {
		return nil, "الباسورد لازم يكون 6 حروف أو أكتر"
	}
	if getAccount(userKey(name)) != nil {
		return nil, "اسم المستخدم ده موجود قبل كده"
	}
	salt := make([]byte, 16)
	rand.Read(salt)
	return &account{Name: name, Salt: hex.EncodeToString(salt), Hash: hex.EncodeToString(hashPw(pw, salt)), Created: time.Now().Format("2006-01-02 15:04")}, ""
}

func checkPw(a *account, pw string) bool {
	salt, _ := hex.DecodeString(a.Salt)
	want, _ := hex.DecodeString(a.Hash)
	return subtle.ConstantTimeCompare(hashPw(pw, salt), want) == 1
}

func startSession(w http.ResponseWriter, key string) {
	t := make([]byte, 24)
	rand.Read(t)
	tok := hex.EncodeToString(t)
	sessMu.Lock()
	sessions[tok] = key
	sessMu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: sessCookie, Value: tok, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
}

// المستخدم الحالي للطلب ("" لو مش داخل)
func sessionUser(r *http.Request) string {
	c, err := r.Cookie(sessCookie)
	if err != nil {
		return ""
	}
	sessMu.Lock()
	k := sessions[c.Value]
	sessMu.Unlock()
	if k == "" {
		return ""
	}
	if a := getAccount(k); a != nil {
		return a.Name
	}
	return ""
}

func readCreds(r *http.Request) (u, p, np string) {
	var b struct{ User, Pass, NewPass string }
	json.NewDecoder(r.Body).Decode(&b)
	return b.User, b.Pass, b.NewPass
}

func registerAuth(mux *http.ServeMux) {
	ensureUsersBucket()
	mux.HandleFunc("/api/auth/state", func(w http.ResponseWriter, r *http.Request) {
		if l := r.URL.Query().Get("lang"); l == "ar" || l == "en" {
			setSetting("lang", l)
		}
		writeJSON(w, 200, map[string]interface{}{"hasUsers": len(listAccounts()) > 0, "user": sessionUser(r), "lang": uiLang(), "theme": uiTheme(), "version": appVersion})
	})
	// أول حساب على الجهاز — مسموح بس لو مفيش ولا حساب
	mux.HandleFunc("/api/auth/setup", func(w http.ResponseWriter, r *http.Request) {
		if len(listAccounts()) > 0 {
			writeJSON(w, 403, map[string]string{"error": "في حساب على الجهاز ده بالفعل — سجّل دخول"})
			return
		}
		u, p, _ := readCreds(r)
		a, msg := newAccount(u, p)
		if a == nil {
			writeJSON(w, 400, map[string]string{"error": msg})
			return
		}
		putAccount(a)
		startSession(w, userKey(a.Name))
		writeJSON(w, 200, map[string]string{"user": a.Name})
	})
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		u, p, _ := readCreds(r)
		a := getAccount(userKey(u))
		if a == nil || !checkPw(a, p) {
			time.Sleep(700 * time.Millisecond)
			writeJSON(w, 401, map[string]string{"error": "اسم المستخدم أو الباسورد غلط"})
			return
		}
		startSession(w, userKey(a.Name))
		writeJSON(w, 200, map[string]string{"user": a.Name})
	})
	mux.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessCookie); err == nil {
			sessMu.Lock()
			delete(sessions, c.Value)
			sessMu.Unlock()
		}
		http.SetCookie(w, &http.Cookie{Name: sessCookie, Value: "", Path: "/", MaxAge: -1})
		writeJSON(w, 200, map[string]string{"ok": "1"})
	})
	// باقي العمليات محتاجة دخول (الـ middleware بيتأكد)
	mux.HandleFunc("/api/auth/users", func(w http.ResponseWriter, r *http.Request) {
		me := sessionUser(r)
		var out []map[string]interface{}
		for _, a := range listAccounts() {
			out = append(out, map[string]interface{}{"name": a.Name, "created": a.Created, "me": a.Name == me})
		}
		writeJSON(w, 200, map[string]interface{}{"users": out})
	})
	mux.HandleFunc("/api/auth/add", func(w http.ResponseWriter, r *http.Request) {
		u, p, _ := readCreds(r)
		a, msg := newAccount(u, p)
		if a == nil {
			writeJSON(w, 400, map[string]string{"error": msg})
			return
		}
		putAccount(a)
		writeJSON(w, 200, map[string]string{"user": a.Name})
	})
	mux.HandleFunc("/api/auth/passwd", func(w http.ResponseWriter, r *http.Request) {
		_, p, np := readCreds(r)
		a := getAccount(userKey(sessionUser(r)))
		if a == nil || !checkPw(a, p) {
			time.Sleep(700 * time.Millisecond)
			writeJSON(w, 400, map[string]string{"error": "الباسورد الحالي غلط"})
			return
		}
		if utf8.RuneCountInString(np) < 6 {
			writeJSON(w, 400, map[string]string{"error": "الباسورد لازم يكون 6 حروف أو أكتر"})
			return
		}
		salt := make([]byte, 16)
		rand.Read(salt)
		a.Salt, a.Hash = hex.EncodeToString(salt), hex.EncodeToString(hashPw(np, salt))
		putAccount(a)
		writeJSON(w, 200, map[string]string{"ok": "1"})
	})
	mux.HandleFunc("/api/auth/remove", func(w http.ResponseWriter, r *http.Request) {
		u, _, _ := readCreds(r)
		if userKey(u) == userKey(sessionUser(r)) {
			writeJSON(w, 400, map[string]string{"error": "مينفعش تمسح الحساب اللي داخل بيه"})
			return
		}
		db.Update(func(tx *bolt.Tx) error { return tx.Bucket(bUsers).Delete([]byte(userKey(u))) })
		sessMu.Lock()
		for t, k := range sessions {
			if k == userKey(u) {
				delete(sessions, t)
			}
		}
		sessMu.Unlock()
		writeJSON(w, 200, map[string]string{"ok": "1"})
	})
}

// requireAuth: أي صفحة أو API غير صفحة الدخول والـ assets محتاجة جلسة
func requireAuth(next http.Handler) http.Handler {
	open := map[string]bool{"/login.html": true, "/api/auth/state": true, "/api/auth/setup": true, "/api/auth/login": true, "/api/auth/logout": true, "/api/version": true}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if open[p] || strings.HasPrefix(p, "/assets/") || sessionUser(r) != "" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(p, "/api/") {
			writeJSON(w, 401, map[string]string{"error": "لازم تسجّل دخول"})
			return
		}
		if p == "/" {
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}
		// صفحة جوه الإطار: نرجّع الشباك كله لصفحة الدخول
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<script>top.location.href='/login.html'</script>`))
	})
}

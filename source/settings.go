package main

// إعدادات الواجهة (الثيم، العملة، اللغة) + بيانات المستخدم للشريط الجانبي

import (
	"encoding/json"
	"net/http"
	"os"
	"os/user"
	"strings"
)

// titleHook: بيغيّر لون شريط عنوان الويندوز (داكن/فاتح) — بيتظبط في main_windows.go
var titleHook func(dark bool)

func userName() string {
	if n := os.Getenv("USERNAME"); n != "" {
		return n
	}
	if u, err := user.Current(); err == nil {
		n := u.Username
		if i := strings.LastIndexAny(n, `\/`); i >= 0 {
			n = n[i+1:]
		}
		return n
	}
	return ""
}

func uiTheme() string {
	if t := getSetting("theme"); t == "light" {
		return t
	}
	return "dark"
}

func uiLang() string {
	if l := getSetting("lang"); l == "en" {
		return l
	}
	return "ar"
}

func uiCurrency() string {
	c := getSetting("currency")
	if c == "" {
		return "ج.م"
	}
	if c == "-" {
		return ""
	}
	return c
}

func settingsJSON() map[string]interface{} {
	return map[string]interface{}{
		"theme": uiTheme(), "lang": uiLang(), "currency": uiCurrency(), "user": userName(),
		"dataDir": appDir(), "dbPath": dbPath(), "lastDir": getSetting("lastDir"), "version": appVersion, "welcome": getSetting("welcome"),
	}
}

func hSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var b struct {
			Theme    *string `json:"theme"`
			Lang     *string `json:"lang"`
			Currency *string `json:"currency"`
			Welcome  *string `json:"welcome"`
		}
		json.NewDecoder(r.Body).Decode(&b)
		if b.Theme != nil && (*b.Theme == "dark" || *b.Theme == "light") {
			setSetting("theme", *b.Theme)
			if titleHook != nil {
				titleHook(*b.Theme == "dark")
			}
		}
		if b.Welcome != nil && (*b.Welcome == "on" || *b.Welcome == "off") {
			setSetting("welcome", *b.Welcome)
		}
		if b.Lang != nil && (*b.Lang == "ar" || *b.Lang == "en") {
			setSetting("lang", *b.Lang)
		}
		if b.Currency != nil {
			c := strings.TrimSpace(*b.Currency)
			if c == "" {
				c = "-"
			}
			if len([]rune(c)) <= 8 {
				setSetting("currency", c)
			}
		}
	}
	m := settingsJSON()
	writeJSON(w, 200, m)
}

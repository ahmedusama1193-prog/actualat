package main

// التحديث التلقائي: البرنامج بيقرا latest.json من مكان ثابت، ولو الإصدار أحدث بينزّله ويستبدل نفسه.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const appVersion = "2.5"

// مكان ملف التحديثات — بيتظبط لما نربط GitHub (فاضي = التحديث مقفول)
var updateManifestURL = "https://raw.githubusercontent.com/ahmedusama1193-prog/actualat/main/latest.json"

// المستودع بقى Private: البرنامج بيقرا التحديثات من GitHub API بمفتاح قراءة بس (Contents: Read-only على actualat بس).
// المفتاح متخزن متلخبط شوية عشان مايبانش كنص صريح جوه الـ exe.
const updRepo = "ahmedusama1193-prog/actualat"

func updToken() string {
	b := make([]byte, len(updKeyX))
	for i, c := range updKeyX {
		b[i] = c ^ byte(0x5a+i%7)
	}
	return string(b)
}

// GET من GitHub API (contents) بالمفتاح؛ لو فشل بيجرب الرابط العام (للنسخ القديمة وقت ما المستودع كان Public)
func ghGet(c *http.Client, path, ref, fallback string) (*http.Response, error) {
	if len(updKeyX) == 0 && fallback != "" {
		return c.Get(fallback) // من غير مفتاح: الرابط العام على طول (الريبو Public)
	}
	u := "https://api.github.com/repos/" + updRepo + "/contents/" + path + "?ref=" + ref
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+updToken())
	req.Header.Set("Accept", "application/vnd.github.raw")
	req.Header.Set("User-Agent", "Actualat/"+appVersion)
	resp, err := c.Do(req)
	if err == nil && resp.StatusCode == 200 {
		return resp, nil
	}
	if resp != nil {
		resp.Body.Close()
	}
	if fallback == "" {
		if err == nil {
			err = fmt.Errorf("مكان التحديث رد بـ %d", resp.StatusCode)
		}
		return nil, err
	}
	return c.Get(fallback)
}

// raw.githubusercontent.com/<owner>/<repo>/<ref>/<path> -> ref, path
func rawParts(u string) (string, string) {
	const pre = "https://raw.githubusercontent.com/" + updRepo + "/"
	if !strings.HasPrefix(u, pre) {
		return "", ""
	}
	rest := strings.TrimPrefix(u, pre)
	i := strings.Index(rest, "/")
	if i < 0 {
		return "", ""
	}
	return rest[:i], rest[i+1:]
}

type UpdateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Current   string `json:"current"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Notes     string `json:"notes"`
	Enabled   bool   `json:"enabled"`
}

var (
	updMu   sync.Mutex
	updLast *UpdateInfo
)

func verParts(v string) []int {
	var out []int
	for _, p := range strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".") {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

func newer(a, b string) bool { // a > b
	pa, pb := verParts(a), verParts(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		x, y := 0, 0
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

var httpc = &http.Client{Timeout: 10 * time.Minute}

func checkUpdate() (*UpdateInfo, error) {
	info := &UpdateInfo{Current: appVersion, Version: appVersion, Enabled: updateManifestURL != ""}
	if updateManifestURL == "" {
		return info, nil
	}
	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := ghGet(c, "latest.json", "main", updateManifestURL+"?t="+strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		return info, fmt.Errorf("مفيش نت أو مكان التحديث مش متاح")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return info, fmt.Errorf("مكان التحديث رد بـ %d", resp.StatusCode)
	}
	var m struct {
		Version string `json:"version"`
		URL     string `json:"url"`
		SHA256  string `json:"sha256"`
		Notes   string `json:"notes"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return info, err
	}
	info.Version, info.URL, info.SHA256, info.Notes = m.Version, m.URL, strings.ToLower(m.SHA256), m.Notes
	info.Available = m.URL != "" && newer(m.Version, appVersion)
	updMu.Lock()
	updLast = info
	updMu.Unlock()
	return info, nil
}

func downloadUpdate(u *UpdateInfo) ([]byte, error) {
	var resp *http.Response
	var err error
	if ref, path := rawParts(u.URL); ref != "" {
		resp, err = ghGet(httpc, path, ref, u.URL)
	} else {
		resp, err = httpc.Get(u.URL)
	}
	if err != nil {
		return nil, fmt.Errorf("التحميل وقف: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("التحميل رد بـ %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 200<<20))
	if err != nil {
		return nil, err
	}
	if len(b) < 1<<20 || string(b[:2]) != "MZ" {
		return nil, fmt.Errorf("الملف اللي نزل مش برنامج سليم")
	}
	if u.SHA256 != "" {
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != u.SHA256 {
			return nil, fmt.Errorf("الملف اللي نزل مش مطابق — جرّب تاني")
		}
	}
	return b, nil
}

func applyUpdate() error {
	updMu.Lock()
	u := updLast
	updMu.Unlock()
	if u == nil || !u.Available {
		var err error
		if u, err = checkUpdate(); err != nil {
			return err
		}
		if !u.Available {
			return fmt.Errorf("انت على آخر إصدار")
		}
	}
	b, err := downloadUpdate(u)
	if err != nil {
		return err
	}
	return replaceSelfAndRestart(b)
}

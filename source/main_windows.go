//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
)

const shortcutName = "اكشوالات"

func installDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", appName)
}

func installedExe() string { return filepath.Join(installDir(), appName+".exe") }

func hiddenCmd(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return c
}

func psq(s string) string { return strings.ReplaceAll(s, "'", "''") }

//go:embed app.ico
var appIco []byte

// ملف الأيقونة باسم فيه رقم الإصدار: مسار جديد مع كل إصدار = ويندوز مايقدرش يعرض نسخة قديمة محفوظة
func iconFile() string {
	return filepath.Join(installDir(), "actualat-"+strings.ReplaceAll(appVersion, ".", "_")+".ico")
}

func writeIconFile() string {
	p := iconFile()
	if _, err := os.Stat(p); err != nil {
		os.MkdirAll(installDir(), 0o755)
		os.WriteFile(p, appIco, 0o644)
		olds, _ := filepath.Glob(filepath.Join(installDir(), "actualat-*.ico"))
		for _, o := range olds {
			if !strings.EqualFold(o, p) {
				os.Remove(o)
			}
		}
	}
	return p
}

// الاختصارات (الديسكتوب + Start) + أي اختصار متثبت في شريط المهام بيشاور على البرنامج
func makeShortcuts(target string) {
	ico := writeIconFile()
	ps := "$s=New-Object -ComObject WScript.Shell;foreach($p in @([Environment]::GetFolderPath('Desktop'),[Environment]::GetFolderPath('Programs'))){" +
		"$l=$s.CreateShortcut((Join-Path $p '" + psq(shortcutName) + ".lnk'));$l.TargetPath='" + psq(target) + "';" +
		"$l.WorkingDirectory='" + psq(filepath.Dir(target)) + "';$l.IconLocation='" + psq(ico) + ",0';$l.Description='Actualat';$l.Save()};" +
		"$t=Join-Path $env:APPDATA 'Microsoft\\Internet Explorer\\Quick Launch\\User Pinned\\TaskBar';" +
		"if(Test-Path $t){Get-ChildItem $t -Filter *.lnk|ForEach-Object{$k=$s.CreateShortcut($_.FullName);" +
		"if($k.TargetPath -ieq '" + psq(target) + "'){$k.IconLocation='" + psq(ico) + ",0';$k.Save()}}}"
	hiddenCmd("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps).Run()
	refreshIconCache()
}

const uninstKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Actualat`

// بيسجل البرنامج في "التطبيقات" بتاعة ويندوز عشان يبان وله زرار إلغاء تثبيت
func registerUninstall(target string) {
	add := func(name, typ, val string) {
		hiddenCmd("reg", "add", uninstKey, "/v", name, "/t", typ, "/d", val, "/f").Run()
	}
	add("DisplayName", "REG_SZ", "اكشوالات · Actualat")
	add("DisplayVersion", "REG_SZ", appVersion)
	add("Publisher", "REG_SZ", "Ahmed Abdulqader")
	add("DisplayIcon", "REG_SZ", target+",0")
	add("InstallLocation", "REG_SZ", installDir())
	add("UninstallString", "REG_SZ", `"`+target+`" uninstall`)
	add("NoModify", "REG_DWORD", "1")
	add("NoRepair", "REG_DWORD", "1")
	if fi, err := os.Stat(target); err == nil {
		add("EstimatedSize", "REG_DWORD", fmt.Sprint(fi.Size()/1024))
	}
}

func askYesNo(title, text string) bool {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(text)
	r, _, _ := pMessageBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x24|0x100) // YESNO + سؤال + الافتراضي "لا"
	return r == 6
}

// إلغاء التثبيت: الاختصارات + التسجيل + فولدر البرنامج. البيانات (السجل والحسابات) بتفضل.
func uninstall() {
	if !askYesNo("اكشوالات", "تشيل اكشوالات من الجهاز ده؟\n\nالسجل والحسابات هيفضلوا محفوظين في:\n"+appDir()+"\nولو نزّلته تاني هترجع زي ما هي.") {
		return
	}
	for _, d := range []string{filepath.Join(os.Getenv("USERPROFILE"), "Desktop"), filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs")} {
		os.Remove(filepath.Join(d, shortcutName+".lnk"))
	}
	hiddenCmd("powershell", "-NoProfile", "-Command", "Remove-Item -LiteralPath ([Environment]::GetFolderPath('Desktop')+'\\"+psq(shortcutName)+".lnk') -ErrorAction SilentlyContinue").Run()
	hiddenCmd("reg", "delete", uninstKey, "/f").Run()
	msgBox("اكشوالات", "اكشوالات اتشال من الجهاز.")
	// الفولدر بيتمسح بعد ما البرنامج يقفل
	c := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe"))
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000,
		CmdLine: `cmd.exe /c ping 127.0.0.1 -n 3 >nul & rmdir /s /q "` + installDir() + `"`}
	c.Start()
}

// أول تشغيل: البرنامج بينقل نفسه لـ LocalAppData\Programs\Actualat ويعمل اختصار على الديسكتوب
func maybeInstall() bool {
	self, err := os.Executable()
	if err != nil {
		return false
	}
	for _, a := range os.Args[1:] {
		if a == "no-install" || a == "--no-install" {
			return false
		}
		if a == "uninstall" {
			uninstall()
			return true
		}
	}
	target := installedExe()
	if strings.EqualFold(filepath.Clean(self), filepath.Clean(target)) {
		os.Remove(filepath.Join(installDir(), appName+".old.exe"))
		go registerUninstall(target)
		_, e1 := os.Stat(filepath.Join(os.Getenv("USERPROFILE"), "Desktop", shortcutName+".lnk"))
		_, e2 := os.Stat(iconFile())
		if e1 != nil || e2 != nil {
			go makeShortcuts(target)
		}
		// بعد التحديث: نعيد عمل الاختصار عشان الأيقونة الجديدة تظهر
		for _, a := range os.Args[1:] {
			if a == "updated" {
				go makeShortcuts(target)
			}
		}
		return false
	}
	b, err := os.ReadFile(self)
	if err != nil {
		return false
	}
	os.MkdirAll(installDir(), 0o755)
	if err := os.WriteFile(target, b, 0o755); err != nil {
		// غالباً البرنامج المتثبت شغال دلوقتي — نشتغل من مكاننا
		return false
	}
	makeShortcuts(target)
	registerUninstall(target)
	exec.Command(target).Start()
	return true
}

// استبدال البرنامج بالإصدار الجديد وإعادة التشغيل
func replaceSelfAndRestart(b []byte) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(self)
	newp := filepath.Join(dir, appName+".new.exe")
	oldp := filepath.Join(dir, appName+".old.exe")
	if err := os.WriteFile(newp, b, 0o755); err != nil {
		return fmt.Errorf("مش قادر أحفظ التحديث: %v", err)
	}
	os.Remove(oldp)
	if err := os.Rename(self, oldp); err != nil {
		return fmt.Errorf("مش قادر أستبدل البرنامج: %v", err)
	}
	if err := os.Rename(newp, self); err != nil {
		os.Rename(oldp, self)
		return fmt.Errorf("مش قادر أستبدل البرنامج: %v", err)
	}
	if err := exec.Command(self, "updated").Start(); err != nil {
		return err
	}
	go func() { time.Sleep(600 * time.Millisecond); os.Exit(0) }()
	return nil
}

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	comdlg32    = syscall.NewLazyDLL("comdlg32.dll")
	ole32       = syscall.NewLazyDLL("ole32.dll")
	dwmapi      = syscall.NewLazyDLL("dwmapi.dll")
	pMessageBox = user32.NewProc("MessageBoxW")
	pGetSave    = comdlg32.NewProc("GetSaveFileNameW")
	pCoInit     = ole32.NewProc("CoInitializeEx")
	pDwmSetAttr = dwmapi.NewProc("DwmSetWindowAttribute")
	shell32     = syscall.NewLazyDLL("shell32.dll")
	pSetAUMID   = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
	pLoadImage  = user32.NewProc("LoadImageW")
	pSendMsg    = user32.NewProc("SendMessageW")
	pGetModule  = syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW")
	pSysMetrics = user32.NewProc("GetSystemMetrics")
	pRedraw     = user32.NewProc("SetWindowPos")
)

// هوية خاصة بالبرنامج في شريط المهام، عشان ويندوز مايجمعهوش تحت أيقونة قديمة متثبتة
func setAppID() {
	id, _ := syscall.UTF16PtrFromString("Ahmed.Actualat")
	pSetAUMID.Call(uintptr(unsafe.Pointer(id)))
}

// أيقونة كبيرة (شريط المهام / Alt+Tab) وصغيرة (شريط العنوان) من أيقونة البرنامج بالمقاس الصح
func setWindowIcons(hwnd uintptr) {
	h, _, _ := pGetModule.Call(0)
	cx, _, _ := pSysMetrics.Call(11) // SM_CXICON
	sx, _, _ := pSysMetrics.Call(49) // SM_CXSMICON
	big, _, _ := pLoadImage.Call(h, 2, 1, cx*2, cx*2, 0)
	if big == 0 {
		big, _, _ = pLoadImage.Call(h, 2, 1, cx, cx, 0)
	}
	small, _, _ := pLoadImage.Call(h, 2, 1, sx, sx, 0)
	if big != 0 {
		pSendMsg.Call(hwnd, 0x80, 1, big) // WM_SETICON, ICON_BIG
	}
	if small != 0 {
		pSendMsg.Call(hwnd, 0x80, 0, small) // ICON_SMALL
	}
}

// ويندوز بيحفظ الأيقونات القديمة — ده بيخليه يقرا الجديدة
func refreshIconCache() {
	hiddenCmd(filepath.Join(os.Getenv("SystemRoot"), "System32", "ie4uinit.exe"), "-show").Run()
}

func msgBox(title, text string) {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(text)
	pMessageBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x40)
}

// شريط عنوان الويندوز داكن/فاتح (Windows 10 20H1+ / 11)
func darkTitle(hwnd uintptr, dark bool) {
	if hwnd == 0 || pDwmSetAttr.Find() != nil {
		return
	}
	v := int32(0)
	if dark {
		v = 1
	}
	pDwmSetAttr.Call(hwnd, 20, uintptr(unsafe.Pointer(&v)), 4)
	pDwmSetAttr.Call(hwnd, 19, uintptr(unsafe.Pointer(&v)), 4)
	// لون الشريط نفسه على ويندوز 11
	col := uint32(0x00ffffff)
	if dark {
		col = 0x00140f0f // BGR لـ #0f0f14 تقريباً
	}
	pDwmSetAttr.Call(hwnd, 35, uintptr(unsafe.Pointer(&col)), 4)
	// SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_FRAMECHANGED
	pRedraw.Call(hwnd, 0, 0, 0, 0, 0, 0x0002|0x0001|0x0004|0x0020)
}

type openFileName struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

// نافذة "حفظ باسم" بتاعة ويندوز — بتفتح على آخر فولدر اتحفظ فيه
func saveDialog(name string) (path string, err error) {
	type ret struct {
		p   string
		err error
	}
	ch := make(chan ret, 1)
	go func() {
		defer func() {
			if e := recover(); e != nil {
				ch <- ret{"", fmt.Errorf("نافذة الحفظ وقفت: %v", e)}
			}
		}()
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		pCoInit.Call(0, 2)
		buf := make([]uint16, 1024)
		n, _ := syscall.UTF16FromString(name)
		copy(buf, n)
		filter := utf16.Encode([]rune("Excel (*.xlsx)\x00*.xlsx\x00\x00"))
		ttl := "حفظ الملف"
		if uiLang() == "en" {
			ttl = "Save file"
		}
		title, _ := syscall.UTF16PtrFromString(ttl)
		ext, _ := syscall.UTF16PtrFromString("xlsx")
		start := getSetting("lastDir")
		if fi, err := os.Stat(start); start == "" || err != nil || !fi.IsDir() {
			home, _ := os.UserHomeDir()
			start = filepath.Join(home, "Desktop")
		}
		dir, _ := syscall.UTF16PtrFromString(start)
		ofn := openFileName{lpstrFilter: &filter[0], lpstrFile: &buf[0], nMaxFile: uint32(len(buf)),
			lpstrTitle: title, lpstrDefExt: ext, lpstrInitialDir: dir, flags: 0x00000002 | 0x00000800 | 0x00080000}
		ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
		r, _, _ := pGetSave.Call(uintptr(unsafe.Pointer(&ofn)))
		runtime.KeepAlive(filter)
		runtime.KeepAlive(buf)
		if r == 0 {
			ch <- ret{"", nil}
			return
		}
		ch <- ret{syscall.UTF16ToString(buf), nil}
	}()
	x := <-ch
	return x.p, x.err
}

// يفتح الفولدر ويحدد الملف
func openFolder(p string) error {
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		return exec.Command("explorer", p).Start()
	}
	return exec.Command("explorer", "/select,", p).Start()
}

func main() {
	setAppID()
	if maybeInstall() {
		return
	}
	url, err := startServer()
	if err != nil {
		msgBox("اكشوالات", "حصل خطأ: "+err.Error())
		return
	}
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  filepath.Join(appDir(), "webview"),
		WindowOptions: webview2.WindowOptions{
			Title: "اكشوالات · Actualat", Width: 1360, Height: 900, Center: true, IconId: 2,
		},
	})
	if w == nil {
		msgBox("اكشوالات", "البرنامج محتاج Microsoft Edge WebView2 Runtime. نزّله من موقع مايكروسوفت وشغّل البرنامج تاني.")
		exec.Command("rundll32", "url.dll,FileProtocolHandler", "https://developer.microsoft.com/microsoft-edge/webview2/").Start()
		return
	}
	defer w.Destroy()
	hwnd := uintptr(w.Window())
	setWindowIcons(hwnd)
	darkTitle(hwnd, uiTheme() == "dark")
	titleHook = func(dark bool) { w.Dispatch(func() { darkTitle(hwnd, dark) }) }
	w.SetSize(1360, 900, webview2.HintNone)
	w.Navigate(url)
	w.Run()
}

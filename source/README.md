# اكشوالات · Actualat — الكود

برنامج ويندوز بيحوّل شيتات صرف المخازن ومستخلصات المقاولين (شيت Sub Contractor في الكوست ريبورت) لقوالب PCS.
Go 1.24 + WebView2، وواجهة HTML جوه البرنامج، وقاعدة بيانات bbolt على الجهاز.

## البناء
1. انسخ `updkey.go.example` باسم `updkey.go` وحط فيه مفتاح التحديثات (الملف ده مش بيترفع هنا).
2. على ويندوز:
```
go install github.com/akavel/rsrc@latest
rsrc -ico app.ico -manifest app.manifest -arch amd64 -o rsrc_windows_amd64.syso
set GOOS=windows& set GOARCH=amd64& set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w -H windowsgui" -o Actualat.exe .
```
`Actualat.exe` هو نفسه ملف الـ Setup: أول ما يشتغل بينسخ نفسه لـ `%LOCALAPPDATA%\Programs\Actualat`،
ويعمل اختصار على الديسكتوب وفي Start، ويسجل نفسه في تطبيقات ويندوز، وبعدين يفتح على شاشة إنشاء الحساب.

## الملفات
| الملف | فيه إيه |
|---|---|
| `main_windows.go` | الشباك، التثبيت، الاختصارات، إلغاء التثبيت، الأيقونة |
| `server.go` | السيرفر المحلي والصفحات |
| `auth.go` | الحسابات وتسجيل الدخول |
| `store.go` / `ipc.go` / `codes.go` | تحويل المخازن والمستخلصات |
| `db.go` / `analytics.go` | السجل والـ Dashboard |
| `update.go` / `updkey.go` | التحديثات من GitHub |
| `web_*.html`, `ui.html`, `assets/web/` | الواجهة (عربي/English) |
| `logo/` | اللوجو والأيقونات |

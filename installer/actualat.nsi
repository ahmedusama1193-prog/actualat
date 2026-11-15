Unicode true
ManifestDPIAware true
!include "MUI2.nsh"

!define APPNAME "Actualat"
!define DISPLAY "اكشوالات · Actualat"
!define VERSION "2.8.1"
!define UNINSTKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\Actualat"

Name "${DISPLAY}"
OutFile "Actualat_Setup_${VERSION}.exe"
InstallDir "$LOCALAPPDATA\Programs\${APPNAME}"
RequestExecutionLevel user
SetCompressor /SOLID lzma
BrandingText "Actualat ${VERSION}"
VIProductVersion "2.8.1.0"
VIAddVersionKey /LANG=1025 "ProductName" "Actualat"
VIAddVersionKey /LANG=1025 "FileDescription" "Actualat Setup"
VIAddVersionKey /LANG=1025 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=1025 "CompanyName" "Ahmed Abdulqader"
VIAddVersionKey /LANG=1025 "LegalCopyright" "Ahmed Abdulqader"

!define MUI_ICON "app.ico"
!define MUI_UNICON "app.ico"
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "تثبيت اكشوالات ${VERSION}"
!define MUI_WELCOMEPAGE_TEXT "البرنامج ده بيحوّل شيتات المخازن والمستخلصات والمالية لقوالب استيراد PCS متكودة.$\r$\n$\r$\nهيتثبت على الجهاز ده للمستخدم الحالي، ومش محتاج صلاحيات أدمن، وبعد كده بيحدّث نفسه لوحده.$\r$\n$\r$\nدوس «التالي» للمتابعة."
!define MUI_FINISHPAGE_RUN "$INSTDIR\${APPNAME}.exe"
!define MUI_FINISHPAGE_RUN_TEXT "افتح اكشوالات دلوقتي"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_LANGUAGE "Arabic"

Section "Install"
  ; لو البرنامج شغال اقفله الأول
  nsExec::Exec 'taskkill /F /IM ${APPNAME}.exe'
  Sleep 800
  SetOutPath "$INSTDIR"
  File "${APPNAME}.exe"
  File /oname=actualat-setup.ico "app.ico"
  CreateShortcut "$DESKTOP\اكشوالات.lnk" "$INSTDIR\${APPNAME}.exe" "" "$INSTDIR\actualat-setup.ico" 0
  CreateShortcut "$SMPROGRAMS\اكشوالات.lnk" "$INSTDIR\${APPNAME}.exe" "" "$INSTDIR\actualat-setup.ico" 0
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayName" "${DISPLAY}"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${UNINSTKEY}" "Publisher" "Ahmed Abdulqader"
  WriteRegStr HKCU "${UNINSTKEY}" "DisplayIcon" "$INSTDIR\${APPNAME}.exe,0"
  WriteRegStr HKCU "${UNINSTKEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${UNINSTKEY}" "UninstallString" '"$INSTDIR\${APPNAME}.exe" uninstall'
  WriteRegDWORD HKCU "${UNINSTKEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINSTKEY}" "NoRepair" 1
SectionEnd

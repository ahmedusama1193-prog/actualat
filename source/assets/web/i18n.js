/* اكشوالات — اللغة (عربي / English)
   الصفحات مكتوبة بالعربي. في وضع English الملف ده بيترجم النصوص في الصفحة وهي شغالة
   (حتى اللي بتتكتب بالـ JavaScript بعدين) ويقلب الاتجاه لشمال ← يمين.
   أي عنصر عليه translate="no" (بيانات المستخدم: أسماء الأصناف والمقاولين والشيتات) مابيتترجمش. */
(function(){
const D={
// ---- 2.1: شاشة البداية
"أهلاً،":"Hi,","خلص":"Done","الشهر ده":"This month","ادخل البرنامج":"Open Actualat","اعرض الشاشة دي لما البرنامج يفتح":"Show this screen when the app opens","للدخول":"to open","البداية":"Start","شاشة البداية":"Start screen","الشاشة اللي فيها اللوجو وحالة الشهر، بتظهر أول ما البرنامج يفتح":"The screen with the logo and this month's status, shown when the app opens","تظهر":"Show","ماتظهرش":"Don't show","الأحد":"Sunday","الاثنين":"Monday","الثلاثاء":"Tuesday","الأربعاء":"Wednesday","الخميس":"Thursday","الجمعة":"Friday","السبت":"Saturday",
// ---- 1.7: الحسابات
"افتكر اسم المستخدم":"Remember my username","تسجيل الدخول":"Sign in","ادخل باسم المستخدم والباسورد بتوعك.":"Enter your username and password.","اسم المستخدم":"Username","الباسورد":"Password","أكد الباسورد":"Confirm password","إظهار الباسورد":"Show password","دخول":"Sign in","إنشاء حساب":"Create account","إنشاء الحساب والدخول":"Create account & sign in","أول مرة البرنامج يفتح على الجهاز ده. اختار اسم مستخدم وباسورد، وهتدخل بيهم كل مرة.":"First time on this computer. Choose a username and password; you'll sign in with them every time.","الحساب بيتحفظ على الجهاز ده بس. لو نسيت الباسورد، أي حساب تاني على نفس الجهاز يقدر يمسح حسابك ويعمله من جديد من الإعدادات.":"The account is stored on this computer only. If you forget the password, another account on the same computer can delete yours and recreate it from Settings.","صرف المخازن ومستخلصات المقاولين، جاهزين كقوالب PCS.":"Store issues and subcontractor IPCs, ready as PCS templates.","الباسورد والتأكيد مش زي بعض":"Password and confirmation don't match","اسم المستخدم أو الباسورد غلط":"Wrong username or password","اسم المستخدم لازم يكون من 3 لـ 32 حرف، من غير مسافات":"Username must be 3–32 characters, no spaces","الباسورد لازم يكون 6 حروف أو أكتر":"Password must be at least 6 characters","اسم المستخدم ده موجود قبل كده":"That username already exists","في حساب على الجهاز ده بالفعل — سجّل دخول":"This computer already has an account — sign in","لازم تسجّل دخول":"Please sign in","الباسورد الحالي غلط":"Current password is wrong","مينفعش تمسح الحساب اللي داخل بيه":"You can't delete the account you're signed in with","تسجيل الخروج":"Sign out","خروج":"Sign out","الحسابات على الجهاز ده":"Accounts on this computer","إضافة حساب":"Add account","لحد تاني بيستخدم نفس الجهاز — كل الحسابات بتشوف نفس السجل":"For someone else on this computer — all accounts share the same history","إضافة":"Add","تغيير الباسورد":"Change password","للحساب اللي انت داخل بيه":"For the account you're signed in with","الباسورد الحالي":"Current password","الباسورد الجديد":"New password","اتعمل":"Created","مسح الحساب":"Delete account","الحساب اتضاف":"Account added","الباسورد اتغير":"Password changed","تمسح الحساب":"Delete account","الحسابات، شكل البرنامج، اللغة، العملة، التحديثات ومكان البيانات":"Accounts, appearance, language, currency, updates and data location",
// ---- 1.6: الإقفال الشهري
"افتح المستودعات، واختار شيت صرف المخازن، وحوّله واحفظ ملف PCS.":"Open Store Issues, pick the store-issue sheet, convert it and save the PCS file.","افتح المستخلصات، واختار الكوست ريبورت، واحفظ المستخلص الحالي والتراكمي.":"Open IPCs, pick the cost report, and save the current and cumulative IPC.","ارجع هنا، وهتلاقي كل شهر قدامه اللي اتعمل واللي ناقص.":"Come back here to see, for each month, what is done and what is missing.",
"الإقفال الشهري":"Monthly close","كل شهر محتاج 3 ملفات PCS: صرف المخازن، والمستخلص الحالي، والمستخلص التراكمي":"Each month needs 3 PCS files: store issues, current IPC and cumulative IPC",
"تحويل صرف مخازن":"Convert store issues","تحويل مستخلصات":"Convert IPCs","لسه مفيش ملفات محفوظة":"No saved files yet",
"الجدول ده بيتملى لوحده من السجل. ابدأ كده:":"This table fills itself from your history. Start like this:","ابدأ بالمستودعات":"Start with store issues",
"الشهر":"Month","صرف المخازن":"Store issues","المستخلص الحالي":"Current IPC","المستخلص التراكمي":"Cumulative IPC","الحالة":"Status",
"آخر الملفات المحفوظة":"Recently saved files","آخر 5 ملفات — كل الملفات في":"Last 5 files — all files in","اتحفظ":"Saved","لسه":"Not yet","حوّل دلوقتي":"Convert now",
"مقفول":"Closed","ناقص":"Missing","من":"of","الشهر الحالي":"This month",
"يناير":"January","فبراير":"February","مارس":"March","أبريل":"April","مايو":"May","يونيو":"June","يوليو":"July","أغسطس":"August","سبتمبر":"September","أكتوبر":"October","نوفمبر":"November","ديسمبر":"December",
"الرسومات بتتعمل من ملفات PCS اللي اتحفظت. حوّل شيت صرف مخازن واحفظه، وهتلاقي هنا أكتر المواد صرفًا بالفلوس وتوزيعها على الشهور.":"Charts are built from the PCS files you've saved. Convert and save a store-issue sheet to see the top materials by value and how spend spreads across months.",
"الرسومات بتتعمل من ملفات PCS اللي اتحفظت. حوّل الكوست ريبورت واحفظ المستخلص، وهتلاقي هنا أكتر البنود والمقاولين صرفًا.":"Charts are built from the PCS files you've saved. Convert the cost report and save the IPC to see the top items and subcontractors by spend.",
"لسه مفيش ملفات اتحفظت. كل ملف PCS تحفظه من المستودعات أو المستخلصات بيتسجل هنا بتاريخه وقيمته، وتقدر تفتح فولدره أو تشيله من السجل.":"Nothing saved yet. Every PCS file you save from Store Issues or IPCs is logged here with its date and value; you can open its folder or remove it from history.",
// ---- الشريط الجانبي / عام
"اكشوالات · Actualat":"Actualat","المستودعات والمستخلصات ← قوالب PCS":"Store issues & IPCs → PCS templates","اكشوالات":"Actualat","الرئيسية":"Home","المستودعات":"Store Issues","المستخلصات":"Subcontractor IPCs",
"السجل":"History","الإعدادات":"Settings","المستخدم":"User","التحديثات":"Updates","بيدوّر…":"Checking…","بيتأكد من الاتصال…":"Checking connection…",
"الإصدار":"Version","جاري تنزيل التحديث… البرنامج هيقفل ويفتح تاني لوحده":"Downloading the update… the app will close and reopen by itself",
"ج.م":"EGP","ر.س":"SAR","تحديث جديد":"New update","اضغط":"click","متصل بالتحديثات":"Connected to updates","من غير نت — البرنامج شغال عادي":"Offline — the app works normally",
"مقدرش يوصل":"Can't reach server","محدّث":"Up to date","خطأ":"Error","افتح فولدر البيانات":"Open data folder","📁 افتح فولدر البيانات":"📁 Open data folder",
"جاري الشغل…":"Working…","حفظ":"Save","افتح":"Open","افتح الفولدر":"Open folder","الكل":"All","مفيش بيانات":"No data",
// ---- Dashboard
"فين الفلوس اتصرفت — من ملفات PCS المحفوظة في السجل":"Where the money went — from the PCS files saved in History","رسم":"Chart","جدول":"Table",
"الفترة":"Period","أعلى":"Top","لسه مفيش ملفات محفوظة هنا":"No saved files here yet",
"أول ما تحفظ ملف PCS من التاب ده هيظهر تحليله هنا.":"Once you save a PCS file from this tab, its analysis will show here.",
"أول ما تحفظ ملف PCS من المستودعات هيظهر تحليله هنا.":"Once you save a PCS file from Store Issues, its analysis will show here.",
"أول ما تحفظ ملف مستخلصات PCS هيظهر تحليله هنا.":"Once you save a PCS IPC file, its analysis will show here.",
"روح للمستودعات":"Go to Store Issues","روح للمستخلصات":"Go to IPCs","إجمالي القيمة":"Total value","عدد الأصناف":"Materials","أكبر صنف":"Top material",
"نصيب أعلى 10":"Top 10 share","نصيب أعلى":"Share of top","من إجمالي القيمة":"of total value","أكتر المواد صرفاً":"Top materials by spend","بالقيمة المالية":"By value",
"الأعمال":"Works","البنود":"Items","المقاولين":"Contractors","حسب Task Code":"By Task Code","توزيع القيمة على بنود PCS":"Value split across PCS task codes",
"حسب الشهر":"By month","قيمة كل شهر في الفترة اللي اخترتها":"Value of each month in the selected period","أكتر البنود صرفاً":"Top items by spend",
"أكتر الأعمال صرفاً":"Top works by spend","أكتر المقاولين صرفاً":"Top contractors by spend","بالقيمة المالية — كل صنف متجمع من كل سطوره":"By value — each material summed across all its lines",
"قيمة الشهر":"Value this month","بالقيمة المالية في المستخلص اللي اخترته":"By value in the selected IPC","سطر":"lines","عدد الأعمال":"Works","عدد المقاولين":"Contractors",
"عدد البنود":"Items","صنف مختلف اتصرف":"distinct materials issued","في الفترة دي":"in this period","أكبر أعمال":"Top work","أكبر مقاول":"Top contractor","أكبر بند":"Top item",
"الصنف":"Material","الكمية":"Qty","البند":"Item","المقاول":"Contractor","السطور":"Lines","الوصف":"Description","النسبة":"Share","القيمة":"Value",
"كل الشهور المحفوظة":"All saved months","(من غير كود)":"(no code)","تراكمي":"cumulative","حالي":"current",
// ---- الرئيسية
"ملخص اللي اتحوّل واتحفظ من المستودعات والمستخلصات":"Summary of what was converted and saved from Store Issues and IPCs","تحويل صرف مخازن":"Convert store issues",
"شيت المخازن ← قالب PCS":"Store sheet → PCS template","تحويل مستخلصات مقاولين":"Convert subcontractor IPCs","الكوست ريبورت ← قالب مستخلصات PCS":"Cost report → PCS IPC template",
"آخر مستخلصات":"Latest IPCs","آخر صرف مخازن":"Latest store issues","ملفات الشهر ده":"Files this month","كل الملفات المحفوظة":"All saved files",
"في قاعدة البيانات على جهازك":"In the database on your PC","شوف التحليل في الـ Dashboard":"See the analysis in the Dashboard",
"أكتر المواد صرفاً في المستودعات، وأكتر الأعمال والبنود في المستخلصات":"Top materials by spend in Store Issues, and top works and items in IPCs",
"آخر قيمة لكل فترة":"Latest value per period","لما تحفظ ملفات هتظهر هنا":"Saved files will show here","آخر الملفات":"Latest files","السجل كله ←":"Full history →",
"لسه مفيش ملفات":"No files yet","لسه ماتحفظش":"Not saved yet","أهلاً":"Hello","مستخلص":"IPCs","بند":"items","مستخلصات":"IPCs","مستودعات":"store",
// ---- السجل
"سجل الملفات المحفوظة":"Saved files history","كل ملف PCS بتحفظه بيتسجل هنا بنسخة منه ومن ملف المراجعة — تقدر تنزّله تاني في أي وقت":"Every PCS file you save is recorded here with a copy of it and its review file — download it again any time",
"قاعدة البيانات شغالة":"Database is running","قاعدة البيانات مش متاحة":"Database not available","تاريخ الحفظ":"Saved on","التصنيف":"Type","اسم الملف":"File name",
"عدد البنود":"Items","القيمة المالية":"Value","الحالة":"Status","لسه مفيش ملفات اتحفظت. أول ما تحفظ ملف PCS هيظهر هنا.":"No files saved yet. Once you save a PCS file it will show here.",
"ملف PCS":"PCS file","المراجعة":"Review","بحث باسم الملف أو الفترة…":"Search by file name or period…","الملف موجود في مكانه":"File is in its place",
"الملف اتنقل أو اتمسح — تقدر تنزّله تاني من هنا":"File was moved or deleted — you can download it again from here","حذف من السجل":"Delete from history",
"محفوظ":"Saved","مش في مكانه":"Moved","تمسح السجل ده؟ (الملف اللي على جهازك مش هيتمسح)":"Delete this record? (The file on your PC will not be deleted)",
// ---- الإعدادات
"شكل البرنامج، اللغة، العملة، التحديثات ومكان البيانات":"Look, language, currency, updates and data location","الشكل واللغة":"Look & language","اللغة":"Language",
"لغة الواجهة — ملفات PCS والمراجعة مش بتتغير":"Interface language — PCS and review files don't change","العربية":"العربية","الثيم":"Theme",
"داكن زي التصميم الجديد، أو فاتح للشغل في النهار":"Dark like the new design, or light for daytime work","داكن":"Dark","فاتح":"Light",
"العملة اللي بتظهر جنب المبالغ":"Currency shown next to amounts","في الرئيسية والـ Dashboard والسجل بس — مش بتغيّر أي رقم في ملفات PCS":"Only on Home, Dashboard and History — doesn't change any number in PCS files",
"من غير":"None","البرنامج بيدوّر على تحديث لوحده كل ما تفتحه":"The app checks for updates every time you open it","دوّر على تحديث":"Check for updates",
"البيانات على جهازك":"Data on your PC","فولدر البيانات":"Data folder","قاعدة البيانات (السجل) + ذاكرة الأكواد — انسخه لو عايز نسخة احتياطية":"Database (history) + code memory — copy it for a backup",
"آخر فولدر اتحفظ فيه":"Last save folder","نافذة الحفظ بتفتح عليه على طول":"The save window opens there directly","مقدرش يوصل لمكان التحديثات — اتأكد من النت":"Can't reach the update server — check your internet",
"انت على آخر إصدار ✓":"You're on the latest version ✓","انت على آخر إصدار":"You're on the latest version","حصل خطأ":"Something went wrong","في إصدار جديد":"New version",
// ---- المستودعات
"تحويل صرف المخازن لقالب PCS — الكميات والقيم زي ما هي في الشيت":"Convert store issues to the PCS template — quantities and values exactly as in the sheet",
"اختار شيت المخازن":"Choose the store sheet","اسحب الملف هنا أو":"Drag the file here or","اضغط للاختيار":"click to choose","xlsx / xlsm — أي شكل لشيت المخازن":"xlsx / xlsm — any store sheet layout",
"الشيت:":"Sheet:","الشيت":"Sheet","صف العناوين:":"Header row:","صف العناوين":"Header row","ربط الأعمدة":"Map columns","عمود PCS":"PCS column","العمود في شيت المخازن":"Column in the store sheet",
"مثال من أول صف":"Sample from first rows","اختار الأعمدة اللي تتكتب في الملاحظات (اختياري)":"pick the columns written into Remarks (optional)",
"— اختار الأعمدة اللي تتكتب في الملاحظات (اختياري)":"— pick the columns written into Remarks (optional)","التالي":"Next","الشهور":"Months","اختار الكل":"Select all","إلغاء الكل":"Clear all",
"Task Code و Cost Code":"Task Code & Cost Code","محتاج مراجعة":"Needs review","بحث بالصنف أو الكود…":"Search by material or code…","كود الصنف":"Item code","القسم":"Section",
"الأكواد اللي بتختارها البرنامج بيفتكرها، والمرة الجاية نفس الصنف بياخد نفس الكود لوحده.":"The app remembers the codes you choose; next time the same material gets the same code automatically.",
"تحويل":"Convert","حفظ ملف PCS… (اختار المكان)":"Save PCS file… (choose location)","✔ البرنامج عارف شكل الشيت ده من قبل وربط الأعمدة لوحده. راجعها ودوس التالي.":"✔ The app knows this sheet layout and mapped the columns itself. Review and click Next.",
"البرنامج ربط الأعمدة اللي عرفها من أسمائها. اختار الباقي من القوايم. (هيفتكر الربط ده المرة الجاية)":"The app mapped the columns it recognised by name. Pick the rest from the lists. (It will remember this mapping next time)",
"— مش موجود —":"— not present —","من الشيت":"from sheet","✔ مطابق لشيت المخازن":"✔ Matches the store sheet","✖ في اختلاف — بلّغني":"✖ Mismatch — report it",
"عدد البنود":"Items","إجمالي الكمية":"Total qty","شيت المخازن":"Store sheet","ذاكرة":"memory","مقترح":"suggested","مش لاقي":"not found","قاعدة":"rule","انت":"you",
"⚠ كود مش في القايمة":"⚠ code not in the list","⚠ كود مش في قايمة PCS":"⚠ code not in the PCS list",
"Date — التاريخ":"Date","Resource Code — كود الصنف":"Resource Code","Resource Name — اسم الصنف":"Resource Name","Description — الوصف":"Description","Unit — الوحدة":"Unit",
"Qty — الكمية":"Qty","Unit Cost — السعر":"Unit Cost","Rate Assumed — سعر تقديري":"Rate Assumed","Amount — القيمة":"Amount","Task Code — تاسك كود":"Task Code",
"Cost Code — كوست كود":"Cost Code","القسم / Task Description":"Section / Task Description","تاريخ الصرف":"Issue date","كود الصنف في PCS لو موجود":"PCS item code if any",
"الاسم اللي PCS بيعرف بيه الصنف":"The name PCS knows the item by","بيتاخد بداله لو اسم الصنف فاضي":"Used when the item name is empty","السعر أو القيمة":"Rate or amount",
"بيتاخد لو السعر فاضي":"Used when the rate is empty","لو فاضي البرنامج بيحطه":"Filled by the app if empty","بيساعد في اقتراح الـ Task Code":"Helps suggest the Task Code",
// ---- المستخلصات
"تحويل شيت Sub Contractor في الكوست ريبورت لقالب مستخلصات PCS":"Convert the cost report's Sub Contractor sheet to the PCS IPC template","اختار الكوست ريبورت":"Choose the cost report",
"xlsx / xlsm — البرنامج بيقرا شيت Sub Contractor بس":"xlsx / xlsm — only the Sub Contractor sheet is read","نوع المستخلص":"IPC type","إجمالي (تراكمي)":"Cumulative",
"الكمية والقيمة التراكمية لحد تاريخ المستخلص — Cumulative Qty":"Cumulative qty and value up to the IPC date — Cumulative Qty","حالي (شهر واحد)":"Current (one month)",
"كمية الشهر اللي تختاره بس — Qty This IPC":"Only the chosen month's qty — Qty This IPC","الشهر:":"Month:","الشهر":"Month","تاريخ المستخلص:":"IPC date:","تاريخ المستخلص":"IPC date",
"Task Code و Cost Code (أكواد PCS)":"Task Code & Cost Code (PCS codes)","بحث بالبند أو العقد أو الكود…":"Search by item, contract or code…","تحديث قوائم أكواد PCS…":"Update PCS code lists…",
"ارفع ملفات الأكواد الجديدة من PCS (نفس شكل Task_Codes / Cost_Codes):":"Upload the new code files from PCS (same layout as Task_Codes / Cost_Codes):",
"وصف التاسك في الشيت":"Task description in sheet","العقد":"Contract","مثال بند":"Sample item","المصدر":"Source",
"أي كود تغيّره البرنامج بيفتكره، والشهر الجاي نفس البند في نفس العقد بياخد نفس الكود لوحده.":"Any code you change is remembered; next month the same item in the same contract gets it automatically.",
"تفاصيل المستخلصات":"IPC details","رقم المستخلص":"IPC No","الصافي":"Net","المحتجز":"Retention","المقاولين/العقود":"Contractors / contracts","صافي الأعمال التراكمي":"Cumulative net works",
"كلهم متربطين، راجعهم بسرعة.":"All mapped — give them a quick review.","هيتنقل كل بند ليه كمية في الشهر ده بس، والمحتجز = نسبة الضمان × قيمة الشهر.":"Every item with a qty in this month is transferred, and retention = retention % × the month's value.",
"هتتنقل الكمية والقيمة التراكمية والمحتجز التراكمي زي ما هما في الشيت. المقاولين اللي مالهمش مستخلص بيتشالوا.":"Cumulative qty, value and retention are transferred exactly as in the sheet. Contractors without an IPC are left out.",
"✔ مطابق لشيت المقاولين":"✔ Matches the subcontractor sheet","بنود الأعمال في ملف PCS":"Work items in the PCS file","الخصومات (Other Deductions)":"Deductions (Other Deductions)",
"مقاولين اتشالوا":"Contractors left out","صافي ملف PCS":"PCS file net","المحتجز (شيت / PCS)":"Retention (sheet / PCS)","مستخلصات · سطور":"IPCs · lines",
"وملف المراجعة جنبه.":"and the review file next to it.","وملف المراجعة جنبه، واتسجل في السجل.":"The review file is next to it, and it's recorded in History.",
// ---- رسائل البرنامج
"مش قادر أفتح الملف":"Can't open the file","مش لاقي شيت Sub Contractor — اختاره من القايمة":"Can't find a Sub Contractor sheet — choose it from the list",
"اختار الكوست ريبورت الأول":"Choose the cost report first","اختار الملف الأول":"Choose the file first","اختار الشهر":"Choose the month","تاريخ المستخلص مش مظبوط":"IPC date is not valid",
"دوس تحويل الأول":"Click Convert first","حوّل الأول":"Convert first","دوس التالي الأول":"Click Next first","اختار شهر واحد على الأقل":"Choose at least one month",
"لازم تربط اسم الصنف أو كوده":"You must map the item name or code","لازم تربط السعر أو القيمة":"You must map the rate or amount","لازم تربط عمود التاريخ":"You must map the date column",
"لازم تربط عمود الكمية":"You must map the qty column","مفيش ولا بند يتنقل في الشهور اللي اخترتها":"No items to transfer in the months you chose",
"مقدرتش أحفظ الملف — لو مفتوح في Excel اقفله وجرّب تاني":"Couldn't save the file — if it's open in Excel, close it and try again","الملف ده مش متسجل":"This file isn't recorded",
"السجل ده مش موجود":"This record doesn't exist","مفيش نت أو مكان التحديث مش متاح":"No internet or update server unavailable","الملف اللي نزل مش برنامج سليم":"The downloaded file isn't a valid program",
"الملف اللي نزل مش مطابق — جرّب تاني":"The downloaded file doesn't match — try again","متاح على ويندوز بس":"Available on Windows only","خطأ غير متوقع":"Unexpected error",
"اتحفظ الملف بس مااتسجلش في السجل":"File saved but not recorded in History",
"مطابق لشيت المخازن":"Matches the store sheet","مطابق لشيت المقاولين":"Matches the subcontractor sheet","في اختلاف — بلّغني":"Mismatch — report it",
// ---- 2.3: تكويد تلقائي + شهر بشهر
"عرض / تعديل الأكواد (اختياري)":"Show / edit codes (optional)","حفظ ملفات PCS… (اختار المكان واسم الفولدر)":"Save PCS files… (choose location and folder name)","جاهز، دوس «تحويل» على طول.":"Ready — just press Convert.",
"أي كود تغيّره البرنامج بيفتكره، والمرة الجاية نفس الصنف بياخد نفس الكود لوحده.":"The app remembers any code you change; next time the same material gets it automatically.",
"صنف معروف":"known item","نفس عيلة الكود":"same code family","صنف شبيه":"similar item","من اسم الصنف":"from item name","من القسم":"from section","افتراضي":"default",
"إجمالي — مرة واحدة":"Cumulative — one file","ملف واحد: الكمية والقيمة التراكمية لحد تاريخ المستخلص — Cumulative Qty":"One file: cumulative qty and value up to the IPC date — Cumulative Qty",
"شهر بشهر (تراكمي)":"Month by month (cumulative)","ملف PCS لكل شهر، تراكمي لحد آخر الشهر — ومعاهم ملف مراجعة مجمّع":"One PCS file per month, cumulative to month end — plus one combined review file",
"اختار الشهور اللي يطلعلها ملف (تاريخ كل مستخلص = آخر الشهر):":"Pick the months to export (each IPC is dated at month end):",
"تاريخ المستخلص":"IPC date","مستخلصات":"IPCs","سطور":"Lines","الصافي التراكمي":"Cumulative net","حركة الشهر":"Month movement","المحتجز التراكمي":"Cumulative retention",
"لما تدوس حفظ: اختار المكان واسم الفولدر، والبرنامج هيحط فيه ملف PCS لكل شهر + ملف المراجعة المجمّع، وكل شهر هيتسجل في السجل لوحده.":"When you save: pick a location and folder name; the app puts one PCS file per month plus the combined review file in it, and records each month in History.",
"وكل شهر اتسجل في السجل لوحده.":"Each month was recorded in History.",
"لكل شهر: الكمية التراكمية = مجموع أعمدة الشهور لحد الشهر ده × الفئة، والمحتجز = نسبة الضمان × القيمة. آخر شهر بيطلع نفس «مرة واحدة» بالظبط. رقم المستخلص = آخر رقم في الشيت ناقص عدد الشهور اللي فيها حركة بعده.":"For each month: cumulative qty = sum of the month columns up to that month × rate, and retention = retention % × value. The last month comes out exactly like “one file”. IPC No = the sheet's latest number minus the months with movement after it.",
"كود مش في القايمة":"code not in the list","كود مش في قايمة PCS":"code not in the PCS list","البرنامج عارف شكل الشيت ده من قبل وربط الأعمدة لوحده. راجعها ودوس التالي.":"The app knows this sheet layout and mapped the columns itself. Review and click Next."
};
// عبارات بتتبدل جوه نص أطول (الأطول الأول) — للنصوص اللي فيها أرقام
const P={
"البرنامج كوّد":"The app coded","صنف لوحده":"items automatically","(مالهمش Task Code أو Cost Code في الشيت) —":"(no Task Code or Cost Code in the sheet) —",
"كل الأكواد اللي اتحطت هتلاقيها في ملف المراجعة (شيت «الأكواد»).":"All assigned codes are listed in the review file (sheet “الأكواد”).",
"صنف اتكوّد تلقائي — تفاصيلهم في شيت «الأكواد» في ملف المراجعة.":"items were coded automatically — details in the “الأكواد” sheet of the review file.",
"ملف — كل شهر مطابق لشيت المقاولين":"files — every month matches the subcontractor sheet","، وآخر شهر = إجمالي الشيت":", and the last month = sheet total",
"صافي آخر ملف":"Net of last file","المحتجز (شيت / آخر ملف)":"Retention (sheet / last file)","لحد":"up to","حركة":"movement",
"ملف PCS + ملف المراجعة المجمّع في:":"PCS files + the combined review file saved in:","اتحفظ":"Saved",
"صنف في الشيت · إجمالي القيمة":"items in the sheet · total value","صنف مالهمش Task Code أو Cost Code في الشيت، فالبرنامج حطلهم أكواد.":"items have no Task Code or Cost Code in the sheet, so the app assigned codes.",
"منهم مقترحين ومحتاجين مراجعة منك (الصفوف الملونة).":"of them are suggestions that need your review (coloured rows).","منهم مقترحين ومحتاجين مراجعة (الصفوف الملونة).":"of them are suggestions that need review (coloured rows).",
"كلهم من الذاكرة":"all from memory","قايمة الأكواد: من الملف نفسه":"Code list: from the file itself","قايمة الأكواد: آخر قايمة استخدمتها":"Code list: the last list you used",
"قايمة الأكواد: آخر قايمة رفعتها":"Code list: the last list you uploaded","قايمة الأكواد: قايمة PCS الأساسية":"Code list: built-in PCS list",
"الشيت مفيهوش Task Code ولا Cost Code، فالبرنامج ربط":"The sheet has no Task Code or Cost Code, so the app mapped","مجموعة بنود بأكواد PCS.":"item groups to PCS codes.",
"انقل كمان":"Also transfer","بند تاريخهم فاضي أو مش مفهوم (قيمتهم":"items with an empty or unclear date (value","بتاريخ:":"dated:",
"… استخدم البحث عشان تلاقي باقي الأصناف":"… use search to find the remaining items","بند مالهمش سعر في الشيت — اتنقلوا بالكمية والقيمة زي ما هما.":"items have no rate in the sheet — transferred with qty and value as they are.",
"بند من غير Task Code.":"items without a Task Code.","بند من غير Cost Code.":"items without a Cost Code.","سطر من غير Task Code.":"lines without a Task Code.","سطر من غير Cost Code.":"lines without a Cost Code.",
"الفترة دي اتصدّرت قبل كده":"This period was exported before","المستخلص ده":"This IPC","اتصدّر قبل كده":"was exported before","مرة (آخر مرة":"time(s) (last on",
"— اتأكد إنك مش هتستوردها في PCS مرتين.":"— make sure you don't import it into PCS twice.","عقد اتشال لأن مالوش مستخلص (التفاصيل في ملف المراجعة).":"contracts left out because they have no IPC (details in the review file).",
"كود مش موجود في قايمة PCS. تكمّل؟":"codes are not in the PCS list. Continue?","اتحفظ:":"Saved:","صافي الأعمال في شيت المقاولين":"Net works in the subcontractor sheet",
"القيمة ≠ الكمية × الفئة (فرق":"Value ≠ qty × rate (diff","— اتنقلت القيمة زي ما هي":"— value transferred as is","كمية بالسالب — راجعها قبل الاستيراد":"Negative qty — review before importing",
"مش لاقي صف العناوين (Item / Invoice No) في شيت":"Can't find the header row (Item / Invoice No) in sheet","عمود ناقص في شيت المقاولين:":"Missing column in the subcontractor sheet:",
"الشيت":"Sheet","فاضي":"is empty","مقدرتش":"Couldn't","مكان التحديث رد بـ":"Update server replied","التحميل وقف:":"Download stopped:","التحميل رد بـ":"Download replied",
"مش قادر أحفظ التحديث:":"Can't save the update:","مش قادر أستبدل البرنامج:":"Can't replace the app:","نافذة الحفظ وقفت:":"Save window failed:",
"صرف مخازن":"Store issues","مستخلصات —":"IPCs —","إجمالي حتى":"Cumulative to","حالي":"Current","شهور":"months","باقي":"Other","مقترح":"suggested",
"مجموعة":"groups","صنف":"items","بند":"items","مستخلص":"IPCs","ملف":"files","صف":"row","سطر":"lines","مرة":"times","في":"",
"يناير":"January","فبراير":"February","مارس":"March","أبريل":"April","مايو":"May","يونيو":"June","يوليو":"July","أغسطس":"August","سبتمبر":"September","أكتوبر":"October","نوفمبر":"November","ديسمبر":"December",
"←":"→"
};
const AR=/[؀-ۿ]/;
const norm=s=>s.replace(/\s+/g,' ').trim();
const esc=s=>s.replace(/[.*+?^${}()|[\]\\]/g,'\\$&');
const pk=Object.keys(P).sort((a,b)=>b.length-a.length);
const PRE=new RegExp('(?<![\\u0600-\\u06FF])('+pk.map(esc).join('|')+')(?![\\u0600-\\u06FF])','g');
function tr(s){
  if(s==null) return s; s=String(s);
  if(!AR.test(s)) return s.replace(/←/g,'→');
  const k=norm(s);
  if(D[k]!=null) return s.replace(s.trim(),D[k]);
  // من غير علامات في الأول/الآخر
  const m=k.match(/^([\s·—:✔⚠✖(…]*)(.*?)([\s·—:.()…]*)$/);
  if(m && D[m[2]]!=null) return s.replace(s.trim(),m[1]+D[m[2]]+m[3]);
  let out=s.replace(PRE,(x)=>P[x]);
  out=out.replace(/  +/g,' ');
  return out;
}
function lang(){ try{ if(window.parent!==window && parent.__lang) return parent.__lang; }catch(e){} return window.__lang||'ar'; }
let cur='ar';
const orig=new WeakMap(), mine=new WeakMap();
function skip(n){ for(let e=n.nodeType===1?n:n.parentElement;e;e=e.parentElement){ if(e.getAttribute&&e.getAttribute('translate')==='no') return true; if(e.tagName==='SCRIPT'||e.tagName==='STYLE') return true; } return false; }
function doText(n){
  if(skip(n)) return;
  if(mine.has(n) && mine.get(n)===n.nodeValue) return;
  const o=n.nodeValue; orig.set(n,o);
  if(cur==='en'){ const t=tr(o); if(t!==o){ mine.set(n,t); n.nodeValue=t; } else mine.delete(n); }
}
const ATTRS=['placeholder','title'];
function doAttrs(el){
  if(skip(el)) return;
  for(const a of ATTRS){
    if(!el.hasAttribute(a)) continue;
    const key='data-i18n-'+a; let o=el.getAttribute(key);
    const v=el.getAttribute(a);
    if(o==null || (el.__i18n&&el.__i18n[a]!==v)){ o=v; el.setAttribute(key,o); }
    const t=cur==='en'?tr(o):o; el.__i18n=el.__i18n||{}; el.__i18n[a]=t;
    if(v!==t) el.setAttribute(a,t);
  }
}
function walk(root){
  if(!root) return;
  if(root.nodeType===3){ doText(root); return; }
  if(root.nodeType!==1 && root.nodeType!==9 && root.nodeType!==11) return;
  if(root.nodeType===1) doAttrs(root);
  const w=document.createTreeWalker(root,NodeFilter.SHOW_TEXT|NodeFilter.SHOW_ELEMENT);
  let n; while((n=w.nextNode())){ if(n.nodeType===3) doText(n); else doAttrs(n); }
}
function restore(root){
  const w=document.createTreeWalker(root,NodeFilter.SHOW_TEXT|NodeFilter.SHOW_ELEMENT);
  let n; while((n=w.nextNode())){
    if(n.nodeType===3){ if(mine.has(n)&&mine.get(n)===n.nodeValue&&orig.has(n)){ n.nodeValue=orig.get(n); } mine.delete(n); }
    else for(const a of ATTRS){ const o=n.getAttribute('data-i18n-'+a); if(o!=null){ n.setAttribute(a,o); if(n.__i18n) n.__i18n[a]=o; } }
  }
}
let titleOrig=null;
function setDir(){ const h=document.documentElement; h.lang=cur; h.dir=cur==='en'?'ltr':'rtl'; }
function apply(l){
  cur=l==='en'?'en':'ar'; setDir();
  if(titleOrig==null) titleOrig=document.title;
  document.title=cur==='en'?tr(titleOrig):titleOrig;
  if(!document.body) return;
  if(cur==='en') walk(document.body); else restore(document.body);
}
cur=lang()==='en'?'en':'ar'; setDir();
const obs=new MutationObserver(ms=>{
  if(cur!=='en') { for(const m of ms) if(m.type==='characterData') { orig.set(m.target,m.target.nodeValue); mine.delete(m.target); } return; }
  for(const m of ms){
    if(m.type==='characterData') doText(m.target);
    else if(m.type==='attributes') doAttrs(m.target);
    else m.addedNodes.forEach(walk);
  }
});
function start(){ apply(cur); obs.observe(document.body,{subtree:true,childList:true,characterData:true,attributes:true,attributeFilter:ATTRS}); }
if(document.readyState==='loading') document.addEventListener('DOMContentLoaded',start); else start();
const _a=window.alert.bind(window), _c=window.confirm.bind(window);
window.alert=m=>_a(cur==='en'?tr(m):m); window.confirm=m=>_c(cur==='en'?tr(m):m);
window.I18N={apply,t:s=>cur==='en'?tr(s):s,get lang(){return cur}};
})();

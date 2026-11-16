package main

// 2.20: قاموس عربي ↔ إنجليزي لأسماء مواد البناء — عشان ملف أسعار إنجليزي يتطابق مع أسماء أصناف عربي
// كل كلمة عربي (أو إنجليزي) بتتحول لكلمة إنجليزي واحدة موحّدة، والمطابقة بتتم على الكلمات الموحّدة.

import "strings"

var arEnRaw = map[string]string{
	// مواسير وقطع
	"ماسوره": "pipe", "مواسير": "pipe", "بايب": "pipe", "كوع": "elbow", "اكواع": "elbow", "كيعان": "elbow",
	"طبه": "cap", "طبات": "cap", "غطا": "cover", "غطاء": "cover", "تي": "tee", "تيه": "tee", "جلبه": "coupling", "جلب": "coupling",
	"وصله": "coupling", "وصلات": "coupling", "كوبلنج": "coupling", "مسلوب": "reducer", "مسلوبه": "reducer", "نقاص": "reducer", "مخفض": "reducer", "ريديوسر": "reducer",
	"محبس": "valve", "محابس": "valve", "بلف": "valve", "بلوف": "valve", "يونيون": "union", "ادابتر": "adapter", "اداپتر": "adapter", "مسنن": "threaded",
	"خرطوم": "hose", "مرن": "flexible", "فلكسبل": "flexible", "جوان": "gasket", "جوانات": "gasket", "حلقه": "ring", "سدادات": "plug", "سداده": "plug",
	"صفايه": "drain", "صفايات": "drain", "بيبه": "trap", "سيفون": "siphon", "مجري": "duct", "دكت": "duct",
	// سباكة وصحي
	"خلاط": "mixer", "خلاطات": "mixer", "حوض": "basin", "مغسله": "basin", "حوض مطبخ": "sink", "كرسي": "toilet", "مرحاض": "toilet", "شطاف": "shattaf",
	"سخان": "heater", "كهرباء": "electrical", "كهربا": "electrical", "كهربائي": "electrical", "عازل": "insulation", "عازله": "insulation", "مضخه": "pump", "طلمبه": "pump", "خزان": "tank", "فلتر": "filter", "عداد": "meter", "مياه": "water", "ميه": "water", "صرف": "drain",
	"حنفيه": "tap", "دش": "shower", "بانيو": "bathtub",
	// كهرباء
	"سلك": "wire", "اسلاك": "wire", "كابل": "cable", "كابلات": "cable", "مفتاح": "switch", "مفاتيح": "switch", "بريزه": "socket", "برايز": "socket",
	"علبه": "box", "علب": "box", "لمبه": "lamp", "لمبات": "lamp", "كشاف": "light", "كشافات": "light", "اناره": "lighting", "اضاءه": "lighting",
	"لوحه": "panel", "لوحات": "panel", "قاطع": "breaker", "قواطع": "breaker", "بريكر": "breaker", "ارضي": "earthing", "تاريض": "earthing",
	"مجلفن": "galvanized", "مجلفنه": "galvanized", "شريط": "tape", "ليد": "led", "انذار": "alarm", "حريق": "fire", "طفايه": "extinguisher",
	"رشاش": "sprinkler", "كاميرا": "camera", "مروحه": "fan", "شفاط": "exhaust", "تكييف": "ac", "مكيف": "ac", "محول": "transformer", "مولد": "generator",
	// أبواب وشبابيك وحدايد
	"باب": "door", "ابواب": "door", "شباك": "window", "شبابيك": "window", "نافذه": "window", "حلق": "frame", "فريم": "frame", "ضلفه": "leaf",
	"مفصله": "hinge", "مفصلات": "hinge", "كالون": "lock", "كوالين": "lock", "قفل": "lock", "مقبض": "handle", "اوكره": "handle", "اكره": "handle",
	"زجاج": "glass", "سكريت": "glass", "سيكوريت": "tempered", "المونيوم": "aluminum", "الومنيوم": "aluminum", "منزلق": "sliding", "سحاب": "sliding",
	"درابزين": "handrail", "عتبه": "threshold", "وزره": "skirting", "نعله": "skirting", "سلم": "ladder", "سقاله": "scaffolding", "سقالات": "scaffolding",
	// حديد ومعادن
	"حديد": "steel", "تسليح": "rebar", "صاج": "sheet", "ستانلس": "stainless", "استانلس": "stainless", "نحاس": "copper", "نحاسي": "copper", "معدن": "metal", "معدني": "metal",
	"زاويه": "angle", "ركن": "corner", "شبكه": "mesh", "شبك": "mesh", "مجرى": "channel", "قطاع": "section",
	// خامات
	"اسمنت": "cement", "اسمنتي": "cement", "رمل": "sand", "زلط": "gravel", "بلوك": "block", "طوب": "brick", "خرسانه": "concrete", "جبس": "gypsum", "جبسم": "gypsum",
	"بورد": "board", "عزل": "insulation", "فوم": "foam", "ممبرين": "membrane", "بيتومين": "bitumen", "صوف": "wool", "صخري": "rock",
	"سيراميك": "ceramic", "بورسلين": "porcelain", "بورسلان": "porcelain", "بلاط": "tile", "رخام": "marble", "جرانيت": "granite", "انترلوك": "interlock", "بردوره": "curb",
	"دهان": "paint", "دهانات": "paint", "بويه": "paint", "معجون": "putty", "سيلر": "sealer", "برايمر": "primer", "سليكون": "silicone", "سيليكون": "silicone",
	"غراء": "glue", "غرا": "glue", "لاصق": "adhesive", "روبه": "grout", "خشب": "wood", "خشبي": "wood", "ابلكاش": "plywood", "كونتر": "plywood", "بلاستيك": "plastic", "بلاستيكي": "plastic",
	"بولي": "poly", "مسمار": "nail", "مسامير": "nail", "برغي": "screw", "براغي": "screw", "صاموله": "nut", "صواميل": "nut", "خابور": "anchor", "خوابير": "anchor", "وردة": "washer", "ورده": "washer",
	"زيت": "oil", "سولار": "diesel", "ديزل": "diesel", "بنزين": "petrol",
	// سلامة وعِدد
	"قفاز": "gloves", "جوانتي": "gloves", "خوذه": "helmet", "بوت": "boots", "حذاء": "boots", "صديري": "vest", "سديري": "vest", "نظاره": "goggles",
	"مفك": "screwdriver", "شاكوش": "hammer", "زرديه": "pliers", "بنسه": "pliers", "متر": "m", "شنيور": "drill", "صاروخ": "grinder", "اسطوانه": "cylinder", "سلندر": "cylinder", "اسطوانات": "cylinder",
	// ألوان وأوصاف
	"ابيض": "white", "اسود": "black", "رمادي": "grey", "رصاصي": "grey", "احمر": "red", "ازرق": "blue", "اخضر": "green", "اصفر": "yellow", "بني": "brown", "شفاف": "clear",
	"خارجي": "outdoor", "داخلي": "indoor", "ارضيات": "floor", "ارضيه": "floor", "سقف": "ceiling", "حائط": "wall", "حوائط": "wall", "جدار": "wall", "قاطوع": "partition", "فاصل": "partition",
	"حار": "hot", "بارد": "cold", "تغذيه": "supply", "مزدوج": "double", "مفرد": "single", "ثقيل": "heavy", "مطبخ": "kitchen", "حمام": "bathroom",
}

var enSyn = map[string]string{
	"aluminium": "aluminum", "porcelin": "porcelain", "outside": "outdoor", "external": "outdoor", "internal": "indoor", "inside": "indoor",
	"galvanised": "galvanized", "gi": "galvanized", "gray": "grey", "reduceers": "reducer", "lamps": "lamp", "luminaire": "light", "bulb": "lamp",
	"ss": "stainless", "cu": "copper", "upvc": "pvc", "cpvc": "pvc", "ppruv": "ppr", "ppr": "ppr", "polyethylene": "poly", "pe": "poly", "hdpe": "poly",
	"tap": "tap", "faucet": "tap", "screws": "screw", "bolt": "screw", "bolts": "screw", "wires": "wire", "cables": "cable", "lights": "light", "lighting": "lighting",
	"rawl": "anchor", "dowel": "anchor", "fischer": "anchor", "drains": "drain", "gully": "drain", "sockets": "socket", "outlet": "socket", "outlets": "socket",
	"cover": "cover", "lid": "cover", "caps": "cap", "end": "cap", "mesh": "mesh", "deformed": "rebar", "bars": "rebar", "bar": "rebar", "wc": "toilet", "closet": "toilet",
	"sink": "sink", "washbasin": "basin", "lavatory": "basin", "frames": "frame", "doors": "door", "windows": "window", "hinges": "hinge", "locks": "lock",
}

var arEn map[string]string

func init() {
	arEn = map[string]string{}
	for k, v := range arEnRaw {
		if !strings.Contains(k, " ") {
			arEn[normAr(k)] = v
		}
	}
}

// canonWord: الكلمة الموحّدة (إنجليزي) للكلمة دي — عربي بالقاموس، إنجليزي بالمرادفات والمفرد
func canonWord(w string) string {
	if v, ok := arEn[w]; ok {
		return v
	}
	if strings.HasPrefix(w, "و") && len([]rune(w)) > 3 {
		if v, ok := arEn[string([]rune(w)[1:])]; ok {
			return v
		}
	}
	if v, ok := enSyn[w]; ok {
		return v
	}
	if w[0] >= 'a' && w[0] <= 'z' && len(w) > 3 {
		switch {
		case strings.HasSuffix(w, "ies"):
			w = w[:len(w)-3] + "y"
		case strings.HasSuffix(w, "ss"), strings.HasSuffix(w, "us"):
		case strings.HasSuffix(w, "es") && (strings.HasSuffix(w, "ches") || strings.HasSuffix(w, "shes") || strings.HasSuffix(w, "xes")):
			w = w[:len(w)-2]
		case strings.HasSuffix(w, "s"):
			w = w[:len(w)-1]
		}
		if v, ok := enSyn[w]; ok {
			return v
		}
	}
	return w
}

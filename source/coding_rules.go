package main

// 2.25 — مكتبة التكويد: MasterFormat 2018 (CSI) ← Task Code و Cost Code من قوايم PCS.
// كل سطر قاعدة: الكود | العنوان EN | العنوان AR | Task | Cost مواد | Cost عمالة | Cost مقاول باطن | Cost معدات | المجموعات
// المجموعات بينها «;» ولازم كل مجموعة يبان منها كلمة واحدة على الأقل (AND)، وجوه المجموعة الكلمات بينها «,» (OR).
// بعد «!» كلمات لو ظهرت القاعدة دي ماتنفعش.
// القاعدة اللي بتطابق مجموعات أكتر بتكسب (مثلاً خرسانة + قواعد ← الأساسات، مش الخرسانة العامة).

var csiRuleSrc = `
01 21 00|Allowances / Provisional Sums|مبالغ احتياطية|2-006-003||||| provisional sum,prime cost,contingency,allowance,مبلغ احتياطي,مبالغ احتياطيه,مبلغ مؤقت,احتياطي
01 31 00|Project Management and Coordination|إدارة المشروع والتنسيق|2-002-001|||||site staff,project manager,site engineer,مدير المشروع,مهندس موقع,جهاز الاشراف,جهاز اداري
01 33 00|Submittal Procedures (Shop & As-built Drawings)|الرسومات التنفيذية واللوحات|2-003-004|1-001-001|||| shop drawing,shop drawings,as built,as-built,رسومات تنفيذيه,رسومات ورشه,لوحات تنفيذيه,ما تم تنفيذه
01 45 00|Quality Control & Testing|ضبط الجودة والاختبارات|2-003-002|1-001-006|||| testing,test,laboratory,lab test,اختبار,اختبارات,معمل,معامل,مكعبات,cubes;!sample panel
01 52 00|Construction Facilities (Site Offices)|منشآت الموقع المؤقتة|2-002-005|2-002-005|||| site office,temporary office,caravan,كرفان,كرافان,مكاتب الموقع,مكتب الموقع,مكاتب مؤقته,مكتب مهندس
01 54 00|Construction Aids (Scaffolding)|معدات مساعدة (سقالات)|2-002-011||||2-004-007| scaffold,scaffolding,سقالات,سقاله,شدات معدنيه للسقالات
01 56 00|Temporary Barriers and Enclosures|الأسوار والحواجز المؤقتة|2-001-009|||||temporary fence,temporary fencing,hoarding,سياج مؤقت,سور مؤقت,اسوار مؤقته,سياج الموقع
01 58 00|Project Identification (Signboards)|لافتة المشروع|2-002-005|||||project sign,signboard,لوحه المشروع,يافطه المشروع,لافته المشروع
01 71 00|Mobilization & Site Setup|التجهيز والحشد للموقع|2-002-005|2-002-005|||| mobilization,mobilisation,demobilization,site setup,تجهيز الموقع,حشد,التجهيزات الاوليه,تجهيزات الموقع
01 71 23|Field Engineering (Surveying & Setting-out)|الأعمال المساحية والتوقيع|2-003-005|1-001-004|||1-001-004| survey,surveying,setting out,setting-out,رفع مساحي,رفع مساحى,توقيع,مساحه,ميزانيه شبكيه
01 74 00|Cleaning and Waste Management|النظافة وإزالة المخلفات|2-002-013|2-002-013|||2-002-018| cleaning,final cleaning,waste,debris,rubbish,نظافه,تنظيف,مخلفات,رتش
00 73 16|Insurance Requirements|التأمينات|3-001-003|3-001-003|||| insurance,car policy,تامين,بوليصه
02 32 00|Geotechnical Investigations|أبحاث التربة والجسات|2-003-005|||1-001-004|1-001-004| soil investigation,geotechnical,borehole,bore hole,جسات,جسه,ابحاث التربه,بحث تربه,ميكانيكا التربه
02 41 16|Structure Demolition (Concrete)|هدم وإزالة الخرسانات|2-001-001|||2-002-201|2-004-002| demolition,demolish,remove,removal,dismantle,هدم,ازاله,تكسير,فك;concrete,rc,خرسانه,خرسانات,سقف,اسقف,اعمده
02 41 19|Selective Demolition (Masonry)|إزالة الحوائط والمباني|2-001-002|||2-002-201|| demolition,demolish,remove,removal,هدم,ازاله,تكسير;wall,walls,block,brick,masonry,حوائط,حائط,مباني,طوب,بلوك
02 41 19|Selective Demolition (Finishes)|إزالة التشطيبات القديمة|2-001-003|||2-002-201|| demolition,demolish,remove,removal,strip,هدم,ازاله,تكسير,كشط,فك;tile,tiles,finishes,plaster,paint,flooring,بلاط,سيراميك,بياض,محاره,دهان,تشطيبات,رخام
02 41 00|Demolition|الهدم والإزالة|2-001-004|||2-002-201|2-004-002| demolition,demolish,هدم,ازاله المباني,تكسير ونقل,نقل المخلفات,ازاله المخلفات,ازاله
02 41 19|MEP Dismantling|فك الشبكات الكهروميكانيكية|2-001-006|||2-002-201|| dismantle,dismantling,remove,removal,فك,ازاله;mep,electrical,plumbing,كهرباء,سباكه,تكييف,كهروميكانيك
03 05 00|Concrete Materials (Cement & Sand)|مواد الخرسانة (أسمنت ورمل)|1-002-002|1-002-006|1-003-002|1-003-202|| cement,portland,opc,src,sand,building sand,اسمنت,أسمنت,رمل,رمله,رمال,جير,lime;!block,blocks,brick,بلوك,طوب,بلاط,tile,tiles,ردم,backfill,fill,مباني,بياض,plaster,بلدوره,بردوره,انترلوك,خرسانه,concrete
31 05 16|Aggregates for Earthwork|الركام (سن وزلط)|1-001-002|1-002-005||1-002-201|| aggregate,aggregates,gravel,crushed stone,base material,سن,زلط,دولوميت,كسر,بقايا محاجر,ردم مستورد
03 05 00|Concrete Admixtures & Chemicals|الإضافات الكيماوية للخرسانة|1-002-002|1-002-006||1-003-202|| admixture,admixtures,plasticizer,superplasticizer,retarder,اضافات,اضافه,ملدن,سوبر بلاستيسايزر,مؤخر شك
03 10 00|Concrete Forming — Labor|عمالة الشدات (نجارة مسلح)|1-002-006|1-002-007|1-003-101|1-003-202|| نجار,نجارين,نجار مسلح,نجارين مسلح,carpenter,carpenters,formwork carpenter
03 20 00|Concrete Reinforcing — Labor|عمالة الحدادة|1-002-002|1-002-003|1-003-101|1-003-202|| حداد,حدادين,حداد مسلح,steel fixer,steel fixers,rebar fixer
03 30 00|Concrete Placing — Labor|عمالة صب الخرسانة|1-002-002|1-002-001|1-003-002|1-003-202|| عمال صب,صباب,concrete gang,pouring crew,concrete labour,concrete labor
04 20 00|Unit Masonry — Labor|عمالة المباني|1-003-001|1-004-001|1-004-101|1-004-206|| بناء,بنا,مباني عماله,mason,masons,bricklayer,block layer
09 24 00|Plastering — Labor|عمالة البياض|1-003-002|1-004-002|1-004-101|1-004-206|| مبيض,مبيضين,محار,محارجي,plasterer,plasterers
09 30 00|Tiling — Labor|عمالة البلاط|1-003-004|1-004-004|1-004-102|1-004-206|| مبلط,مبلطين,بلاط عماله,tiler,tilers,tile layer,tile fixer
09 91 00|Painting — Labor|عمالة الدهانات|1-003-003|1-004-003|1-004-103|1-004-206|| نقاش,نقاشين,دهان عماله,painter,painters
22 00 00|Plumbing — Labor|عمالة السباكة|1-006-001|1-007-001|1-007-101|1-007-202|| سباك,سباكين,plumber,plumbers,pipe fitter
26 05 00|Electrical — Labor|عمالة الكهرباء|1-005-012|1-006-001|1-006-101|1-006-202|| كهربائي فني,فني كهرباء,كهربائيه فنيين,electrician,electricians,wireman
23 00 00|HVAC — Labor|عمالة التكييف|1-006-003|1-006-005|1-007-101|1-007-201|| فني تكييف,فنيين تكييف,hvac technician,ac technician
05 50 00|Welding & Steel Fabrication — Labor|عمالة اللحام والمشغولات|1-003-017|1-002-004||1-004-207|| لحام,لحامين,welder,welders,fabricator
01 50 00|Temporary Facilities and Controls (General Labor)|العمالة العامة والمساعدين|2-002-013|2-002-014|1-003-102||| عامل,عمال,عامل عادي,عماله عاديه,عماله عامه,مساعد,مساعدين,helper,helpers,general labor,general labour,unskilled,labourer,laborer,ملاحظ,foreman
31 23 16|Earthmoving Equipment|معدات الحفر والنقل|1-001-001|||1-002-201|2-004-002| excavator,excavators,loader,loaders,backhoe,bulldozer,dozer,grader,dump truck,tipper,jcb,bobcat,حفار,حفارات,بوكلين,لودر,لوادر,بلدوزر,جريدر,قلاب,قلابات,تريلا
31 23 23|Compaction Equipment|معدات الدمك|1-001-002|||1-002-201|2-004-002| roller,rollers,compactor,plate compactor,هراس,هراسات,مندالة,دكاك
01 54 00|Construction Aids — Cranes & Lifting|الأوناش ومعدات الرفع|2-004-001||||2-004-001| crane,cranes,tower crane,mobile crane,ونش,اوناش,كرين,رافعه,رافعات,manlift,boom lift,forklift,فورك
03 30 00|Concrete Equipment|معدات الخرسانة|1-002-002|||1-003-202|2-004-002| concrete pump,mixer truck,transit mixer,طلمبه خرسانه,مضخه خرسانه,خلاطه,خلاطه خرسانه,vibrator,هزاز
01 51 00|Temporary Utilities (Water, Power, Fuel)|المرافق المؤقتة (مياه وكهرباء ووقود)|2-002-009|2-002-010|||2-004-006| diesel,fuel,solar oil,petrol,gasoline,سولار,ديزل,بنزين,وقود,water tanker,وايت مياه,فنطاس,temporary power,كهرباء مؤقته
01 35 23|Owner Safety Requirements (PPE & HSE)|مهمات السلامة|2-002-010|2-002-012|||| ppe,safety,helmet,helmets,safety shoes,vest,vests,gloves,goggles,harness,سلامه,امن صناعي,خوذه,خوذ,حذاء سيفتي,جوانتي,قفازات,سديري,نظاره,حزام امان
01 74 00|Small Tools & Consumables|العدد والمستهلكات|2-004-003|2-002-014|||2-004-005| tools,small tools,consumables,شنيور,صاروخ,عده,عدد يدويه,مستهلكات,ادوات,اسطوانات قطع,disc,discs
03 01 00|Concrete Repair & Strengthening|تقوية وإصلاح الخرسانات|2-001-005|1-002-006||1-003-202|| strengthening,repair,retrofit,jacketing,carbon fiber,cfrp,تقويه,ترميم,اصلاح,تدعيم,قميص خرساني,الياف كربون
03 10 00|Concrete Forming and Accessories|الشدات الخشبية والمعدنية|1-002-006|1-002-007|1-003-101|1-003-202|2-004-005| formwork,shuttering,forms,شدات,شده,شداد,نجاره مسلح,نجاره خرسانه,فرم
03 15 00|Concrete Accessories (Joints & Waterstops)|إكسسوارات الخرسانة (فواصل ووتر ستوب)|1-002-006|1-004-012||1-004-203|| waterstop,water stop,water-stop,ووتر ستوب,ووترستوب,expansion joint,joint filler,فاصل تمدد,فواصل تمدد,فواصل حركه,فلين فواصل
03 20 00|Concrete Reinforcing|حديد التسليح|1-002-002|1-002-003|1-003-101|1-003-202|| rebar,reinforcement,reinforcing,reinforcing steel,steel bars,deformed bars,حديد تسليح,حديد التسليح,حداده,تسليح,شبك حديد,wire mesh,سلك رباط
03 30 00|Cast-in-Place Concrete (Plain / Blinding)|الخرسانة العادية|1-002-001|1-002-001|1-003-002|1-003-202|| concrete,خرسانه,خرسانات,pcc,p.c;plain,blinding,lean,pcc,p.c,عاديه,عادية,نظافه,فرشه خرسانه,بلوكات خرسانيه عاديه
03 30 00|Cast-in-Place Concrete (Foundations)|خرسانة الأساسات|1-002-007|1-002-001|1-003-002|1-003-202|| concrete,rc,r.c,خرسانه,خرسانات;footing,footings,foundation,foundations,raft,tie beam,tie beams,ground beam,grade beam,strap beam,pile cap,pile caps,neck column,necks,substructure,قواعد,قاعده,اساسات,اساس,لبشه,ميد,ميده,سملات,رقاب,رقاب الاعمده,هامات,حوائط ساند,retaining wall
03 30 00|Cast-in-Place Concrete|الخرسانة المسلحة المصبوبة بالموقع|1-002-002|1-002-001|1-003-002|1-003-202|| concrete,rc,r.c,rcc,reinforced concrete,خرسانه,خرسانات,مسلحه,اعمده خرسانيه,اسقف خرسانيه;!precast,سابقه الصب,demolition,هدم,ازاله
03 35 00|Concrete Finishing|تشطيب أسطح الخرسانة|1-002-002|1-002-006|1-003-002|1-003-202|| power float,helicopter,trowel finish,topping,screed,سكريد,هليكوبتر,تسويه سطح الخرسانه,ضهاره
03 39 00|Concrete Curing|معالجة الخرسانة|1-002-008|1-002-011|1-003-002||| curing,curing compound,معالجه الخرسانه,ماده معالجه,مواد معالجه,معالجه
03 38 00|Post-Tensioned Concrete|الخرسانة سابقة الإجهاد|1-002-004|1-002-003||1-003-201|| post tension,post-tension,post tensioned,post-tensioned,prestressed,pre-stressed,pt slab,سابق الاجهاد,سابقه الاجهاد,بوست تنشن
03 40 00|Precast Concrete|الخرسانة سابقة الصب|1-002-003|1-002-008||1-003-202|| precast,pre-cast,hollow core,hollowcore,سابقه الصب,سابق الصب,هولوكور,هولو كور,بريكاست
03 41 00|Precast Structural Concrete (Barriers)|حواجز خرسانية سابقة الصب|1-002-003|1-002-008||1-003-202|| new jersey,jersey barrier,concrete barrier,حواجز خرسانيه,حاجز خرساني,نيوجيرسي
03 60 00|Grouting|الجراوت|1-002-002|1-002-006||1-003-202|| grout,grouting,non shrink,non-shrink,جراوت,جروت
04 20 00|Unit Masonry|المباني (البلوك والطوب)|1-003-001|1-004-001|1-004-101|1-004-206|| block,blocks,blockwork,block work,brick,bricks,brickwork,masonry,cmu,aac,بلوك,بلوكات,طوب,طوبه,مباني,مبانى,حوائط بلوك,اسمنتي مصمت,مفرغ,ايتونج;!demolition,هدم,ازاله,precast
04 20 00|Unit Masonry — Boundary Walls|الأسوار المباني|1-004-004|1-004-001|1-004-101|1-005-203|| block,blocks,blockwork,brick,masonry,بلوك,طوب,مباني;boundary,compound wall,fence wall,سور,اسوار,السور الخارجي
04 40 00|Stone Assemblies (Stone Cladding)|تكسيات الحجر|1-003-005|1-004-005|1-004-102|1-004-206|| stone cladding,natural stone,cut stone,حجر,احجار,هاشمي,حجر طبيعي,حجر صناعي,تكسيه حجر
04 42 00|Exterior Stone Cladding (Marble/Granite)|تكسيات واجهات رخام وجرانيت|1-003-005|1-004-005|1-004-102|1-004-206|| marble,granite,رخام,جرانيت;cladding,facade,facades,wall cladding,تكسيه,تكسيات,واجهات,واجهه
05 10 00|Structural Metal Framing|المنشآت المعدنية|1-002-005|1-002-004||1-004-207|| structural steel,steel structure,steel structures,steel frame,منشات معدنيه,منشات حديديه,هيكل معدني,هياكل معدنيه,استيل ستراكشر,قطاعات حديد;!rebar,reinforcement,تسليح
05 12 00|Structural Steel Framing (Trusses)|الجمالونات|1-003-020|1-002-004||1-004-207|| truss,trusses,جمالون,جمالونات,هنجر,hangar,shed
05 50 00|Metal Fabrications|المشغولات المعدنية|1-003-017|1-002-004||1-004-207|| metal fabrication,metal works,metalwork,steel plate,embedded plate,base plate,angle,channel,مشغولات معدنيه,مشغولات حديد,حدايد,حلي,كريتال,برجوله حديد,مظلات حديد,steel grating,جريلات حديد,حديد مشغول;!drain,drains,صرف,جاليتراب
05 51 00|Metal Stairs and Ladders|السلالم المعدنية|1-003-022|1-002-004||1-004-207|| cat ladder,steel ladder,metal stair,metal stairs,steel stair,سلم بحاره,سلم معدني,سلالم معدنيه,سلم حديد
05 52 00|Metal Railings|الدرابزينات|1-003-014|1-004-014||1-004-207|| railing,railings,handrail,handrails,balustrade,guardrail,درابزين,درابزينات,ترابزين,كوبستيه,هاندريل
06 10 00|Rough Carpentry|النجارة الإنشائية|1-002-006|1-002-007|1-003-101||| rough carpentry,timber,lumber,خشب موسكي,خشب لتزانه,عروق خشب
06 40 00|Architectural Woodwork (Joinery)|النجارة المعمارية|1-003-008|1-004-008||1-004-207|| joinery,woodwork,wooden,wood,cladding wood,خشب,خشبي,خشبيه,نجاره,ابلكاش,mdf,ام دي اف,قشره,دواليب,خزائن;!formwork,شدات,door,doors,باب,ابواب
06 64 00|Plastic Paneling (WPC Cladding)|تكسيات WPC|1-003-007|1-004-007||1-004-206|| wpc,wood plastic,composite decking,دبليو بي سي
07 11 00|Dampproofing (Below-grade Waterproofing)|عزل الأساسات والأعمال تحت الأرض|1-002-006|1-004-012||1-004-203|| waterproof,waterproofing,dampproof,damp proof,tanking,bitumen,bituminous,membrane,عزل مائي,عزل رطوبه,عزل الرطوبه,بيتومين,بيتومينى,بيتوميني,سيكا,ممبرين,عزل;foundation,footing,footings,raft,substructure,basement,retaining,tank,اساسات,قواعد,لبشه,ميد,سملات,بدروم,حوائط ساند,خزان,خزانات,تحت منسوب,اسفل
07 13 00|Sheet Waterproofing|العزل المائي|1-002-006|1-004-012||1-004-203|| waterproof,waterproofing,membrane,bitumen,bituminous,عزل مائي,عزل رطوبه,بيتومين,ممبرين,رولات عزل,لفائف;!roof,roofs,سطح,اسطح,السطح
07 14 00|Fluid-Applied Waterproofing (Wet Areas)|عزل الحمامات والمطابخ|1-003-012|1-004-012||1-004-203|| waterproof,waterproofing,cementitious,عزل,سيكا,ادبيوند;wet area,wet areas,toilet,toilets,bathroom,kitchen,حمام,حمامات,مطبخ,مطابخ,دورات مياه
07 20 00|Thermal Protection (Insulation)|العزل الحراري|1-003-012|1-004-012||1-004-203|| thermal insulation,insulation board,polystyrene,extruded polystyrene,xps,eps,polyurethane foam,rock wool,rockwool,glass wool,عزل حراري,عزل حرارى,بوليسترين,فوم,صوف صخري,صوف زجاجي,بولي يوريثان
07 50 00|Membrane Roofing (Roof Insulation)|عزل الأسطح|1-003-012|1-004-012||1-004-203|| waterproof,waterproofing,insulation,membrane,bitumen,عزل,ممبرين,بيتومين,foam,فوم;roof,roofs,roofing,سطح,اسطح,السطح,الاسطح,روف
07 41 00|Roof Panels (Sandwich Panels & Sheeting)|ساندوتش بانل والتغطيات المعدنية|1-003-021|1-004-012||1-004-207|| sandwich panel,sandwich panels,roof sheeting,metal sheet,corrugated,ساندوتش,ساندويتش,صاج مموج,الواح صاج,تغطيه صاج
07 42 43|Composite Wall Panels (ACP Cladding)|تكسيات ألومنيوم كلادينج|1-003-025|1-004-015||1-004-205|| acp,aluminium composite,aluminum composite,composite panel,cladding panel,alucobond,كلادينج,الوكوبوند,الكوبوند
07 46 00|GRC / GRP Cladding|تكسيات GRC|1-003-011|1-004-011||1-004-205|| grc,grp,gfrc,glass fiber reinforced,glass fibre reinforced,جي ار سي
07 84 00|Firestopping|سد فتحات الحريق|1-006-004|1-004-012||1-007-202|| firestop,fire stop,fire stopping,fire sealant,firestopping
07 92 00|Joint Sealants|مواد مانعة للتسرب والفواصل|1-003-012|1-004-012||1-004-203|| sealant,sealants,silicone,mastic,polysulphide,polysulfide,سيلانت,سليكون,سيليكون,ماستيك,بولي سلفيد
08 11 13|Hollow Metal Doors and Frames|الأبواب المعدنية|1-003-009|1-004-009||1-004-207|| door,doors,باب,ابواب,بوابه;steel,metal,iron,fire rated,fire-rated,حديد,معدني,معدنيه,صاج,حريق,مقاوم للحريق
08 11 16|Aluminum Doors and Frames|أبواب الألومنيوم|1-003-010|1-004-010||1-004-202|| door,doors,باب,ابواب;aluminium,aluminum,الومنيوم,ألومنيوم,الوميتال
08 14 00|Wood Doors|الأبواب الخشبية|1-003-008|1-004-008||1-004-207|| door,doors,باب,ابواب,ضلفه;wood,wooden,timber,hdf,mdf,خشب,خشبي,خشبيه,حلق,قشره,ابلكاش;!steel,metal,حديد,معدني,الومنيوم
08 14 00|Doors (General)|الأبواب|1-003-008|1-004-008||1-004-207|| door,doors,doorset,باب,ابواب,ضلفه
08 33 00|Coiling Doors and Shutters|الأبواب الصاج المتحركة (شتر)|1-003-009|1-004-009||1-004-207|| rolling shutter,roller shutter,shutter,shutters,coiling door,rolling door,شتر,شيش حصيره,رولينج,رول اب,سكسونيا
08 44 00|Curtain Wall and Glazed Assemblies|الحوائط الستارية|1-003-016|1-004-010||1-004-202|| curtain wall,curtain walls,structural glazing,glass facade,حوائط ستاريه,حائط ستاري,كيرتن وول,واجهات زجاجيه,واجهه زجاجيه
08 41 00|Entrances and Storefronts|الواجهات الزجاجية والمداخل|1-003-010|1-004-010||1-004-202|| storefront,shop front,entrance,skylight,سكاي لايت,منور زجاج
08 51 13|Aluminum Windows|شبابيك الألومنيوم|1-003-010|1-004-010||1-004-202|| window,windows,شباك,شبابيك,نوافذ,نافذه,شيش,ستاره الومنيوم,سلايد,مفصلي,uPVC;!door,باب
08 51 13|Aluminum Windows|شبابيك الألومنيوم|1-003-010|1-004-010||1-004-202|| window,windows,شباك,شبابيك,نوافذ,نافذه;aluminium,aluminum,الومنيوم,ألومنيوم,sliding,glazed,سلايد,upvc,زجاج
08 51 13|Aluminum Works|أعمال الألومنيوم|1-003-010|1-004-010||1-004-202|| aluminium,aluminum,الومنيوم,ألومنيوم,الوميتال,قطاعات الومنيوم
08 71 00|Door Hardware|إكسسوارات الأبواب|1-003-008|1-004-008||1-004-207|| ironmongery,door hardware,hardware,lock,locks,hinge,hinges,handle,handles,closer,door closer,كالون,كوالين,مقابض,مفصلات,مفصله,اكره,اوكر,دوربات
08 80 00|Glazing|الزجاج|1-003-010|1-004-010||1-004-202|| glass,glazing,tempered,laminated,double glazed,securit,زجاج,سيكوريت,سكوريت,مزدوج,دبل جلاس;!fiber,fibre,wool,صوف,window,windows,شباك,شبابيك,نوافذ,curtain wall,door,doors,باب,ابواب
08 83 00|Mirrors|المرايات|1-003-013|1-004-010||1-004-202|| mirror,mirrors,مرايه,مرايات,مرآه
09 24 00|Cement Plastering|البياض (المحارة)|1-003-002|1-004-002|1-004-101|1-004-206|| plaster,plastering,render,rendering,cement plaster,بياض,محاره,لياسه,لياسة,طرطشه,بؤج,بؤوج,بطانه;!gypsum,جبس,جبسي
09 26 00|Gypsum Plastering|البياض الجبسي|1-003-002|1-004-002|1-004-101|1-004-206|| gypsum plaster,جبس بياض,بياض جبس,محاره جبس,جبس اسبراي,spray plaster
09 27 00|Plaster Fabrications (Cornices & Decorations)|الكرانيش والديكورات الجبسية|1-003-024|1-004-006||1-004-201|| cornice,cornices,moulding,molding,gypsum decoration,كرانيش,كرانيشه,كورنيش,ديكورات جبس,جبس بلدي,بانوهات,زخارف جبس,وزرات جبس
09 29 00|Gypsum Board (Partitions)|قواطع الجبسم بورد|1-003-027|1-004-006||1-004-201|| gypsum board,gypsumboard,drywall,plasterboard,dry wall,جبسن بورد,جبس بورد,جيبسون بورد,جبسم بورد,جبسوم بورد;partition,partitions,wall,walls,قواطع,قاطوع,حوائط,حائط
09 29 00|Gypsum Board|الجبسم بورد|1-003-006|1-004-006||1-004-201|| gypsum board,gypsumboard,drywall,plasterboard,جبسن بورد,جبس بورد,جيبسون بورد,جبسم بورد,جبسوم بورد,بورد
09 51 00|Acoustical Ceilings (Suspended Ceilings)|الأسقف المعلقة|1-003-006|1-004-006||1-004-201|| false ceiling,suspended ceiling,ceiling tiles,ceiling tile,acoustic ceiling,metal ceiling,armstrong,اسقف معلقه,سقف معلق,اسقف ساقطه,سقف ساقط,بلاطات اسقف,اسقف معدنيه,ارمسترونج
09 30 00|Tiling (Ceramic & Porcelain)|السيراميك والبورسلين|1-003-004|1-004-004|1-004-102|1-004-206|| ceramic,porcelain,tile,tiles,tiling,mosaic,بلاط,سيراميك,بورسلين,بورسلان,موزايكو,موزاييك,وزره,وزرات,skirting,نعلات;!ceiling,سقف,اسقف,roof
09 63 40|Stone Flooring (Marble & Granite)|الرخام والجرانيت|1-003-005|1-004-005|1-004-102|1-004-206|| marble,granite,quartz,رخام,جرانيت,كوارتز,مرمر,ترابيزه رخام,سلالم رخام,دواسه,قايم ونايم
09 66 00|Terrazzo Flooring|بلاط الترازو|1-003-004|1-004-004|1-004-102|1-004-206|| terrazzo,ترازو,تيرازو
09 64 00|Wood Flooring|الأرضيات الخشبية (باركيه)|1-003-019|1-004-004||1-004-206|| parquet,wood flooring,laminate flooring,hdf flooring,باركيه,باركية,ارضيات خشب,ارضيه خشب
09 65 00|Resilient Flooring (Vinyl)|أرضيات الفينيل|1-003-019|1-004-004||1-004-206|| vinyl,pvc flooring,rubber flooring,linoleum,فينيل,فنيل,مطاط
09 67 00|Fluid-Applied Flooring (Epoxy)|أرضيات الإيبوكسي|1-003-019|1-004-003||1-004-206|| epoxy,polyurethane flooring,self leveling,self-levelling,ايبوكسي,ابوكسي,ايبوكسى,سيلف ليفلنج
09 68 00|Carpeting|الموكيت|1-003-019|1-004-004||1-004-206|| carpet,carpets,carpeting,موكيت,سجاد
09 69 00|Access Flooring|الأرضيات المرفوعة|1-003-019|1-004-004||1-004-206|| raised floor,access floor,access flooring,ارضيات مرفوعه
09 72 00|Wall Coverings|ورق الحائط|1-003-003|1-004-003|1-004-103|1-004-206|| wallpaper,wall paper,wall covering,ورق حائط,ورق حوائط,ورق جدران
09 81 00|Acoustic Insulation|العزل الصوتي|1-003-023|1-004-012||1-004-203|| acoustic,acoustical,sound insulation,soundproof,عزل صوتي,عزل صوت
09 91 00|Painting|الدهانات|1-003-003|1-004-003|1-004-103|1-004-206|| paint,painting,paints,emulsion,enamel,primer,putty,varnish,coating,coat,دهان,دهانات,بلاستيك,معجون,سيلر,ورنيش,زيت,جوتن,سايبس,سكوتش,طلاء,بوية,بويه,لاكيه,دوكو,رولاتو,جرافياتو,ستوكو;!road marking,روف,epoxy floor
09 96 00|High-Performance Coatings|الدهانات الخاصة|1-003-003|1-004-003|1-004-103|1-004-206|| anti corrosion,anticorrosion,intumescent,fire retardant paint,epoxy paint,epoxy coating,دهان ايبوكسي,دهان مقاوم,مقاوم للحريق دهان
10 14 00|Signage|اللوحات الإرشادية|1-003-026|2-002-012||1-004-207|| signage,signs,sign,wayfinding,لوحات ارشاديه,لوحه ارشاديه,يافطه,يافطات,لافتات,لافته,ارقام الفلل,حروف بارزه
10 21 00|Toilet Compartments and Cubicles|كبائن الحمامات|1-003-027|1-004-006||1-004-201|| cubicle,cubicles,toilet partition,toilet partitions,hpl,كبائن,كابينه حمام,قواطع حمامات,اتش بي ال
10 28 00|Toilet and Bath Accessories|إكسسوارات الحمامات|1-003-013|1-004-013||1-007-202|| toilet accessories,bath accessories,tissue holder,paper holder,towel rail,towel bar,soap dispenser,grab bar,hand dryer,اكسسوارات حمام,اكسسوارات الحمامات,حامل مناديل,حامل فوطه,شماعه,صابونه,مجفف ايدي
10 44 00|Fire Protection Specialties (Extinguishers)|طفايات الحريق|1-006-004|1-006-006||1-007-202|| extinguisher,extinguishers,fire extinguisher,fire blanket,طفايه,طفايات,بطانيه حريق
11 31 00|Residential Appliances|الأجهزة المنزلية|1-006-010|1-004-016|||| appliance,appliances,oven,cooker,hob,hood,dishwasher,refrigerator,fridge,washing machine,بوتجاز,فرن,شفاط مطبخ,ثلاجه,غساله
12 20 00|Window Treatments|الستائر والتظليل|1-003-018|1-004-016||1-004-207|| curtain,curtains,blind,blinds,roller blind,shading,ستائر,ستاره,ستارة,بلاك اوت,رول,تظليل;!curtain wall,حوائط ستاريه,ستاره الومنيوم
12 35 30|Residential Casework (Kitchen Cabinets)|المطابخ والدواليب|1-003-008|1-004-008||1-004-207|| kitchen cabinet,kitchen cabinets,kitchen unit,kitchen units,wardrobe,wardrobes,cabinets,closet,مطبخ,مطابخ,دولاب,دواليب,وحدات مطبخ,خزائن ملابس;!sink,حوض,خلاط,waterproof,عزل,tile,سيراميك,بلاط
12 36 00|Countertops & Vanity Units|وحدات المغاسل والكاونترات|1-003-028|1-004-008||1-004-207|| vanity,vanity unit,vanity cabinet,countertop,counter top,وحده مغسله,وحدات مغاسل,دولاب مغسله,كاونتر,كونتر
12 50 00|Furniture|الأثاث|1-003-028|1-004-016||1-004-207|| furniture,furnishing,sofa,chair,chairs,desk,desks,table,tables,bed,beds,اثاث,موبيليا,كرسي,كراسي,ترابيزه,ترابيزات,سرير,كنب,انتريه
13 11 00|Swimming Pools|حمامات السباحة|1-006-009|1-007-001||1-007-202|| swimming pool,pool,pools,حمام سباحه,حمامات سباحه,بيسين,مسبح,جاكوزي,jacuzzi
13 12 00|Fountains and Water Features|النوافير|1-006-009|1-007-001||1-007-202|| fountain,fountains,water feature,نافوره,نوافير,شلال
14 21 00|Electric Traction Elevators|المصاعد|1-003-015|1-006-008||1-004-204|| elevator,elevators,lift,lifts,passenger lift,مصعد,مصاعد,اسانسير,اسانسيرات;!lifting,رفع
14 31 00|Escalators|السلالم الكهربائية|1-003-015|1-006-008||1-004-204|| escalator,escalators,travelator,سلم كهربائي,سلالم كهربائيه,سلالم متحركه
21 13 00|Fire-Suppression Sprinkler Systems|رشاشات الحريق|1-006-004|1-006-006|1-007-101|1-007-202|| sprinkler,sprinklers,رشاش,رشاشات,سبرينكلر;!irrigation,ري,زراعه
21 12 00|Fire-Suppression Standpipes (Hose Reels & Cabinets)|صناديق وخراطيم الحريق|1-006-004|1-006-006|1-007-101|1-007-202|| hose reel,hose reels,fire hose,fire cabinet,landing valve,standpipe,خرطوم حريق,صندوق حريق,صناديق حريق,بكره خرطوم,حنفيه حريق,fire hydrant,هيدرانت
21 30 00|Fire Pumps|طلمبات الحريق|1-006-004|1-006-006||1-007-202|| fire pump,fire pumps,jockey pump,jockey,طلمبه حريق,طلمبات حريق,مضخه حريق,مضخات حريق,طلمبات الحريق
21 00 00|Fire Suppression (General)|مكافحة الحريق|1-006-004|1-006-006|1-007-101|1-007-202|| fire fighting,firefighting,fire suppression,fire protection,مكافحه حريق,مكافحه الحريق,اطفاء,اطفاء الحريق,fm200,fm-200,co2 system;!alarm,انذار,detector,كاشف,extinguisher,طفايه
22 05 23|General-Duty Valves for Plumbing|محابس السباكة|1-006-002|1-007-002|1-007-101|1-007-202|| valve,valves,gate valve,ball valve,check valve,محبس,محابس,بلف,بلوف;!fire,حريق
22 11 00|Facility Water Distribution|تغذية المياه|1-006-002|1-007-001|1-007-101|1-007-202|| water supply,water distribution,cold water,hot water,potable water,domestic water,ppr,pp-r,cpvc,تغذيه,تغذيه المياه,مياه باردة,مياه بارده,مياه ساخنه,مواسير تغذيه,بي بي ار,حنفيه,حنفيات,نقطه مياه;!external,خارجي,خارجيه,network,شبكه
22 11 23|Domestic Water Pumps|طلمبات المياه|1-006-005|1-006-007||1-007-202|| pump,pumps,booster,booster pump,submersible,طلمبه,طلمبات,مضخه,مضخات,بوستر;!fire,حريق,sump,غاطس صرف
22 12 00|Facility Potable-Water Storage Tanks|خزانات المياه|1-006-005|1-006-007||1-007-202|| water tank,water tanks,storage tank,grp tank,tank,tanks,خزان,خزانات,تنك,خزان علوي,خزان ارضي;!septic,بياره,fuel,سولار,ديزل
22 13 00|Facility Sanitary Sewerage|الصرف الصحي الداخلي|1-006-001|1-007-001|1-007-101|1-007-202|| drainage,soil pipe,waste pipe,vent pipe,floor drain,floor trap,gully trap,clean out,cleanout,upvc,صرف صحي,صرف,مواسير صرف,جاليتراب,جاليتراب,بيبه,بيبات,سيفون ارضي,كلين اوت,تهويه صرف,عمود صرف,عمود هوايه;!external,خارجي,خارجيه,network,شبكه,storm,امطار
22 14 00|Facility Storm Drainage|صرف مياه الأمطار الداخلي|1-006-001|1-007-001|1-007-101|1-007-202|| rain water,rainwater,roof drain,roof drains,downpipe,down pipe,storm drainage,مياه امطار,مياه الامطار,صرف الامطار,مزراب,مزاريب,جرجوري;!external,خارجي,خارجيه,network,شبكه
22 33 00|Electric Domestic Water Heaters|السخانات|1-006-002|1-007-002||1-007-202|| water heater,water heaters,heater,boiler,calorifier,سخان,سخانات,بويلر,سخان شمسي,solar heater
22 40 00|Plumbing Fixtures|الأجهزة الصحية|1-003-013|1-004-013|1-007-101|1-007-202|| sanitary ware,sanitaryware,sanitary fixtures,plumbing fixtures,water closet,wc,toilet,toilets,closet,basin,washbasin,wash basin,lavatory,sink,sinks,urinal,bidet,bathtub,bath tub,shower,mixer,mixers,faucet,tap,taps,اجهزه صحيه,اطقم صحيه,طقم حمام,مرحاض,قاعده افرنجي,تواليت,حوض,احواض,مغسله,مغاسل,مبولة,مبوله,بانيو,دش,خلاط,خلاطات,شطاف,حوض مطبخ,بيديه;!accessories,اكسسوارات,partition,cubicle,كبائن,vanity unit,دولاب
22 60 00|Medical Gas Systems|الغازات الطبية|1-006-006|1-007-001||1-007-202|| medical gas,medical gases,oxygen,vacuum,غازات طبيه,اكسجين,شفط طبي
23 81 00|Unitary HVAC Equipment (Split / Package / VRF)|وحدات التكييف|1-006-003|1-006-005|1-007-101|1-007-201|| air conditioning,air conditioner,air-conditioning,split,split unit,package unit,packaged unit,vrf,vrv,ducted split,hvac,تكييف,تكييفات,مكيف,مكيفات,سبليت,سبلت,باكدج,كونسيلد,دكت سبليت,وحده تكييف,وحدات تكييف;!electrical connection,تغذيه كهرباء
23 64 00|Packaged Water Chillers|التشيلرات|1-006-003|1-006-005||1-007-201|| chiller,chillers,تشيلر,تشيلرات,cooling tower,برج تبريد
23 73 00|Air Handling Units & Fan Coils|وحدات مناولة الهواء والفان كويل|1-006-003|1-006-005||1-007-201|| ahu,air handling,air handling unit,fcu,fan coil,fan coils,فان كويل,وحده مناوله هواء,مناوله هواء
23 31 00|HVAC Ducts and Casings|مجاري الهواء|1-006-003|1-007-003|1-007-101|1-007-201|| duct,ducts,ductwork,duct work,دكت,دكتات,مجاري هواء,مجاري الهواء,مجرى هواء,صاج مجلفن للدكت
23 37 00|Air Outlets and Inlets (Grilles & Diffusers)|فتحات التكييف|1-006-003|1-007-003||1-007-201|| grille,grilles,diffuser,diffusers,louver,louvers,louvre,register,جريل,جريلات,دفيوزر,دفيوزرات,فتحات تكييف,لوفر
23 34 00|HVAC Fans (Exhaust & Ventilation)|مراوح الشفط والتهوية|1-006-003|1-007-003|1-007-101|1-007-201|| exhaust fan,extract fan,ventilation fan,fan,fans,ventilation,شفاط,شفاطات,مروحه شفط,مراوح شفط,مراوح,تهويه;!fan coil,فان كويل,kitchen hood,شفاط مطبخ
23 23 00|Refrigerant Piping|مواسير الفريون النحاس|1-006-003|1-007-001|1-007-101|1-007-201|| refrigerant,copper pipe,copper pipes,copper piping,نحاس,مواسير نحاس,فريون
23 07 00|HVAC Insulation|عزل مجاري التكييف|1-006-003|1-007-003||1-007-201|| insulation,عزل,armaflex,ارمافلكس;duct,ducts,pipe,pipes,hvac,دكت,مواسير,تكييف
25 00 00|Integrated Automation (BMS / Smart Home)|التحكم الآلي والمباني الذكية|1-005-011|1-006-009||1-006-202|| bms,building management,smart home,home automation,knx,scada,سمارت هوم,منزل ذكي,مباني ذكيه,تحكم ذكي,تحكم الي
26 05 26|Grounding and Bonding|التأريض|1-005-006|1-006-001|1-006-101|1-006-202|| earthing,grounding,earth pit,earth rod,ground rod,تاريض,ارضي,ارضى,حفره ارضي,بير ارضي,اسلاك ارضي
26 41 00|Lightning Protection|الحماية من الصواعق|1-005-006|1-006-001|1-006-101|1-006-202|| lightning,lightning protection,air terminal,صواعق,صاعقه,مانعه صواعق
26 05 19|Low-Voltage Power Conductors and Cables|الكابلات والأسلاك|1-005-005|1-006-001|1-006-101|1-006-202|| cable,cables,wire,wires,wiring,conductor,xlpe,pvc cable,armoured,armored,كابل,كابلات,سلك,اسلاك,سلوك,موصلات,تسليك;!tray,trays,ladder,حامل,حوامل,low current,تيار خفيف,data,cat6,tv,telephone
26 05 33|Raceways and Boxes (Conduits & Cable Trays)|المواسير ومسارات الكابلات|1-005-005|1-006-002|1-006-101|1-006-202|| conduit,conduits,cable tray,cable trays,cable ladder,trunking,raceway,back box,pull box,مواسير كهرباء,ماسوره كهرباء,خراطيم كهرباء,حوامل كابلات,حامل كابلات,مسارات كابلات,تراي,علب كهرباء,بوكس;!low current,تيار خفيف
26 24 16|Panelboards (Distribution Boards)|لوحات التوزيع|1-005-003|1-006-003|1-006-101|1-006-202|| panel,panels,panelboard,distribution board,db,mdb,smdb,sdb,switchboard,switchgear,ats,لوحه,لوحات,لوحه توزيع,لوحات توزيع,لوحه رئيسيه,لوحه فرعيه,طبلون,قاطع,قواطع كهرباء,breaker,breakers,mccb,mcb;!signage,ارشاديه,sign,اسم المشروع,solar,شمسي,acp,sandwich,ساندوتش
26 27 26|Wiring Devices (Sockets & Switches)|البرايز والمفاتيح|1-005-002|1-006-004|1-006-101|1-006-202|| socket,sockets,socket outlet,outlet,outlets,power point,switch,switches,isolator,برايز,بريزه,بريزة,مخرج قوي,مخارج قوي,مفتاح,مفاتيح,نقطه قوي,نقاط قوي,نقطه بريزه,بواط
26 27 26|Wiring Devices (Power Points)|نقاط القوى|1-005-002|1-006-004|1-006-101|1-006-202|| point,points,نقطه,نقاط;power,socket,قوي,بريزه,برايز,مخرج
26 51 00|Interior Lighting|الإنارة الداخلية|1-005-001|1-006-004|1-006-101|1-006-202|| lighting,light,lights,luminaire,luminaires,light fixture,downlight,spot light,spotlight,led,chandelier,emergency light,exit sign,اناره,انارة,كشاف,كشافات,سبوت,سبوتات,لمبه,لمبات,نجفه,نجف,ابليك,وحده اناره,وحدات اناره,طوارئ,نيون,بانل ليد,شريط ليد;!external,خارجي,خارجيه,street,garden,حدائق,شوارع,اعمده اناره,bollard
26 51 00|Lighting Points|نقاط الإنارة|1-005-001|1-006-004|1-006-101|1-006-202|| point,points,نقطه,نقاط;lighting,light,اناره,انارة
26 56 00|Exterior Lighting|الإنارة الخارجية|1-005-010|1-006-004|1-006-101|1-006-202|| lighting,light,lights,luminaire,pole,poles,bollard,اناره,انارة,كشاف,كشافات,اعمده,عمود,بولارد;external,exterior,outdoor,street,garden,landscape,site,خارجي,خارجيه,شوارع,حدائق,حديقه,طرق,موقع عام,اعمده اناره,عمود اناره
26 12 00|Medium-Voltage Transformers|المحولات|1-005-007|1-006-003||1-006-202|| transformer,transformers,substation,rmu,ring main unit,kiosk,محول,محولات,محطه محولات,غرفه محولات,كشك,mv,medium voltage,جهد متوسط
26 32 13|Engine Generators|المولدات|1-005-007|1-006-003||1-006-202|| generator,generators,genset,diesel generator,مولد,مولدات,ماكينه توليد
26 31 00|Photovoltaic Collectors (Solar)|الطاقة الشمسية|1-005-012|1-006-003||1-006-202|| solar,photovoltaic,pv panel,pv panels,طاقه شمسيه,الواح شمسيه,خلايا شمسيه;!heater,سخان
26 05 00|Common Work Results for Electrical|أعمال كهربائية عامة|1-005-012|1-006-001|1-006-101|1-006-202|| electrical,electric,كهرباء,كهربائي,كهربائيه,كهربا;!heater,heaters,سخان,سخانات,water,مياه
22 00 00|Plumbing (General)|الأعمال الصحية|1-006-001|1-007-001|1-007-101|1-007-202|| plumbing,sanitary works,plumbing works,اعمال صحيه,الاعمال الصحيه,صحي,صحيه,سباكه,السباكه
09 00 00|Finishes (General)|التشطيبات|1-003-002|1-004-002|1-004-101|1-004-206|| finishes,finishing works,تشطيبات,التشطيبات,اعمال التشطيب
26 05 00|HVAC Electrical Connections|تغذية كهرباء التكييف|1-005-009|1-006-001|1-006-101|1-006-202|| electrical connection,power supply,feeder,تغذيه كهرباء,تغذيه كهربائيه,توصيل كهرباء;hvac,ac,air condition,تكييف,مكيف,مكيفات,سبليت,معدات
27 05 28|Communications Pathways (Light-Current Infrastructure)|البنية التحتية للتيار الخفيف|1-005-008|1-006-009|1-006-101|1-006-201|| conduit,conduits,pipe,pipes,box,boxes,manhole,manholes,infrastructure,مواسير,خراطيم,علب,غرف تفتيش,بنيه تحتيه,تأسيس,تاسيس;low current,light current,elv,telecom,data,تيار خفيف,اتصالات,تليفون,داتا
27 10 00|Structured Cabling (Data & Telephone)|الشبكات والتليفونات|1-005-004|1-006-009|1-006-101|1-006-201|| data,network,structured cabling,cat6,cat 6,cat6a,fiber optic,fibre optic,telephone,tel,rj45,access point,wifi,شبكه بيانات,شبكات,داتا,تليفون,تليفونات,هاتف,فايبر,نقطه داتا,نقطه تليفون,راوتر,سويتش شبكه
27 41 00|Audio-Video Systems (TV / Satellite / Intercom)|التلفزيون والدش والإنتركم|1-005-004|1-006-009|1-006-101|1-006-201|| tv,television,satellite,matv,iptv,intercom,video intercom,sound system,public address,تلفزيون,تليفزيون,دش,ستلايت,انتركم,انتر كم,اذاعه داخليه,صوتيات,سماعات
28 13 00|Access Control|التحكم في الدخول|1-005-004|1-006-009||1-006-201|| access control,card reader,turnstile,barrier gate,بوابه الكترونيه,تحكم في الدخول,كارت,بصمه
28 23 00|Video Surveillance (CCTV)|كاميرات المراقبة|1-005-004|1-006-009|1-006-101|1-006-201|| cctv,camera,cameras,surveillance,nvr,dvr,كاميرا,كاميرات,مراقبه,كاميرات مراقبه
28 46 00|Fire Detection and Alarm|إنذار الحريق|1-005-004|1-006-009|1-006-101|1-006-201|| fire alarm,smoke detector,heat detector,detector,detectors,manual call point,break glass,sounder,facp,انذار حريق,انذار الحريق,كاشف دخان,كاشف حراره,كواشف,كاشف,لوحه انذار,جرس انذار,اجراس,مضيئه انذار
28 00 00|Low Current Systems (General)|أنظمة التيار الخفيف|1-005-004|1-006-009|1-006-101|1-006-201|| low current,light current,elv,weak current,تيار خفيف,التيار الخفيف
31 10 00|Site Clearing|تطهير الموقع|2-001-004|||1-002-201|2-004-002| site clearing,clearing,grubbing,clear site,تطهير,تطهير الموقع,ازاله الحشائش,نظافه الموقع قبل
31 22 00|Grading (Leveling)|التسوية|1-001-003|1-002-005||1-002-201|2-004-002| grading,leveling,levelling,cut and fill,cut & fill,تسويه,تسوية,تسويات,تسويه الموقع,مناسيب
31 23 16|Excavation|الحفر|1-001-001|||1-002-201|2-004-002| excavation,excavate,excavating,digging,trench,trenches,حفر,الحفر,حفريات,خنادق,تكسير صخر,rock excavation;!backfill,ردم
31 23 19|Dewatering|نزح المياه|1-001-001|||1-002-202|2-004-003| dewatering,de-watering,well point,نزح,نزح المياه,سحب المياه الجوفيه,ويل بوينت
31 23 23|Fill (Backfilling & Replacement)|الردم والإحلال|1-001-002|1-002-005||1-002-201|2-004-002| backfill,backfilling,fill,filling,compaction,compacted,replacement,sub-base,selected material,ردم,الردم,احلال,إحلال,دمك,دك,رمله,رمال ردم,زلط,سن,بيس كورس,base course;!asphalt,اسفلت,paving,interlock,انترلوك,road,طرق
31 31 16|Termite Control|مكافحة النمل الأبيض|1-001-003|||1-002-203|| termite,anti-termite,anti termite,pest control,النمل الابيض,نمل ابيض,مكافحه النمل,مبيد
31 32 00|Soil Stabilization (Soil Treatment)|معالجة التربة|1-001-003|1-002-005||1-002-203|| soil treatment,soil stabilization,soil improvement,stabilization,معالجه التربه,تحسين التربه,تثبيت التربه,معالجه تربه
31 50 00|Excavation Support and Protection|سند جوانب الحفر|2-001-007|||1-002-202|| shoring,sheet pile,sheet piles,sheet piling,secant,diaphragm wall,سند جوانب,سند الجوانب,ستاره لوحيه,شيت بايل,حوائط لوحيه
31 62 00|Driven / Bored Piles|الخوازيق|1-002-007|1-002-001||1-003-202|| pile,piles,piling,bored pile,bored piles,خازوق,خوازيق;!sheet,لوحيه,pile cap,هامات
32 11 23|Aggregate Base Courses|طبقات الأساس (البيس كورس)|1-004-003|1-002-005||1-005-001|2-004-002| base course,sub base,subbase,sub-base,aggregate base,بيس كورس,بيسكورس,سب بيس,طبقه اساس;road,roads,pavement,paving,asphalt,طرق,طريق,رصف,اسفلت,ممرات,شوارع
32 12 16|Asphalt Paving|الأسفلت|1-004-003|1-002-005||1-005-001|2-004-002| asphalt,asphaltic,bituminous paving,wearing course,binder course,prime coat,tack coat,اسفلت,أسفلت,رصف اسفلتي,طبقه رابطه,طبقه سطحيه,ام سي,mc
32 13 13|Concrete Paving|الرصف الخرساني|1-004-003|1-002-001||1-005-001|| concrete paving,concrete pavement,rigid pavement,رصف خرساني,بلاطات رصف خرساني
32 14 00|Unit Paving (Interlock)|الإنترلوك والبلاطات|1-004-002|1-002-010||1-005-001|| interlock,interlocking,paving block,paving blocks,paver,pavers,انترلوك,إنترلوك,بلاط انترلوك,بلاط ارصفه,بلاط رصف,بلاط موزايكو خارجي,كنستون
32 16 00|Curbs and Gutters|البردورات|1-004-002|1-002-009||1-005-001|| curb,curbs,kerb,kerbs,kerbstone,curbstone,gutter,بردوره,بردورات,بلدوره,بلدورات,شرديه
32 17 23|Pavement Markings|دهانات الطرق|1-004-003|1-004-003||1-005-001|| road marking,road markings,pavement marking,thermoplastic,line marking,دهانات طرق,دهان طرق,خطوط طرق,ترموبلاستيك,عيون قطط
32 31 00|Fences and Gates|الأسوار والبوابات|1-004-004|1-004-014||1-005-203|| fence,fences,fencing,gate,gates,chain link,boundary wall,سور,اسوار,سياج,بوابه,بوابات,شبك سور,سلك شائك;!temporary,مؤقت,مؤقته
32 32 00|Retaining Walls|الحوائط الساندة|1-004-004|1-002-001||1-003-202|| retaining wall,retaining walls,gabion,gabions,حائط ساند,حوائط ساند,حوائط سانده,جابيون;!concrete,خرسانه
32 33 00|Site Furnishings (Pergolas, Benches, Shades)|فرش الموقع (برجولات ومظلات ومقاعد)|1-004-001|1-004-016||1-005-001|| pergola,pergolas,bench,benches,shade,shades,canopy,trash bin,litter bin,برجوله,برجولات,مظله,مظلات,مقاعد,بنش,سلات مهملات,كانوبي
32 18 00|Athletic and Recreational Surfacing (Playgrounds)|ملاعب ومناطق ألعاب|1-004-001|1-004-016||1-005-001|| playground,play area,kids area,rubber tiles,artificial grass,synthetic turf,العاب اطفال,منطقه العاب,نجيل صناعي,ارضيات مطاط,ملعب,ملاعب
32 84 00|Planting Irrigation|شبكات الري|1-006-008|1-007-001|1-007-101|1-005-001|| irrigation,drip,sprinkler irrigation,bubbler,ري,الري,شبكه ري,شبكات ري,تنقيط,رشاشات ري,نقاط ري,بابلر
32 91 00|Planting Preparation (Topsoil)|تجهيز التربة الزراعية|1-004-001|1-002-005||1-005-001|| topsoil,top soil,planting soil,agricultural soil,soil mix,تربه زراعيه,طمي,سماد,بيتموس,تربه صالحه
32 92 00|Turf and Grasses|النجيل|1-004-001|1-004-016||1-005-001|| grass,turf,lawn,sod,نجيل,نجيله,حشيش
32 93 00|Plants (Trees & Shrubs)|الأشجار والشجيرات|1-004-001|1-004-016||1-005-001|| tree,trees,palm,palms,shrub,shrubs,plants,planting,hedge,groundcover,شجره,اشجار,نخيل,نخله,شجيرات,نباتات,زراعه اشجار,سياج نباتي,مغطيات;!grass,turf,نجيل
32 90 00|Landscaping (Soft & Hard)|اللاندسكيب|1-004-001|1-004-016||1-005-001|| landscape,landscaping,softscape,hardscape,لاندسكيب,لاند سكيب,تنسيق الموقع,تنسيق حدائق,حدائق,هاردسكيب
33 11 00|Water Utility Distribution|شبكات المياه الخارجية|1-004-006|1-007-001||1-005-202|2-004-002| water,potable,hdpe,ductile,مياه,تغذيه,مواسير مياه,اتش دي بي اي;external,network,networks,main,mains,infrastructure,خارجي,خارجيه,شبكه,شبكات,خط رئيسي,بنيه تحتيه,موقع عام
33 31 00|Sanitary Sewerage Utilities|شبكات الصرف الصحي الخارجية|1-004-006|1-007-001||1-005-202|2-004-002| sewer,sewerage,sanitary,drainage,صرف,صرف صحي,مجاري;external,network,networks,manhole,manholes,infrastructure,خارجي,خارجيه,شبكه,شبكات,غرف تفتيش,غرفه تفتيش,مطابق,بنيه تحتيه,موقع عام
33 05 61|Manholes and Inspection Chambers|غرف التفتيش|1-004-006|1-002-001||1-005-202|| manhole,manholes,inspection chamber,inspection chambers,غرفه تفتيش,غرف تفتيش,مطبق,مطابق,منهول;!electrical,كهرباء,telecom,اتصالات,تيار خفيف
33 36 00|Utility Septic Tanks|البيارات وخزانات التحليل|1-004-006|1-002-001||1-005-202|| septic,septic tank,soakaway,soak away,بياره,بيارات,خزان تحليل,ترانش
33 41 00|Storm Utility Drainage Piping|شبكات تصريف الأمطار الخارجية|1-004-005|1-007-001||1-005-202|2-004-002| storm,stormwater,storm water,rain,catch basin,catch basins,culvert,امطار,مياه امطار,سيول,مصارف,مصرف,بالوعه,بالوعات,عباره;external,network,networks,site,road,خارجي,خارجيه,شبكه,شبكات,موقع عام,طرق,catch basin,بالوعات
33 71 00|Electrical Utility Transmission and Distribution|شبكات الكهرباء الخارجية|1-004-007|1-006-001||1-005-202|2-004-002| electrical,power,cable,cables,كهرباء,كابلات,جهد;external,network,networks,underground,infrastructure,duct bank,خارجي,خارجيه,شبكه,شبكات,تحت الارض,بنيه تحتيه,موقع عام
33 81 00|Communications Utilities|شبكات الاتصالات الخارجية|1-004-007|1-006-009||1-005-202|| telecom,communication,communications,telephone,fiber,اتصالات,تليفونات,فايبر;external,network,networks,manhole,infrastructure,خارجي,خارجيه,شبكه,شبكات,بنيه تحتيه,موقع عام
`

// عناوين الـ Divisions (MasterFormat 2018)
var csiDivTitle = map[string][2]string{
	"00": {"Procurement and Contracting Requirements", "متطلبات التعاقد"},
	"01": {"General Requirements", "المتطلبات العامة"},
	"02": {"Existing Conditions", "الأوضاع القائمة"},
	"03": {"Concrete", "الخرسانة"},
	"04": {"Masonry", "المباني"},
	"05": {"Metals", "المعادن"},
	"06": {"Wood, Plastics, and Composites", "الأخشاب واللدائن"},
	"07": {"Thermal and Moisture Protection", "العزل الحراري والمائي"},
	"08": {"Openings", "الفتحات (أبواب وشبابيك)"},
	"09": {"Finishes", "التشطيبات"},
	"10": {"Specialties", "التخصصات"},
	"11": {"Equipment", "المعدات"},
	"12": {"Furnishings", "التأثيث"},
	"13": {"Special Construction", "إنشاءات خاصة"},
	"14": {"Conveying Equipment", "معدات النقل الرأسي"},
	"21": {"Fire Suppression", "مكافحة الحريق"},
	"22": {"Plumbing", "السباكة"},
	"23": {"Heating, Ventilating, and Air Conditioning (HVAC)", "التكييف والتهوية"},
	"25": {"Integrated Automation", "التحكم الآلي"},
	"26": {"Electrical", "الكهرباء"},
	"27": {"Communications", "الاتصالات"},
	"28": {"Electronic Safety and Security", "الأمن والسلامة الإلكترونية"},
	"31": {"Earthwork", "أعمال التربة"},
	"32": {"Exterior Improvements", "أعمال الموقع الخارجية"},
	"33": {"Utilities", "المرافق"},
	"34": {"Transportation", "النقل"},
	"35": {"Waterway and Marine Construction", "الأعمال البحرية"},
}

// تقسيم الـ 16 Division القديم (CSI 1995) ← Divisions الجديدة — لو عنوان الشيت/القسم فيه «Div 15» مثلاً
var oldDivMap = map[int][]string{
	1: {"01"}, 2: {"02", "31", "32", "33"}, 3: {"03"}, 4: {"04"}, 5: {"05"}, 6: {"06", "12"}, 7: {"07"}, 8: {"08"},
	9: {"09"}, 10: {"10"}, 11: {"11"}, 12: {"12"}, 13: {"13"}, 14: {"14"}, 15: {"21", "22", "23"}, 16: {"26", "27", "28", "25"},
}

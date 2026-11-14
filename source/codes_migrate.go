package main

// نقل الأكواد لقوائم PCS الجديدة (نوفمبر 2026)
// Task Codes: باكدج "Design & Supervision" (1-001) اتشال، فكل الأعمال المباشرة نزلت رقم:
//   Earth 1-002→1-001 · Skeleton 1-003→1-002 · Finishing 1-004→1-003 · Site 1-005→1-004
//   Electrical 1-006→1-005 · Mechanical 1-007→1-006 · التصميم/الإشراف → 2-003-004 · الجسات → 2-003-005
// Cost Codes: 1-002-002 (خرسانة جاهزة مسلحة) اتدمج في 1-002-001 (خرسانة جاهزة)
// الذاكرة (ipc_memory / store_memory) بتتنقل مرة واحدة، وبيتعمل منها نسخة قبل النقل.

import (
	"os"
	"path/filepath"
	"time"
)

const codesRev = 2

var migTaskMap = map[string]string{
	"1-001-001": "2-003-004", "1-001-002": "2-003-004", "1-001-003": "2-003-005", "1-002-001": "1-001-001",
	"1-002-002": "1-001-002", "1-002-003": "1-001-003", "1-003-001": "1-002-001", "1-003-002": "1-002-002",
	"1-003-003": "1-002-003", "1-003-004": "1-002-004", "1-003-005": "1-002-005", "1-003-006": "1-002-006",
	"1-003-007": "1-002-007", "1-004-001": "1-003-001", "1-004-002": "1-003-002", "1-004-003": "1-003-003",
	"1-004-004": "1-003-004", "1-004-005": "1-003-005", "1-004-006": "1-003-006", "1-004-007": "1-003-007",
	"1-004-008": "1-003-008", "1-004-009": "1-003-009", "1-004-010": "1-003-010", "1-004-011": "1-003-011",
	"1-004-012": "1-003-012", "1-004-013": "1-003-013", "1-004-014": "1-003-014", "1-004-015": "1-003-015",
	"1-004-016": "1-003-016", "1-004-017": "1-003-017", "1-004-018": "1-003-018", "1-004-019": "1-003-019",
	"1-004-020": "1-003-020", "1-004-021": "1-003-021", "1-004-022": "1-003-022", "1-004-023": "1-003-023",
	"1-004-024": "1-003-024", "1-004-025": "1-003-025", "1-004-026": "1-003-026", "1-004-027": "1-003-027",
	"1-005-001": "1-004-001", "1-005-002": "1-004-002", "1-005-003": "1-004-003", "1-006-001": "1-005-001",
	"1-006-002": "1-005-002", "1-006-003": "1-005-003", "1-006-004": "1-005-004", "1-006-005": "1-005-005",
	"1-006-006": "1-005-006", "1-006-007": "1-005-007", "1-006-008": "1-005-008", "1-006-009": "1-005-009",
	"1-006-010": "1-005-010", "1-006-011": "1-005-011", "1-006-012": "1-005-012", "1-007-001": "1-006-001",
	"1-007-002": "1-006-002", "1-007-003": "1-006-003", "1-007-004": "1-006-004", "1-007-005": "1-006-005",
	"1-007-006": "1-006-006", "1-007-007": "1-006-007", "1-007-008": "1-006-008", "1-007-009": "1-006-009",
	"1-007-010": "1-006-010",
}

var migCostMap = map[string]string{
	"1-002-002": "1-002-001",
}

func migTask(c string) string {
	if v, ok := migTaskMap[c]; ok {
		return v
	}
	return c
}

func migCost(c string) string {
	if v, ok := migCostMap[c]; ok {
		return v
	}
	return c
}

func backupFile(p string) {
	if b, err := os.ReadFile(p); err == nil {
		bak := p[:len(p)-len(filepath.Ext(p))] + ".before-codes2" + filepath.Ext(p)
		if _, err := os.Stat(bak); err != nil {
			os.WriteFile(bak, b, 0o644)
		}
	}
}

// ملفات أكواد قديمة اترفعت من البرنامج قبل كده بتغلب على القوائم الجديدة اللي جوه البرنامج — بنركنها
func migrateCodeFiles() {
	if getSetting("codesV2") != "" {
		return
	}
	tp := filepath.Join(appDir(), "Task_Codes.xlsx")
	if b, err := os.ReadFile(tp); err == nil {
		if cs, err := readCodeSheet(b); err == nil && codesHave(cs, "1-007-001") {
			os.Rename(tp, filepath.Join(appDir(), "Task_Codes.old.xlsx"))
		}
	}
	cp := filepath.Join(appDir(), "Cost_Codes.xlsx")
	if b, err := os.ReadFile(cp); err == nil {
		if cs, err := readCodeSheet(b); err == nil && !codesHave(cs, "1-002-012") {
			os.Rename(cp, filepath.Join(appDir(), "Cost_Codes.old.xlsx"))
		}
	}
	setSetting("codesV2", time.Now().Format("2006-01-02 15:04"))
}

func codesHave(cs []*Code, code string) bool {
	for _, c := range cs {
		if c.Code == code {
			return true
		}
	}
	return false
}

// ذاكرة المستخلصات (الـ lock ماسكه loadMem)
func migrateIPCMemLocked() bool {
	if mem.CodesRev >= codesRev {
		return false
	}
	backupFile(memPath())
	for k, v := range mem.Task {
		mem.Task[k] = migTask(v)
	}
	for k, v := range mem.Cost {
		mem.Cost[k] = migCost(v)
	}
	mem.CodesRev = codesRev
	return true
}

// ذاكرة المستودعات (الـ lock ماسكه loadStoreMem)
func migrateStoreMemLocked() bool {
	if smem.CodesRev >= codesRev {
		return false
	}
	backupFile(smemPath())
	for k, v := range smem.Codes {
		smem.Codes[k] = [2]string{migTask(v[0]), migCost(v[1])}
	}
	for k, v := range smem.Learned {
		v.Task, v.Cost = migTask(v.Task), migCost(v.Cost)
		smem.Learned[k] = v
	}
	for k, v := range smem.Sections {
		smem.Sections[k] = migTask(v)
	}
	smem.CodesRev = codesRev
	return true
}

// ملفات اتحفظت في السجل قبل التحويل: الـ Dashboard بيقراها بالأكواد الجديدة
func migrateAnaLines(run *Run, ls []anaLine) {
	cut := getSetting("codesV2")
	if cut == "" || run.Created >= cut {
		return
	}
	for i := range ls {
		ls[i].Task = migTask(ls[i].Task)
		ls[i].Cost = migCost(ls[i].Cost)
	}
}

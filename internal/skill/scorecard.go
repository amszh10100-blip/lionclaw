package skill

// SecurityScore 计算 Skill 的安全评分
// 0 warnings/errors → A
// 1-2 warnings → B
// 3+ warnings → C
// 1 error → D
// 2+ errors → F
func SecurityScore(warnings, errors int) string {
	if errors >= 2 {
		return "F"
	}
	if errors == 1 {
		return "D"
	}
	if warnings >= 3 {
		return "C"
	}
	if warnings >= 1 {
		return "B"
	}
	return "A"
}

func FormatScore(score string) string {
	switch score {
	case "A":
		return "[🛡️ A — 安全]"
	case "B":
		return "[✅ B — 较安全]"
	case "C":
		return "[⚠️ C — 需关注]"
	case "D":
		return "[❌ D — 警告]"
	case "F":
		return "[❌ F — 危险]"
	}
	return "[❓ 未知]"
}

package score

func Arcade(ragePercent int, combo int) int {
	if ragePercent < 0 {
		ragePercent = 0
	}
	if ragePercent > 100 {
		ragePercent = 100
	}
	if combo < 0 {
		combo = 0
	}

	// Base score moves smoothly with rage; combo adds arcade-style bursts.
	score := 1 + (ragePercent * 7) + (combo * 13)
	if score > 999 {
		return 999
	}
	if score < 1 {
		return 1
	}
	return score
}

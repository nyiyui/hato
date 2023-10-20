package preset

func ScaleKmH(a int64) int64 {
	return int64(float64(a) * 1e9 / (60 * 60) / 150)
	// a : km/h
	// return : µm/s
}

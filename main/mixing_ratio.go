package main

import "math"

//--------------------------------------
// 重量絶対湿度の計算
//--------------------------------------

// 重量絶対湿度 MR [g/kg(DA)] を 気温 TMP [℃] と 気圧 PRES [hPa] を用いて補正します。
func CorrectMR(MR float64, TMP float64, PRES float64) float64 {
	//  飽和水蒸気量（重量絶対湿度） [g/kg(DA)]
	MR_sat := mixingRatio(PRES, TMP)

	// 重量絶対湿度の補正
	MR_corr := math.Min(MR, MR_sat) // 飽和水蒸気量（重量絶対湿度）を最大とする

	return MR_corr
}

// 気圧 PRES [hPa] と 気温 TMP [℃] から 重量絶対湿度 [g/kg(DA)] を求める。
func mixingRatio(PRES float64, TMP float64) float64 {
	// 絶対温度 [K]
	T := TMP + 273.15

	// 飽和水蒸気圧 [hPa]
	eSAT := eSAT(T)

	// 飽和水蒸気量 [g/m3]
	aT := aT(eSAT, T)

	// 重量絶対湿度 [g/kg(DA)]
	MR := aT / ((PRES / 100) / (2.87 * T))

	return MR
}

// 絶対温度 T [K] から 飽和水蒸気圧 [hPa] を求める。
func eSAT(T float64) float64 {
	return math.Exp(-5800.2206/T+
		1.3914993-0.048640239*T+
		0.41764768*math.Pow(10, -4)*math.Pow(T, 2)-
		0.14452093*math.Pow(10, -7)*math.Pow(T, 3)+
		6.5459673*math.Log(T)) / 100
}

// 飽和水蒸気圧 eSAT [hPa] と 絶対温度 T [K] から 飽和水蒸気量 aT [g/m^3] を求める。
func aT(eSAT float64, T float64) float64 {
	return (217 * eSAT) / T
}

// 飽和水蒸気量 aT [g/m^3] と 相対湿度 RH [%] から 容積絶対湿度 [g/m^3] を求める。
func VH(aT float64, RH float64) float64 {
	return aT * (RH / 100)
}

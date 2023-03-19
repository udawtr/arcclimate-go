package main

import "math"

//重量絶対湿度の計算モジュール

func get_corrected_mixing_ratio(MR float64, TMP float64, PRES float64) float64 {
	// """重量絶対湿度の標高補正

	// Args:
	//   MR(np.ndarray): 補正前重量絶対湿度 (Mixing Ratio) [g/kg(DA)]
	//   TMP(np.ndarray): 気温 [C]
	//   PRES(np.ndarray): 気圧 [hPa]

	// Returns:
	//   np.ndarray: 重量絶対湿度の標高補正後のMR [g/kg(DA)]
	// """
	//  飽和水蒸気量（重量絶対湿度） [g/kg(DA)]
	MR_sat := get_mixing_ratio(PRES, TMP)

	// 重量絶対湿度の補正
	MR_corr := math.Min(MR, MR_sat) // 飽和水蒸気量（重量絶対湿度）を最大とする

	return MR_corr
}

func get_mixing_ratio(PRES float64, TMP float64) float64 {
	// """重量絶対湿度を求める

	// Args:
	//     PRES (np.ndarray): 気圧 [hPa]
	//     TMP (np.ndarray): 気温 [C]

	// Returns:
	//     np.ndarray: 重量絶対湿度 [g/kg(DA)]
	// """

	// 絶対温度 [K]
	T := TMP + 273.15

	// 飽和水蒸気圧 [hPa]
	eSAT := get_eSAT(T)

	// 飽和水蒸気量 [g/m3]
	aT := get_aT(eSAT, T)

	// 重量絶対湿度 [g/kg(DA)]
	MR := aT / ((PRES / 100) / (2.87 * T))

	return MR
}

func get_eSAT(T float64) float64 {
	// """Wexler-Hylandの式 飽和水蒸気圧 eSAT

	// Args:
	//   T(np.ndarray): 絶対温度 [K]

	// Returns:
	//   np.ndarray: 飽和水蒸気圧 [hPa]
	// """
	return math.Exp(-5800.2206/T+
		1.3914993-0.048640239*T+
		0.41764768*math.Pow(10, -4)*math.Pow(T, 2)-
		0.14452093*math.Pow(10, -7)*math.Pow(T, 3)+
		6.5459673*math.Log(T)) / 100
}

func get_aT(eSAT float64, T float64) float64 {
	// """飽和水蒸気量 a(T) Saturated water vapor amount

	// Args:
	//   eSAT(np.ndarray): 飽和水蒸気圧 [hPa]
	//   T(np.ndarray): 絶対温度 [K]

	// Returns:
	//   np.ndarray: 飽和水蒸気量 [g/m3]
	// """
	return (217 * eSAT) / T
}

func get_VH(aT float64, RH float64) float64 {
	// """容積絶対湿度 volumetric humidity

	// Args:
	//   aT(np.ndarray): 飽和水蒸気量 [g/m3]
	//   RH(np.ndarray): 相対湿度 [%]

	// Returns:
	//   np.ndarray: 容積絶対湿度 [g/m3]
	// """
	return aT * (RH / 100)
}

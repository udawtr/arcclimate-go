package main

import "math"

//気圧に関するモジュール

// 気圧の標高補正を行います。
// 引数:
// PRES: 補正前の気圧 [hPa]
// ele_gap: 標高差 [m]
// TMP: 気温 [℃]
// 戻り値:
// 標高補正後の気圧 [hPa]
// ただし、気温減率の平均値を0.0065℃/mとする。
func get_corrected_PRES(PRES float64, ele_gap float64, TMP float64) float64 {
	return PRES * math.Pow(1-((ele_gap*0.0065)/(TMP+273.15)), 5.257)
}

package main

import "math"

//--------------------------------------
// 気圧
//--------------------------------------

// 気圧 PRES [hPa] を 標高差 ele_gap [m] と 気温 TMP [℃]を用いて補正します。
// ただし、気温減率の平均値を0.0065℃/mとします。
func CorrectPRES(PRES float64, ele_gap float64, TMP float64) float64 {
	return PRES * math.Pow(1-((ele_gap*0.0065)/(TMP+273.15)), 5.257)
}

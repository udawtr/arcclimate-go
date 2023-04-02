package main

//--------------------------------------
// 温度補正
//--------------------------------------

// 気温 TMP [℃] を、 標高差 ele_gap [m] を用いて補正します。
// ただし、気温減率の平均値を0.0065℃/m とします。
func CorrectTMP(TMP float64, ele_gap float64) float64 {
	return TMP + ele_gap*-0.0065
}

package main

// 気温の標高補正をおこないます。基準値の気温をTMP[℃]とし、
// 標高差が ele_gap [m]ある地点の温度を計算します。
// ただし、気温減率の平均値を0.0065℃/m とします。
func CorrectTMP(TMP float64, ele_gap float64) float64 {
	return TMP + ele_gap*-0.0065
}

package main

import "math"

//--------------------------------------
// 相対湿度、水蒸気分圧および露点温度の計算
//--------------------------------------

// 重量絶対湿度(補正前) MR [g/kg(DA)], 気温 TMP [℃], 気圧 PRES [Pa] から相対湿度 RH [%] と水蒸気分圧 Pw [hPa] を求める
func func_RH_eSAT(MR float64, TMP float64, PRES float64) (RH float64, Pw float64) {

	P := PRES / 100   // hpa
	T := TMP + 273.15 // 絶対温度
	VH := MR * (P / (T * 2.87))

	eSAT := math.Exp(-5800.2206/T+1.3914993-0.048640239*T+0.41764768*math.Pow(10, -4)*math.Pow(T, 2)-0.14452093*math.Pow(10, -7)*math.Pow(T, 3)+6.5459673*math.Log(T)) / 100 // hPa
	aT := (217 * eSAT) / T
	RH = VH / aT * 100
	Pw = RH / 100 * eSAT // hPa

	return RH, Pw
}

// 水蒸気分圧 Pw [hPa] から気温（露点温度）DT [℃]を求める。ただし、0.039 <= Pw(hpa) < 6.112（-50～0℃の時）
// パソコンによる空気調和計算法 著:宇田川光弘,オーム社, 1986.12 より
// 0.039 <= Pw(hpa) < 6.112（-50～0℃の時）
func func_DT_0(Pw float64) float64 {
	Y := math.Log(Pw * 100) // Pa
	return -60.662 + 7.4624*Y + 0.20594*math.Pow(Y, 2) + 0.016321*math.Pow(Y, 3)
}

// 水蒸気分圧 PW [hPa] から気温（露点温度） DT [℃]を求める近似式。ただし、6.112 <= Pw(hpa) <= 123.50（0～50℃の時）
// パソコンによる空気調和計算法 著:宇田川光弘,オーム社, 1986.12 より
func func_DT_50(Pw float64) float64 {
	Y := math.Log(Pw * 100) // Pa
	return -77.199 + 13.198*Y - 0.63772*math.Pow(Y, 2) + 0.071098*math.Pow(Y, 3)
}

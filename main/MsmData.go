package main

import "time"

// MSMファイルから読み取ったデータ
type MsmData struct {
	date      []time.Time //参照時刻。日本標準時JST
	TMP       []float64   //参照時刻時点の気温の瞬時値 (単位:℃)
	MR        []float64   //参照時刻時点の重量絶対湿度の瞬時値 (単位:g/kgDA)
	DSWRF_est []float64   //参照時刻の前1時間の推定日射量の積算値 (単位:MJ/m2)
	DSWRF_msm []float64   //参照時刻の前1時間の日射量の積算値 (単位:MJ/m2)
	Ld        []float64   //参照時刻の前1時間の下向き大気放射量の積算値 (単位:MJ/m2)
	VGRD      []float64   //南北風(V軸) (単位:m/s)
	UGRD      []float64   //東西風(U軸) (単位:m/s)
	PRES      []float64   //気圧 (単位:hPa)
	APCP01    []float64   //参照時刻の前1時間の降水量の積算値 (単位:mm/h)
}

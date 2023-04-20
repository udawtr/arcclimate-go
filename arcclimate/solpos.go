package arcclimate

import (
	"math"
	"time"
)

// """緯度経度と日時データから太陽位置および大気外法線面日射量の計算を行う
// Args:
//
//	lat(float64): 推計対象地点の緯度（10進法）
//	lon(float64): 推計対象地点の経度（10進法）
//	date(pd.Series): 計算対象の時刻データ
//
// Returns:
//
//	pd.DataFrame: 大気外法線面日射量、太陽高度角および方位角のデータフレーム
//	              df[IN0:大気外法線面日射量,
//	                 h:太陽高度角,
//	                 Sinh:太陽高度角のサイン,
//	                 A:太陽方位角]
//
// """
func get_sun_position(lat float64, lon float64, date []time.Time) []SunPositionRecord {

	//参照時刻の前1時間の太陽高度および方位角を取得する（1/10時間ずつ計算した平均値）
	count := [...]float64{1.0, 0.9, 0.8, 0.7, 0.6, 0.5, 0.4, 0.3, 0.2, 0.1}

	const J0 = 4.921              //太陽定数[MJ/m²h] 4.921
	dlt0 := degreeToRad(-23.4393) //冬至の日赤緯

	const lons = 135.0         //標準時の地点の経度
	latrad := degreeToRad(lat) //緯度
	Cos_lat := math.Cos(latrad)
	Sin_lat := math.Sin(latrad)

	var h [10]float64 //hの容器
	var A [10]float64 //Aの容器

	df := make([]SunPositionRecord, len(date))
	for i := 0; i < len(df); i++ {
		DY := date[i].Year()
		//year := time.Date(DY, 1, 1, 0, 0, 0, 0, time.UTC)
		nday := float64(date[i].YearDay()) //年間通日+1
		Tm := float64(date[i].Hour())      //標準時

		n := float64(DY - 1968)

		d0 := 3.71 + 0.2596*n - math.Floor((n+3)/4)                               //近日点通過日
		m := 360 * (nday - d0) / 365.2596                                         //平均近点離角
		eps := 12.3901 + 0.0172*(n+m/360)                                         //近日点と冬至点の角度
		v := m + 1.914*math.Sin(degreeToRad(m)) + 0.02*math.Sin(degreeToRad(2*m)) //真近点離角
		veps := degreeToRad(v + eps)
		Et := (m - v) - radToDegree(math.Atan(0.043*math.Sin(2*veps)/(1.0-0.043*math.Cos(2*veps)))) //近時差

		sindlt := math.Cos(veps) * math.Sin(dlt0)          //赤緯の正弦
		cosdlt := math.Sqrt(math.Abs(1.0 - sindlt*sindlt)) //赤緯の余弦

		IN0 := J0 * (1 + 0.033*math.Cos(degreeToRad(v))) //IN0 大気外法線面日射量

		for idx, j := range count {
			tm := Tm - j
			t := 15*(tm-12) + (lon - lons) + Et //時角
			trad := degreeToRad(t)
			Sinh := Sin_lat*sindlt + Cos_lat*cosdlt*math.Cos(trad) //太陽高度角の正弦
			Cosh := math.Sqrt(1 - Sinh*Sinh)
			SinA := cosdlt * math.Sin(trad) / Cosh
			CosA := (Sinh*Sin_lat - sindlt) / (Cosh * Cos_lat)

			h[idx] = math.Asin(Sinh)
			A[idx] = math.Atan2(SinA, CosA) + math.Pi
		}

		//太陽高度[rad]
		var h_avg float64
		for i := 0; i < 10; i++ {
			h_avg += h[i]
		}
		h_avg /= 10

		//太陽高度角のサイン
		Sinh := math.Sin(h_avg)

		//太陽方位角[rad]
		var A_avg float64
		for i := 0; i < 10; i++ {
			A_avg += A[i]
		}
		A_avg /= 10

		df[i] = SunPositionRecord{
			IN0:  IN0,
			h:    radToDegree(h_avg),
			Sinh: Sinh,
			A:    radToDegree(A_avg),
		}
	}

	return df
}

type SunPositionRecord struct {
	IN0  float64 //IN0 大気外法線面日射量
	h    float64 //太陽高度(1時間平均), deg
	Sinh float64 //太陽高度角のサイン
	A    float64 //太陽方位角(1時間平均), deg
}

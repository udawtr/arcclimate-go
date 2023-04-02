package main

import (
	"math"
)

//--------------------------------------
// 距離の重みづけ平均の係数を算出
//--------------------------------------

// 推計対象地点の緯度（10進法）lat, 経度 lon から MSM4地点(SW,SE,NW,NE)の重みを返す。
func MsmWeights(lat float64, lon float64) [4]float64 {

	// 補間計算 リストはいずれもSW南西,SE南東,NW北西,NE北東の順
	// 入力した緯度経度から周囲のMSMまでの距離を算出して、距離の重みづけ係数をリストで返す
	distances := latLonMsmDistances(lat, lon)

	// MSM4地点のと目標座標の距離から4地点のウェイトを計算
	weights := weightsFromDistances(distances)

	return weights
}

// 推計対象地点の緯度（10進法）lat, 経度 lon からMSM4地点(SW,SE,NW,NE)と推計対象地点の距離を返す。
func latLonMsmDistances(lat float64, lon float64) [4]float64 {
	const lat_unit = 0.05   // MSMの緯度間隔
	const lon_unit = 0.0625 // MSMの経度間隔

	lat0 := lat
	lon0 := lon

	// メッシュ周囲のMSM位置（緯度経度）の取得
	lat_S := math.Floor(lat/lat_unit) * lat_unit // 南は切り下げ
	lon_W := math.Floor(lon/lon_unit) * lon_unit // 西は切り下げ

	// 緯度経度差から距離の重みづけ平均の係数を算出

	// 南西（左下）との距離
	MSM_SW := vincentyInverse(lat0, lon0, lat_S, lon_W)

	// 南東（右下）との距離
	MSM_SE := vincentyInverse(lat0, lon0, lat_S, lon_W+lon_unit)

	// 北西（左上）との距離
	MSM_NW := vincentyInverse(lat0, lon0, lat_S+lat_unit, lon_W)

	// 北東（右上）との距離
	MSM_NE := vincentyInverse(lat0, lon0, lat_S+lat_unit, lon_W+lon_unit)

	return [4]float64{MSM_SW, MSM_SE, MSM_NW, MSM_NE}
}

// vincenty法(逆解法)を用いて、地点1(緯度lat1,経度lon1)と地点2(緯度lat2,経度lon2)の楕円体上の距離[m]を求めます。
// ただし、計算に失敗した場合は、パニックが発生します。
//
// 参照)
//
//	https://ja.wikipedia.org/wiki/Vincenty法
//	https://vldb.gsi.go.jp/sokuchi/surveycalc/surveycalc/bl2stf.html
func vincentyInverse(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	// 反復計算の上限回数
	const ITERATION_LIMIT = 10000

	// 差異が無ければ0.0を返す
	if math.Abs(lat1-lat2) < 1e-9 && math.Abs(lon1-lon2) < 1e-9 {
		return 0.0
	}

	// 長軸半径と扁平率から短軸半径を算出する
	// 楕円体はGRS80の値
	a := 6378137.0         // 長軸半径(GRS80)
	ƒ := 1 / 298.257222101 // 扁平率(GRS80)
	b := (1 - ƒ) * a

	p1 := degreeToRad(lat1) // φ1
	p2 := degreeToRad(lat2) // φ2
	r1 := degreeToRad(lon1) // λ1
	r2 := degreeToRad(lon2) // λ2

	// 更成緯度(補助球上の緯度)
	U1 := math.Atan((1 - ƒ) * math.Tan(p1))
	U2 := math.Atan((1 - ƒ) * math.Tan(p2))

	sinU1 := math.Sin(U1)
	sinU2 := math.Sin(U2)
	cosU1 := math.Cos(U1)
	cosU2 := math.Cos(U2)

	// 2点間の経度差
	L := r2 - r1

	// λをLで初期化
	ramda := L

	// λを収束計算。反復回数の上限を10000回に設定
	var ramada_p, cos2A, sinS, cos2Sm, cosS, sigma float64
	for i := 0; i < ITERATION_LIMIT; i++ {
		sinR := math.Sin(ramda)
		cosR := math.Cos(ramda)
		sinS = math.Sqrt((cosU2*sinR)*(cosU2*sinR) + (cosU1*sinU2-sinU1*cosU2*cosR)*(cosU1*sinU2-sinU1*cosU2*cosR))
		cosS = sinU1*sinU2 + cosU1*cosU2*cosR
		sigma = math.Atan2(sinS, cosS)
		sinA := cosU1 * cosU2 * sinR / sinS
		cos2A = 1 - (sinA * sinA)
		cos2Sm = cosS - 2*sinU1*sinU2/cos2A
		C := ƒ / 16 * cos2A * (4 + ƒ*(4-3*cos2A))

		ramada_p = ramda
		ramda = L + (1-C)*ƒ*sinA*(sigma+C*sinS*(cos2Sm+C*cosS*(-1+2*(cos2Sm*cos2Sm))))

		// 偏差が.000000000001以下ならbreak
		//fmt.Printf("%.10f\t%.10f\t%.10f\n", ramda, ramada_p, ramda-ramada_p)
		if math.Abs(ramda-ramada_p) <= 1e-12 {
			break
		}
	}

	// 偏差が.000000000001以下ならbreak
	if math.Abs(ramda-ramada_p) > 1e-12 {
		// 計算が収束しなかった場合はNoneを返す
		panic("計算が収束しなかった")
	}

	// λが所望の精度まで収束したら以下の計算を行う
	u2 := cos2A * (a*a - b*b) / (b * b)
	A := 1 + u2/16384*(4096+u2*(-768+u2*(320-175*u2)))
	B := u2 / 1024 * (256 + u2*(-128+u2*(74-47*u2)))
	dS := B * sinS * (cos2Sm + B/4*(cosS*(-1+2*(cos2Sm*cos2Sm))-B/6*cos2Sm*(-3+4*(sinS*sinS))*(-3+4*(cos2Sm*cos2Sm))))

	// 2点間の楕円体上の距離
	s := b * A * (sigma - dS)

	return s
}

// 基準地点からの距離 distances [m]に応じて、それぞれの距離離れた地点の重みづけを計算します。
// 全ても重みを合算すると常に1になるように計算します。
func weightsFromDistances(distances [4]float64) [4]float64 {

	weights := [4]float64{0.0, 0.0, 0.0, 0.0}

	//ピンポイント地点がある場合
	for i := 0; i < 4; i++ {
		if distances[i] == 0.0 {
			weights[i] = 1.0
			return weights
		}
	}

	var total_distance_inv float64 = 0.0
	for i := 0; i < 4; i++ {
		total_distance_inv += 1.0 / distances[i]
	}
	for i := 0; i < 4; i++ {
		weights[i] = 1.0 / distances[i] / total_distance_inv
	}

	return weights
}

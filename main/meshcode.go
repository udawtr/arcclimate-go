package main

import (
	"math"
	"strconv"
)

//--------------------------------------
// メッシュコード処理
// ref: 『統計に用いる標準地域メッシュおよび標準地域メッシュコード』
//--------------------------------------

// 経度 lon, 緯度 lat からメッシュコード(1 次、2 次、3 次)を取得
func MeshCodeFromLatLon(lat float64, lon float64) int {
	lt := lat * 3.0 / 2.0
	lg := lon
	y1 := math.Floor(lt)
	x1 := math.Floor(lg)

	lt = (lt - y1) * 8.0
	lg = (lg - x1) * 8.0
	y2 := math.Floor(lt)
	x2 := math.Floor(lg)

	lt = (lt - y2) * 10.0
	lg = (lg - x2) * 10.0
	y3 := math.Floor(lt)
	x3 := math.Floor(lg)

	code1 := 0
	code1 += int(y1) % 100 * 100
	code1 += int(x1) % 100 * 1

	code2 := 0
	code2 += int(y2) * 10
	code2 += int(x2) * 1

	code3 := 0
	code3 += int(y3) * 10
	code3 += int(x3) * 1

	return code1*10000 + code2*100 + code3
}

// メッシュコード meshcode から緯度(10進数) lat, 経度(10進数) lon への変換
func get_mesh_latlon(meshcode string) (lat float64, lon float64) {
	// メッシュコードから緯度経度を計算(中心ではなく南西方向の座標が得られる)
	b := []byte(meshcode)
	y1, _ := strconv.ParseFloat(string(b[:2]), 64)
	x1, _ := strconv.ParseFloat(string(b[2:4]), 64)
	y2, _ := strconv.ParseFloat(string(b[4]), 64)
	x2, _ := strconv.ParseFloat(string(b[5]), 64)
	y3, _ := strconv.ParseFloat(string(b[6]), 64)
	x3, _ := strconv.ParseFloat(string(b[7]), 64)

	// 南西方向の座標からメッシュ中心の緯度を算出
	lat = ((y1*80 + y2*10 + y3) * 30 / 3600) + 15.0/3600

	// 南西方向の座標からメッシュ中心の経度を算出
	lon = (((x1*80 + x2*10 + x3) * 45 / 3600) + 100) + 22.5/3600

	return lat, lon
}

package main

import (
	"math"
	"strconv"
)

//--------------------------------------
// メッシュコード処理
// ref: 『統計に用いる標準地域メッシュおよび標準地域メッシュコード』
//--------------------------------------

// メッシュ周囲のMSM位置（緯度経度）と番号（北始まり0～、西始まり0～）の取得
// Args:
//
//	lat(float64): 推計対象地点の緯度（10進法）
//	lon(float64): 推計対象地点の経度（10進法）
//
// Returns:
//
//	Tuple[int, int, int, int]: メッシュ周囲のMSM位置（緯度経度）と番号（北始まり0～、西始まり0～）
func Meshcode1d(lat float64, lon float64) (int, int, int, int) {
	lat_unit := 0.05   // MSMの緯度間隔
	lon_unit := 0.0625 // MSMの経度間隔

	// 緯度⇒メッシュ番号
	lat_S := math.Floor(lat/lat_unit) * lat_unit // 南は切り下げ
	MSM_S := int(math.Round((47.6 - lat_S) / lat_unit))
	MSM_N := int(MSM_S - 1)

	// 経度⇒メッシュ番号
	lon_W := math.Floor(lon/lon_unit) * lon_unit // 西は切り下げ
	MSM_W := int(math.Round((lon_W - 120) / lon_unit))
	MSM_E := int(MSM_W + 1)

	return MSM_S, MSM_N, MSM_W, MSM_E
}

// 経度 lon, 緯度 lat からメッシュコード(1 次、2 次、3 次)を取得
func MeshCodeFromLatLon(lat float64, lon float64) (int, int) {
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

	return code1, code2*100 + code3
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

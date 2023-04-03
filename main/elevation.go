package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
)

//--------------------------------------
// 標高
//--------------------------------------

// 緯度 lat, 経度 lonの地点の標高[m]の取得します。
// 取得の方法 mode_elevation は、 "mesh" または "api" を指定します。
// "mesh"の場合は、標高補正に3次メッシュ（1㎞メッシュ）の平均標高データ mesh_elevation_master を使用します。
// "api"の場合は、国土地理院のAPIを使用します。そのため、平均標高データ mesh_elevation_masterは不要です。
func ElevationFromLatLon(
	lat float64,
	lon float64,
	mode_elevation string,
	mesh_elevation_master map[int]map[int]float64) float64 {

	var elevation float64

	if mode_elevation == "mesh" {
		// 標高補正に3次メッシュ（1㎞メッシュ）の平均標高データを使用する場合
		elevation = elevationFromMesh(lat, lon, mesh_elevation_master)

		log.Printf("入力された緯度・経度が含まれる3次メッシュの平均標高 %fm で計算します", elevation)

	} else if mode_elevation == "api" {
		// 国土地理院のAPIを使用して入力した緯度f経度位置の標高を返す
		log.Printf("入力された緯度・経度位置の標高データを国土地理院のAPIから取得します")
		var err error
		elevation, err = elevationFromCyberjapandata2(lat, lon)
		if err == nil {
			log.Printf("成功  標高 %fm で計算します", elevation)
		} else {
			// 国土地理院のAPIから標高データを取得できなかった場合の判断
			// 標高補正に3次メッシュ（1㎞メッシュ）の平均標高データにフォールバック
			// ref: https://maps.gsi.go.jp/development/elevation_s.html
			// ref: https://github.com/gsi-cyberjapan/elevation-php/blob/master/getelevation.php
			elevation = elevationFromMesh(lat, lon, mesh_elevation_master)
			log.Printf("国土地理院のAPIから標高データを取得できなかったため、\n"+
				"入力された緯度・経度が含まれる3次メッシュの平均標高 %fm で計算します", elevation)
		}
	} else {
		panic(mode_elevation)
	}

	return elevation
}

// 3次メッシュ（1㎞メッシュ）の平均標高データ mesh_elevation_master を用いて、緯度 lat, 経度 lonの地点の標高[m]の取得します。
func elevationFromMesh(
	lat float64,
	lon float64,
	mesh_elevation_master map[int]map[int]float64,
) float64 {
	meshcode1d, meshcode23d := MeshCodeFromLatLon(lat, lon)
	elevation := mesh_elevation_master[meshcode1d][meshcode23d]
	return elevation
}

// 国土地理院のAPI を用いて、緯度 lat, 経度 lonの地点の標高[m]の取得します。
func elevationFromCyberjapandata2(lat float64, lon float64) (float64, error) {
	cyberjapandata2_endpoint := "http://cyberjapandata2.gsi.go.jp/general/dem/scripts/getelevation.php"
	url := fmt.Sprintf("%s?lon=%f&lat=%f&outtype=%s", cyberjapandata2_endpoint, lon, lat, "JSON")

	resp, err := http.Get(url)
	if err != nil {
		return math.NaN(), err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		panic(err)
	}

	var eleApiRes ElevationApiResnponse
	if err := json.Unmarshal(body, &eleApiRes); err != nil {
		return math.NaN(), err
	}

	return eleApiRes.Elevation.(float64), nil
}

type ElevationApiResnponse struct {
	Elevation interface{} `json:"elevation"`
	HSrc      interface{} `json:"hsrc"`
}

package main

import (
	"bytes"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
)

type ElevationMaster struct {

	//MSM4地点の平均標高を取得するため2次メッシュコードまでの標高
	DfMsmEle [][]float64

	// ピンポイントの標高のため、3次メッシュコードまで含んだ標高
	DfMeshEle map[int]map[int]float64
}

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
	mesh_elevation_master *ElevationMaster) float64 {

	var elevation float64

	if mode_elevation == "mesh" {
		// 標高補正に3次メッシュ（1㎞メッシュ）の平均標高データを使用する場合
		elevation = mesh_elevation_master.Elevation3d(lat, lon)

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
			elevation = mesh_elevation_master.Elevation3d(lat, lon)
			log.Printf("国土地理院のAPIから標高データを取得できなかったため、\n"+
				"入力された緯度・経度が含まれる3次メッシュの平均標高 %fm で計算します", elevation)
		}
	} else {
		panic(mode_elevation)
	}

	return elevation
}

// 3次メッシュ（1㎞メッシュ）の平均標高データ mesh_elevation_master を用いて、緯度 lat, 経度 lonの地点の標高[m]の取得します。
func (mesh_elevation_master *ElevationMaster) Elevation3d(lat float64, lon float64) float64 {
	meshcode1d, meshcode23d := MeshCodeFromLatLon(lat, lon)
	elevation := mesh_elevation_master.DfMeshEle[meshcode1d][meshcode23d]
	return elevation
}

func (msm_elevation_master *ElevationMaster) Elevation2d(codeSN int, codeWE int) float64 {
	return msm_elevation_master.DfMsmEle[codeSN][codeWE]
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

//go:embed data/*.csv
var f embed.FS

// 経度 lon, 緯度 lat の補完に必要なマスタ読み取り
func NewElevationMaster(lat float64, lon float64) *ElevationMaster {
	ele := &ElevationMaster{
		DfMsmEle:  make([][]float64, 0),
		DfMeshEle: make(map[int]map[int]float64),
	}

	mesh1d, _ := MeshCodeFromLatLon(lat, lon)

	// MSM地点の標高データの読込
	ele.ReadMsmElevation()

	// 3次メッシュの標高データの読込
	ele.Read3dMeshElevation(mesh1d)

	return ele
}

// 2次メッシュコードまでの標高データを読み取り
func (ele *ElevationMaster) ReadMsmElevation() {
	// Open the CSV file
	content, err := f.ReadFile("data/MSM_elevation.csv")
	if err != nil {
		panic(err)
	}

	// Create a new CSV reader
	reader := csv.NewReader(bytes.NewBuffer(content))

	// Read all records at once
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	// Print the records
	elemap := make([][]float64, len(records))
	for i, record := range records {
		elemap[i] = make([]float64, len(record))
		for j := 0; j < len(record); j++ {
			elemap[i][j], err = strconv.ParseFloat(record[j], 64)
			if err != nil {
				panic(err)
			}
		}
	}

	ele.DfMsmEle = elemap
}

func (ele *ElevationMaster) Read3dMeshElevation(meshcode_1d int) {
	// Open the CSV file
	content, err := f.ReadFile(fmt.Sprintf("data/mesh_3d_ele_%d.csv", meshcode_1d))
	if err != nil {
		panic(err)
	}

	// Create a new CSV reader
	reader := csv.NewReader(bytes.NewBuffer(content))

	// Skip a header
	_, _ = reader.Read()

	// Read all records at once
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	// Print the records
	elemap := make(map[int]float64, len(records))
	for _, record := range records {
		meshcode, err := strconv.Atoi(record[0])
		if err != nil {
			panic(err)
		}
		elevation, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			panic(err)
		}
		elemap[meshcode] = elevation
	}

	ele.DfMeshEle[meshcode_1d] = elemap
}

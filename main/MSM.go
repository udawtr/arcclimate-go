package main

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// メッシュ周囲のMSM位置（緯度経度）と番号（北始まり0～、西始まり0～）の取得
// Args:
//
//	lat(float64): 推計対象地点の緯度（10進法）
//	lon(float64): 推計対象地点の経度（10進法）
//
// Returns:
//
//	Tuple[int, int, int, int]: メッシュ周囲のMSM位置（緯度経度）と番号（北始まり0～、西始まり0～）
func get_MSM(lat float64, lon float64) (int, int, int, int) {
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

func load_msm_files(lat float64, lon float64, msm_file_dir string) ([]string, []MsmData) {
	// """MSMファイルを読み込みます。必要に応じてダウンロードを行います。

	// Args:
	//   lat(float64): 推計対象地点の緯度（10進法）
	//   lon(float64): 推計対象地点の経度（10進法）
	//   msm_file_dir(str): MSMファイルの格納ディレクトリ

	// Returns:
	//   msm_list(list[str]): 読み込んだMSMファイルの一覧
	//   df_msm_list(list[pd.DataFrame]): 読み込んだデータフレームのリスト
	// """
	// 計算に必要なMSMを算出して、ダウンロード⇒ファイルpathをリストで返す

	// 保存先ディレクトリの作成
	os.Mkdir(msm_file_dir, os.ModePerm)

	// 必要なMSMファイル名の一覧を緯度経度から取得
	msm_list := get_msm_requirements(lat, lon)

	// ダウンロードが必要なMSMの一覧を取得
	msm_list_missed := get_missing_msm(msm_list[:], msm_file_dir)

	// ダウンロード
	download_msm_files(msm_list_missed, msm_file_dir)

	// MSMファイル読み込み
	df_msm_list := make([]MsmData, len(msm_list))
	c := make(chan MsmAndIndex, 4)
	for index, msm := range msm_list {
		// MSMファイルのパス
		// MSMファイル読み込み
		// 負の日射量が存在した際に日射量を0とする
		go load_msm(index, msm_file_dir, msm, c)
	}

	for i := 0; i < len(msm_list); i++ {
		ret := <-c
		df_msm_list[ret.Index] = ret.Msm
	}

	return msm_list, df_msm_list
}

type MsmAndIndex struct {
	Index int
	Msm   MsmData
}

func load_msm(index int, msm_file_dir string, msm string, c chan MsmAndIndex) {
	msm_path := filepath.Join(msm_file_dir, fmt.Sprintf("%s.csv.gz", msm))

	log.Printf("MSMファイル読み込み: %s", msm_path)

	f, ferr := os.Open(msm_path)
	if ferr != nil {
		log.Fatal(ferr)
		panic(ferr)
	}
	defer f.Close()

	gf, gerr := gzip.NewReader(f)
	if gerr != nil {
		fmt.Println("gzipエラー")
		panic(gerr)
	}
	defer gf.Close()

	csvReader := csv.NewReader(gf)
	_, _ = csvReader.Read()
	data, cerr := csvReader.ReadAll()
	if cerr != nil {
		log.Fatal(cerr)
		panic(cerr)
	}

	df_msm := MsmData{
		date:      make([]time.Time, len(data)),
		TMP:       make([]float64, len(data)),
		MR:        make([]float64, len(data)),
		DSWRF_est: make([]float64, len(data)),
		DSWRF_msm: make([]float64, len(data)),
		Ld:        make([]float64, len(data)),
		VGRD:      make([]float64, len(data)),
		UGRD:      make([]float64, len(data)),
		PRES:      make([]float64, len(data)),
		APCP01:    make([]float64, len(data)),
	}
	for i, row := range data {

		date, _ := time.Parse("2006-01-02 15:04:05", row[0])
		TMP, _ := strconv.ParseFloat(row[1], 64)
		MR, _ := strconv.ParseFloat(row[2], 64)
		DSWRF_est, _ := strconv.ParseFloat(row[3], 64)
		DSWRF_msm, _ := strconv.ParseFloat(row[4], 64)
		Ld, _ := strconv.ParseFloat(row[5], 64)
		VGRD, _ := strconv.ParseFloat(row[6], 64)
		UGRD, _ := strconv.ParseFloat(row[7], 64)
		PRES, _ := strconv.ParseFloat(row[8], 64)
		APCP01, _ := strconv.ParseFloat(row[9], 64)

		if DSWRF_msm < 0.0 {
			DSWRF_msm = 0.0
		}
		if DSWRF_est < 0.0 {
			DSWRF_est = 0.0
		}

		df_msm.date[i] = date
		df_msm.TMP[i] = TMP
		df_msm.MR[i] = MR
		df_msm.DSWRF_est[i] = DSWRF_est
		df_msm.DSWRF_msm[i] = DSWRF_msm
		df_msm.Ld[i] = Ld
		df_msm.VGRD[i] = VGRD
		df_msm.UGRD[i] = UGRD
		df_msm.PRES[i] = PRES
		df_msm.APCP01[i] = APCP01
	}
	c <- MsmAndIndex{index, df_msm}
}

func download_msm_files(msm_list []string, output_dir string) error {
	// """MSMファイルのダウンロード

	// Args:
	//   msm_list(Iterable[str]): ダウンロードするMSM名 ex)159-338
	//   output_dir(str): ダウンロード先ディレクトリ名 ex) ./msm/
	// """

	// ダウンロード元URL
	dl_url := "https://s3.us-west-1.wasabisys.com/arcclimate/msm_2011_2020/"

	dl := func(msm string, c chan error) {
		var err error = nil
		src_url := fmt.Sprintf("%s%s.csv.gz", dl_url, msm)
		save_path := filepath.Join(output_dir, fmt.Sprintf("%s.csv.gz", msm))

		log.Printf("MSMダウンロード %s => %s", src_url, save_path)

		// Get the data
		resp, err := http.Get(src_url)
		if err != nil {
			c <- err
			return
		}
		defer resp.Body.Close()

		// Create the file
		var out *os.File
		out, err = os.Create(save_path)
		if err != nil {
			c <- err
			return
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			c <- err
			return
		}

		c <- nil
	}

	c := make(chan error, 4)
	for _, msm := range msm_list {
		go dl(msm, c)
	}

	for i := 0; i < len(msm_list); i++ {
		err := <-c
		if err != nil {
			return err
		}
	}

	return nil
}

func get_missing_msm(msm_list []string, msm_file_dir string) []string {
	// """存在しないMSMファイルの一覧を取得

	// Args:
	//   msm_list(Iterable[str]): MSM名一覧
	//   msm_file_dir(str): MSMファイルの格納ディレクトリ

	// Returns:
	//   不足しているMSM名一覧
	// """

	missing_list := make([]string, 0)

	for _, msm := range msm_list {

		msm_path := filepath.Join(msm_file_dir, fmt.Sprintf("%s.csv.gz", msm))
		if !fileExists(msm_path) {
			missing_list = append(missing_list, msm)
		}
	}

	return missing_list
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func get_msm_requirements(lat float64, lon float64) []string {
	// """必要なMSMファイル名の一覧を取得

	// Args:
	//   lat(float64): 推計対象地点の緯度（10進法）
	//   lon(float64): 推計対象地点の経度（10進法）

	// Returns:
	//   Tuple[str, str, str, str]: 隣接する4地点のMSMファイル名のタプル
	// """
	MSM_S, MSM_N, MSM_W, MSM_E := get_MSM(lat, lon)

	// 周囲4地点のメッシュ地点番号
	MSM_SW := fmt.Sprintf("%d-%d", MSM_S, MSM_W)
	MSM_SE := fmt.Sprintf("%d-%d", MSM_S, MSM_E)
	MSM_NW := fmt.Sprintf("%d-%d", MSM_N, MSM_W)
	MSM_NE := fmt.Sprintf("%d-%d", MSM_N, MSM_E)

	return []string{MSM_SW, MSM_SE, MSM_NW, MSM_NE}
}

func get_msm_elevations(lat float64, lon float64, msm_elevation_master [][]float64) [4]float64 {
	// """計算に必要なMSMを算出して、MSM位置の標高を探してタプルで返す

	// Args:
	//   lat: 推計対象地点の緯度（10進法）
	//   lon: 推計対象地点の経度（10進法）
	//   msm_elevation_master: MSM地点の標高データ [m]

	// Returns:
	//   Tuple[float64, float64, float64, float64]: 4地点の標高をタプルで返します(SW, SE, NW, NE)
	// """

	MSM_S, MSM_N, MSM_W, MSM_E := get_MSM(lat, lon)

	ele_SW := msm_elevation_master[MSM_S][MSM_W] // SW
	ele_SE := msm_elevation_master[MSM_S][MSM_E] // SE
	ele_NW := msm_elevation_master[MSM_N][MSM_W] // NW
	ele_NE := msm_elevation_master[MSM_N][MSM_E] // NE

	return [4]float64{ele_SW, ele_SE, ele_NW, ele_NE}
}

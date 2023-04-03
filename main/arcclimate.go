// ArcClimate
package main

import (
	"bytes"
	"embed"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/akamensky/argparse"
	"github.com/hhkbp2/go-logging"
)

// 直散分離に関するデータ
type AAA struct {
	SH float64 //水平面天空日射量
	DN float64 //法線面直達日射量
	DT float64 //露点温度
}

// 緯度lat,経度lonで表される推計対象地点の周囲のMSMデータを利用して空間補間計算を行います。
// 出力する気象データの期間は開始年start_yearから終了年end_yearまでです。
//
//	msm_elevation_master(pd.DataFrame):  MSM地点の標高データ
//	mesh_elevation_master(pd.DataFrame): 3次メッシュの標高データ
//	msms(Tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame, pd.DataFrame]): 4地点のMSMデータ
//	mode_elevation(str, Optional): 'mesh':標高補正に3次メッシュ（1㎞メッシュ）の平均標高データを使用する
//	                     'api':国土地理院のAPIを使用する
//	mode(str, Optional): "normal"=補正のみ
//	                     "EA"=拡張アメダス方式に準じた標準年データを作成する (funcault value = 'api')
//	use_est(bool, Optional): 標準年データの検討に日射量の推計値を使用する（使用しない場合2018年以降のデータのみで作成） (funcault value = True)
//
// Returns:
//
//	pd.DataFrame: MSMデータフレーム
func interpolate(
	lat float64,
	lon float64,
	start_year int,
	end_year int,
	msm_elevation_master [][]float64,
	mesh_elevation_master map[int]map[int]float64,
	msms [4]MsmData,
	mode_elevation string,
	mode string,
	use_est bool,
	mode_separate string) *MsmTarget {

	// 周囲4地点のMSMデータフレームから標高補正したMSMデータフレームを作成
	var msm *MsmTarget = _get_interpolated_msm(
		lat,
		lon,
		msms,
		msm_elevation_master,
		mesh_elevation_master,
		mode_elevation,
		mode_separate)

	// ベクトル風速から16方位の風向風速を計算
	msm._convert_wind16()

	var df_save *MsmTarget

	if mode == "normal" {
		// 保存用に年月日をフィルタ
		local, _ := time.LoadLocation("Local")
		start_time := time.Date(start_year, 1, 1, 0, 0, 0, 0, local)
		end_time := time.Date(end_year+1, 1, 1, 0, 0, 0, 0, local)
		start_index := sort.Search(len(msm.date), func(i int) bool {
			return msm.date[i].After(start_time) || msm.date[i].Equal(start_time)
		})
		end_index := sort.Search(len(msm.date), func(i int) bool {
			return msm.date[i].After(end_time) || msm.date[i].Equal(end_time)
		})
		df_save = &MsmTarget{
			date:      msm.date[start_index:end_index],
			TMP:       msm.TMP[start_index:end_index],
			MR:        msm.MR[start_index:end_index],
			DSWRF_est: msm.DSWRF_est[start_index:end_index],
			DSWRF_msm: msm.DSWRF_msm[start_index:end_index],
			Ld:        msm.Ld[start_index:end_index],
			VGRD:      msm.VGRD[start_index:end_index],
			UGRD:      msm.UGRD[start_index:end_index],
			PRES:      msm.PRES[start_index:end_index],
			APCP01:    msm.APCP01[start_index:end_index],
			RH:        msm.RH[start_index:end_index],
			Pw:        msm.Pw[start_index:end_index],
			DT:        msm.DT[start_index:end_index],
			AAA_est:   msm.AAA_est[start_index:end_index],
			AAA_msm:   msm.AAA_msm[start_index:end_index],
			NR:        msm.NR[start_index:end_index],
			w_spd:     msm.w_spd[start_index:end_index],
			w_dir:     msm.w_dir[start_index:end_index],
		}

	} else if mode == "EA" {
		// 標準年の計算
		df_save, _ = msm.calc_EA(
			start_year,
			end_year,
			use_est)

		// ベクトル風速から16方位の風向風速を再計算
		df_save._convert_wind16()
	} else {
		panic(mode)
	}

	return df_save
}

// """標高補正
// Args:
//
//	lat(float): 推計対象地点の緯度（10進法）
//	lon(float): 推計対象地点の経度（10進法）
//	msms(Tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame, pd.DataFrame]): 4地点のMSMデータフレーム
//	msm_elevation_master(pd.DataFrame): MSM地点の標高データマスタ
//	mesh_elevation_master(pd.DataFrame): 3次メッシュの標高データ
//	mode_elevation(str, Optional): 'mesh':標高補正に3次メッシュ（1㎞メッシュ）の平均標高データを使用する
//	                               'api':国土地理院のAPIを使用する (funcault)
//
// Returns:
//
//	pd.DataFrame: 標高補正されたMSMデータフレーム
//
// """
func _get_interpolated_msm(
	lat float64,
	lon float64,
	msms [4]MsmData,
	msm_elevation_master [][]float64,
	mesh_elevation_master map[int]map[int]float64,
	mode_elevation string,
	mode_separate string) *MsmTarget {
	logger := logging.GetLogger("arcclimate")
	logger.Infof("補間計算を実行します")

	// 緯度経度から標高を取得
	ele_target := ElevationFromLatLon(
		lat,
		lon,
		mode_elevation,
		mesh_elevation_master,
	)

	// 補間計算 リストはいずれもSW南西,SE南東,NW北西,NE北東の順
	// 入力した緯度経度から周囲のMSMまでの距離を算出して、距離の重みづけ係数をリストで返す
	weights := MsmWeights(lat, lon)

	// 計算に必要なMSMを算出して、MSM位置の標高を探してリストで返す
	elevations := get_msm_elevations(lat, lon, msm_elevation_master)

	// 周囲のMSMの気象データを読み込んで標高補正後に按分する
	msm_target := _get_prportional_divided_msm_df(
		&msms,
		weights,
		elevations,
		ele_target)

	// 相対湿度・飽和水蒸気圧・露点温度の計算
	msm_target._get_relative_humidity()

	// 水平面全天日射量の直散分離
	get_separate(msm_target, lat, lon, ele_target, mode_separate)

	// 大気放射量の単位をMJ/m2に換算
	msm_target._convert_Ld_w_to_mj()

	// 夜間放射量の計算
	msm_target._get_Nocturnal_Radiation()

	return msm_target
}

// 周囲のMSMの気象データを読み込んで標高補正し加算
// Args:
//
//	msms(Tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame, pd.DataFrame]): 4地点のMSMデータフレーム(タプル)
//	weights(Tuple[float, float, float, float]): 4地点の重み(タプル)
//	elevations(Tuple[float, float, float, float]): 4地点のMSM平均標高[m](タプル)
//	ele_target: 目標地点の標高 [m]
//
// Returns:
//
//	pd.DataFrame: 標高補正により重みづけ補正されたMSMデータフレーム
func _get_prportional_divided_msm_df(
	msms *[4]MsmData,
	weights [4]float64,
	elevations [4]float64,
	ele_target float64) *MsmTarget {

	// 標高補正 (SW,SE,NW,NE)
	msm_SW := msms[0]._get_corrected_msm(elevations[0], ele_target)
	msm_SE := msms[1]._get_corrected_msm(elevations[1], ele_target)
	msm_NW := msms[2]._get_corrected_msm(elevations[2], ele_target)
	msm_NE := msms[3]._get_corrected_msm(elevations[3], ele_target)

	// 重みづけによる按分
	l := len(msm_SW.date)
	w_SW, w_SE, w_NW, w_NE := weights[0], weights[1], weights[2], weights[3]

	msm_target := MsmTarget{
		date:      make([]time.Time, l),
		TMP:       make([]float64, l),
		MR:        make([]float64, l),
		DSWRF_est: make([]float64, l),
		DSWRF_msm: make([]float64, l),
		Ld:        make([]float64, l),
		VGRD:      make([]float64, l),
		UGRD:      make([]float64, l),
		PRES:      make([]float64, l),
		APCP01:    make([]float64, l),
	}
	for i := 0; i < l; i++ {
		msm_target.date[i] = msm_SW.date[i]
		msm_target.TMP[i] = w_SW*msm_SW.TMP[i] + w_SE*msm_SE.TMP[i] + w_NW*msm_NW.TMP[i] + w_NE*msm_NE.TMP[i]
		msm_target.MR[i] = w_SW*msm_SW.MR[i] + w_SE*msm_SE.MR[i] + w_NW*msm_NW.MR[i] + w_NE*msm_NE.MR[i]
		msm_target.DSWRF_est[i] = w_SW*msm_SW.DSWRF_est[i] + w_SE*msm_SE.DSWRF_est[i] + w_NW*msm_NW.DSWRF_est[i] + w_NE*msm_NE.DSWRF_est[i]
		msm_target.DSWRF_msm[i] = w_SW*msm_SW.DSWRF_msm[i] + w_SE*msm_SE.DSWRF_msm[i] + w_NW*msm_NW.DSWRF_msm[i] + w_NE*msm_NE.DSWRF_msm[i]
		msm_target.Ld[i] = w_SW*msm_SW.Ld[i] + w_SE*msm_SE.Ld[i] + w_NW*msm_NW.Ld[i] + w_NE*msm_NE.Ld[i]
		msm_target.VGRD[i] = w_SW*msm_SW.VGRD[i] + w_SE*msm_SE.VGRD[i] + w_NW*msm_NW.VGRD[i] + w_NE*msm_NE.VGRD[i]
		msm_target.UGRD[i] = w_SW*msm_SW.UGRD[i] + w_SE*msm_SE.UGRD[i] + w_NW*msm_NW.UGRD[i] + w_NE*msm_NE.UGRD[i]
		msm_target.PRES[i] = w_SW*msm_SW.PRES[i] + w_SE*msm_SE.PRES[i] + w_NW*msm_NW.PRES[i] + w_NE*msm_NE.PRES[i]
		msm_target.APCP01[i] = w_SW*msm_SW.APCP01[i] + w_SE*msm_SE.APCP01[i] + w_NW*msm_NW.APCP01[i] + w_NE*msm_NE.APCP01[i]
	}

	return &msm_target
}

// MSMデータフレーム内の気温、気圧、重量絶対湿度を標高補正
// Args:
//
//	df_msm(pd.DataFrame): MSMデータフレーム
//	ele(float): 平均標高 [m]
//	elevation(float): 目標地点の標高 [m]
//
// Returns:
//
//	pd.DataFrame: 補正後のMSMデータフレーム
func (msm *MsmData) _get_corrected_msm(elevation float64, ele_target float64) *MsmData {

	// 標高差
	ele_gap := ele_target - elevation

	for i := 0; i < len(msm.date); i++ {

		TMP := msm.TMP[i]
		PRES := msm.PRES[i]
		MR := msm.MR[i]

		// 気温補正
		TMP_corr := CorrectTMP(TMP, ele_gap)

		// 気圧補正
		PRES_corr := CorrectPRES(PRES, ele_gap, TMP_corr)

		// 重量絶対湿度補正
		MR_corr := CorrectMR(MR, TMP_corr, PRES_corr)

		// 補正値をデータフレームに戻す
		msm.TMP[i] = TMP_corr
		msm.PRES[i] = PRES_corr
		msm.MR[i] = MR_corr
	}

	// なぜ 気圧消すのか？
	// msm.drop(['PRES'], axis=1, inplace=True)

	return msm
}

// ベクトル風速から16方位の風向風速を計算
//
// Args:
//
//	df(pd.DataFrame): MSMデータフレーム
func (msm *MsmTarget) _convert_wind16() {
	msm.w_spd = make([]float64, len(msm.date))
	msm.w_dir = make([]float64, len(msm.date))

	for i := 0; i < len(msm.date); i++ {
		// 風向風速の計算
		w_spd16, w_dir16 := Wind16(msm.UGRD[i], msm.VGRD[i])

		// 風速(16方位)
		msm.w_spd[i] = w_spd16

		// 風向(16方位)
		msm.w_dir[i] = w_dir16
	}
}

// 大気放射量の単位をW/m2からMJ/m2に換算
// Args:
//
//	df(pd.DataFrame): MSMデータフレーム
func (msm_target *MsmTarget) _convert_Ld_w_to_mj() {
	for i := 0; i < len(msm_target.date); i++ {
		msm_target.Ld[i] = msm_target.Ld[i] * (3.6 / 1000)
	}
}

// 夜間放射量[MJ/m2]の計算
// Args:
// df(pd.DataFrame): MSMデータフレーム
func (msm_target *MsmTarget) _get_Nocturnal_Radiation() {

	msm_target.NR = make([]float64, len(msm_target.date))

	sigma := 5.67 * math.Pow10(-8) // シュテファン-ボルツマン定数[W/m2・K4]
	for i := 0; i < len(msm_target.date); i++ {
		TMP := msm_target.TMP[i]
		Ld := msm_target.Ld[i]

		NR := ((sigma * math.Pow(TMP+273.15, 4)) * (3600 * math.Pow10(-6))) - Ld

		msm_target.NR[i] = NR
	}
}

// 相対湿度、飽和水蒸気圧、露点温度の計算
//
//	msm(pd.DataFrame): MSMデータフレーム
func (msm_target *MsmTarget) _get_relative_humidity() {

	msm_target.RH = make([]float64, len(msm_target.date))
	msm_target.Pw = make([]float64, len(msm_target.date))
	msm_target.DT = make([]float64, len(msm_target.date))

	for i := 0; i < len(msm_target.date); i++ {

		MR := msm_target.MR[i]
		PRES := msm_target.PRES[i]
		TMP := msm_target.TMP[i]

		RH, Pw := func_RH_eSAT(MR, TMP, PRES)

		msm_target.RH[i] = RH
		msm_target.Pw[i] = Pw

		// 露点温度が計算できない場合にはnanとする
		DT := math.NaN()

		if 6.112 <= Pw && Pw <= 123.50 {
			// 水蒸気分圧から露点温度を求める 6.112 <= Pw(hpa) <= 123.50（0～50℃）
			DT = func_DT_50(Pw)
		} else if 0.039 <= Pw && Pw <= 6.112 {
			// 水蒸気分圧から露点温度を求める 0.039 <= Pw(hpa) < 6.112（-50～0℃）
			DT = func_DT_0(Pw)
		}

		msm_target.DT[i] = DT
	}
}

// 初期化処理
//
// Args:
//
//	lat(float): 推計対象地点の緯度（10進法）
//	lon(float): 推計対象地点の経度（10進法）
//	msm_file_dir(str): MSMファイルの格納ディレクトリ
//
// Returns:
//
//	以下の要素を含む辞書
//	- msm_list(list[str]): 読み込んだMSMファイルの一覧
//	- df_msm_ele(pd.DataFrame): MSM地点の標高データ
//	- df_mesh_ele(pd.DataFrame): 3次メッシュの標高データ
//	- df_msm_list(list[pd.DataFrame]): 読み込んだデータフレームのリスト
func init_arcclimate(lat float64, lon float64, msm_file_dir string) ArcclimateConf {
	// MSM地点の標高データの読込
	df_msm_ele := read_msm_elevation()

	// 3次メッシュの標高データの読込
	mesh1d, _ := MeshCodeFromLatLon(lat, lon)
	df_mesh_ele := make(map[int]map[int]float64)
	df_mesh_ele[mesh1d] = read_3d_mesh_elevation(mesh1d)

	// MSMファイルの読込 (0.2s; 4 MSM from cache)
	MSM_list, df_msm_list := load_msm_files(lat, lon, msm_file_dir)

	return ArcclimateConf{
		MSM_list,
		df_msm_ele,
		df_mesh_ele,
		df_msm_list,
	}
}

//go:embed data/*.csv
var f embed.FS

func read_msm_elevation() [][]float64 {
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

	return elemap
}

func read_3d_mesh_elevation(meshcode_1d int) map[int]float64 {
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

	return elemap
}

type ArcclimateConf struct {
	MsmList   []string
	DfMsmEle  [][]float64             //MSM4地点の平均標高を取得するため2次メッシュコードまでの標高
	DfMeshEle map[int]map[int]float64 //ピンポイントの標高のため、3次メッシュコードまで含んだ標高
	DfMsmList []MsmData
}

func main() {
	// コマンドライン引数の処理
	parser := argparse.NewParser("ArcClimate", "Creates a design meteorological data set for any specified point")

	lat := parser.FloatPositional(&argparse.Options{
		Default: 35.658,
		Help:    "推計対象地点の緯度（10進法）"})

	lon := parser.FloatPositional(&argparse.Options{
		Default: 139.741,
		Help:    "推計対象地点の経度（10進法）"})

	filename := parser.String("o", "output", &argparse.Options{
		Default: "",
		Help:    "保存ファイルパス"})

	start_year := parser.Int("", "start_year", &argparse.Options{
		Default: 2011,
		Help:    "出力する気象データの開始年（標準年データの検討期間も兼ねる）"})

	end_year := parser.Int("", "end_year", &argparse.Options{
		Default: 2020,
		Help:    "出力する気象データの終了年（標準年データの検討期間も兼ねる）"})

	mode := parser.Selector("", "mode", []string{"normal", "EA"}, &argparse.Options{
		Default: "normal",
		Help:    "計算モードの指定 標準=normal(デフォルト), 標準年=EA"})

	format := parser.Selector("f", "file", []string{"CSV", "EPW", "HAS"}, &argparse.Options{
		Default: "CSV",
		Help:    "出力形式 CSV, EPW or HAS"})

	mode_elevation := parser.Selector("", "mode_elevation", []string{"mesh", "api"}, &argparse.Options{
		Default: "api",
		Help:    "標高判定方法 API=api(デフォルト), メッシュデータ=mesh"})

	disable_est := parser.Flag("", "disable_est", &argparse.Options{
		Help: "標準年データの検討に日射量の推計値を使用しない（使用しない場合2018年以降のデータのみで作成）"})

	msm_file_dir := parser.String("", "msm_file_dir", &argparse.Options{
		Default: ".msm_cache",
		Help:    "MSMファイルの格納ディレクトリ"})

	mode_separate := parser.Selector("", "mode_separate", []string{"Nagata", "Watanabe", "Erbs", "Udagawa", "Perez"}, &argparse.Options{
		Default: "Perez",
		Help:    "直散分離の方法"})

	log := parser.Selector("", "log", []string{"DEBUG", "INFO", "WARN", "ERROR", "CRITICAL"}, &argparse.Options{
		Default: "ERROR",
		Help:    "ログレベルの設定"})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	// ログレベル設定
	logger := logging.GetLogger("arcclimate")
	if *log == "DEBUG" {
		logger.SetLevel(logging.LevelDebug)
	} else if *log == "INFO" {
		logger.SetLevel(logging.LevelInfo)
	} else if *log == "WARN" {
		logger.SetLevel(logging.LevelWarn)
	} else if *log == "ERROR" {
		logger.SetLevel(logging.LevelError)
	} else if *log == "CRITICAL" {
		logger.SetLevel(logging.LevelCritical)
	}

	// MSMフォルダの作成
	os.MkdirAll(*msm_file_dir, os.ModePerm)

	// 初期化 (0.36s)
	conf := init_arcclimate(
		*lat,
		*lon,
		*msm_file_dir)

	// EA方式かつ日射量の推計値を使用しない場合に開始年が2018年以上となっているか確認
	if *mode == "EA" {
		if *disable_est {
			if *start_year < 2018 {
				logging.Infof("--disable_estを設定した場合は開始年を2018年以降にする必要があります")
				fmt.Fprintln(os.Stderr, "Error: If \"disable_est\" is set, the start year must be 2018 or later")
				os.Exit(1)
			} else {
				*disable_est = false
			}
		}
	}

	// 補間処理 (0.3s)
	df_save := interpolate(
		*lat,
		*lon,
		*start_year,
		*end_year,
		conf.DfMsmEle,
		conf.DfMeshEle,
		[4]MsmData(conf.DfMsmList),
		*mode_elevation,
		*mode,
		!*disable_est,
		*mode_separate)

	// 保存
	var buf *bytes.Buffer = bytes.NewBuffer([]byte{})
	if *format == "CSV" {
		df_save.to_csv(buf)
	} else if *format == "EPW" {
		df_save.to_epw(buf, *lat, *lon)
	} else if *format == "HAS" {
		df_save.to_has(buf)
	}

	if *filename == "" {
		fmt.Print(buf.String())
	} else {
		logger.Infof("CSV保存: %s", *filename)
		err := os.WriteFile(*filename, buf.Bytes(), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	logger.Infof("計算が終了しました")
}

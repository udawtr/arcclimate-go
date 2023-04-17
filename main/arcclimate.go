// ArcClimate
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/akamensky/argparse"
	"github.com/hhkbp2/go-logging"
)

// 緯度lat,経度lonで表される推計対象地点の周囲のMSMデータを利用して空間補間計算を行います。
// 標準年の計算を行う場合は mode = "EA" とし、それ以外の場合は EA = "normal" とします。
// 標準年データの検討に日射量の推計値を使用する場合は useEst = True とします。（使用しない場合2018年以降のデータのみで作成）
// 出力する気象データの期間は開始年startYearから終了年endYearまでです。ただし、標準年の計算をする場合は、検討期間として解釈します。
func Interpolate(
	lat float64,
	lon float64,
	startYear int,
	endYear int,
	modeEle string,
	mode string,
	useEst bool,
	modeSep string,
	msmFileDir string) *MsmTarget {

	log.Printf("データ読み込み")

	// MSM地点の標高データの読込
	ele := NewElevationMaster(lat, lon)

	// 必要なMSMファイル名の一覧を緯度経度から取得
	msmList := RequiredMsmList(lat, lon)

	// MSMファイルの読込 (0.2s; 4 MSM from cache)
	msms := LoadMsmFiles(msmList, msmFileDir)

	log.Printf("補正計算")

	// 周囲4地点のMSMデータフレームから標高補正したMSMデータフレームを作成
	msm := PrportionalDivided(lat, lon, msms, ele, modeEle, modeSep)

	if mode == "normal" {
		// 保存用に年月日をフィルタ
		return msm.ExctactMsmYear(startYear, endYear)
	} else if mode == "EA" {
		// 標準年の計算
		log.Printf("標準年計算 %d-%d", startYear, endYear)
		return msm.EA(startYear, endYear, useEst)
	}

	panic(mode)
}

// 緯度 lat, 経度 lon の周囲4地点のメッシュ地点番号を返します。
func RequiredMsmList(lat float64, lon float64) []string {
	MSM_S, MSM_N, MSM_W, MSM_E := Meshcode1d(lat, lon)

	// 周囲4地点のメッシュ地点番号
	MSM_SW := fmt.Sprintf("%d-%d", MSM_S, MSM_W)
	MSM_SE := fmt.Sprintf("%d-%d", MSM_S, MSM_E)
	MSM_NW := fmt.Sprintf("%d-%d", MSM_N, MSM_W)
	MSM_NE := fmt.Sprintf("%d-%d", MSM_N, MSM_E)

	return []string{MSM_SW, MSM_SE, MSM_NW, MSM_NE}
}

// 緯度 lat, 経度 lon から計算に必要なMSMを決定し、各地点の標高を返す。
// 各MSMファイルの標高は eleMstr に格納されている値を使用する。
func Elevations(lat float64, lon float64, eleMstr *ElevationMaster) [4]float64 {

	MSM_S, MSM_N, MSM_W, MSM_E := Meshcode1d(lat, lon)

	ele_SW := eleMstr.Elevation2d(MSM_S, MSM_W) // SW
	ele_SE := eleMstr.Elevation2d(MSM_S, MSM_E) // SE
	ele_NW := eleMstr.Elevation2d(MSM_N, MSM_W) // NW
	ele_NE := eleMstr.Elevation2d(MSM_N, MSM_E) // NE

	return [4]float64{ele_SW, ele_SE, ele_NW, ele_NE}
}

// 緯度 lat, 経度 lon の標高補正を行います。
func PrportionalDivided(
	lat float64,
	lon float64,
	msms MsmDataSet,
	eleMstr *ElevationMaster,
	modeEle string,
	modeSep string) *MsmTarget {
	logger := logging.GetLogger("arcclimate")
	logger.Infof("補間計算を実行します")

	// 緯度経度から標高を取得
	ele_target := ElevationFromLatLon(
		lat,
		lon,
		modeEle,
		eleMstr,
	)

	// 補間計算 リストはいずれもSW南西,SE南東,NW北西,NE北東の順
	// 入力した緯度経度から周囲のMSMまでの距離を算出して、距離の重みづけ係数をリストで返す
	weights := MsmWeights(lat, lon)

	// 計算に必要なMSMを算出して、MSM位置の標高を探してリストで返す
	elevations := Elevations(lat, lon, eleMstr)

	// 周囲のMSMの気象データを読み込んで標高補正後に按分する
	log.Print("周囲のMSMの気象データを読み込んで標高補正後に按分する")
	msm_target := msms.PrportionalDivided(weights, elevations, ele_target)

	// 相対湿度・飽和水蒸気圧・露点温度の計算
	log.Print("相対湿度・飽和水蒸気圧・露点温度の計算")
	msm_target.RH_Pw_DT()

	// 水平面全天日射量の直散分離
	log.Print("水平面全天日射量の直散分離")
	msm_target.SeparateSolarRadiation(lat, lon, ele_target, modeSep)

	// 大気放射量の単位をMJ/m2に換算
	log.Print("大気放射量の単位をMJ/m2に換算")
	msm_target.ConvertLdUnit()

	// 夜間放射量の計算
	log.Print("夜間放射量の計算")
	msm_target.CalcNocturnalRadiation()

	// ベクトル風速から16方位の風向風速を計算
	log.Print("ベクトル風速から16方位の風向風速を計算")
	msm_target.WindVectorToDirAndSpeed()

	return msm_target
}

// 周囲のMSMの気象データから目標地点(標高 ele_target [m])の気象データを作成する。
// 按分には、目標地点と各周辺の地点の平均標高 elevations [m] と 地点間の距離から求めた重み weights を用いる。
func (msms *MsmDataSet) PrportionalDivided(
	weights [4]float64,
	elevations [4]float64,
	ele_target float64) *MsmTarget {

	// 標高補正 (SW,SE,NW,NE)
	msm_SW := msms.Data[0].CorrectedMsm_TMP_PRES_MR(elevations[0], ele_target)
	msm_SE := msms.Data[1].CorrectedMsm_TMP_PRES_MR(elevations[1], ele_target)
	msm_NW := msms.Data[2].CorrectedMsm_TMP_PRES_MR(elevations[2], ele_target)
	msm_NE := msms.Data[3].CorrectedMsm_TMP_PRES_MR(elevations[3], ele_target)

	// 重みづけによる按分
	l := msm_SW.Length()
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
		row_SW := msm_SW.Rows[i]
		row_SE := msm_SE.Rows[i]
		row_NW := msm_NW.Rows[i]
		row_NE := msm_NE.Rows[i]
		msm_target.date[i] = row_SW.date
		msm_target.TMP[i] = w_SW*row_SW.TMP + w_SE*row_SE.TMP + w_NW*row_NW.TMP + w_NE*row_NE.TMP
		msm_target.MR[i] = w_SW*row_SW.MR + w_SE*row_SE.MR + w_NW*row_NW.MR + w_NE*row_NE.MR
		msm_target.DSWRF_est[i] = w_SW*row_SW.DSWRF_est + w_SE*row_SE.DSWRF_est + w_NW*row_NW.DSWRF_est + w_NE*row_NE.DSWRF_est
		msm_target.DSWRF_msm[i] = w_SW*row_SW.DSWRF_msm + w_SE*row_SE.DSWRF_msm + w_NW*row_NW.DSWRF_msm + w_NE*row_NE.DSWRF_msm
		msm_target.Ld[i] = w_SW*row_SW.Ld + w_SE*row_SE.Ld + w_NW*row_NW.Ld + w_NE*row_NE.Ld
		msm_target.VGRD[i] = w_SW*row_SW.VGRD + w_SE*row_SE.VGRD + w_NW*row_NW.VGRD + w_NE*row_NE.VGRD
		msm_target.UGRD[i] = w_SW*row_SW.UGRD + w_SE*row_SE.UGRD + w_NW*row_NW.UGRD + w_NE*row_NE.UGRD
		msm_target.PRES[i] = w_SW*row_SW.PRES + w_SE*row_SE.PRES + w_NW*row_NW.PRES + w_NE*row_NE.PRES
		msm_target.APCP01[i] = w_SW*row_SW.APCP01 + w_SE*row_SE.APCP01 + w_NW*row_NW.APCP01 + w_NE*row_NE.APCP01
	}

	return &msm_target
}

// 大気放射量 Ld の単位をW/m2からMJ/m2に換算
func (msm_target *MsmTarget) ConvertLdUnit() {
	for i := 0; i < len(msm_target.date); i++ {
		msm_target.Ld[i] = msm_target.Ld[i] * (3.6 / 1000)
	}
}

var sigma float64

func init() {
	sigma = 5.67 * 0.00000001 // シュテファン-ボルツマン定数[W/m2・K4]
}

// 夜間放射量 NR [MJ/m2] を 気温 TMP [℃], 大気放射量 Ld [MJ/m2] から計算
func (msm_target *MsmTarget) CalcNocturnalRadiation() {

	msm_target.NR = make([]float64, len(msm_target.date))

	for i := 0; i < len(msm_target.date); i++ {
		TMP := msm_target.TMP[i]
		Ld := msm_target.Ld[i]

		NR := ((sigma * pow4(TMP+273.15)) * (3600 * 0.000001)) - Ld

		msm_target.NR[i] = NR
	}
}

func pow4(v float64) float64 {
	return v * v * v * v
}

func main() {
	log.SetFlags(log.Lmicroseconds)

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

	startYear := parser.Int("", "start_year", &argparse.Options{
		Default: 2011,
		Help:    "出力する気象データの開始年（標準年データの検討期間も兼ねる）"})

	endYear := parser.Int("", "end_year", &argparse.Options{
		Default: 2020,
		Help:    "出力する気象データの終了年（標準年データの検討期間も兼ねる）"})

	mode := parser.Selector("", "mode", []string{"normal", "EA"}, &argparse.Options{
		Default: "normal",
		Help:    "計算モードの指定 標準=normal(デフォルト), 標準年=EA"})

	format := parser.Selector("f", "file", []string{"CSV", "EPW", "HAS"}, &argparse.Options{
		Default: "CSV",
		Help:    "出力形式 CSV, EPW or HAS"})

	modeEle := parser.Selector("", "mode_elevation", []string{"mesh", "api"}, &argparse.Options{
		Default: "api",
		Help:    "標高判定方法 API=api(デフォルト), メッシュデータ=mesh"})

	disableEst := parser.Flag("", "disable_est", &argparse.Options{
		Help: "標準年データの検討に日射量の推計値を使用しない（使用しない場合2018年以降のデータのみで作成）"})

	msmFileDir := parser.String("", "msm_file_dir", &argparse.Options{
		Default: ".msm_cache",
		Help:    "MSMファイルの格納ディレクトリ"})

	modeSep := parser.Selector("", "mode_separate", []string{"Nagata", "Watanabe", "Erbs", "Udagawa", "Perez"}, &argparse.Options{
		Default: "Perez",
		Help:    "直散分離の方法"})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	// MSMフォルダの作成
	os.MkdirAll(*msmFileDir, os.ModePerm)

	// EA方式かつ日射量の推計値を使用しない場合に開始年が2018年以上となっているか確認
	if *mode == "EA" {
		if *disableEst {
			if *startYear < 2018 {
				log.Printf("--disable_estを設定した場合は開始年を2018年以降にする必要があります")
				fmt.Fprintln(os.Stderr, "Error: If \"disable_est\" is set, the start year must be 2018 or later")
				os.Exit(1)
			} else {
				*disableEst = false
			}
		}
	}

	// 補間処理 (0.3s)
	res := Interpolate(
		*lat,
		*lon,
		*startYear,
		*endYear,
		*modeEle,
		*mode,
		!*disableEst,
		*modeSep,
		*msmFileDir,
	)

	// 保存
	var buf *bytes.Buffer = bytes.NewBuffer([]byte{})
	if *format == "CSV" {
		res.to_csv(buf)
	} else if *format == "EPW" {
		res.to_epw(buf, *lat, *lon)
	} else if *format == "HAS" {
		res.to_has(buf)
	}

	if *filename == "" {
		fmt.Print(buf.String())
	} else {
		log.Printf("CSV保存: %s", *filename)
		err := os.WriteFile(*filename, buf.Bytes(), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	log.Printf("計算が終了しました")
}

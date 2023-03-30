package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/akamensky/argparse"
	"github.com/hhkbp2/go-logging"
)

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

//直散分離に関するモジュール

type AAA struct {
	SH float64 //水平面天空日射量
	DN float64 //法線面直達日射量
	DT float64 //露点温度
}

// 推定結果データ
type MsmTarget struct {
	date []time.Time //1.参照時刻。日本標準時JST

	//標高補正のみのデータ項目
	TMP       []float64 //2.参照時刻時点の気温の瞬時値 (単位:℃)
	MR        []float64 //3.参照時刻時点の重量絶対湿度の瞬時値 (単位:g/kgDA)
	DSWRF_est []float64 //4.参照時刻の前1時間の推定日射量の積算値 (単位:MJ/m2)
	DSWRF_msm []float64 //5.参照時刻の前1時間の日射量の積算値 (単位:MJ/m2)
	Ld        []float64 //6.大気放射量 W/m2 or MJ/m2
	VGRD      []float64 //7.南北風(V軸) (単位:m/s)
	UGRD      []float64 //8.東西風(U軸) (単位:m/s)
	PRES      []float64 //9.気圧 (単位:hPa)
	APCP01    []float64 //10.参照時刻の前1時間の降水量の積算値 (単位:mm/h)

	DSWRF []float64 //標準年の計算時に使用するDSWRF

	//追加項目
	w_spd []float64 //11.参照時刻時点の風速の瞬時値 (単位:m/s)
	w_dir []float64 //12.参照時刻時点の風向の瞬時値 (単位:°)

	NR []float64 //夜間放射量[MJ/m2]

	RH []float64 //	float64: 相対湿度[%]
	Pw []float64 //	float64: 水蒸気分圧 [hpa]

	//計算対象時刻の露点温度(℃)
	DT []float64

	//直散分離
	AAA_est []AAA
	AAA_msm []AAA
}

func interpolate(
	lat float64,
	lon float64,
	start_year int,
	end_year int,
	msm_elevation_master [][]float64,
	mesh_elevation_master map[int]float64,
	msms [4]MsmData,
	mode_elevation string,
	mode string,
	use_est bool,
	mode_separate string) *MsmTarget {
	// """対象地点の周囲のMSMデータを利用して空間補間計算を行う

	// Args:
	//   lat(float): 推計対象地点の緯度（10進法）
	//   lon(float): 推計対象地点の経度（10進法）
	//   start_year(int): 出力する気象データの開始年（標準年データの検討期間も兼ねる）
	//   end_year(int): 出力する気象データの終了年（標準年データの検討期間も兼ねる）
	//   msm_elevation_master(pd.DataFrame):  MSM地点の標高データ
	//   mesh_elevation_master(pd.DataFrame): 3次メッシュの標高データ
	//   msms(Tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame, pd.DataFrame]): 4地点のMSMデータ
	//   mode_elevation(str, Optional): 'mesh':標高補正に3次メッシュ（1㎞メッシュ）の平均標高データを使用する
	//                        'api':国土地理院のAPIを使用する
	//   mode(str, Optional): "normal"=補正のみ
	//                        "EA"=拡張アメダス方式に準じた標準年データを作成する (funcault value = 'api')
	//   use_est(bool, Optional): 標準年データの検討に日射量の推計値を使用する（使用しない場合2018年以降のデータのみで作成） (funcault value = True)

	// Returns:
	//   pd.DataFrame: MSMデータフレーム
	// """

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
	_convert_wind16(msm)

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
			NR:        msm.NR[start_index:end_index],
			w_spd:     msm.w_spd[start_index:end_index],
			w_dir:     msm.w_dir[start_index:end_index],
		}

	} else if mode == "EA" {
		// 標準年の計算
		df_save, _ = calc_EA(
			msm,
			start_year,
			end_year,
			use_est)

		// ベクトル風速から16方位の風向風速を再計算
		_convert_wind16(df_save)
	} else {
		panic(mode)
	}

	return df_save
}

func _get_interpolated_msm(
	lat float64,
	lon float64,
	msms [4]MsmData,
	msm_elevation_master [][]float64,
	mesh_elevation_master map[int]float64,
	mode_elevation string,
	mode_separate string) *MsmTarget {
	// """標高補正

	// Args:
	//   lat(float): 推計対象地点の緯度（10進法）
	//   lon(float): 推計対象地点の経度（10進法）
	//   msms(Tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame, pd.DataFrame]): 4地点のMSMデータフレーム
	//   msm_elevation_master(pd.DataFrame): MSM地点の標高データマスタ
	//   mesh_elevation_master(pd.DataFrame): 3次メッシュの標高データ
	//   mode_elevation(str, Optional): 'mesh':標高補正に3次メッシュ（1㎞メッシュ）の平均標高データを使用する
	//                                  'api':国土地理院のAPIを使用する (funcault)

	// Returns:
	//   pd.DataFrame: 標高補正されたMSMデータフレーム
	// """
	logger := logging.GetLogger("arcclimate")
	logger.Infof("補間計算を実行します")

	// 緯度経度から標高を取得
	ele_target := get_latlon_elevation(
		lat,
		lon,
		mode_elevation,
		mesh_elevation_master,
	)

	// 補間計算 リストはいずれもSW南西,SE南東,NW北西,NE北東の順
	// 入力した緯度経度から周囲のMSMまでの距離を算出して、距離の重みづけ係数をリストで返す
	weights := get_msm_weights(lat, lon)

	// 計算に必要なMSMを算出して、MSM位置の標高を探してリストで返す
	elevations := get_msm_elevations(lat, lon, msm_elevation_master)

	// 周囲のMSMの気象データを読み込んで標高補正後に按分する
	msm_target := _get_prportional_divided_msm_df(
		&msms,
		weights,
		elevations,
		ele_target)

	// 相対湿度・飽和水蒸気圧・露点温度の計算
	_get_relative_humidity(msm_target)

	// 水平面全天日射量の直散分離
	get_separate(msm_target, lat, lon, ele_target, mode_separate)

	// 大気放射量の単位をMJ/m2に換算
	_convert_Ld_w_to_mj(msm_target)

	// 夜間放射量の計算
	_get_Nocturnal_Radiation(msm_target)

	return msm_target
}

func _get_prportional_divided_msm_df(
	msms *[4]MsmData,
	weights [4]float64,
	elevations [4]float64,
	ele_target float64) *MsmTarget {
	// """周囲のMSMの気象データを読み込んで標高補正し加算

	// Args:
	//   msms(Tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame, pd.DataFrame]): 4地点のMSMデータフレーム(タプル)
	//   weights(Tuple[float, float, float, float]): 4地点の重み(タプル)
	//   elevations(Tuple[float, float, float, float]): 4地点のMSM平均標高[m](タプル)
	//   ele_target: 目標地点の標高 [m]

	// Returns:
	//   pd.DataFrame: 標高補正により重みづけ補正されたMSMデータフレーム
	// """

	// 標高補正 (SW,SE,NW,NE)
	msm_SW := _get_corrected_msm(&msms[0], elevations[0], ele_target)
	msm_SE := _get_corrected_msm(&msms[1], elevations[1], ele_target)
	msm_NW := _get_corrected_msm(&msms[2], elevations[2], ele_target)
	msm_NE := _get_corrected_msm(&msms[3], elevations[3], ele_target)

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

func _get_corrected_msm(msm *MsmData, elevation float64, ele_target float64) *MsmData {
	// """MSMデータフレーム内の気温、気圧、重量絶対湿度を標高補正

	// Args:
	//   df_msm(pd.DataFrame): MSMデータフレーム
	//   ele(float): 平均標高 [m]
	//   elevation(float): 目標地点の標高 [m]

	// Returns:
	//   pd.DataFrame: 補正後のMSMデータフレーム
	// """

	// 標高差
	ele_gap := ele_target - elevation

	for i := 0; i < len(msm.date); i++ {

		TMP := msm.TMP[i]
		PRES := msm.PRES[i]
		MR := msm.MR[i]

		// 気温補正
		TMP_corr := get_corrected_TMP(TMP, ele_gap)

		// 気圧補正
		PRES_corr := get_corrected_PRES(PRES, ele_gap, TMP_corr)

		// 重量絶対湿度補正
		MR_corr := get_corrected_mixing_ratio(MR, TMP_corr, PRES_corr)

		// 補正値をデータフレームに戻す
		msm.TMP[i] = TMP_corr
		msm.PRES[i] = PRES_corr
		msm.MR[i] = MR_corr
	}

	// なぜ 気圧消すのか？
	// msm.drop(['PRES'], axis=1, inplace=True)

	return msm
}

func _convert_wind16(msm *MsmTarget) {
	// """ベクトル風速から16方位の風向風速を計算

	// Args:
	//   df(pd.DataFrame): MSMデータフレーム
	// """

	msm.w_spd = make([]float64, len(msm.date))
	msm.w_dir = make([]float64, len(msm.date))

	for i := 0; i < len(msm.date); i++ {
		// 風向風速の計算
		w_spd16, w_dir16 := get_wind16(msm.UGRD[i], msm.VGRD[i])

		// 風速(16方位)
		msm.w_spd[i] = w_spd16

		// 風向(16方位)
		msm.w_dir[i] = w_dir16
	}
}

func _convert_Ld_w_to_mj(msm_target *MsmTarget) {
	// """大気放射量の単位をW/m2からMJ/m2に換算

	// Args:
	//   df(pd.DataFrame): MSMデータフレーム
	// """

	for i := 0; i < len(msm_target.date); i++ {
		msm_target.Ld[i] = msm_target.Ld[i] * (3.6 / 1000)
	}
}

func _get_Nocturnal_Radiation(msm_target *MsmTarget) {
	// """夜間放射量[MJ/m2]の計算
	// Args:
	// df(pd.DataFrame): MSMデータフレーム
	// """

	msm_target.NR = make([]float64, len(msm_target.date))

	sigma := 5.67 * math.Pow10(-8) // シュテファン-ボルツマン定数[W/m2・K4]
	for i := 0; i < len(msm_target.date); i++ {
		TMP := msm_target.TMP[i]
		Ld := msm_target.Ld[i]

		NR := ((sigma * math.Pow(TMP+273.15, 4)) * (3600 * math.Pow10(-6))) - Ld

		msm_target.NR[i] = NR
	}
}

func _get_relative_humidity(msm_target *MsmTarget) {
	// """相対湿度、飽和水蒸気圧、露点温度の計算
	//   msm(pd.DataFrame): MSMデータフレーム
	// """

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

func init_arcclimate(lat float64, lon float64, path_MSM_ele string, path_mesh_ele string, msm_file_dir string) ArcclimateConf {
	// """初期化処理

	// Args:
	//   lat(float): 推計対象地点の緯度（10進法）
	//   lon(float): 推計対象地点の経度（10進法）
	//   path_MSM_ele(str): MSM地点の標高データのファイルパス
	//   path_mesh_ele(str): 3次メッシュの標高データのファイルパス
	//   msm_file_dir(str): MSMファイルの格納ディレクトリ

	// Returns:
	//   以下の要素を含む辞書
	//   - msm_list(list[str]): 読み込んだMSMファイルの一覧
	//   - df_msm_ele(pd.DataFrame): MSM地点の標高データ
	//   - df_mesh_ele(pd.DataFrame): 3次メッシュの標高データ
	//   - df_msm_list(list[pd.DataFrame]): 読み込んだデータフレームのリスト
	// """

	// ロガーの作成
	logger := logging.GetLogger("arcclimate")

	// MSM地点の標高データの読込
	logger.Infof("MSM地点の標高データ読込: %s", path_MSM_ele)
	df_msm_ele := read_msm_elevation(path_MSM_ele)

	// 3次メッシュの標高データの読込
	logger.Infof("3次メッシュの標高データ読込: %s", path_mesh_ele)
	df_mesh_ele := read_3d_mesh_elevation(path_mesh_ele)

	// MSMファイルの読込
	MSM_list, df_msm_list := load_msm_files(lat, lon, msm_file_dir)

	return ArcclimateConf{
		MSM_list,
		df_msm_ele,
		df_mesh_ele,
		df_msm_list,
	}
}

func read_msm_elevation(path_MSM_ele string) [][]float64 {
	// Open the CSV file
	file, err := os.Open(path_MSM_ele)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

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

func read_3d_mesh_elevation(path_mesh_ele string) map[int]float64 {
	// Open the CSV file
	file, err := os.Open(path_mesh_ele)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

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
	DfMsmEle  [][]float64
	DfMeshEle map[int]float64
	DfMsmList []MsmData
}

func to_has(df *MsmTarget, out *bytes.Buffer) {
	// """HASP形式への変換

	// Args:
	//   df(pd.DataFrame): MSMデータフレーム
	//   out(io.StringIO): 出力先のテキストストリーム

	// Note:
	//   法線面直達日射量、水平面天空日射量、水平面夜間日射量は0を出力します。
	//   曜日の祝日判定を行っていません。
	// """

	for d := 0; d < 365; d++ {
		off := d * 24

		// 年,月,日,曜日
		year := df.date[off].Year() % 100
		month := df.date[off].Month()
		day := df.date[off].Day()
		weekday := df.date[off].Weekday() + 2 // 月2,...,日8
		if weekday == 8 {                     // 日=>1
			weekday = 1
		}
		// 注)祝日は処理していない

		// 2列	2列	2列	1列
		// 年	月	日	曜日
		day_signature := fmt.Sprintf("%2d%2d%2d%1d", year, month, day, weekday)

		// 外気温 (×0.1℃-50℃)
		for h := 0; h < 24; h++ {
			TMP := int(df.TMP[off+h]*10) + 50
			out.Write([]byte(fmt.Sprintf("%3d", TMP)))
		}
		out.Write([]byte(fmt.Sprintf("%s1\n", day_signature)))

		// 絶対湿度 (0.1g/kg(DA))
		for h := 0; h < 24; h++ {
			MR := int(df.MR[off+h] * 10)
			out.Write([]byte(fmt.Sprintf("%3d", MR)))
		}
		out.Write([]byte(fmt.Sprintf("%s2\n", day_signature)))

		// 日射量
		out.Write([]byte(fmt.Sprintf("  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0%s3\n", day_signature)))
		out.Write([]byte(fmt.Sprintf("  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0%s4\n", day_signature)))
		out.Write([]byte(fmt.Sprintf("  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0  0%s5\n", day_signature)))

		// 風向 (0:無風,1:NNE,...,16:N)
		for h := 0; h < 24; h++ {
			w_dir := int(df.w_dir[off+h]/22.5) + 1
			if w_dir == 0 {
				// 真北の場合を0から16へ変更
				w_dir = 16
			}
			if df.w_spd[off+h] == 0 {
				w_dir = 0 // 無風の場合は0
			}

			out.Write([]byte(fmt.Sprintf("%3d", w_dir)))
		}
		out.Write([]byte(fmt.Sprintf("%s6\n", day_signature)))

		// 風速 (0.1m/s)
		for h := 0; h < 24; h++ {
			w_spd := int(df.w_dir[off+h] * 10)
			out.Write([]byte(fmt.Sprintf("%3d", w_spd)))
		}
		out.Write([]byte(fmt.Sprintf("%s7\n", day_signature)))
	}
}

func to_epw(msm *MsmTarget, out *bytes.Buffer, lat float64, lon float64) {
	// """初期化処理

	// Args:
	//   df(pd.DataFrame): MSMデータフレーム
	//   out(io.StringIO): 出力先のテキストストリーム
	//   lat(float): 推計対象地点の緯度（10進法）
	//   lon(float): 推計対象地点の経度（10進法）

	// Note:
	//   "EnergyPlus Auxilary Programs"を参考に記述されました。
	//   外気温(単位:℃)、風向(単位:°)、風速(単位:m/s)、降水量の積算値(単位:mm/h)のみを出力します。
	//   それ以外の値については、"missing"に該当する値を出力します。
	// """

	// LOCATION
	// 国名,緯度,経度,タイムゾーンのみ出力
	out.Write([]byte(fmt.Sprintf("LOCATION,-,-,JPN,-,-,%.2f,%.2f,9.0,0.0\n", lat, lon)))

	// DESIGN CONDITION
	// 設計条件なし
	out.Write([]byte("DESIGN CONDITIONS,0\n"))

	// TYPICAL/EXTREME PERIODS
	// 期間指定なし
	out.Write([]byte("TYPICAL/EXTREME PERIODS,0\n"))

	// GROUND TEMPERATURES
	// 地中温度無し
	out.Write([]byte("GROUND TEMPERATURES,0\n"))

	// HOLIDAYS/DAYLIGHT SAVINGS
	// 休日/サマータイム
	out.Write([]byte("HOLIDAYS/DAYLIGHT SAVINGS,No,0,0,0\n"))

	// COMMENT 1
	out.Write([]byte("COMMENTS 1\n"))

	// COMMENT 2
	out.Write([]byte("COMMENTS 2\n"))

	// DATA HEADER
	out.Write([]byte("DATA PERIODS,1,1,Data,Sunday,1/1,12/31\n"))

	for i := 0; i < len(msm.date); i++ {
		// N1: 年
		// N2: 月
		// N3: 日
		// N4: 時
		// N5: 分 = 0
		// N6: Dry Bulb Temperature
		// N7-N19: missing
		// N20: w_dir
		// N21: w_spd
		// N22-N32: missing
		// N33: APCP01
		// N34: missing
		out.Write([]byte(fmt.Sprintf("%d,%d,%d,%d,60,-,%.1f,99.9,999,999999,999,9999,9999,9999,9999,9999,999999,999999,999999,9999,%d,%.1f,99,99,9999,99999,9,999999999,999,0.999,999,99,999,%.1f,99\n", msm.date[i].Year(), msm.date[i].Month(), msm.date[i].Day(), msm.date[i].Hour()+1, msm.TMP[i], int(msm.w_dir[i]), msm.w_spd[i], msm.APCP01[i])))
	}
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

	// 初期化
	conf := init_arcclimate(
		*lat,
		*lon,
		filepath.Join("data", "MSM_elevation.csv"),
		filepath.Join("data", "mesh_3d_elevation.csv"),
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

	// 補間処理
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
	_ = df_save

	// 保存
	var buf *bytes.Buffer = bytes.NewBuffer([]byte{})
	if *format == "CSV" {
		//Write Header
		buf.WriteString("date")
		buf.WriteString(",TMP")
		buf.WriteString(",MR")
		if df_save.DSWRF_est != nil {
			buf.WriteString(",DSWRF_est")
		}
		if df_save.DSWRF_msm != nil {
			buf.WriteString(",DSWRF_msm")
		}
		buf.WriteString(",Ld")
		buf.WriteString(",VGRD")
		buf.WriteString(",UGRD")
		buf.WriteString(",PRES")
		buf.WriteString(",APCP01")
		buf.WriteString(",RH")
		buf.WriteString(",Pw")
		if df_save.DT != nil {
			buf.WriteString(",DT")
		}
		// buf.WriteString(",DN_est")
		// buf.WriteString(",SH_est")
		// buf.WriteString(",DN_msm")
		// buf.WriteString(",SH_msm")
		if df_save.NR != nil {
			buf.WriteString(",NR")
		}
		buf.WriteString(",w_spd")
		buf.WriteString(",w_dir")
		buf.WriteString("\n")

		//Write Data
		writeFloat := func(v float64) {
			buf.WriteString(",")
			buf.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		}
		for i := 0; i < len(df_save.date); i++ {
			buf.WriteString(df_save.date[i].Format("2006-01-02 15:04:05"))
			writeFloat(df_save.TMP[i])
			writeFloat(df_save.MR[i])
			if df_save.DSWRF_est != nil {
				writeFloat(df_save.DSWRF_est[i])
			}
			if df_save.DSWRF_msm != nil {
				writeFloat(df_save.DSWRF_msm[i])
			}
			writeFloat(df_save.Ld[i])
			writeFloat(df_save.VGRD[i])
			writeFloat(df_save.UGRD[i])
			writeFloat(df_save.PRES[i])
			writeFloat(df_save.APCP01[i])
			writeFloat(df_save.RH[i])
			writeFloat(df_save.Pw[i])
			if df_save.DT != nil {
				writeFloat(df_save.DT[i])
			}
			if df_save.NR != nil {
				writeFloat(df_save.NR[i])
			}
			writeFloat(df_save.w_spd[i])
			writeFloat(df_save.w_dir[i])
			buf.WriteString("\n")
		}
		// // u,v軸のベクトル風データのフィルタ
		// if !vector_wind {
		// 	// df_save.drop(['VGRD', 'UGRD'], axis=1, inplace=True)
		// }
		// df_save.to_csv(out, line_terminator='\n')
	} else if *format == "EPW" {
		to_epw(df_save, buf, *lat, *lon)
	} else if *format == "HAS" {
		to_has(df_save, buf)
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

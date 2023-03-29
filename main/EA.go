package main

import (
	"math"
	"sort"
	"time"
)

//標準年の計算モジュール

// 拡張アメダス(MetDS(株)気象データシステム社)の
// 標準年データの2010年版の作成方法を参考とした
// ※2020年版は作成方法が変更されている
//
// 参考文献
// 二宮 秀與 他
// 外皮・躯体と設備・機器の総合エネルギーシミュレーションツール
// 「BEST」の開発(その172)30 年拡張アメダス気象データ
// 空気調和・衛生工学会大会 学術講演論文集 2016.5 (0), 13-16, 2016

func calc_EA(df_msm *MsmTarget, start_year int, end_year int, use_est bool) (*MsmTarget, []int) {
	// """標準年の計算
	// Args:
	//   df(pd.DataFrame): MSMデータフレーム
	//   start_year(int): 標準年データの検討開始年
	//   end_year(int): 標準年データの検討終了年
	//   use_est(bool): 標準年データの検討に日射量の推計値を使用する(使用しない場合2018年以降のデータのみで作成)

	// Returns:
	//   Tuple[pd.DataFrame, List[int]]: 標準年MSMデータフレームおよび選択された年のリストのタプル
	// """

	var df_targ MsmTarget

	if use_est {
		// * 標準年データの検討に日射量の推計値を使用する
		//   -> `DSWRF_msm`列を削除し、`DSWRF_est`列を`DSWRF`列へ変更(推計値データを採用)
		df_msm.DSWRF = append([]float64{}, df_msm.DSWRF_est...)

		local, _ := time.LoadLocation("Local")
		start_time := time.Date(start_year, 1, 1, 0, 0, 0, 0, local)
		end_time := time.Date(end_year, 12, 31, 23, 0, 0, 0, local)
		df_targ = exctactMsm(df_msm, start_time, end_time)

		// TODO: drop, rename処理はcopyの後の方がよさそう

	} else {
		// * 2018年以降のデータのみで作成
		// * `DSWRF_est`列を削除し、`DSWRF_msm`列を`DSWRF`列へ変更(MSMデータを採用)
		df_msm.DSWRF = append([]float64{}, df_msm.DSWRF_msm...)

		if start_year < 2018 {
			start_year = 2018
		}

		local, _ := time.LoadLocation("Local")
		start_time := time.Date(start_year, 1, 1, 0, 0, 0, 0, local)
		end_time := time.Date(end_year, 12, 31, 23, 0, 0, 0, local)
		df_targ = exctactMsm(df_msm, start_time, end_time)

		// TODO: drop, rename処理はcopyの後の方がよさそう
		// TODO: copy処理は if/elseの両方で同じに見える
	}

	// ベクトル風速`w_spd`, 16方位の風向風速`w_dir`の列を削除
	df_msm.w_spd = nil
	df_msm.w_dir = nil

	// groupのターゲットを追加
	// カラム [TMP, MR, DSWF, Ld, VGRD, UGRD, PRES, APCP01, MR_sat, w_spd, w_dir]
	//   ↓
	// カラム [TMP, MR, DSWF, Ld, VGRD, UGRD, PRES, APCP01, MR_sat, w_spd, w_dir, y, m, d]
	//                            TMP        MR  DSWRF        Ld      VGRD      UGRD          PRES    APCP01    MR_sat     w_spd  w_dir     y   m   d
	// date
	// 2011-01-01 00:00:00  -6.776232  1.969510    0.0  0.772116 -3.051297  0.837700  98383.696285  0.021976  2.344268  3.139605  337.5  2011  01  01
	// 2011-01-01 01:00:00  -5.862659  2.085005    0.0  0.793957 -3.203325  0.251010  98437.952674  0.001623  2.511178  3.203325  360.0  2011  01  01
	// df_targ["y"] = df_targ.index.strftime('%Y')
	// df_targ["m"] = df_targ.index.strftime('%m')
	// df_targ["d"] = df_targ.index.strftime('%d')

	// df_temp : 月平均値による信頼区間の判定
	//         y   m   TMP  DSWRF    MR  APCP01  w_spd
	// 0    2011  01  True   True  True    True   True
	// 1    2012  01  True   True  True    True   True
	// 2    2013  01  True   True  True    True   True
	df_temp_ci := get_temp_ci(&df_targ)

	// df_fs : fs計算による信頼区間の判定
	//         y   m    TMP  DSWRF     MR  APCP01  w_spd    TMP_FS
	// 0    2011  01  False  False  False   False  False  0.076795
	// 1    2012  01  False   True  False    True  False  0.149116
	df_fs_ci := get_fs_ci(&df_targ)

	// 信頼区間の判定結果を合成
	df_ci := make(map[YearMonth]GroupData4, 120)
	for ym := range df_temp_ci {
		df_ci[ym] = GroupData4{
			TMP_mean:    df_temp_ci[ym].TMP,
			TMP_dev:     df_temp_ci[ym].TMP_dev,
			DSWRF_mean:  df_temp_ci[ym].DSWRF,
			MR_mean:     df_temp_ci[ym].MR,
			APCP01_mean: df_temp_ci[ym].APCP01,
			w_spd_mean:  df_temp_ci[ym].w_spd,
			TMP_fs:      df_fs_ci[ym].TMP,
			DSWRF_fs:    df_fs_ci[ym].DSWRF,
			MR_fs:       df_fs_ci[ym].MR,
			APCP01_fs:   df_fs_ci[ym].APCP01,
			w_spd_fs:    df_fs_ci[ym].w_spd,
		}
	}

	// 月別に代表的な年を取得
	rep_years := _get_representative_years(df_ci)

	// 月別に代表的な年から接合した1年間のデータを作成
	df_patchwork := patch_representataive_years(df_msm, rep_years)

	// 接合部の円滑化
	for _, v := range get_smoothing_months(rep_years) {
		target := v.TargetMonth
		before_year := v.BeforeYear
		after_year := v.AfterYear
		_smooth_month_gaps(target, before_year, after_year, df_msm, df_patchwork)
	}

	if use_est {
		df_patchwork.DSWRF_est = df_patchwork.DSWRF
		df_patchwork.DSWRF = nil
	} else {
		df_patchwork.DSWRF_msm = df_patchwork.DSWRF
		df_patchwork.DSWRF = nil
	}

	return df_patchwork, rep_years
}

func exctactMsm(df_msm *MsmTarget, start_time time.Time, end_time time.Time) MsmTarget {
	start_index := sort.Search(len(df_msm.date), func(i int) bool {
		return df_msm.date[i].After(start_time) || df_msm.date[i].Equal(start_time)
	})
	end_index := sort.Search(len(df_msm.date), func(i int) bool {
		return df_msm.date[i].After(end_time) || df_msm.date[i].Equal(end_time)
	})
	msm := MsmTarget{
		date:   append([]time.Time{}, df_msm.date[start_index:end_index+1]...),
		TMP:    append([]float64{}, df_msm.TMP[start_index:end_index+1]...),
		MR:     append([]float64{}, df_msm.MR[start_index:end_index+1]...),
		DSWRF:  append([]float64{}, df_msm.DSWRF[start_index:end_index+1]...),
		Ld:     append([]float64{}, df_msm.Ld[start_index:end_index+1]...),
		VGRD:   append([]float64{}, df_msm.VGRD[start_index:end_index+1]...),
		UGRD:   append([]float64{}, df_msm.UGRD[start_index:end_index+1]...),
		PRES:   append([]float64{}, df_msm.PRES[start_index:end_index+1]...),
		APCP01: append([]float64{}, df_msm.APCP01[start_index:end_index+1]...),
		RH:     append([]float64{}, df_msm.RH[start_index:end_index+1]...),
		Pw:     append([]float64{}, df_msm.Pw[start_index:end_index+1]...),
	}
	if df_msm.w_spd != nil {
		msm.w_spd = append([]float64{}, df_msm.w_spd[start_index:end_index+1]...)
	}
	if df_msm.w_dir != nil {
		msm.w_dir = append([]float64{}, df_msm.w_dir[start_index:end_index+1]...)
	}

	return msm
}

func filterMsmLeapYear29th(df_msm *MsmTarget) MsmTarget {
	date := []time.Time{}
	TMP := []float64{}
	MR := []float64{}
	DSWRF := []float64{}
	Ld := []float64{}
	VGRD := []float64{}
	UGRD := []float64{}
	PRES := []float64{}
	APCP01 := []float64{}
	RH := []float64{}
	Pw := []float64{}
	// w_spd := []float64{}
	// w_dir := []float64{}

	for i := 0; i < len(df_msm.date); i++ {
		if !(df_msm.date[i].Month() == 2 && df_msm.date[i].Day() == 29) {
			date = append(date, df_msm.date[i])
			TMP = append(TMP, df_msm.TMP[i])
			MR = append(MR, df_msm.MR[i])
			DSWRF = append(DSWRF, df_msm.DSWRF[i])
			Ld = append(Ld, df_msm.Ld[i])
			VGRD = append(VGRD, df_msm.VGRD[i])
			UGRD = append(UGRD, df_msm.UGRD[i])
			PRES = append(PRES, df_msm.PRES[i])
			APCP01 = append(APCP01, df_msm.APCP01[i])
			RH = append(RH, df_msm.RH[i])
			Pw = append(Pw, df_msm.Pw[i])
			// w_spd = append(w_spd, df_msm.w_dir[i])
			// w_dir = append(w_dir, df_msm.w_dir[i])
		}
	}

	return MsmTarget{
		date:   date,
		TMP:    TMP,
		MR:     MR,
		DSWRF:  DSWRF,
		Ld:     Ld,
		VGRD:   VGRD,
		UGRD:   UGRD,
		PRES:   PRES,
		APCP01: APCP01,
		RH:     RH,
		Pw:     Pw,
		// w_spd:     w_spd,
		// w_dir:     w_dir,
	}
}

func grouping(msm *MsmTarget) map[YearMonth]GroupData0_1 {
	// """月偏差値,月平均,年月平均のMSMデータフレームの作成

	// Args:
	//   df: MSMデータフレーム

	// Returns:
	//   pd.DataFrame: 月偏差値,月平均,年月平均のMSMデータフレーム
	// """

	//月インデックス領域確保
	index_m := make(map[int][]int, 12)
	for m := 1; m <= 12; m++ {
		index_m[m] = make([]int, 0, int(len(msm.date)/11))
	}

	//年月インデックス領域確保
	index_ym := make(map[YearMonth][]int, 10)
	for y := 2010; y <= 2021; y++ {
		for m := 1; m <= 12; m++ {
			ym := YearMonth{y, m}
			index_ym[ym] = make([]int, 0, int(len(msm.date)/11))
		}
	}

	//インデックス生成
	for i := 0; i < len(msm.date); i++ {
		y := msm.date[i].Year()
		m := int(msm.date[i].Month())
		ym := YearMonth{y, m}
		index_m[m] = append(index_m[m], i)
		index_ym[ym] = append(index_ym[ym], i)
	}

	//平均を求める関数の定義
	getMean := func(list []float64, index []int) float64 {
		n := len(index)

		var sum float64
		for i := 0; i < n; i++ {
			sum += list[index[i]]
		}
		avg := sum / float64(n)

		return avg
	}

	//標準偏差を求める関数の定義
	getStdDev := func(list []float64, index []int) float64 {
		n := len(index)
		avg := getMean(list, index)

		var sum_dev float64
		for i := 0; i < n; i++ {
			sum_dev += math.Pow(list[index[i]]-avg, 2)
		}

		std_dev := math.Sqrt(sum_dev / float64(n))

		return std_dev
	}

	df_temp_m_mean := make(map[int]GroupData0, 12)
	df_temp_m_std := make(map[int]GroupData0, 12)

	for m := range index_m {
		//月平均
		df_temp_m_mean[m] = GroupData0{
			TMP:    getMean(msm.TMP, index_m[m]),
			DSWRF:  getMean(msm.DSWRF, index_m[m]),
			MR:     getMean(msm.MR, index_m[m]),
			APCP01: getMean(msm.APCP01, index_m[m]),
			w_spd:  getMean(msm.w_spd, index_m[m]),
		}
		//月標準偏差
		df_temp_m_std[m] = GroupData0{
			TMP:    getStdDev(msm.TMP, index_m[m]),
			DSWRF:  getStdDev(msm.DSWRF, index_m[m]),
			MR:     getStdDev(msm.MR, index_m[m]),
			APCP01: getStdDev(msm.APCP01, index_m[m]),
			w_spd:  getStdDev(msm.w_spd, index_m[m]),
		}
	}

	df_temp_ym_mean := make(map[YearMonth]GroupData0, 120)
	for ym := range index_ym {
		if len(index_ym[ym]) > 0 {
			//年月平均
			df_temp_ym_mean[ym] = GroupData0{
				TMP:    getMean(msm.TMP, index_ym[ym]),
				DSWRF:  getMean(msm.DSWRF, index_ym[ym]),
				MR:     getMean(msm.MR, index_ym[ym]),
				APCP01: getMean(msm.APCP01, index_ym[ym]),
				w_spd:  getMean(msm.w_spd, index_ym[ym]),
			}
		}
	}

	df_temp := make(map[YearMonth]GroupData0_1, 120)
	for ym := range df_temp_ym_mean {
		df_temp[ym] = GroupData0_1{
			// TMP
			TMP_mean_m:  df_temp_m_mean[ym.Month].TMP,
			TMP_mean_ym: df_temp_ym_mean[ym].TMP,
			TMP_std_m:   df_temp_m_std[ym.Month].TMP,
			// DSWF
			DSWRF_mean_m:  df_temp_m_mean[ym.Month].DSWRF,
			DSWRF_mean_ym: df_temp_ym_mean[ym].DSWRF,
			DSWRF_std_m:   df_temp_m_std[ym.Month].DSWRF,
			// MR
			MR_mean_m:  df_temp_m_mean[ym.Month].MR,
			MR_mean_ym: df_temp_ym_mean[ym].MR,
			MR_std_m:   df_temp_m_std[ym.Month].MR,
			// APCP01
			APCP01_mean_m:  df_temp_m_mean[ym.Month].APCP01,
			APCP01_mean_ym: df_temp_ym_mean[ym].APCP01,
			APCP01_std_m:   df_temp_m_std[ym.Month].APCP01,
			// w_spd
			w_spd_mean_m:  df_temp_m_mean[ym.Month].w_spd,
			w_spd_mean_ym: df_temp_ym_mean[ym].w_spd,
			w_spd_std_m:   df_temp_m_std[ym.Month].w_spd,
		}
	}

	return df_temp
}

func get_temp_ci(msm *MsmTarget) map[YearMonth]GroupData {
	// """気象パラメータごとに決められた信頼区間に入っているかの判定

	// Args:
	//   df: MSMデータフレーム
	// Returns:
	//   pd.DataFrame: 各項目が想定信頼区間に入っているかを真偽値で格納したデータフレーム
	//                 カラム = y,m,TMP_dev,TMP,DSWRF,MR,APCP01,w_spd
	//                  y,mは年月、TMP_devは月平均気温の分散値、その他は真偽値
	// """

	// df_temp(pd.DataFrame): 気象パラメータごとの年月平均,月平均,月標準偏差
	//                         カラム = y,m,[TMP,VGRD,PRESS,MR_sat]*[mean_ym,mean_y,std_m]
	var df_temp map[YearMonth]GroupData0_1 = grouping(msm)

	// 気象パラメータと基準となる標準偏差(σ)の倍率
	const std_rate_TMP = 1.0
	const std_rate_DSWRF = 1.0
	const std_rate_MR = 1.0
	const std_rate_APCP01 = 1.5
	const std_rate_w_spd = 1.5

	df_ret := make(map[YearMonth]GroupData, 120)

	for ym, v := range df_temp {
		// 月平均と年月平均の差分(絶対値)計算 => "XXX_mean"
		// 月平均と年月平均の差分 "XXX_mean" が月標準偏差σ以下か？ => "XXX"

		//TMP
		TMP_dev := math.Abs(v.TMP_mean_m - v.TMP_mean_ym)
		TMP := TMP_dev <= std_rate_TMP*v.TMP_std_m

		//DSWRF
		DSWRF_dev := math.Abs(v.DSWRF_mean_m - v.DSWRF_mean_ym)
		DSWRF := DSWRF_dev <= std_rate_DSWRF*v.DSWRF_std_m

		//MR
		MR_dev := math.Abs(v.MR_mean_m - v.MR_mean_ym)
		MR := MR_dev <= std_rate_MR*v.MR_std_m

		//APCP01
		APCP01_dev := math.Abs(v.APCP01_mean_m - v.APCP01_mean_ym)
		APCP01 := APCP01_dev <= std_rate_APCP01*v.APCP01_std_m

		//w_spd
		w_spd_dev := math.Abs(v.w_spd_mean_m - v.w_spd_mean_ym)
		w_spd := w_spd_dev <= std_rate_w_spd*v.w_spd_std_m

		df_ret[ym] = GroupData{
			TMP:     TMP,
			TMP_dev: TMP_dev,
			DSWRF:   DSWRF,
			MR:      MR,
			APCP01:  APCP01,
			w_spd:   w_spd,
		}
	}

	// 各項目が想定信頼区間に入っているかを真偽値で格納したデータフレーム
	//         y   m   TMP_dev   TMP   DSWRF  MR   APCP01  w_spd
	// 0    2011  01     0.01   True   True  True   True   True
	// 1    2012  01     0.01   True   True  True   True   True
	// 2    2013  01     0.01   True   True  True   True   True
	//return df_temp.loc[:, ["y", "m", "TMP_dev", "TMP", "DSWRF", "MR", "APCP01", "w_spd"]]
	return df_ret
}

type GroupData struct {
	TMP, DSWRF, MR, APCP01, w_spd bool
	TMP_dev                       float64
}

type GroupData4AndYear struct {
	Year                                                   []int
	TMP_mean, DSWRF_mean, MR_mean, APCP01_mean, w_spd_mean []bool
	TMP_fs, DSWRF_fs, MR_fs, APCP01_fs, w_spd_fs           []bool
	TMP_dev                                                []float64
}

type GroupData4 struct {
	TMP_mean, DSWRF_mean, MR_mean, APCP01_mean, w_spd_mean bool
	TMP_fs, DSWRF_fs, MR_fs, APCP01_fs, w_spd_fs           bool
	TMP_dev                                                float64
}

type GroupData0_1 struct {
	TMP_mean_m, DSWRF_mean_m, MR_mean_m, APCP01_mean_m, w_spd_mean_m      float64
	TMP_mean_ym, DSWRF_mean_ym, MR_mean_ym, APCP01_mean_ym, w_spd_mean_ym float64
	TMP_std_m, DSWRF_std_m, MR_std_m, APCP01_std_m, w_spd_std_m           float64
}

type GroupData0 struct {
	TMP, DSWRF, MR, APCP01, w_spd float64
}

func get_fs_ci(df *MsmTarget) map[YearMonth]FSData {
	// """FS(Finkelstein Schafer statistics)計算

	// Args:
	//   df: 1時間ごとの気象パラメータが入ったデータフレーム
	//       カラム=date,y,m,d,TMP,DSWRF,MR,APCP01,w_spd

	// Returns:
	//   DataFrame: カラム=y,m,TMP,DSWRF,MR,APCP01,w_spd,TMP_FS
	// """

	// 気象パラメータと信頼区間(σ)
	const std_rate_TMP = 1.0
	const std_rate_DSWRF = 1.0
	const std_rate_MR = 1.0
	const std_rate_APCP01 = 1.5
	const std_rate_w_spd = 1.5

	//インデックス生成
	var g_ymd_mean GroupData2
	for i := 0; i < len(df.date); i += 24 {
		y := df.date[i].Year()
		m := int(df.date[i].Month())
		d := df.date[i].Day()
		g_ymd_mean.Year = append(g_ymd_mean.Year, y)
		g_ymd_mean.Month = append(g_ymd_mean.Month, m)
		g_ymd_mean.Day = append(g_ymd_mean.Day, d)
	}

	//平均を求める関数の定義
	getMean24H := func(list []float64, index int) float64 {
		var sum float64
		for i := 0; i < 24; i++ {
			sum += list[index+i]
		}
		avg := sum / 24.0

		return avg
	}

	// 年月日ごとの平均を生成
	getMeanForYearMonthGroupDay := func(list []float64, g *GroupData2) []float64 {
		mean_list := make([]float64, len(g.Day))
		for i := 0; i < len(g.Day); i++ {
			mean_list[i] = getMean24H(list, i*24)
		}
		return mean_list
	}

	// 日平均計算
	g_ymd_mean.TMP_mean_ymd = getMeanForYearMonthGroupDay(df.TMP, &g_ymd_mean)
	g_ymd_mean.DSWRF_mean_ymd = getMeanForYearMonthGroupDay(df.DSWRF, &g_ymd_mean)
	g_ymd_mean.MR_mean_ymd = getMeanForYearMonthGroupDay(df.MR, &g_ymd_mean)
	g_ymd_mean.APCP01_mean_ymd = getMeanForYearMonthGroupDay(df.APCP01, &g_ymd_mean)
	g_ymd_mean.w_spd_mean_ymd = getMeanForYearMonthGroupDay(df.w_spd, &g_ymd_mean)

	// FS値,FS値の偏差,FS値の偏差が指定範囲内に入っているか
	TMP_FS := make_fs(&g_ymd_mean, func(msm *GroupData2, i int) float64 { return msm.TMP_mean_ymd[i] }, std_rate_TMP)
	DSWRF_FS := make_fs(&g_ymd_mean, func(msm *GroupData2, i int) float64 { return msm.DSWRF_mean_ymd[i] }, std_rate_DSWRF)
	MR_FS := make_fs(&g_ymd_mean, func(msm *GroupData2, i int) float64 { return msm.MR_mean_ymd[i] }, std_rate_MR)
	APCP01_FS := make_fs(&g_ymd_mean, func(msm *GroupData2, i int) float64 { return msm.APCP01_mean_ymd[i] }, std_rate_APCP01)
	w_spd_FS := make_fs(&g_ymd_mean, func(msm *GroupData2, i int) float64 { return msm.w_spd_mean_ymd[i] }, std_rate_w_spd)

	FS := make(map[YearMonth]FSData)
	for ym := range TMP_FS {
		FS[ym] = FSData{
			TMP:    TMP_FS[ym],
			DSWRF:  DSWRF_FS[ym],
			MR:     MR_FS[ym],
			APCP01: APCP01_FS[ym],
			w_spd:  w_spd_FS[ym],
		}
	}

	return FS
}

type YearMonthDay struct {
	Year, Month, Day int
}

type FSData struct {
	TMP, DSWRF, MR, APCP01, w_spd bool
}

type GroupData2 struct {
	Year, Month, Day []int

	TMP_mean_ymd, DSWRF_mean_ymd, MR_mean_ymd, APCP01_mean_ymd, w_spd_mean_ymd []float64
}

func make_fs(g_ymd_mean *GroupData2, key func(*GroupData2, int) float64, std_rate float64) map[YearMonth]bool {
	// """特定の気象パラメータに対するFS(Finkelstein Schafer statistics)計算

	// Args:
	//   g_ymd_mean: 日平均値の入ったデータフレーム
	//   key: FS値を計算するカラム名
	//   std_rate: FS値の偏差が std_rate * σ 以下であれば カラムkeyにTrueを設定します

	// Returns:
	//   DataFrame: カラム=y,m,<key>,<key>_FS,<key>_FS_std
	// """
	// 月ごとの累積度数分布(CDF)の計算
	cdf_ALL := make_cdf(
		g_ymd_mean,
		func(msm *GroupData2, i int) int { return msm.Month[i] },
		key)

	// 年月ごとの累積度数分布(CDF)の計算
	cdf_year := make_cdf(
		g_ymd_mean,
		func(msm *GroupData2, i int) int { return msm.Year[i]*100 + msm.Month[i] },
		key)

	// 日ごとのFS値の計算
	FS := make([]float64, len(g_ymd_mean.Day))
	for i := 0; i < len(FS); i++ {
		FS[i] = math.Abs(cdf_ALL[i] - cdf_year[i])
	}

	// 年月インデックス
	ym_list := make([]YearMonthIndex, 0, 120)
	y, m := 0, 0
	for i := 0; i < len(g_ymd_mean.Day); i++ {
		if g_ymd_mean.Year[i] != y || g_ymd_mean.Month[i] != m {
			y = g_ymd_mean.Year[i]
			m = g_ymd_mean.Month[i]
			ym_list = append(ym_list, YearMonthIndex{y, m, i})
		}
	}

	// 年月ごとのFS値の平均を計算 : <key>_FS
	fs_ym := make([]float64, len(ym_list))
	for i, ym := range ym_list {
		var start, end int
		start = ym.Index
		if i < len(ym_list)-1 {
			end = ym_list[i+1].Index
		} else {
			end = len(g_ymd_mean.Day)
		}
		fs_ym[i] = mean(FS[start:end])
	}

	// 月ごとにFS値の偏差 : <key>_FS_std
	fs_m_list := make(map[int][]float64, 12)
	for i, ym := range ym_list {
		m := ym.Month
		if _, ok := fs_m_list[m]; !ok {
			fs_m_list[m] = make([]float64, 0, 20)
		}
		fs_m_list[m] = append(fs_m_list[m], fs_ym[i])
	}
	fs_std_m := make(map[int]float64)
	for m := range fs_m_list {
		list_sq := make([]float64, len(fs_m_list[m]))
		for i := 0; i < len(list_sq); i++ {
			list_sq[i] = math.Pow(fs_m_list[m][i], 2.0)
		}
		fs_std_m[m] = math.Sqrt(mean(list_sq))
	}

	// 年月ごとにFS値の偏差が指定の範囲に収まっているか
	typical := make(map[YearMonth]bool)
	for i, ymi := range ym_list {
		ym := YearMonth{ymi.Year, ymi.Month}
		if fs_ym[i] <= std_rate*fs_std_m[ym.Month] {
			typical[ym] = true
		} else {
			typical[ym] = false
		}
	}

	return typical
}

func mean(list []float64) float64 {
	sum := 0.0
	for i := 0; i < len(list); i++ {
		sum += list[i]
	}
	return sum / float64(len(list))
}

type YearMonth struct {
	Year, Month int
}

type YearMonthIndex struct {
	Year, Month int
	Index       int
}

func make_cdf(g_ymd_mean *GroupData2, by func(*GroupData2, int) int, key func(*GroupData2, int) float64) []float64 {
	// """特定の気象パラメータに対するCDF計算
	// g_ymd_mean に 名前が<key>_<suffix> のカラムを追加し、CDFを格納する。

	// Args:
	//   g_ymd_mean: 日平均値の入ったデータフレーム
	//   by: CDF計算時にグループ化するカラム名のリスト
	//   key: CDF計算時対象のカラム名
	//   suffix: CDF計算結果を格納するカラム名に付与するサフィックス
	// """
	// g_ymd_mean_m = g_ymd_mean.groupby(by, as_index=False)
	// for _, group in g_ymd_mean_m:
	//     g = group.sort_values(key).reset_index()
	//     N = len(g)
	//     g.loc[:, "cdf"] = [(i + 1) / N for i in range(N)]
	//     g = g.sort_values('index').set_index('index')
	//     g_ymd_mean.loc[list(g.index), key + suffix] = g["cdf"].values

	cdf := make([]float64, len(g_ymd_mean.Day))

	//月or年月ごとの分割: オリジナルの配列のインデックスをmapに格納
	//(分割基準は by 関数で指定される)
	indexMap := make(map[int][]int, 120)
	for i := 0; i < len(g_ymd_mean.Day); i++ {
		// ex) by = func(msm MsmTarget, i int) int { return msm.date.Month() }
		k := by(g_ymd_mean, i)
		if _, ok := indexMap[k]; !ok {
			indexMap[k] = []int{}
		}
		indexMap[k] = append(indexMap[k], i)
	}

	for k := range indexMap {
		// オリジナルの配列のインデックスと値の配列を用意
		iv := make([]IndexAndValue, len(indexMap[k]))
		for i := 0; i < len(iv); i++ {
			// ex) key = func(msm MsmTarget, i int) float64 { return msm.TMP[i] }
			iv[i] = IndexAndValue{i, key(g_ymd_mean, indexMap[k][i])}
		}

		//値(TMP,DSWRF,MR,APCP01 or w_spd)で並べ替え
		sort.Slice(iv, func(i, j int) bool { return iv[i].Value < iv[j].Value })

		//CDFの計算
		for i := 0; i < len(iv); i++ {
			cdf[indexMap[k][iv[i].Index]] = (float64(i) + 1.0) / float64(len(iv))
		}
	}

	return cdf
}

type IndexAndValue struct {
	Index int
	Value float64
}

type GroupData3 struct {
	Year, Month       int
	TMP_dev, TMP_mean []float64
	DSWRF_mean        []float64
	MR_mean           []float64
	APCP01_mean       []float64
	w_spd_mean        []float64
	TMP_fs            []float64
	DSWRF_fs          []float64
	MR_fs             []float64
	APCP01_fs         []float64
	w_spd_fs          []float64
}

// **** 代表年の決定と接合処理 ****

func _get_representative_years(df_ci map[YearMonth]GroupData4) []int {
	// """

	// Args:
	//   df_ci(pd.DataFrame): 信頼区間判定結果の入ったデータフレーム

	// Returns:
	//   List[int]: 月別の代表的な年
	// """

	// 選定は、気温(偏差)=>水平面全天日射量(偏差)=>絶対湿度(偏差)=>降水量(偏差)=>風速(偏差)=>
	// 気温(FS)=>水平面全天日射量(FS)=>絶対湿度(FS)=>降水量(FS)=>風速(FS)の順に判定を行う
	// 最終的に複数が候補となった場合は気温(偏差)が最も0に近い年を選定する。

	select_year := []int{}

	//月ごとのグループ化
	g_m := make(map[int]*GroupData4AndYear, 12)
	for ym, v := range df_ci {
		if _, ok := g_m[ym.Month]; !ok {
			g_m[ym.Month] = &GroupData4AndYear{
				Year:        []int{},
				TMP_mean:    []bool{},
				DSWRF_mean:  []bool{},
				MR_mean:     []bool{},
				APCP01_mean: []bool{},
				w_spd_mean:  []bool{},
				TMP_fs:      []bool{},
				DSWRF_fs:    []bool{},
				MR_fs:       []bool{},
				APCP01_fs:   []bool{},
				w_spd_fs:    []bool{},
				TMP_dev:     []float64{},
			}
		}
		g_m[ym.Month].Year = append(g_m[ym.Month].Year, ym.Year)
		g_m[ym.Month].TMP_mean = append(g_m[ym.Month].TMP_mean, v.TMP_mean)
		g_m[ym.Month].TMP_dev = append(g_m[ym.Month].TMP_dev, v.TMP_dev)
		g_m[ym.Month].DSWRF_mean = append(g_m[ym.Month].DSWRF_mean, v.DSWRF_mean)
		g_m[ym.Month].MR_mean = append(g_m[ym.Month].MR_mean, v.MR_mean)
		g_m[ym.Month].APCP01_mean = append(g_m[ym.Month].APCP01_mean, v.APCP01_mean)
		g_m[ym.Month].w_spd_mean = append(g_m[ym.Month].w_spd_mean, v.w_spd_mean)
		g_m[ym.Month].TMP_fs = append(g_m[ym.Month].TMP_fs, v.TMP_fs)
		g_m[ym.Month].DSWRF_fs = append(g_m[ym.Month].DSWRF_fs, v.DSWRF_fs)
		g_m[ym.Month].MR_fs = append(g_m[ym.Month].MR_fs, v.MR_fs)
		g_m[ym.Month].APCP01_fs = append(g_m[ym.Month].APCP01_fs, v.APCP01_fs)
		g_m[ym.Month].w_spd_fs = append(g_m[ym.Month].w_spd_fs, v.w_spd_fs)
	}

	get_mean_int := func(list []int) float64 {
		sum := 0.0
		for i := 0; i < len(list); i++ {
			sum += float64(list[i])
		}
		return sum / float64(len(list))
	}

	true_index := func(list []bool, filter_index []int) []int {
		true_index := []int{}
		for _, i := range filter_index {
			if list[i] {
				true_index = append(true_index, i)
			}
		}
		return true_index
	}

	get_min := func(list []float64, filter_index []int) float64 {
		min := math.MaxFloat64
		for _, i := range filter_index {
			v := list[i]
			if v < min {
				min = v
			}
		}
		return min
	}

	for _, group := range g_m {
		center_y := get_mean_int(group.Year)
		var temp_index []int

		//絞り込み判定指標
		select_list := [][]bool{
			group.TMP_mean,
			group.DSWRF_mean,
			group.MR_mean,
			group.APCP01_mean,
			group.w_spd_mean,
			group.TMP_fs,
			group.DSWRF_fs,
			group.MR_fs,
			group.APCP01_fs,
			group.w_spd_fs,
		}

		//絞り込み途中の候補インデックス
		filter_index := []int{}
		for i := 0; i < len(group.Year); i++ {
			filter_index = append(filter_index, i)
		}

		// 判定指標でループ(候補が単一の年になるまで繰り返す)
		for _, select_slice := range select_list {

			temp_index := true_index(select_slice, filter_index)
			if len(temp_index) == 0 {
				// group_temp(selectがTrueの年)が0個
				// =>group(前selectがTrueの年)の中から気温(偏差)が最も小さい年を選定

				// TMP_devが最小の年を抜粋
				break
			}

			// group_temp(selectがTrueの年)が1個 => 代表年として選定
			if len(temp_index) == 1 {
				break
			} else {
				filter_index = temp_index
			}
		}

		// 判定指標がw_spd_fs(最後の判定指標)の時 => 気温(偏差)で判定
		// or 途中で候補が消失した場合
		if len(filter_index) != 1 {
			// TMP_devが最小の年を抜粋
			temp_index = []int{}
			TMP_dev_min := get_min(group.TMP_dev, filter_index)
			for _, i := range filter_index {
				if group.TMP_dev[i] == TMP_dev_min {
					temp_index = append(temp_index, i)
				}
			}

			if len(temp_index) > 1 {
				// TMP_devの最小が複数残った場合 => 対象期間の中心(平均)に近い年を選定

				y_abs := make([]float64, len(group.Year))
				for _, i := range temp_index {
					y_abs[i] = math.Abs(float64(group.Year[i]) - center_y)
				}
				y_abs_min := math.MaxFloat64
				for _, i := range temp_index {
					if y_abs[i] > y_abs_min {
						y_abs_min = y_abs[i]
					}
				}
				temp_index = []int{}
				for _, i := range temp_index {
					if y_abs[i] == y_abs_min {
						temp_index = append(temp_index, i)
					}
				}

				// 対象期間の中心(平均)に近い年が複数残った場合 => 若い年を選定
				if len(temp_index) > 1 {
					sort.Slice(temp_index, func(i, j int) bool { return group.Year[temp_index[i]] < group.Year[temp_index[j]] })
				}
			}
		}

		// 絞り込んだ一覧表の先頭の年を採用
		select_year = append(select_year, group.Year[temp_index[0]])
	}

	return select_year
}

func patch_representataive_years(df *MsmTarget, rep_years []int) *MsmTarget {
	// """標準年の作成
	// 月別に代表的な年から接合した1年間のデータを作成します。

	// Args:
	//   df(pd.DataFrame):
	//   rep_years(List[int]): 月別の代表的な年

	// Returns:
	//   pd.DataFrame: 標準年のMSMデータフレーム
	// """
	df_EA := MsmTarget{
		date:   []time.Time{},
		TMP:    []float64{},
		MR:     []float64{},
		DSWRF:  []float64{},
		Ld:     []float64{},
		VGRD:   []float64{},
		UGRD:   []float64{},
		PRES:   []float64{},
		APCP01: []float64{},
		RH:     []float64{},
		Pw:     []float64{},
	}

	// 月日数
	mdays := [...]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

	local, _ := time.LoadLocation("Local")

	// 月別に代表的な年のデータを抜き出す
	for i, year := range rep_years {

		month := time.Month(i + 1)

		// 当該代表年月の開始日とその次月開始日
		start_date := time.Date(year, month, 1, 0, 0, 0, 0, local)
		end_date := start_date.AddDate(0, 0, mdays[int(month)-1])

		start_index := sort.Search(len(df.date), func(i int) bool {
			return df.date[i].After(start_date) || df.date[i].Equal(start_date)
		})
		end_index := sort.Search(len(df.date), func(i int) bool {
			return df.date[i].After(end_date) || df.date[i].Equal(end_date)
		})

		// 抜き出した代表データ
		df_temp := MsmTarget{
			date:   df.date[start_index : end_index+1],
			TMP:    df.TMP[start_index : end_index+1],
			MR:     df.MR[start_index : end_index+1],
			DSWRF:  df.DSWRF[start_index : end_index+1],
			Ld:     df.Ld[start_index : end_index+1],
			VGRD:   df.VGRD[start_index : end_index+1],
			UGRD:   df.UGRD[start_index : end_index+1],
			PRES:   df.PRES[start_index : end_index+1],
			APCP01: df.APCP01[start_index : end_index+1],
			RH:     df.RH[start_index : end_index+1],
			Pw:     df.Pw[start_index : end_index+1],
		}

		// 接合
		df_EA.date = append(df_EA.date, df_temp.date...)
		df_EA.TMP = append(df_EA.TMP, df_temp.TMP...)
		df_EA.MR = append(df_EA.MR, df_temp.MR...)
		df_EA.DSWRF = append(df_EA.DSWRF, df_temp.DSWRF...)
		df_EA.Ld = append(df_EA.Ld, df_temp.Ld...)
		df_EA.VGRD = append(df_EA.VGRD, df_temp.VGRD...)
		df_EA.UGRD = append(df_EA.UGRD, df_temp.UGRD...)
		df_EA.PRES = append(df_EA.PRES, df_temp.PRES...)
		df_EA.APCP01 = append(df_EA.APCP01, df_temp.APCP01...)
		df_EA.RH = append(df_EA.RH, df_temp.RH...)
		df_EA.Pw = append(df_EA.Pw, df_temp.Pw...)
	}

	for i := 0; i < len(df_EA.date); i++ {
		df_EA.date[i] = time.Date(1970, df_EA.date[i].Month(), df_EA.date[i].Day(), df_EA.date[i].Hour(), 0, 0, 0, local)
	}

	return &df_EA
}

// **** 円滑化処理 ****

func get_smoothing_months(rep_years []int) []SmootingMonth {
	// """円滑化が必要な月の取得

	// Args:
	//   rep_years(List[int]): 月別の代表的な年

	// Returns:
	//   Generator[int, int, int]: 円滑化が必要な月,前月の代表年,対象月の代表年のタプル
	// """

	mlist := []SmootingMonth{}

	for i := 0; i < 12; i++ {
		// 1月から計算
		target := time.Month(i + 1)

		// 前月の代表年
		var before_year int
		if i > 0 {
			before_year = int(rep_years[i-1])
		} else {
			before_year = int(rep_years[len(rep_years)-1])
		}

		// 対象月の代表年
		after_year := int(rep_years[i])

		// 前月と対象月では対象年が異なる または 前月が2月かつその代表年が閏年の場合
		before_year_date := time.Date(before_year, time.December, 31, 0, 0, 0, 0, time.Local)
		if before_year != after_year || (target == 3 && before_year_date.YearDay() == 366) {
			mlist = append(mlist, SmootingMonth{target, before_year, after_year})
		}
	}

	return mlist
}

type SmootingMonth struct {
	TargetMonth time.Month
	BeforeYear  int
	AfterYear   int
}

func _smooth_month_gaps(after_month time.Month, before_year int, after_year int, df_temp *MsmTarget, df_EA *MsmTarget) {
	// """月別に代表的な年からの接合部を滑らかに加工する

	// Args:
	//   after_month(int): 対象月
	//   after_year(int): 対象月の代表年
	//   before_year(int): 前月の代表年
	//   df_temp(pd.DataFrame): 全期間のデータ
	//   df_EA(pd.DataFrame): 1年分にあらく結合したデータ
	// """
	// [1, 0.92, 0.83, ...., 0.0], [0.0, 0.08, 0.17, 0.25, ..., 1.0]

	// before_coef = np.linspace(1, 0, 13, endpoint=True)
	// after_coef = np.linspace(0, 1, 13, endpoint=True)
	var before_coef, after_coef [13]float64
	for i := 0; i < 13; i++ {
		after_coef[i] = float64(i) * float64(i) / 12.0
		before_coef[i] = 1.0 - after_coef[i]
	}

	// 対象月の1970年における対象月の1日
	center := time.Date(1970, after_month, 1, 0, 0, 0, 0, time.Local)

	// 前月の代表年における対象月の1日
	before := time.Date(before_year, after_month, 1, 0, 0, 0, 0, time.Local)

	// 対象月の代表年における対象月の1日
	after := time.Date(int(after_year), after_month, 1, 0, 0, 0, 0, time.Local)

	var timestamp [13]time.Time
	var df_before, df_after MsmTarget

	if after_month == 1 {
		// 12月と1月の結合(年をまたぐ)

		// 前月の代表年における12月31日18時
		before_start := time.Date(int(before_year), 12, 31, 18, 0, 0, 0, time.Local)

		// 前月の代表年の翌年の1月1日6時
		before_end := time.Date(int(before_year+1), 1, 1, 6, 0, 0, 0, time.Local)

		// 前月の代表年の12月31日18時から翌年1月1日6時までのMSMデータフレーム
		df_before = exctactMsm(df_temp, before_start, before_end)

		// 対象月の代表年の前年の12月31日18時
		after_start := time.Date(int(after_year-1), 12, 31, 18, 0, 0, 0, time.Local)

		// 対象月の代表年の1月1日6時
		after_end := time.Date(int(after_year), 1, 1, 6, 0, 0, 0, time.Local)

		// 対象月の代表年の前年12月31日18時から翌年1月1日6時までのMSMデータフレーム
		df_after = exctactMsm(df_temp, after_start, after_end)

		// 1970年12月31日18時-23時 および 1月1日0時-6時
		// 1970年12月31日18時-23時
		for i := 0; i < 6; i++ {
			timestamp[i] = time.Date(1970, 12, 31, 18+i, 0, 0, 0, time.Local)
		}

		// 1970年1月1日0時-6時
		for i := 0; i < 7; i++ {
			timestamp[i+6] = time.Date(1970, 1, 1, i, 0, 0, 0, time.Local)
		}

		// 2月と3月の結合(うるう年の回避)
	} else if after_month == 3 {

		// 結合する2つの月の若い月(前月)の代表年における2月28日18時(はじまり)
		before_start := time.Date(before_year, 2, 28, 18, 0, 0, 0, time.Local)

		// 前月の代表年における3月1日6時(おわり)
		before_end := time.Date(before_year, 3, 1, 6, 0, 0, 0, time.Local)

		// 前月の代表年における2月28日18時から3月1日6時までのMSMデータフレーム
		df_before = exctactMsm(df_temp, before_start, before_end)

		// 結合する2つの月の遅い月(対象月)の代表年における2月28日18時(はじまり)
		after_start := time.Date(after_year, 2, 28, 18, 0, 0, 0, time.Local)

		// 対象月の代表年における3月1日6時(おわり)
		after_end := after.Add(time.Hour * 6)

		// 対象月の代表年における2月28日18時から3月1日6時までのMSMデータフレーム
		df_after = exctactMsm(df_temp, after_start, after_end)

		// MSMデータフレームから2月29日を除外
		df_before = filterMsmLeapYear29th(&df_before)
		df_after = filterMsmLeapYear29th(&df_after)

		// 対象月の1970年における対象月の1日の前日18時から翌日6時まで
		for i := -6; i <= 6; i++ {
			timestamp[i+6] = center.Add(time.Duration(i) * time.Hour)
		}
	} else {
		// 前月の代表年における対象月の1日の前月末日18時
		before_start := before.Add(time.Hour * -6)

		// 前月の代表年における対象月の1日6時
		before_end := before.Add(time.Hour * 6)

		// 前月の代表年における対象月の1日の前月末日18時から1日6時までのMSMデータフレーム
		df_before = exctactMsm(df_temp, before_start, before_end)

		// 対象月の代表年における対象月の1日の前月末日18時
		after_start := after.Add(time.Hour * -6)

		// 対象月の代表年における対象月の1日6時
		after_end := after.Add(time.Hour * 6)

		// 対象月の代表年における対象月の1日の前月末日18時から1日6時までのMSMデータフレーム
		df_after = exctactMsm(df_temp, after_start, after_end)

		// 対象月の1970年における対象月の1日の前日18時から翌日6時まで
		for i := -6; i <= 6; i++ {
			timestamp[i+6] = center.Add(time.Duration(i) * time.Hour)
		}
	}

	// 前月の代表年における月末から翌月にかけての13時間 -> 係数を1,0.92,... と掛ける。
	// 対象月の代表年における前月末18時からの13時間 -> 係数を0,0.08,,... と掛ける。
	// 以上を合算する。
	date := [13]time.Time{}
	TMP := [13]float64{}
	MR := [13]float64{}
	DSWRF := [13]float64{}
	Ld := [13]float64{}
	VGRD := [13]float64{}
	UGRD := [13]float64{}
	PRES := [13]float64{}
	APCP01 := [13]float64{}
	RH := [13]float64{}
	Pw := [13]float64{}
	// w_spd := [13]float64{}
	// w_dir := [13]float64{}

	for i := 0; i < 13; i++ {
		date[i] = timestamp[i] //タイムスタンプは例外
		TMP[i] = df_before.TMP[i]*before_coef[i] + df_after.TMP[i]*after_coef[i]
		MR[i] = df_before.MR[i]*before_coef[i] + df_after.MR[i]*after_coef[i]
		DSWRF[i] = df_before.DSWRF[i]*before_coef[i] + df_after.DSWRF[i]*after_coef[i]
		Ld[i] = df_before.Ld[i]*before_coef[i] + df_after.Ld[i]*after_coef[i]
		VGRD[i] = df_before.VGRD[i]*before_coef[i] + df_after.VGRD[i]*after_coef[i]
		UGRD[i] = df_before.UGRD[i]*before_coef[i] + df_after.UGRD[i]*after_coef[i]
		PRES[i] = df_before.PRES[i]*before_coef[i] + df_after.PRES[i]*after_coef[i]
		APCP01[i] = df_before.APCP01[i]*before_coef[i] + df_after.APCP01[i]*after_coef[i]
		RH[i] = df_before.RH[i]*before_coef[i] + df_after.RH[i]*after_coef[i]
		Pw[i] = df_before.Pw[i]*before_coef[i] + df_after.Pw[i]*after_coef[i]
		// w_spd[i] = df_before.w_spd[i]*before_coef[i] + df_after.w_spd[i]*after_coef[i]
		// w_dir[i] = df_before.w_dir[i]*before_coef[i] + df_after.w_dir[i]*after_coef[i]
	}

	dateIndex := make(map[time.Time]int, 13)
	for i := 0; i < len(df_EA.date); i++ {
		dateIndex[df_EA.date[i]] = i
	}

	for i := 0; i < 13; i++ {
		index := dateIndex[date[i]]
		df_EA.TMP[index] = TMP[i]
		df_EA.MR[index] = MR[i]
		df_EA.DSWRF[index] = DSWRF[i]
		df_EA.Ld[index] = Ld[i]
		df_EA.VGRD[index] = VGRD[i]
		df_EA.UGRD[index] = UGRD[i]
		df_EA.PRES[index] = PRES[i]
		df_EA.APCP01[index] = APCP01[i]
		df_EA.RH[index] = RH[i]
		df_EA.Pw[index] = Pw[i]
		// df_EA.w_spd[index] = w_spd[i]
		// df_EA.w_dir[index] = w_dir[i]
	}
}

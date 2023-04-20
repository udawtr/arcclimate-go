package arcclimate

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// """MSMファイルを読み込みます。必要に応じてダウンロードを行います。
// Args:
//
//	lat(float64): 推計対象地点の緯度（10進法）
//	lon(float64): 推計対象地点の経度（10進法）
//	msm_file_dir(str): MSMファイルの格納ディレクトリ
//
// Returns:
//
//	MsmDataSet: 読み込んだデータフレームのリスト
//
// """
func LoadMsmFiles(msm_list []string, msm_file_dir string) MsmDataSet {
	// 計算に必要なMSMを算出して、ダウンロード⇒ファイルpathをリストで返す

	// 保存先ディレクトリの作成
	os.Mkdir(msm_file_dir, os.ModePerm)

	// MSMファイル読み込み
	df_msm_list := make([]MsmData, len(msm_list))
	c := make(chan MsmAndIndex, 4)
	for index, msm := range msm_list {
		// MSMファイルのパス
		// MSMファイル読み込み
		// 負の日射量が存在した際に日射量を0とする
		go load_msm(index, msm_file_dir, msm, c, true, msm_file_dir)
	}

	for i := 0; i < len(msm_list); i++ {
		ret := <-c
		df_msm_list[ret.Index] = ret.Msm
		log.Printf("MSM読み込み完了 %s", ret.Msm.name)
	}

	return MsmDataSet{Data: df_msm_list}
}

type MsmAndIndex struct {
	Index int
	Msm   MsmData
}

func load_msm(index int, msm_file_dir string, msm string, c chan MsmAndIndex, saveCache bool, output_dir string) {
	msm_path := filepath.Join(msm_file_dir, fmt.Sprintf("%s.csv.gz", msm))

	var gf *gzip.Reader
	var gerr error
	if !fileExists(msm_path) {
		// ダウンロード元URL
		dl_url := "https://s3.ap-northeast-1.wasabisys.com/arcclimate-ja/msm_2011_2020/"
		//dl_url := "https://storage.googleapis.com/arcclimate-msm/"

		var err error = nil
		src_url := fmt.Sprintf("%s%s.csv.gz", dl_url, msm)
		save_path := filepath.Join(output_dir, fmt.Sprintf("%s.csv.gz", msm))

		if saveCache {
			log.Printf("MSMダウンロード %s => %s", src_url, save_path)
		} else {
			log.Printf("MSMダウンロード %s", src_url)
		}

		// Get the data
		resp, err := http.Get(src_url)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		if saveCache {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln(err)
			}

			// Create the file
			var out *os.File
			out, err = os.Create(save_path)
			if err != nil {
				panic(err)
			}
			defer out.Close()

			// Write the body to file
			_, err = out.Write(b)
			if err != nil {
				panic(err)
			}

			buf := bytes.NewBuffer(b)
			gf, gerr = gzip.NewReader(buf)
		} else {
			gf, gerr = gzip.NewReader(resp.Body)
		}
	} else {
		log.Printf("MSMファイル読み込み: %s", msm_path)
		f, ferr := os.Open(msm_path)
		if ferr != nil {
			log.Fatal(ferr)
			panic(ferr)
		}
		defer f.Close()
		gf, gerr = gzip.NewReader(f)
	}

	if gerr != nil {
		fmt.Println("gzipエラー")
		panic(gerr)
	}
	defer gf.Close()

	csvReader := csv.NewReader(gf)
	csvReader.ReuseRecord = true
	_, _ = csvReader.Read()

	var rows [87687]MsmDataRow

	for i := 0; i < len(rows); i++ {
		row, cerr := csvReader.Read()
		if cerr == io.EOF {
			break
		}
		if cerr != nil {
			log.Fatal(cerr)
			panic(cerr)
		}

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

		rows[i] = MsmDataRow{
			date:      date,
			TMP:       TMP,
			MR:        MR,
			DSWRF_est: DSWRF_est,
			DSWRF_msm: DSWRF_msm,
			Ld:        Ld,
			VGRD:      VGRD,
			UGRD:      UGRD,
			PRES:      PRES,
			APCP01:    APCP01,
		}
	}

	df_msm := MsmData{
		name: msm,
		Rows: &rows,
	}

	c <- MsmAndIndex{index, df_msm}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

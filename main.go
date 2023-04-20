// ArcClimate
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/akamensky/argparse"
	"github.com/udawtr/arcclimate_go/arcclimate"
)

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
	res := arcclimate.Interpolate(
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
		res.ToCSV(buf)
	} else if *format == "EPW" {
		res.ToEPW(buf, *lat, *lon)
	} else if *format == "HAS" {
		res.ToHAS(buf)
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

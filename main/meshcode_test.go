package main // mainパッケージであることを宣言

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_MeshCodeFromLatLon(t *testing.T) {
	// 緯度経度を設定する
	lat := 36.0
	lon := 138.0

	// メッシュコードを取得する
	meshcode := MeshCodeFromLatLon(lat, lon)

	// 正しいメッシュコードが取得できることを確認する
	assert.Equal(t, 54380000, meshcode)
}

func Test_get_mesh_latlon(t *testing.T) {
	// メッシュコードを取得する
	lat, lon := get_mesh_latlon("54380000")

	// 正しい緯度経度が取得できることを確認する
	assert.InDelta(t, 36, lat, 0.01)
	assert.InDelta(t, 138, lon, 0.01)
}

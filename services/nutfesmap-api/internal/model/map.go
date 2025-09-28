package model

import "time"

// DB行と1:1のドメインモデル（外部I/Oのjsonタグは持たせない）
type Map struct {
	ID            string     `db:"id"` // ex: "map_..."（UUIDベース）
	Name          string     `db:"name"`
	ImageData     string     `db:"image_data"` // base64文字列
	NaturalWidth  int        `db:"natural_width"`
	NaturalHeight int        `db:"natural_height"`
	ParentMapID   *string    `db:"parent_map_id"`
	HasFloors     bool       `db:"has_floors"`
	FloorCount    int        `db:"floor_count"`
	CreatedAt     time.Time  `db:"created_at"`
	ModifiedAt    time.Time  `db:"modified_at"`
	DeletedAt     *time.Time `db:"deleted_at"`
}

// 子マップの軽量参照（リスト用）
// SELECT id, name, has_floors, floor_count ... を想定
type MapChildRef struct {
	ID         string `db:"id"`
	Name       string `db:"name"`
	HasFloors  bool   `db:"has_floors"`
	FloorCount int    `db:"floor_count"`
}

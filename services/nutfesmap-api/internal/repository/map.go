package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"nutfesmap/internal/model"
)

type MapRepository struct{ DB *sql.DB }

// Request DTO（外部契約）
type MapCreateRequest struct {
	Name          string  `json:"name"          validate:"required,min=1"`
	ImageData     string  `json:"imageData"     validate:"required"` // base64
	NaturalWidth  int     `json:"naturalWidth"  validate:"required,min=1"`
	NaturalHeight int     `json:"naturalHeight" validate:"required,min=1"`
	ParentMapID   *string `json:"parentMapId"`
	HasFloors     bool    `json:"hasFloors"`
	FloorCount    int     `json:"floorCount"    validate:"min=0"`
}

// Response DTO（外部契約）
type MapChildRefDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	HasFloors  bool   `json:"hasFloors"`
	FloorCount int    `json:"floorCount"`
}

type MapResponse struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	ImageData     string           `json:"imageData"`
	NaturalWidth  int              `json:"naturalWidth"`
	NaturalHeight int              `json:"naturalHeight"`
	ParentMapID   *string          `json:"parentMapId,omitempty"`
	HasFloors     bool             `json:"hasFloors"`
	FloorCount    int              `json:"floorCount"`
	ChildrenCount int              `json:"childrenCount"`
	Children      []MapChildRefDTO `json:"children"`
	CreatedAt     time.Time        `json:"createdAt"`
	ModifiedAt    time.Time        `json:"modifiedAt"`
}

func NewMapRepository(db *sql.DB) *MapRepository { return &MapRepository{DB: db} }

// 1件挿入
func (r *MapRepository) Insert(ctx context.Context, m *model.Map) error {
	now := time.Now().UTC()
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height,
			parent_map_id, has_floors, floor_count, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`,
		m.ID, m.Name, m.ImageData, m.NaturalWidth, m.NaturalHeight,
		m.ParentMapID, m.HasFloors, m.FloorCount, now, now,
	)
	return err
}

// 本体 + 子の件数 + 子の軽量一覧をまとめて取得（レスポンス組み立て向け）
type MapAggregate struct {
	Base          *model.Map
	ChildrenCount int
	Children      []model.MapChildRef
}

func (r *MapRepository) FindAggregate(ctx context.Context, id string) (*MapAggregate, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`, id)

	var base model.Map
	if err := row.Scan(
		&base.ID, &base.Name, &base.ImageData, &base.NaturalWidth, &base.NaturalHeight,
		&base.ParentMapID, &base.HasFloors, &base.FloorCount, &base.CreatedAt, &base.ModifiedAt, &base.DeletedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// 子の件数
	var cnt int
	if err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`, base.ID).Scan(&cnt); err != nil {
		return nil, err
	}

	// 子の軽量一覧
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`, base.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	children := make([]model.MapChildRef, 0)
	for rows.Next() {
		var c model.MapChildRef
		if err := rows.Scan(&c.ID, &c.Name, &c.HasFloors, &c.FloorCount); err != nil {
			return nil, err
		}
		children = append(children, c)
	}

	return &MapAggregate{
		Base:          &base,
		ChildrenCount: cnt,
		Children:      children,
	}, nil
}

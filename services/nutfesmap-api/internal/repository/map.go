package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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

// 親マップ（parent_map_id IS NULL）をまとめて取得し、各親の ChildrenCount と Children を埋めて返す
func (r *MapRepository) FindIndexAggregates(ctx context.Context) ([]*MapAggregate, error) {
	// 1) 親一覧（削除されていない最上位マップ）
	parentRows, err := r.DB.QueryContext(ctx, `
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer parentRows.Close()

	var parents []*MapAggregate
	var parentIDs []string

	for parentRows.Next() {
		var base model.Map
		if err := parentRows.Scan(
			&base.ID, &base.Name, &base.ImageData, &base.NaturalWidth, &base.NaturalHeight,
			&base.ParentMapID, &base.HasFloors, &base.FloorCount, &base.CreatedAt, &base.ModifiedAt, &base.DeletedAt,
		); err != nil {
			return nil, err
		}
		parents = append(parents, &MapAggregate{
			Base:          &base,
			ChildrenCount: 0,
			Children:      []model.MapChildRef{},
		})
		parentIDs = append(parentIDs, base.ID)
	}
	if err := parentRows.Err(); err != nil {
		return nil, err
	}
	if len(parents) == 0 {
		return parents, nil
	}

	// parentID -> Aggregate のインデックス
	parentIdx := make(map[string]*MapAggregate, len(parents))
	for _, ag := range parents {
		parentIdx[ag.Base.ID] = ag
	}

	// 2) 子件数を一括で取得
	// SELECT parent_map_id, COUNT(*) FROM maps WHERE parent_map_id IN (...) AND deleted_at IS NULL GROUP BY parent_map_id;
	cntQuery, cntArgs := buildParentInQuery(`
		SELECT parent_map_id, COUNT(*)
		  FROM maps
		 WHERE deleted_at IS NULL
		   AND parent_map_id IN ({{IN}})
		 GROUP BY parent_map_id
	`, parentIDs)

	cntRows, err := r.DB.QueryContext(ctx, cntQuery, cntArgs...)
	if err != nil {
		return nil, err
	}
	defer cntRows.Close()

	for cntRows.Next() {
		var pid string
		var cnt int
		if err := cntRows.Scan(&pid, &cnt); err != nil {
			return nil, err
		}
		if ag, ok := parentIdx[p_id(pid)]; ok { // p_id は下のヘルパ（単純に戻すだけ）
			ag.ChildrenCount = cnt
		}
	}
	if err := cntRows.Err(); err != nil {
		return nil, err
	}

	// 3) 子の軽量一覧を一括で取得
	// SELECT id, name, has_floors, floor_count, parent_map_id FROM maps WHERE parent_map_id IN (...) AND deleted_at IS NULL ORDER BY name;
	childQuery, childArgs := buildParentInQuery(`
		SELECT id, name, has_floors, floor_count, parent_map_id
		  FROM maps
		 WHERE deleted_at IS NULL
		   AND parent_map_id IN ({{IN}})
		 ORDER BY name ASC
	`, parentIDs)

	cRows, err := r.DB.QueryContext(ctx, childQuery, childArgs...)
	if err != nil {
		return nil, err
	}
	defer cRows.Close()

	for cRows.Next() {
		var c model.MapChildRef
		var pid string
		if err := cRows.Scan(&c.ID, &c.Name, &c.HasFloors, &c.FloorCount, &pid); err != nil {
			return nil, err
		}
		if ag, ok := parentIdx[p_id(pid)]; ok {
			ag.Children = append(ag.Children, c)
		}
	}
	if err := cRows.Err(); err != nil {
		return nil, err
	}

	return parents, nil
}

// 親IDの IN 句を安全に生成するヘルパ
// tmpl 内の "{{IN}}" を "?, ?, ?" に置換し、引数スライスを返す
func buildParentInQuery(tmpl string, ids []string) (string, []any) {
	if len(ids) == 0 {
		// 呼び出し側でゼロ件を弾いているのでここには通常来ないが、念のため
		return strings.ReplaceAll(tmpl, "{{IN}}", "NULL"), nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	q := strings.ReplaceAll(tmpl, "{{IN}}", placeholders)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return q, args
}

// p_id: プレーンな親IDを返すだけ（将来の前処理フック用に分離）
func p_id(s string) string { return s }

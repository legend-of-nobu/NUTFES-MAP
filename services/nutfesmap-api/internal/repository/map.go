package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

type MapUpdateRequest struct {
	Name          OptionalString `json:"name"`
	ImageData     OptionalString `json:"imageData"`
	NaturalWidth  OptionalInt    `json:"naturalWidth"`
	NaturalHeight OptionalInt    `json:"naturalHeight"`
	ParentMapID   OptionalString `json:"parentMapId"` // ""（null相当）で親を外す
	HasFloors     OptionalBool   `json:"hasFloors"`
	FloorCount    OptionalInt    `json:"floorCount"`
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

// --- 地図メタ取得（/maps/{mapId} GET）用: Aggregate -> Response 変換付きの取得関数を追加 ---

// FindMapResponseByID は、指定IDの Map 本体と直下の子メタ情報をまとめて取得し、外部契約DTOに整形して返す。
// 見つからなかった場合は (nil, nil) を返す。
func (r *MapRepository) FindMapResponseByID(ctx context.Context, id string) (*MapResponse, error) {
	ag, err := r.FindAggregate(ctx, id)
	if err != nil {
		return nil, err
	}
	if ag == nil || ag.Base == nil {
		return nil, nil
	}
	return toMapResponse(ag), nil
}

// 内部: MapAggregate を MapResponse に詰め替える
func toMapResponse(ag *MapAggregate) *MapResponse {
	base := ag.Base

	// 子の軽量一覧を DTO に詰め替え
	children := make([]MapChildRefDTO, 0, len(ag.Children))
	for _, c := range ag.Children {
		children = append(children, MapChildRefDTO{
			ID:         c.ID,
			Name:       c.Name,
			HasFloors:  c.HasFloors,
			FloorCount: c.FloorCount,
		})
	}

	return &MapResponse{
		ID:            base.ID,
		Name:          base.Name,
		ImageData:     base.ImageData,
		NaturalWidth:  base.NaturalWidth,
		NaturalHeight: base.NaturalHeight,
		ParentMapID:   base.ParentMapID,
		HasFloors:     base.HasFloors,
		FloorCount:    base.FloorCount,
		ChildrenCount: ag.ChildrenCount,
		Children:      children,
		CreatedAt:     base.CreatedAt,
		ModifiedAt:    base.ModifiedAt,
	}
}

// -------- 部分更新ロジック --------

// UpdatePartial は maps テーブルを部分更新し、更新後の MapResponse を返す。
// - 指定IDが存在しない場合は (nil, sql.ErrNoRows) を返す。
// - バリデーションエラーは (nil, ErrBadRequest) など任意のエラーで返す（ここでは fmt.Errorf を使用）。
func (r *MapRepository) UpdatePartial(ctx context.Context, id string, req *MapUpdateRequest) (*MapResponse, error) {
	// 1) 現在値の取得（has_floors と floor_count は整合性チェックに使う）
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`, id)

	var cur model.Map
	if err := row.Scan(
		&cur.ID, &cur.Name, &cur.ImageData, &cur.NaturalWidth, &cur.NaturalHeight,
		&cur.ParentMapID, &cur.HasFloors, &cur.FloorCount, &cur.CreatedAt, &cur.ModifiedAt, &cur.DeletedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	// 2) 更新後の値（仮）を作る（受け取っていないフィールドは現状維持）
	newName := cur.Name
	if req.Name.Set {
		newName = strings.TrimSpace(req.Name.Value)
		if len(newName) == 0 {
			return nil, fmt.Errorf("name must not be empty")
		}
	}

	newImage := cur.ImageData
	if req.ImageData.Set {
		// 画像差し替え。空文字の場合は許容しない（base64想定）。
		if strings.TrimSpace(req.ImageData.Value) == "" {
			return nil, fmt.Errorf("imageData must not be empty when provided")
		}
		newImage = req.ImageData.Value
	}

	newW := cur.NaturalWidth
	if req.NaturalWidth.Set {
		if req.NaturalWidth.Value < 1 {
			return nil, fmt.Errorf("naturalWidth must be >= 1")
		}
		newW = req.NaturalWidth.Value
	}

	newH := cur.NaturalHeight
	if req.NaturalHeight.Set {
		if req.NaturalHeight.Value < 1 {
			return nil, fmt.Errorf("naturalHeight must be >= 1")
		}
		newH = req.NaturalHeight.Value
	}

	// parent_map_id は ""（= JSON null）なら NULL にする
	var parentID *string = cur.ParentMapID
	if req.ParentMapID.Set {
		if req.ParentMapID.Value == "" {
			parentID = nil
		} else {
			val := req.ParentMapID.Value
			parentID = &val
		}
	}

	newHasFloors := cur.HasFloors
	if req.HasFloors.Set {
		newHasFloors = req.HasFloors.Value
	}

	newFloorCount := cur.FloorCount
	if req.FloorCount.Set {
		if req.FloorCount.Value < 0 {
			return nil, fmt.Errorf("floorCount must be >= 0")
		}
		newFloorCount = req.FloorCount.Value
	}

	// hasFloors と floorCount の整合性
	// - hasFloors=true なら floorCount>=1 を要求
	// - hasFloors=false なら floorCount は 0 に揃える（要求がなければ自動で0にする）
	if newHasFloors {
		if newFloorCount < 1 {
			return nil, fmt.Errorf("floorCount must be >= 1 when hasFloors is true")
		}
	} else {
		newFloorCount = 0
	}

	// 3) 実際に変更があるフィールドだけ UPDATE を組み立てる
	set := make([]string, 0, 8)
	args := make([]any, 0, 9)

	if req.Name.Set && newName != cur.Name {
		set = append(set, "name = ?")
		args = append(args, newName)
	}
	if req.ImageData.Set && newImage != cur.ImageData {
		set = append(set, "image_data = ?")
		args = append(args, newImage)
	}
	if req.NaturalWidth.Set && newW != cur.NaturalWidth {
		set = append(set, "natural_width = ?")
		args = append(args, newW)
	}
	if req.NaturalHeight.Set && newH != cur.NaturalHeight {
		set = append(set, "natural_height = ?")
		args = append(args, newH)
	}
	if req.ParentMapID.Set {
		if parentID == nil {
			set = append(set, "parent_map_id = NULL")
		} else {
			set = append(set, "parent_map_id = ?")
			args = append(args, *parentID)
		}
	}
	// has_floors / floor_count はセットの有無にかかわらず、上で整合済みの new* と現状が違うなら反映
	if newHasFloors != cur.HasFloors {
		set = append(set, "has_floors = ?")
		args = append(args, newHasFloors)
	}
	if newFloorCount != cur.FloorCount {
		set = append(set, "floor_count = ?")
		args = append(args, newFloorCount)
	}

	// 変更なしならそのまま現在のレスポンスを返す
	if len(set) == 0 {
		return toMapResponse(&MapAggregate{
			Base:          &cur,
			ChildrenCount: 0, // 下で再取得するのでここは未使用
			Children:      []model.MapChildRef{},
		}), nil
	}

	// modified_at 更新
	set = append(set, "modified_at = ?")
	now := time.Now().UTC()
	args = append(args, now)

	// WHERE
	args = append(args, id)

	q := "UPDATE maps SET " + strings.Join(set, ", ") + " WHERE id = ? AND deleted_at IS NULL"
	res, err := r.DB.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		// 競合的に消された、等
		return nil, sql.ErrNoRows
	}

	// 4) 更新後の統合レスポンスを返す
	return r.FindMapResponseByID(ctx, id)
}

// p_id: プレーンな親IDを返すだけ（将来の前処理フック用に分離）
func p_id(s string) string { return s }

// -------- Optional ユーティリティ（JSONで「指定されたか」を判定するための型） --------

type OptionalString struct {
	Set   bool
	Value string
}

func (o *OptionalString) UnmarshalJSON(b []byte) error {
	o.Set = true
	// null の場合は空文字のまま Set=true にしておく（空文字は NULL にしたい箇所で解釈）
	if string(b) == "null" {
		o.Value = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	o.Value = s
	return nil
}

type OptionalInt struct {
	Set   bool
	Value int
}

func (o *OptionalInt) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = 0
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	o.Value = n
	return nil
}

type OptionalBool struct {
	Set   bool
	Value bool
}

func (o *OptionalBool) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = false
		return nil
	}
	var v bool
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = v
	return nil
}

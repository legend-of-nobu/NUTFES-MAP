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

func NewMapRepository(db *sql.DB) *MapRepository { return &MapRepository{DB: db} }

//
// ====== リクエスト/レスポンス DTO ======
//

type MapCreateRequest struct {
	// Swagger準拠: parentMapId のみ
	// null なら root 作成、rootId を指定すると空の floor を1つ作成
	ParentMapID *string `json:"parentMapId"`
}

type MapUpdateRequest struct {
	Name          OptionalString `json:"name"`
	ImageData     OptionalString `json:"imageData"`
	NaturalWidth  OptionalInt    `json:"naturalWidth"`
	NaturalHeight OptionalInt    `json:"naturalHeight"`
	ParentMapID   OptionalString `json:"parentMapId"`
}

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

// FloorStack用DTO（Swagger準拠）
type FloorItemDTO struct {
	FloorIndex int         `json:"floorIndex"`
	Map        MapResponse `json:"map"`
}

type FloorStackResponse struct {
	RootMapID  string         `json:"rootMapId"`
	RootName   string         `json:"rootName"`
	FloorCount int            `json:"floorCount"`
	Items      []FloorItemDTO `json:"items"`
}

//
// ====== 作成（root/floor） ======
//

// 既存 CreateEmptyMapTx は「汎用の子/ルート作成（集約値を更新しない）」に変更
func (r *MapRepository) CreateEmptyMapTx(ctx context.Context, id string, parentID *string) error {
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	now := time.Now().UTC()

	// 親指定がある場合は、親の存在だけを検証（親が root である必要はない）
	if parentID != nil {
		var cnt int
		if err := tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM maps WHERE id = ? AND deleted_at IS NULL FOR UPDATE",
			*parentID,
		).Scan(&cnt); err != nil {
			_ = tx.Rollback()
			return err
		}
		if cnt == 0 {
			_ = tx.Rollback()
			return fmt.Errorf("parent not found: %s", *parentID)
		}
	}

	var parent any
	if parentID == nil {
		parent = nil
	} else {
		parent = *parentID
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO maps
		  (id, name, image_data, natural_width, natural_height, parent_map_id, has_floors, floor_count, created_at, modified_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		id, "", nil, 0, 0, parent, false, 0, now, now,
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// 新規追加: フロア専用の作成（親＝フロアのルート：通常はエリア）
// 親の has_floors=true, floor_count++ を更新する
func (r *MapRepository) CreateEmptyFloorTx(ctx context.Context, id string, floorRootID string) error {
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	now := time.Now().UTC()

	// フロア親（=フロアスタックのルート＝エリア）の存在チェック（FOR UPDATE）
	{
		var cnt int
		if err := tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM maps WHERE id = ? AND deleted_at IS NULL FOR UPDATE",
			floorRootID,
		).Scan(&cnt); err != nil {
			_ = tx.Rollback()
			return err
		}
		if cnt == 0 {
			_ = tx.Rollback()
			return fmt.Errorf("floor root not found: %s", floorRootID)
		}
	}

	// フロア（空Map）を挿入（parent_map_id = floorRootID）
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO maps
		  (id, name, image_data, natural_width, natural_height, parent_map_id, has_floors, floor_count, created_at, modified_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		id, "", nil, 0, 0, floorRootID, false, 0, now, now,
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	// 親（エリア）を更新
	if _, err := tx.ExecContext(ctx, `
		UPDATE maps
		   SET has_floors = TRUE,
		       floor_count = floor_count + 1,
		       modified_at = ?
		 WHERE id = ? AND deleted_at IS NULL
	`, now, floorRootID); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// 仕様に合わせた入口: /maps POST の実体
func (r *MapRepository) CreateByRequest(ctx context.Context, newID string, req *MapCreateRequest) error {
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	now := time.Now().UTC()

	// 親なし（=root作成）
	if req.ParentMapID == nil {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO maps (id, name, image_data, natural_width, natural_height, parent_map_id, has_floors, floor_count, created_at, modified_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
			newID, "", nil, 0, 0, nil, false, 0, now, now,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
		return tx.Commit()
	}

	// 親あり（=通常の子マップ作成）。親の存在チェック（存在しなければ404）
	{
		var cnt int
		if err := tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM maps WHERE id = ? AND deleted_at IS NULL FOR UPDATE",
			*req.ParentMapID,
		).Scan(&cnt); err != nil {
			_ = tx.Rollback()
			return err
		}
		if cnt == 0 {
			_ = tx.Rollback()
			return fmt.Errorf("parent not found: %s", *req.ParentMapID)
		}
	}

	// floor の INSERT
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO maps (id, name, image_data, natural_width, natural_height, parent_map_id, has_floors, floor_count, created_at, modified_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
		newID, "", nil, 0, 0, *req.ParentMapID, false, 0, now, now,
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

//
// ====== 取得（単体・index・階スタック） ======
//

type MapAggregate struct {
	Base          *model.Map
	ChildrenCount int
	Children      []model.MapChildRef
}

func (r *MapRepository) FindAggregate(ctx context.Context, id string) (*MapAggregate, error) {
	row := r.DB.QueryRowContext(ctx, "SELECT id, COALESCE(name, ''), COALESCE(image_data, ''), COALESCE(natural_width, 0), COALESCE(natural_height, 0), parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at FROM maps WHERE id = ? AND deleted_at IS NULL LIMIT 1", id)

	var base model.Map
	var imgStr string
	var parentNS sql.NullString
	var deletedAtN sql.NullTime

	if err := row.Scan(
		&base.ID, &base.Name, &imgStr, &base.NaturalWidth, &base.NaturalHeight,
		&parentNS, &base.HasFloors, &base.FloorCount,
		&base.CreatedAt, &base.ModifiedAt, &deletedAtN,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if parentNS.Valid {
		v := parentNS.String
		base.ParentMapID = &v
	}
	if deletedAtN.Valid {
		t := deletedAtN.Time
		base.DeletedAt = &t
	}
	base.ImageData = imgStr

	// 子件数
	var cnt int
	if err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL", base.ID).Scan(&cnt); err != nil {
		return nil, err
	}

	// 子一覧
	rows, err := r.DB.QueryContext(ctx, "SELECT id, COALESCE(name, ''), has_floors, floor_count FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL ORDER BY name", base.ID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &MapAggregate{Base: &base, ChildrenCount: cnt, Children: children}, nil
}

// Index（root一覧＋各rootの軽量な子情報）
func (r *MapRepository) FindIndexAggregates(ctx context.Context) ([]*MapAggregate, error) {
	parentRows, err := r.DB.QueryContext(ctx, "SELECT id, COALESCE(name, ''), COALESCE(image_data, ''), COALESCE(natural_width, 0), COALESCE(natural_height, 0), parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at FROM maps WHERE parent_map_id IS NULL AND deleted_at IS NULL ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer parentRows.Close()

	var parents []*MapAggregate
	var parentIDs []string
	for parentRows.Next() {
		var base model.Map
		var imgStr string
		var parentNS sql.NullString
		var deletedAtN sql.NullTime

		if err := parentRows.Scan(
			&base.ID, &base.Name, &imgStr, &base.NaturalWidth, &base.NaturalHeight,
			&parentNS, &base.HasFloors, &base.FloorCount,
			&base.CreatedAt, &base.ModifiedAt, &deletedAtN,
		); err != nil {
			return nil, err
		}
		if parentNS.Valid {
			v := parentNS.String
			base.ParentMapID = &v
		}
		if deletedAtN.Valid {
			t := deletedAtN.Time
			base.DeletedAt = &t
		}
		base.ImageData = imgStr

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

	parentIdx := make(map[string]*MapAggregate, len(parents))
	for _, ag := range parents {
		parentIdx[ag.Base.ID] = ag
	}

	// 子件数
	cntQuery, cntArgs := buildParentInQuery("SELECT parent_map_id, COUNT(*) FROM maps WHERE deleted_at IS NULL AND parent_map_id IN ({{IN}}) GROUP BY parent_map_id", parentIDs)
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
		if ag, ok := parentIdx[p_id(pid)]; ok {
			ag.ChildrenCount = cnt
		}
	}
	if err := cntRows.Err(); err != nil {
		return nil, err
	}

	// 子の軽量一覧
	childQuery, childArgs := buildParentInQuery("SELECT id, COALESCE(name, ''), has_floors, floor_count, parent_map_id FROM maps WHERE deleted_at IS NULL AND parent_map_id IN ({{IN}}) ORDER BY name ASC", parentIDs)
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

// anyID がエリア(root)ならその直下、フロアならその親エリア直下を 1F.. の順で返す
func (r *MapRepository) FindFloorStackByAnyID(ctx context.Context, anyID string) (*FloorStackResponse, error) {
	// 1) anyID の aggregate をまず読む（ここで floor の場合は子件数=0 の想定）
	ag, err := r.FindAggregate(ctx, anyID)
	if err != nil {
		return nil, err
	}
	if ag == nil || ag.Base == nil {
		return nil, nil
	}

	// 2) floorRoot を決定する
	//    - anyID が floor のとき: 親の has_floors が TRUE であれば親を root とする
	//    - それ以外（親が floor container でない or 親が存在しない）場合は anyID を root とみなす
	floorRootID := anyID
	rootAg := ag
	if ag.Base.ParentMapID != nil {
		parentID := *ag.Base.ParentMapID
		parentAg, err := r.FindAggregate(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if parentAg != nil && parentAg.Base != nil && parentAg.Base.HasFloors {
			floorRootID = parentID
			rootAg = parentAg
		}
	}
	if rootAg == nil || rootAg.Base == nil {
		rootAg = ag
	}

	// 4) root 直下のフロアを created_at ASC で読む（SQL はテスト定数と完全一致）
	rows, err := r.DB.QueryContext(ctx, `
		SELECT
		  id, COALESCE(name, ''), COALESCE(image_data, ''),
		  COALESCE(natural_width, 0), COALESCE(natural_height, 0),
		  parent_map_id, has_floors, floor_count, created_at, modified_at
		FROM maps
		WHERE parent_map_id = ? AND deleted_at IS NULL
		ORDER BY created_at ASC, id ASC
	`, floorRootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]FloorItemDTO, 0, 8)
	idx := 0
	for rows.Next() {
		idx++
		var m model.Map
		var img string
		var parentNS sql.NullString
		if err := rows.Scan(
			&m.ID, &m.Name, &img, &m.NaturalWidth, &m.NaturalHeight,
			&parentNS, &m.HasFloors, &m.FloorCount, &m.CreatedAt, &m.ModifiedAt,
		); err != nil {
			return nil, err
		}
		if parentNS.Valid {
			v := parentNS.String
			m.ParentMapID = &v
		}
		m.ImageData = img

		items = append(items, FloorItemDTO{
			FloorIndex: idx,
			Map:        *toMapResponse(&MapAggregate{Base: &m}),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &FloorStackResponse{
		RootMapID:  rootAg.Base.ID,
		RootName:   rootAg.Base.Name, // ← ここで必ずルート名が入る
		FloorCount: len(items),
		Items:      items,
	}, nil
}

func buildParentInQuery(tmpl string, ids []string) (string, []any) {
	if len(ids) == 0 {
		return strings.ReplaceAll(tmpl, "{{IN}}", "NULL"), nil
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	q := strings.ReplaceAll(tmpl, "{{IN}}", ph)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return q, args
}

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

func toMapResponse(ag *MapAggregate) *MapResponse {
	base := ag.Base
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
		ChildrenCount: len(ag.Children),
		Children:      children,
		CreatedAt:     base.CreatedAt,
		ModifiedAt:    base.ModifiedAt,
	}
}

//
// ====== 更新（PATCH） ======
//

func (r *MapRepository) UpdatePartial(ctx context.Context, id string, req *MapUpdateRequest) (*MapResponse, error) {
	row := r.DB.QueryRowContext(ctx, "SELECT id, COALESCE(name, ''), COALESCE(image_data, ''), COALESCE(natural_width, 0), COALESCE(natural_height, 0), parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at FROM maps WHERE id = ? AND deleted_at IS NULL LIMIT 1", id)

	var cur model.Map
	var imgStr string
	var parentNS sql.NullString
	var deletedAtN sql.NullTime

	if err := row.Scan(
		&cur.ID, &cur.Name, &imgStr, &cur.NaturalWidth, &cur.NaturalHeight,
		&parentNS, &cur.HasFloors, &cur.FloorCount,
		&cur.CreatedAt, &cur.ModifiedAt, &deletedAtN,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if parentNS.Valid {
		v := parentNS.String
		cur.ParentMapID = &v
	}
	if deletedAtN.Valid {
		t := deletedAtN.Time
		cur.DeletedAt = &t
	}
	cur.ImageData = imgStr

	newName := cur.Name
	if req.Name.Set {
		newName = strings.TrimSpace(req.Name.Value)
		if newName == "" {
			return nil, fmt.Errorf("name must not be empty")
		}
	}

	newImage := cur.ImageData
	if req.ImageData.Set {
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

	var parentID *string = cur.ParentMapID
	if req.ParentMapID.Set {
		if req.ParentMapID.Value == "" {
			parentID = nil
		} else {
			v := req.ParentMapID.Value
			parentID = &v
		}
	}

	set := make([]string, 0, 6)
	args := make([]any, 0, 8)

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

	if len(set) == 0 {
		return toMapResponse(&MapAggregate{
			Base:          &cur,
			ChildrenCount: 0,
			Children:      []model.MapChildRef{},
		}), nil
	}

	set = append(set, "modified_at = ?")
	now := time.Now().UTC()
	args = append(args, now, id)

	q := "UPDATE maps SET " + strings.Join(set, ", ") + " WHERE id = ? AND deleted_at IS NULL"
	res, err := r.DB.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if aff, _ := res.RowsAffected(); aff == 0 {
		return nil, sql.ErrNoRows
	}

	return r.FindMapResponseByID(ctx, id)
}

//
// ====== 削除（CASCADE）／最上階削除 ======
//

// ルートを含む再帰削除（pinsも含めてCTEで削除）
func (r *MapRepository) DeleteCascade(ctx context.Context, rootID string) (int64, int64, error) {
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM maps WHERE id = ? AND deleted_at IS NULL LIMIT 1", rootID).Scan(&exists); err != nil {
		_ = tx.Rollback()
		return 0, 0, err
	}
	if exists == 0 {
		_ = tx.Rollback()
		return 0, 0, sql.ErrNoRows
	}

	resPins, err := tx.ExecContext(ctx, "WITH RECURSIVE submaps AS (SELECT id FROM maps WHERE id = ? AND deleted_at IS NULL UNION ALL SELECT m.id FROM maps m JOIN submaps s ON m.parent_map_id = s.id WHERE m.deleted_at IS NULL) DELETE p FROM pins p JOIN submaps sm ON p.map_id = sm.id", rootID)
	if err != nil {
		_ = tx.Rollback()
		return 0, 0, err
	}
	pinsDeleted, _ := resPins.RowsAffected()

	resMaps, err := tx.ExecContext(ctx, "WITH RECURSIVE submaps AS (SELECT id FROM maps WHERE id = ? AND deleted_at IS NULL UNION ALL SELECT m.id FROM maps m JOIN submaps s ON m.parent_map_id = s.id WHERE m.deleted_at IS NULL) DELETE m FROM maps m JOIN submaps sm ON m.id = sm.id", rootID)
	if err != nil {
		_ = tx.Rollback()
		return 0, 0, err
	}
	mapsDeleted, _ := resMaps.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return mapsDeleted, pinsDeleted, nil
}

// 最上階のみ削除可：anyID（rootでもfloorでも可）＋ floorIndex（1..floor_count）
func (r *MapRepository) DeleteTopFloorByIndex(ctx context.Context, anyID string, floorIndex int) error {
	if floorIndex < 1 {
		return fmt.Errorf("floorIndex must be >= 1")
	}

	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// anyID の親をロック取得（rootなら NULL が返る）
	var parentID sql.NullString
	if err := tx.QueryRowContext(ctx, `
		SELECT parent_map_id FROM maps WHERE id = ? AND deleted_at IS NULL FOR UPDATE
	`, anyID).Scan(&parentID); err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return err
	}

	// floor root の決定（root のときは anyID、floor のときは親ID）
	floorRootID := anyID
	if parentID.Valid {
		floorRootID = parentID.String
	}

	// root の floor_count をロック（テストの SQL と一致：parent_map_id IS NULL を含む）
	var floorCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT floor_count FROM maps
		WHERE id = ? AND parent_map_id IS NULL AND deleted_at IS NULL
		FOR UPDATE
	`, floorRootID).Scan(&floorCount); err != nil {
		_ = tx.Rollback()
		return err
	}
	if floorCount == 0 {
		_ = tx.Rollback()
		return fmt.Errorf("no floors to delete")
	}
	if floorIndex != floorCount {
		_ = tx.Rollback()
		return fmt.Errorf("only top floor (index=%d) can be deleted", floorCount)
	}

	// 最上階（created_at ASC の floorIndex 件目 = OFFSET floorIndex-1）のIDを取得
	var targetID string
	if err := tx.QueryRowContext(ctx, `
		SELECT id
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY created_at ASC, id ASC
		 LIMIT 1 OFFSET ?
	`, floorRootID, floorIndex-1).Scan(&targetID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// pins -> map の順で削除
	if _, err := tx.ExecContext(ctx, `DELETE FROM pins WHERE map_id = ?`, targetID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM maps WHERE id = ?`, targetID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// root の集約更新
	now := time.Now().UTC()
	newCount := floorCount - 1
	hasFloors := newCount > 0
	if _, err := tx.ExecContext(ctx, `
		UPDATE maps
		   SET floor_count = ?, has_floors = ?, modified_at = ?
		 WHERE id = ? AND deleted_at IS NULL
	`, newCount, hasFloors, now, floorRootID); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

//
// ====== ユーティリティ／Optional ======
//

func p_id(s string) string { return s }

// Optional types
type OptionalString struct {
	Set   bool
	Value string
}

func (o *OptionalString) UnmarshalJSON(b []byte) error {
	o.Set = true
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

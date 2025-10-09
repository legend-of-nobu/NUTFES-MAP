package repository_test

import (
	"context"
	"testing"
	"time"

	repository "nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
)

// --- helpers ---

func newMock(t *testing.T) (*repository.MapRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	r := repository.NewMapRepository(db)
	cleanup := func() { _ = db.Close() }
	return r, mock, cleanup
}

// -----------------------------------------------------------------------------
// SQL（リポジトリ実装と**完全一致**の文字列）
// -----------------------------------------------------------------------------

const selectOneSQL = "SELECT id, COALESCE(name, ''), COALESCE(image_data, ''), COALESCE(natural_width, 0), COALESCE(natural_height, 0), parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at FROM maps WHERE id = ? AND deleted_at IS NULL LIMIT 1"

const selectParentsSQL = "SELECT id, COALESCE(name, ''), COALESCE(image_data, ''), COALESCE(natural_width, 0), COALESCE(natural_height, 0), parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at FROM maps WHERE parent_map_id IS NULL AND deleted_at IS NULL ORDER BY created_at DESC"

const countChildrenSQL = "SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL"

const selectChildrenSQL = "SELECT id, COALESCE(name, ''), has_floors, floor_count FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL ORDER BY name"

// 実装の CreateEmptyMapTx は「parent_map_id IS NULL」を付けない形で存在チェックしている
const rootExistLockSQL = "SELECT COUNT(*) FROM maps WHERE id = ? AND deleted_at IS NULL FOR UPDATE"

const insertMapSQL = "INSERT INTO maps (id, name, image_data, natural_width, natural_height, parent_map_id, has_floors, floor_count, created_at, modified_at) VALUES (?,?,?,?,?,?,?,?,?,?)"

const idxCountSQL_2 = "SELECT parent_map_id, COUNT(*) FROM maps WHERE deleted_at IS NULL AND parent_map_id IN (?,?) GROUP BY parent_map_id"

const idxChildrenSQL_2 = "SELECT id, COALESCE(name, ''), has_floors, floor_count, parent_map_id FROM maps WHERE deleted_at IS NULL AND parent_map_id IN (?,?) ORDER BY name ASC"

const deleteExistsSQL = "SELECT COUNT(*) FROM maps WHERE id = ? AND deleted_at IS NULL LIMIT 1"

const deletePinsCTE = "WITH RECURSIVE submaps AS (SELECT id FROM maps WHERE id = ? AND deleted_at IS NULL UNION ALL SELECT m.id FROM maps m JOIN submaps s ON m.parent_map_id = s.id WHERE m.deleted_at IS NULL) DELETE p FROM pins p JOIN submaps sm ON p.map_id = sm.id"

const deleteMapsCTE = "WITH RECURSIVE submaps AS (SELECT id FROM maps WHERE id = ? AND deleted_at IS NULL UNION ALL SELECT m.id FROM maps m JOIN submaps s ON m.parent_map_id = s.id WHERE m.deleted_at IS NULL) DELETE m FROM maps m JOIN submaps sm ON m.id = sm.id"

// FindFloorStackByAnyID が floors を読む際のクエリ（実装のインデント・改行そのまま）
const floorsOfRootSQL = `
		SELECT
		  id, COALESCE(name, ''), COALESCE(image_data, ''),
		  COALESCE(natural_width, 0), COALESCE(natural_height, 0),
		  parent_map_id, has_floors, floor_count, created_at, modified_at
		FROM maps
		WHERE parent_map_id = ? AND deleted_at IS NULL
		ORDER BY created_at ASC, id ASC
	`

// DeleteTopFloorByIndex で anyID -> root 解決に使うロック付クエリ
const parentResolveLockSQL = `
		SELECT parent_map_id FROM maps WHERE id = ? AND deleted_at IS NULL FOR UPDATE
	`

// DeleteTopFloorByIndex で root の floor_count をロック取得
const rootFloorCountLockSQL = `
		SELECT floor_count FROM maps
		WHERE id = ? AND parent_map_id IS NULL AND deleted_at IS NULL
		FOR UPDATE
	`

// DeleteTopFloorByIndex で “最上階（created_at ASC の floorIndex 件目）” を取得
const selectNthFloorSQL = `
		SELECT id
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY created_at ASC, id ASC
		 LIMIT 1 OFFSET ?
	`

// DeleteTopFloorByIndex の更新（UPDATE maps ...）
const decRootFloorSQL = `
		UPDATE maps
		   SET floor_count = ?, has_floors = ?, modified_at = ?
		 WHERE id = ? AND deleted_at IS NULL
	`

// -----------------------------------------------------------------------------
// CreateByRequest（/maps POST の実体）
// -----------------------------------------------------------------------------

func TestMapRepository_CreateByRequest_CreateRoot_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	newID := "map_root_1"
	var parent *string = nil

	mock.ExpectBegin()

	// INSERT（has_floors=false, floor_count=0 を明示）
	mock.ExpectExec(insertMapSQL).
		WithArgs(newID, "", nil, 0, 0, nil, false, 0, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	if err := r.CreateByRequest(context.Background(), newID, &repository.MapCreateRequest{ParentMapID: parent}); err != nil {
		t.Fatalf("CreateByRequest(root) returned err: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_CreateByRequest_CreateChild_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	newID := "map_floor_1"
	rootID := "root_1"

	mock.ExpectBegin()

	// 親 root の存在チェック + FOR UPDATE
	mock.ExpectQuery(rootExistLockSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	// 子マップ行の INSERT（親ID設定, has_floors=false, floor_count=0）
	mock.ExpectExec(insertMapSQL).
		WithArgs(newID, "", nil, 0, 0, rootID, false, 0, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	if err := r.CreateByRequest(context.Background(), newID, &repository.MapCreateRequest{ParentMapID: &rootID}); err != nil {
		t.Fatalf("CreateByRequest(child) returned err: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_CreateByRequest_ParentNotFound_Rollback(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	newID := "map_floor_x"
	rootID := "missing_root"

	mock.ExpectBegin()

	// 親 root 無し（0件）
	mock.ExpectQuery(rootExistLockSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(0))

	mock.ExpectRollback()

	if err := r.CreateByRequest(context.Background(), newID, &repository.MapCreateRequest{ParentMapID: &rootID}); err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// FindAggregate
// -----------------------------------------------------------------------------

func TestMapRepository_FindAggregate_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	// 1) 本体行
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	now := time.Now().UTC()
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Campus 2025", "IMGPNG", 4096, 3072,
		nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 2) 子件数
	mock.ExpectQuery(countChildrenSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 3) 子一覧
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("map_201", "1F", false, 0).
		AddRow("map_202", "2F", false, 0)
	mock.ExpectQuery(selectChildrenSQL).
		WithArgs(id).
		WillReturnRows(childRows)

	agg, err := r.FindAggregate(context.Background(), id)
	if err != nil {
		t.Fatalf("FindAggregate returned err: %v", err)
	}
	if agg == nil || agg.Base == nil {
		t.Fatalf("FindAggregate returned nil aggregate")
	}
	if agg.Base.ID != id || agg.ChildrenCount != 2 || len(agg.Children) != 2 {
		t.Fatalf("unexpected aggregate: %#v", agg)
	}
	if agg.Base.ImageData != "IMGPNG" {
		t.Fatalf("unexpected image_data: %s", agg.Base.ImageData)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindAggregate_NoRows(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_not_found"
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(mainCols))

	agg, err := r.FindAggregate(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if agg != nil {
		t.Fatalf("expected nil, got: %#v", agg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// FindIndexAggregates
// -----------------------------------------------------------------------------

func TestMapRepository_FindIndexAggregates_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	now := time.Now().UTC()

	// 1) 親一覧（2件）
	parentCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	parentRows := sqlmock.NewRows(parentCols).
		AddRow("p1", "Campus A", "IMG_A", 4096, 3072, nil, true, 3, now, now, nil).
		AddRow("p2", "Campus B", "IMG_B", 2048, 1536, nil, false, 0, now, now, nil)

	mock.ExpectQuery(selectParentsSQL).
		WillReturnRows(parentRows)

	// 2) 子件数（親ごとに集約）
	mock.ExpectQuery(idxCountSQL_2).
		WithArgs("p1", "p2").
		WillReturnRows(
			sqlmock.NewRows([]string{"parent_map_id", "count"}).
				AddRow("p1", 2).
				AddRow("p2", 1),
		)

	// 3) 子の軽量一覧
	childCols := []string{"id", "name", "has_floors", "floor_count", "parent_map_id"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("c11", "1F", false, 0, "p1").
		AddRow("c12", "2F", false, 0, "p1").
		AddRow("c21", "展示エリア", false, 0, "p2")

	mock.ExpectQuery(idxChildrenSQL_2).
		WithArgs("p1", "p2").
		WillReturnRows(childRows)

	ags, err := r.FindIndexAggregates(context.Background())
	if err != nil {
		t.Fatalf("FindIndexAggregates returned err: %v", err)
	}
	if len(ags) != 2 {
		t.Fatalf("want 2 parents, got %d", len(ags))
	}
	if ags[0].Base.ID != "p1" || ags[0].ChildrenCount != 2 || len(ags[0].Children) != 2 {
		t.Fatalf("unexpected aggregate for p1: %+v", ags[0])
	}
	if ags[1].Base.ID != "p2" || ags[1].ChildrenCount != 1 || len(ags[1].Children) != 1 {
		t.Fatalf("unexpected aggregate for p2: %+v", ags[1])
	}
	if ags[0].Base.ImageData != "IMG_A" || ags[1].Base.ImageData != "IMG_B" {
		t.Fatalf("unexpected images: %s / %s", ags[0].Base.ImageData, ags[1].Base.ImageData)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// FindMapResponseByID
// -----------------------------------------------------------------------------

func TestMapRepository_FindMapResponseByID_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	now := time.Now().UTC()

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Campus 2025", "RAWIMG", 4096, 3072,
		nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(mainRow)

	mock.ExpectQuery(countChildrenSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 子一覧
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("map_201", "1F", false, 0).
		AddRow("map_202", "2F", false, 0)
	mock.ExpectQuery(selectChildrenSQL).
		WithArgs(id).
		WillReturnRows(childRows)

	res, err := r.FindMapResponseByID(context.Background(), id)
	if err != nil {
		t.Fatalf("FindMapResponseByID returned err: %v", err)
	}
	if res == nil {
		t.Fatalf("FindMapResponseByID returned nil")
	}
	if res.ID != id ||
		res.Name != "Campus 2025" ||
		res.ImageData != "RAWIMG" ||
		res.NaturalWidth != 4096 ||
		res.NaturalHeight != 3072 ||
		res.ParentMapID != nil ||
		!res.HasFloors ||
		res.FloorCount != 3 {
		t.Fatalf("unexpected base fields: %+v", res)
	}
	if res.ChildrenCount != 2 || len(res.Children) != 2 {
		t.Fatalf("unexpected children meta: count=%d len=%d", res.ChildrenCount, len(res.Children))
	}
	if res.Children[0].ID != "map_201" || res.Children[0].Name != "1F" {
		t.Fatalf("unexpected first child: %+v", res.Children[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindMapResponseByID_NoRows(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_not_found"

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(mainCols))

	res, err := r.FindMapResponseByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil, got: %#v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// UpdatePartial（PATCH）
// -----------------------------------------------------------------------------

func TestMapRepository_UpdatePartial_UpdateSomeFields_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	now := time.Now().UTC()

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Old Name", "OLD", 1024, 768,
		nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 更新：name, natural_width, parent_map_id, modified_at
	updateSQL := "UPDATE maps SET name = ?, natural_width = ?, parent_map_id = ?, modified_at = ? WHERE id = ? AND deleted_at IS NULL"
	mock.ExpectExec(updateSQL).
		WithArgs("New Campus", 2048, "parent_1", sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	afterRow := sqlmock.NewRows(mainCols).AddRow(
		id, "New Campus", "OLD", 2048, 768,
		"parent_1", false, 0, now.Add(-time.Hour), now.Add(time.Minute), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(afterRow)

	mock.ExpectQuery(countChildrenSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	childCols := []string{"id", "name", "has_floors", "floor_count"}
	mock.ExpectQuery(selectChildrenSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(childCols))

	req := &repository.MapUpdateRequest{
		Name:         repository.OptionalString{Set: true, Value: "New Campus"},
		NaturalWidth: repository.OptionalInt{Set: true, Value: 2048},
		ParentMapID:  repository.OptionalString{Set: true, Value: "parent_1"},
	}

	got, err := r.UpdatePartial(context.Background(), id, req)
	if err != nil {
		t.Fatalf("UpdatePartial returned err: %v", err)
	}
	if got == nil || got.ID != id || got.Name != "New Campus" || got.NaturalWidth != 2048 {
		t.Fatalf("unexpected updated response: %+v", got)
	}
	if got.ParentMapID == nil || *got.ParentMapID != "parent_1" {
		t.Fatalf("expected parent_map_id=parent_1, got: %+v", got.ParentMapID)
	}
	if got.ImageData != "OLD" {
		t.Fatalf("unexpected image after update: %s", got.ImageData)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_UpdatePartial_ClearParentToNULL_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_456"
	now := time.Now().UTC()

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Bldg", "IMG", 3000, 2000,
		"parent_old", true, 5, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(mainRow)

	updateSQL := "UPDATE maps SET parent_map_id = NULL, modified_at = ? WHERE id = ? AND deleted_at IS NULL"
	mock.ExpectExec(updateSQL).
		WithArgs(sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	afterRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Bldg", "IMG", 3000, 2000,
		nil, true, 5, now.Add(-time.Hour), now.Add(time.Minute), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(afterRow)

	mock.ExpectQuery(countChildrenSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	childCols := []string{"id", "name", "has_floors", "floor_count"}
	mock.ExpectQuery(selectChildrenSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(childCols))

	req := &repository.MapUpdateRequest{
		ParentMapID: repository.OptionalString{Set: true, Value: ""},
	}

	got, err := r.UpdatePartial(context.Background(), id, req)
	if err != nil {
		t.Fatalf("UpdatePartial returned err: %v", err)
	}
	if got.ParentMapID != nil {
		t.Fatalf("expected parent_map_id=NULL, got: %+v", got.ParentMapID)
	}
	if got.ImageData != "IMG" {
		t.Fatalf("unexpected image: %s", got.ImageData)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_UpdatePartial_NoChange_NoUpdate(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_same"
	now := time.Now().UTC()

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Same", "IMG", 1000, 800,
		nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(mainRow)

	req := &repository.MapUpdateRequest{}

	got, err := r.UpdatePartial(context.Background(), id, req)
	if err != nil {
		t.Fatalf("UpdatePartial returned err: %v", err)
	}
	if got == nil || got.ID != id || got.Name != "Same" || got.NaturalWidth != 1000 || got.NaturalHeight != 800 {
		t.Fatalf("unexpected response for no-change patch: %+v", got)
	}
	if got.ImageData != "IMG" {
		t.Fatalf("unexpected image: %s", got.ImageData)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// DeleteCascade
// -----------------------------------------------------------------------------

func TestMapRepository_DeleteCascade_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "map_root"

	mock.ExpectBegin()

	mock.ExpectQuery(deleteExistsSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	mock.ExpectExec(deletePinsCTE).
		WithArgs(rootID).
		WillReturnResult(sqlmock.NewResult(0, 5))

	mock.ExpectExec(deleteMapsCTE).
		WithArgs(rootID).
		WillReturnResult(sqlmock.NewResult(0, 3))

	mock.ExpectCommit()

	mapsDel, pinsDel, err := r.DeleteCascade(context.Background(), rootID)
	if err != nil {
		t.Fatalf("DeleteCascade returned err: %v", err)
	}
	if mapsDel != 3 || pinsDel != 5 {
		t.Fatalf("unexpected affected rows: maps=%d pins=%d", mapsDel, pinsDel)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// FindFloorStackByAnyID（root指定）
// -----------------------------------------------------------------------------

func TestMapRepository_FindFloorStackByAnyID_AsRoot_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	now := time.Now().UTC()
	rootID := "root_A"

	// 1) anyID=root の aggregate
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	// 1) anyID=root の aggregate
	rootRow1 := sqlmock.NewRows(mainCols).AddRow(
		rootID, "講義棟エリア", "IMG", 3000, 2000, nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).WithArgs(rootID).WillReturnRows(rootRow1)
	mock.ExpectQuery(countChildrenSQL).WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(selectChildrenSQL).WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// 2) floors（created_at ASC）
	floorCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at",
	}
	floorRows := sqlmock.NewRows(floorCols).
		AddRow("f1", "1F", "", 0, 0, rootID, false, 0, now.Add(-3*time.Hour), now.Add(-3*time.Hour)).
		AddRow("f2", "2F", "", 0, 0, rootID, false, 0, now.Add(-2*time.Hour), now.Add(-2*time.Hour)).
		AddRow("f3", "3F", "", 0, 0, rootID, false, 0, now.Add(-1*time.Hour), now.Add(-1*time.Hour))
	mock.ExpectQuery(floorsOfRootSQL).WithArgs(rootID).WillReturnRows(floorRows)

	got, err := r.FindFloorStackByAnyID(context.Background(), rootID)
	if err != nil {
		t.Fatalf("FindFloorStackByAnyID(root) err: %v", err)
	}
	if got == nil || got.RootMapID != rootID || got.FloorCount != 3 || len(got.Items) != 3 {
		t.Fatalf("unexpected stack: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// FindFloorStackByAnyID（floor指定→親rootへ解決）
// -----------------------------------------------------------------------------

func TestMapRepository_FindFloorStackByAnyID_AsFloor_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	now := time.Now().UTC()
	rootID := "root_A"
	floorID := "f2"

	// 1) anyID=floor の aggregate（親あり）
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	floorRow := sqlmock.NewRows(mainCols).AddRow(
		floorID, "2F", "", 0, 0, rootID, false, 0, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).WithArgs(floorID).WillReturnRows(floorRow)
	mock.ExpectQuery(countChildrenSQL).WithArgs(floorID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(selectChildrenSQL).WithArgs(floorID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// 2) root の aggregate
	rootRow := sqlmock.NewRows(mainCols).AddRow(
		rootID, "講義棟エリア", "IMG", 3000, 2000, nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).WithArgs(rootID).WillReturnRows(rootRow)
	mock.ExpectQuery(countChildrenSQL).WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(selectChildrenSQL).WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// 3) floors 一覧
	floorCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at",
	}
	floorRows := sqlmock.NewRows(floorCols).
		AddRow("f1", "1F", "", 0, 0, rootID, false, 0, now.Add(-3*time.Hour), now.Add(-3*time.Hour)).
		AddRow("f2", "2F", "", 0, 0, rootID, false, 0, now.Add(-2*time.Hour), now.Add(-2*time.Hour)).
		AddRow("f3", "3F", "", 0, 0, rootID, false, 0, now.Add(-1*time.Hour), now.Add(-1*time.Hour))
	mock.ExpectQuery(floorsOfRootSQL).WithArgs(rootID).WillReturnRows(floorRows)

	got, err := r.FindFloorStackByAnyID(context.Background(), floorID)
	if err != nil {
		t.Fatalf("FindFloorStackByAnyID(floor) err: %v", err)
	}
	if got == nil || got.RootMapID != rootID || got.FloorCount != 3 {
		t.Fatalf("unexpected stack: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// FindFloorStackByAnyID（フロア無し）
// -----------------------------------------------------------------------------

func TestMapRepository_FindFloorStackByAnyID_NoFloors_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	now := time.Now().UTC()
	rootID := "root_empty"

	// 1) anyID=root の aggregate
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	// 1) anyID=root の aggregate
	rootRow1 := sqlmock.NewRows(mainCols).AddRow(
		rootID, "屋外エリア", "IMG", 2000, 1500, nil, false, 0, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).WithArgs(rootID).WillReturnRows(rootRow1)
	mock.ExpectQuery(countChildrenSQL).WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(selectChildrenSQL).WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// 2) floors は 0 件
	floorCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at",
	}
	mock.ExpectQuery(floorsOfRootSQL).WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows(floorCols))

	got, err := r.FindFloorStackByAnyID(context.Background(), rootID)
	if err != nil {
		t.Fatalf("FindFloorStackByAnyID(root) err: %v", err)
	}
	if got == nil || got.RootMapID != rootID || got.FloorCount != 0 || len(got.Items) != 0 {
		t.Fatalf("unexpected stack: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// FindFloorStackByAnyID（エリア: 親はあるが has_floors=false）
// -----------------------------------------------------------------------------

func TestMapRepository_FindFloorStackByAnyID_AreaChild_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	now := time.Now().UTC()
	indexID := "index_root"
	areaID := "area_child"

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}

	// 1) anyID=area の aggregate（親は index, has_floors=false）
	areaRow := sqlmock.NewRows(mainCols).AddRow(
		areaID, "講義棟エリア", "", 2048, 1536,
		indexID, false, 0, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).WithArgs(areaID).WillReturnRows(areaRow)
	mock.ExpectQuery(countChildrenSQL).WithArgs(areaID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(selectChildrenSQL).WithArgs(areaID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// 2) 親 index の aggregate（has_floors=false のため root には採用されない）
	indexRow := sqlmock.NewRows(mainCols).AddRow(
		indexID, "IndexMap", "", 4096, 3072,
		nil, false, 0, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).WithArgs(indexID).WillReturnRows(indexRow)
	mock.ExpectQuery(countChildrenSQL).WithArgs(indexID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(selectChildrenSQL).WithArgs(indexID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// 3) floors クエリは area 配下を対象とする
	floorCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at",
	}
	floorRows := sqlmock.NewRows(floorCols).
		AddRow("floor1", "1F", "", 0, 0, areaID, false, 0, now.Add(-time.Hour), now.Add(-time.Hour))
	mock.ExpectQuery(floorsOfRootSQL).WithArgs(areaID).WillReturnRows(floorRows)

	got, err := r.FindFloorStackByAnyID(context.Background(), areaID)
	if err != nil {
		t.Fatalf("FindFloorStackByAnyID(area) err: %v", err)
	}
	if got == nil || got.RootMapID != areaID || got.FloorCount != 1 {
		t.Fatalf("unexpected stack: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// DeleteTopFloorByIndex（最上階OK）
// -----------------------------------------------------------------------------

func TestMapRepository_DeleteTopFloorByIndex_TopOK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "root_A"

	mock.ExpectBegin()

	// anyID=root → parent 解決（NULL）
	mock.ExpectQuery(parentResolveLockSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"parent_map_id"}).AddRow(nil))

	// root の floor_count=3 をロック取得
	mock.ExpectQuery(rootFloorCountLockSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"floor_count"}).AddRow(3))

	// 最上階 index=3 → OFFSET=2 で id 取得
	mock.ExpectQuery(selectNthFloorSQL).
		WithArgs(rootID, 2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("f3"))

	// pins 削除 → map 削除
	mock.ExpectExec("DELETE FROM pins WHERE map_id = ?").
		WithArgs("f3").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("DELETE FROM maps WHERE id = ?").
		WithArgs("f3").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// root の集約更新（2, true）と modified_at
	mock.ExpectExec(decRootFloorSQL).
		WithArgs(2, true, sqlmock.AnyArg(), rootID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	if err := r.DeleteTopFloorByIndex(context.Background(), rootID, 3); err != nil {
		t.Fatalf("DeleteTopFloorByIndex returned err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// DeleteTopFloorByIndex（最上階以外 → エラー）
// -----------------------------------------------------------------------------

func TestMapRepository_DeleteTopFloorByIndex_NotTop_Error(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "root_A"

	mock.ExpectBegin()

	// anyID=root → parent 解決（NULL）
	mock.ExpectQuery(parentResolveLockSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"parent_map_id"}).AddRow(nil))

	// floor_count=3 だが、指定 index=2（最上階ではない）
	mock.ExpectQuery(rootFloorCountLockSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"floor_count"}).AddRow(3))

	// トップ以外なのでロールバック
	mock.ExpectRollback()

	if err := r.DeleteTopFloorByIndex(context.Background(), rootID, 2); err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// DeleteTopFloorByIndex（floor指定→親root解決）
// -----------------------------------------------------------------------------

func TestMapRepository_DeleteTopFloorByIndex_FromFloor_TopOK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "root_A"
	floorID := "f3" // anyID が floor のケース

	mock.ExpectBegin()

	// anyID=floor → parent 解決で root を取得
	mock.ExpectQuery(parentResolveLockSQL).
		WithArgs(floorID).
		WillReturnRows(sqlmock.NewRows([]string{"parent_map_id"}).AddRow(rootID))

	// root の floor_count=3 をロック取得
	mock.ExpectQuery(rootFloorCountLockSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"floor_count"}).AddRow(3))

	// 最上階 index=3 → OFFSET=2
	mock.ExpectQuery(selectNthFloorSQL).
		WithArgs(rootID, 2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("f3"))

	mock.ExpectExec("DELETE FROM pins WHERE map_id = ?").
		WithArgs("f3").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM maps WHERE id = ?").
		WithArgs("f3").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(decRootFloorSQL).
		WithArgs(2, true, sqlmock.AnyArg(), rootID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	if err := r.DeleteTopFloorByIndex(context.Background(), floorID, 3); err != nil {
		t.Fatalf("DeleteTopFloorByIndex(floor) err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// DeleteTopFloorByIndex（フロア無し → エラー）
// -----------------------------------------------------------------------------

func TestMapRepository_DeleteTopFloorByIndex_NoFloors_Error(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "root_empty"

	mock.ExpectBegin()

	// anyID=root → parent 解決（NULL）
	mock.ExpectQuery(parentResolveLockSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"parent_map_id"}).AddRow(nil))

	// floor_count=0
	mock.ExpectQuery(rootFloorCountLockSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"floor_count"}).AddRow(0))

	// ロールバック
	mock.ExpectRollback()

	if err := r.DeleteTopFloorByIndex(context.Background(), rootID, 1); err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

package repository_test

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"nutfesmap/internal/model"
	repository "nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
)

// --- helpers ---

func newMock(t *testing.T) (*repository.MapRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	r := repository.NewMapRepository(db)
	cleanup := func() { _ = db.Close() }
	return r, mock, cleanup
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

// 改行/複数空白を \s+ に畳み、正規表現として安全にマッチできるようにする
func rx(s string) string {
	s = strings.TrimSpace(s)
	s = regexp.QuoteMeta(s)
	// 改行/タブ/スペースの差異を吸収
	s = strings.ReplaceAll(s, "\\\n", "\\s+")
	s = strings.ReplaceAll(s, "\\t", "\\s+")
	s = strings.ReplaceAll(s, "\\  ", "\\s+")
	s = strings.ReplaceAll(s, "\\ ", "\\s+")
	return s
}

// -----------------------------------------------------------------------------
// CreateEmpty（空マップ作成：ゼロ値を明示挿入） LONGTEXT対応
// -----------------------------------------------------------------------------

func TestMapRepository_CreateEmpty_NoParent_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_empty_1"

	mock.ExpectExec(rx(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height, parent_map_id, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?)
	`)).
		WithArgs(id, "", nil, 0, 0, nil, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := r.CreateEmpty(context.Background(), id, nil); err != nil {
		t.Fatalf("CreateEmpty returned err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_CreateEmpty_WithParent_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_empty_2"
	parent := "parent_1"

	mock.ExpectExec(rx(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height, parent_map_id, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?)
	`)).
		WithArgs(id, "", nil, 0, 0, parent, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := r.CreateEmpty(context.Background(), id, &parent); err != nil {
		t.Fatalf("CreateEmpty returned err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_CreateEmpty_ExecError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_empty_err"

	mock.ExpectExec(rx(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height, parent_map_id, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?)
	`)).
		WithArgs(id, "", nil, 0, 0, nil, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(assertErr("insert failed"))

	if err := r.CreateEmpty(context.Background(), id, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Insert（現在は CreateEmpty と同じゼロ値明示挿入に合わせる） LONGTEXT対応
// -----------------------------------------------------------------------------

func TestMapRepository_Insert_Minimal_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	m := &model.Map{
		ID:          "map_min_1",
		ParentMapID: nil,
	}

	mock.ExpectExec(rx(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height, parent_map_id, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?)
	`)).
		WithArgs(m.ID, "", nil, 0, 0, nil, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := r.Insert(context.Background(), m)
	if err != nil {
		t.Fatalf("Insert returned err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unfulfilled expectations: %v", err)
	}
}

func TestMapRepository_Insert_Minimal_WithParent_ExecError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	parent := "parent_x"
	m := &model.Map{
		ID:          "map_min_err",
		ParentMapID: &parent,
	}

	mock.ExpectExec(rx(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height, parent_map_id, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?)
	`)).
		WithArgs(m.ID, "", nil, 0, 0, parent, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(assertErr("insert failed"))

	err := r.Insert(context.Background(), m)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// SQL 断片（実装に合わせて COALESCE(image_data,'') を使用）
// -----------------------------------------------------------------------------

const selectOneSQL = `
SELECT
  id,
  COALESCE(name, ''),
  COALESCE(image_data, ''),
  COALESCE(natural_width, 0),
  COALESCE(natural_height, 0),
  parent_map_id,
  has_floors,
  floor_count,
  created_at,
  modified_at,
  deleted_at
FROM maps
WHERE id = ? AND deleted_at IS NULL
LIMIT 1
`

const selectParentsSQL = `
SELECT
  id,
  COALESCE(name, ''),
  COALESCE(image_data, ''),
  COALESCE(natural_width, 0),
  COALESCE(natural_height, 0),
  parent_map_id,
  has_floors,
  floor_count,
  created_at,
  modified_at,
  deleted_at
FROM maps
WHERE parent_map_id IS NULL
  AND deleted_at IS NULL
ORDER BY created_at DESC
`

// -----------------------------------------------------------------------------
// FindAggregate
// -----------------------------------------------------------------------------

func TestMapRepository_FindAggregate_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	// 1) 本体行（image_data は LONGTEXT 文字列）
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	now := time.Now().UTC()
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Campus 2025", "IMGPNG", 4096, 3072,
		nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 2) 子件数
	mock.ExpectQuery(rx(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 3) 子一覧（COALESCE(name,'')）
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("map_201", "1F", false, 0).
		AddRow("map_202", "2F", false, 0)
	mock.ExpectQuery(rx(`
		SELECT id, COALESCE(name, ''), has_floors, floor_count
		FROM maps
		WHERE parent_map_id = ? AND deleted_at IS NULL
		ORDER BY name
	`)).
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
	// LONGTEXT はそのまま返ってくる（base64 変換なし）
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
	mock.ExpectQuery(rx(selectOneSQL)).
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

func TestMapRepository_FindAggregate_QueryError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnError(assertErr("select failed"))

	agg, err := r.FindAggregate(context.Background(), id)
	if err == nil {
		t.Fatalf("expected error, got nil; agg=%#v", agg)
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

	mock.ExpectQuery(rx(selectParentsSQL)).
		WillReturnRows(parentRows)

	// 2) 子件数（親ごとに集約）
	mock.ExpectQuery(rx(`
		SELECT parent_map_id, COUNT(*)
		FROM maps
		WHERE deleted_at IS NULL
		  AND parent_map_id IN (?,?)
		GROUP BY parent_map_id
	`)).
		WithArgs("p1", "p2").
		WillReturnRows(
			sqlmock.NewRows([]string{"parent_map_id", "count"}).
				AddRow("p1", 2).
				AddRow("p2", 1),
		)

	// 3) 子の軽量一覧（COALESCE(name,'')）
	childCols := []string{"id", "name", "has_floors", "floor_count", "parent_map_id"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("c11", "1F", false, 0, "p1").
		AddRow("c12", "2F", false, 0, "p1").
		AddRow("c21", "展示エリア", false, 0, "p2")

	mock.ExpectQuery(rx(`
		SELECT id, COALESCE(name, ''), has_floors, floor_count, parent_map_id
		FROM maps
		WHERE deleted_at IS NULL
		  AND parent_map_id IN (?,?)
		ORDER BY name ASC
	`)).
		WithArgs("p1", "p2").
		WillReturnRows(childRows)

	ags, err := r.FindIndexAggregates(context.Background())
	if err != nil {
		t.Fatalf("FindIndexAggregates returned err: %v", err)
	}
	if len(ags) != 2 {
		t.Fatalf("want 2 parents, got %d", len(ags))
	}
	// p1
	if ags[0].Base.ID != "p1" || ags[0].ChildrenCount != 2 || len(ags[0].Children) != 2 {
		t.Fatalf("unexpected aggregate for p1: %+v", ags[0])
	}
	// p2
	if ags[1].Base.ID != "p2" || ags[1].ChildrenCount != 1 || len(ags[1].Children) != 1 {
		t.Fatalf("unexpected aggregate for p2: %+v", ags[1])
	}
	// 画像（LONGTEXT文字列）検証
	if ags[0].Base.ImageData != "IMG_A" || ags[1].Base.ImageData != "IMG_B" {
		t.Fatalf("unexpected images: %s / %s", ags[0].Base.ImageData, ags[1].Base.ImageData)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindIndexAggregates_NoParents(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	parentCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mock.ExpectQuery(rx(selectParentsSQL)).
		WillReturnRows(sqlmock.NewRows(parentCols))

	ags, err := r.FindIndexAggregates(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(ags) != 0 {
		t.Fatalf("want 0 parents, got %d", len(ags))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindIndexAggregates_ParentQueryError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	mock.ExpectQuery(rx(selectParentsSQL)).
		WillReturnError(assertErr("parent select failed"))

	ags, err := r.FindIndexAggregates(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil; ags=%#v", ags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindIndexAggregates_CountQueryError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	now := time.Now().UTC()
	parentCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	parentRows := sqlmock.NewRows(parentCols).
		AddRow("p1", "Campus A", "A", 4096, 3072, nil, true, 3, now, now, nil).
		AddRow("p2", "Campus B", "B", 2048, 1536, nil, false, 0, now, now, nil)

	mock.ExpectQuery(rx(selectParentsSQL)).
		WillReturnRows(parentRows)

	mock.ExpectQuery(rx(`
		SELECT parent_map_id, COUNT(*)
		FROM maps
		WHERE deleted_at IS NULL
		  AND parent_map_id IN (?,?)
		GROUP BY parent_map_id
	`)).
		WithArgs("p1", "p2").
		WillReturnError(assertErr("count select failed"))

	ags, err := r.FindIndexAggregates(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil; ags=%#v", ags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindIndexAggregates_ChildQueryError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	now := time.Now().UTC()
	parentCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	parentRows := sqlmock.NewRows(parentCols).
		AddRow("p1", "Campus A", "A", 4096, 3072, nil, true, 3, now, now, nil).
		AddRow("p2", "Campus B", "B", 2048, 1536, nil, false, 0, now, now, nil)

	mock.ExpectQuery(rx(selectParentsSQL)).
		WillReturnRows(parentRows)

	mock.ExpectQuery(rx(`
		SELECT parent_map_id, COUNT(*)
		FROM maps
		WHERE deleted_at IS NULL
		  AND parent_map_id IN (?,?)
		GROUP BY parent_map_id
	`)).
		WithArgs("p1", "p2").
		WillReturnRows(
			sqlmock.NewRows([]string{"parent_map_id", "count"}).
				AddRow("p1", 2).
				AddRow("p2", 1),
		)

	mock.ExpectQuery(rx(`
		SELECT id, COALESCE(name, ''), has_floors, floor_count, parent_map_id
		FROM maps
		WHERE deleted_at IS NULL
		  AND parent_map_id IN (?,?)
		ORDER BY name ASC
	`)).
		WithArgs("p1", "p2").
		WillReturnError(assertErr("children select failed"))

	ags, err := r.FindIndexAggregates(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil; ags=%#v", ags)
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
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnRows(mainRow)

	mock.ExpectQuery(rx(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 子一覧（COALESCE(name,'')）
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("map_201", "1F", false, 0).
		AddRow("map_202", "2F", false, 0)
	mock.ExpectQuery(rx(`
		SELECT id, COALESCE(name, ''), has_floors, floor_count
		FROM maps
		WHERE parent_map_id = ? AND deleted_at IS NULL
		ORDER BY name
	`)).
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
	mock.ExpectQuery(rx(selectOneSQL)).
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

func TestMapRepository_FindMapResponseByID_QueryError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"

	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnError(assertErr("select failed"))

	res, err := r.FindMapResponseByID(context.Background(), id)
	if err == nil {
		t.Fatalf("expected error, got nil; res=%#v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindMapResponseByID_NoChildren(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	now := time.Now().UTC()

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Empty Child Map", "X", 1024, 768,
		nil, false, 0, now, now, nil,
	)
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnRows(mainRow)

	mock.ExpectQuery(rx(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 子一覧=0行（COALESCE(name,'')）
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	mock.ExpectQuery(rx(`
		SELECT id, COALESCE(name, ''), has_floors, floor_count
		FROM maps
		WHERE parent_map_id = ? AND deleted_at IS NULL
		ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(childCols))

	res, err := r.FindMapResponseByID(context.Background(), id)
	if err != nil {
		t.Fatalf("FindMapResponseByID returned err: %v", err)
	}
	if res == nil {
		t.Fatalf("FindMapResponseByID returned nil")
	}
	if res.ChildrenCount != 0 || len(res.Children) != 0 {
		t.Fatalf("expected no children; got count=%d len=%d", res.ChildrenCount, len(res.Children))
	}
	if res.HasFloors || res.FloorCount != 0 {
		t.Fatalf("unexpected floors flags: hasFloors=%v floorCount=%d", res.HasFloors, res.FloorCount)
	}
	if res.ImageData != "X" {
		t.Fatalf("unexpected image: %s", res.ImageData)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// UpdatePartial（PATCH） LONGTEXT対応（image_data は文字列のまま）
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
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnRows(mainRow)

	mock.ExpectExec(rx(`
		UPDATE maps
		SET name = ?, natural_width = ?, parent_map_id = ?, modified_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`)).
		WithArgs("New Campus", 2048, "parent_1", sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	afterRow := sqlmock.NewRows(mainCols).AddRow(
		id, "New Campus", "OLD", 2048, 768,
		"parent_1", false, 0, now.Add(-time.Hour), now.Add(time.Minute), nil,
	)
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnRows(afterRow)

	mock.ExpectQuery(rx(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	childCols := []string{"id", "name", "has_floors", "floor_count"}
	mock.ExpectQuery(rx(`
		SELECT id, COALESCE(name, ''), has_floors, floor_count
		FROM maps
		WHERE parent_map_id = ? AND deleted_at IS NULL
		ORDER BY name
	`)).
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
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnRows(mainRow)

	mock.ExpectExec(rx(`
		UPDATE maps
		SET parent_map_id = NULL, modified_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`)).
		WithArgs(sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	afterRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Bldg", "IMG", 3000, 2000,
		nil, true, 5, now.Add(-time.Hour), now.Add(time.Minute), nil,
	)
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnRows(afterRow)

	mock.ExpectQuery(rx(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	childCols := []string{"id", "name", "has_floors", "floor_count"}
	mock.ExpectQuery(rx(`
		SELECT id, COALESCE(name, ''), has_floors, floor_count
		FROM maps
		WHERE parent_map_id = ? AND deleted_at IS NULL
		ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(childCols))

	req := &repository.MapUpdateRequest{
		// JSON null を想定（Value="" で NULL 解釈）
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
	mock.ExpectQuery(rx(selectOneSQL)).
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
	// UPDATE/再読込無し
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_UpdatePartial_NotFound(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "missing"

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(mainCols))

	req := &repository.MapUpdateRequest{Name: repository.OptionalString{Set: true, Value: "X"}}

	got, err := r.UpdatePartial(context.Background(), id, req)
	if err == nil {
		t.Fatalf("expected error, got nil; res=%#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_UpdatePartial_UpdateExecError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_err"
	now := time.Now().UTC()

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Old", "IMG", 100, 100, nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(rx(selectOneSQL)).
		WithArgs(id).
		WillReturnRows(mainRow)

	mock.ExpectExec(rx(`
		UPDATE maps
		SET name = ?, modified_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`)).
		WithArgs("New", sqlmock.AnyArg(), id).
		WillReturnError(assertErr("update failed"))

	req := &repository.MapUpdateRequest{
		Name: repository.OptionalString{Set: true, Value: "New"},
	}

	got, err := r.UpdatePartial(context.Background(), id, req)
	if err == nil {
		t.Fatalf("expected exec error, got nil; res=%#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// -----------------------------------------------------------------------------
// DeleteCascade（DELETE）
// -----------------------------------------------------------------------------

func TestMapRepository_DeleteCascade_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "map_root"

	mock.ExpectBegin()

	mock.ExpectQuery(rx(`
		SELECT COUNT(*)
		FROM maps
		WHERE id = ? AND deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	mock.ExpectExec(rx(`
		WITH RECURSIVE submaps AS (
			SELECT id
			FROM maps
			WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT m.id
			FROM maps m
			JOIN submaps s ON m.parent_map_id = s.id
			WHERE m.deleted_at IS NULL
		)
		DELETE p FROM pins p
		JOIN submaps sm ON p.map_id = sm.id
	`)).
		WithArgs(rootID).
		WillReturnResult(sqlmock.NewResult(0, 5))

	mock.ExpectExec(rx(`
		WITH RECURSIVE submaps AS (
			SELECT id
			FROM maps
			WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT m.id
			FROM maps m
			JOIN submaps s ON m.parent_map_id = s.id
			WHERE m.deleted_at IS NULL
		)
		DELETE m FROM maps m
		JOIN submaps sm ON m.id = sm.id
	`)).
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

func TestMapRepository_DeleteCascade_NotFound(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "missing"

	mock.ExpectBegin()
	mock.ExpectQuery(rx(`
		SELECT COUNT(*)
		FROM maps
		WHERE id = ? AND deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(0))
	mock.ExpectRollback()

	mapsDel, pinsDel, err := r.DeleteCascade(context.Background(), rootID)
	if err == nil {
		t.Fatalf("expected sql.ErrNoRows, got nil; maps=%d pins=%d", mapsDel, pinsDel)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_DeleteCascade_ExistSelectError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "map_x"

	mock.ExpectBegin()
	mock.ExpectQuery(rx(`
		SELECT COUNT(*)
		FROM maps
		WHERE id = ? AND deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(rootID).
		WillReturnError(assertErr("exist select failed"))
	mock.ExpectRollback()

	_, _, err := r.DeleteCascade(context.Background(), rootID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_DeleteCascade_DeletePinsError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "map_err_pins"

	mock.ExpectBegin()
	mock.ExpectQuery(rx(`
		SELECT COUNT(*)
		FROM maps
		WHERE id = ? AND deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	mock.ExpectExec(rx(`
		WITH RECURSIVE submaps AS (
			SELECT id
			FROM maps
			WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT m.id
			FROM maps m
			JOIN submaps s ON m.parent_map_id = s.id
			WHERE m.deleted_at IS NULL
		)
		DELETE p FROM pins p
		JOIN submaps sm ON p.map_id = sm.id
	`)).
		WithArgs(rootID).
		WillReturnError(assertErr("delete pins failed"))

	mock.ExpectRollback()

	_, _, err := r.DeleteCascade(context.Background(), rootID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_DeleteCascade_DeleteMapsError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "map_err_maps"

	mock.ExpectBegin()
	mock.ExpectQuery(rx(`
		SELECT COUNT(*)
		FROM maps
		WHERE id = ? AND deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	mock.ExpectExec(rx(`
		WITH RECURSIVE submaps AS (
			SELECT id
			FROM maps
			WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT m.id
			FROM maps m
			JOIN submaps s ON m.parent_map_id = s.id
			WHERE m.deleted_at IS NULL
		)
		DELETE p FROM pins p
		JOIN submaps sm ON p.map_id = sm.id
	`)).
		WithArgs(rootID).
		WillReturnResult(sqlmock.NewResult(0, 2))

	mock.ExpectExec(rx(`
		WITH RECURSIVE submaps AS (
			SELECT id
			FROM maps
			WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT m.id
			FROM maps m
			JOIN submaps s ON m.parent_map_id = s.id
			WHERE m.deleted_at IS NULL
		)
		DELETE m FROM maps m
		JOIN submaps sm ON m.id = sm.id
	`)).
		WithArgs(rootID).
		WillReturnError(assertErr("delete maps failed"))

	mock.ExpectRollback()

	_, _, err := r.DeleteCascade(context.Background(), rootID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_DeleteCascade_CommitError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	rootID := "map_commit_err"

	mock.ExpectBegin()
	mock.ExpectQuery(rx(`
		SELECT COUNT(*)
		FROM maps
		WHERE id = ? AND deleted_at IS NULL
		LIMIT 1
	`)).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	mock.ExpectExec(rx(`
		WITH RECURSIVE submaps AS (
			SELECT id
			FROM maps
			WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT m.id
			FROM maps m
			JOIN submaps s ON m.parent_map_id = s.id
			WHERE m.deleted_at IS NULL
		)
		DELETE p FROM pins p
		JOIN submaps sm ON p.map_id = sm.id
	`)).
		WithArgs(rootID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(rx(`
		WITH RECURSIVE submaps AS (
			SELECT id
			FROM maps
			WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT m.id
			FROM maps m
			JOIN submaps s ON m.parent_map_id = s.id
			WHERE m.deleted_at IS NULL
		)
		DELETE m FROM maps m
		JOIN submaps sm ON m.id = sm.id
	`)).
		WithArgs(rootID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit().WillReturnError(assertErr("commit failed"))

	_, _, err := r.DeleteCascade(context.Background(), rootID)
	if err == nil {
		t.Fatalf("expected commit error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

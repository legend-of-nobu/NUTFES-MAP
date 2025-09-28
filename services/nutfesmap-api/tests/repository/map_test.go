package repository_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"nutfesmap/internal/model"
	repository "nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
)

// --- helpers ---

// newMock は MapRepository と sqlmock をまとめて返す
func newMock(t *testing.T) (*repository.MapRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	r := repository.NewMapRepository(db)
	cleanup := func() { _ = db.Close() }
	return r, mock, cleanup
}

// assertErr はテスト内で分かりやすい固定エラーを作る小道具
type assertErr string

func (e assertErr) Error() string { return string(e) }

// --- Insert ---

func TestMapRepository_Insert_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	// Arrange
	m := &model.Map{
		ID:            "map_123",
		Name:          "Campus 2025",
		ImageData:     "iVBORw0K...",
		NaturalWidth:  4096,
		NaturalHeight: 3072,
		ParentMapID:   nil,
		HasFloors:     true,
		FloorCount:    3,
	}

	// Execの引数のうち created_at / modified_at は time.Now().UTC() なので AnyArg() で受ける
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height,
			parent_map_id, has_floors, floor_count, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`)).
		WithArgs(
			m.ID, m.Name, m.ImageData, m.NaturalWidth, m.NaturalHeight,
			m.ParentMapID, m.HasFloors, m.FloorCount, sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Act
	err := r.Insert(context.Background(), m)

	// Assert
	if err != nil {
		t.Fatalf("Insert returned err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unfulfilled expectations: %v", err)
	}
}

func TestMapRepository_Insert_ExecError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	m := &model.Map{
		ID:            "map_123",
		Name:          "Campus 2025",
		ImageData:     "iVBORw0K...",
		NaturalWidth:  4096,
		NaturalHeight: 3072,
		HasFloors:     false,
		FloorCount:    0,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height,
			parent_map_id, has_floors, floor_count, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`)).
		WithArgs(
			m.ID, m.Name, m.ImageData, m.NaturalWidth, m.NaturalHeight,
			m.ParentMapID, m.HasFloors, m.FloorCount, sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(assertErr("insert failed"))

	err := r.Insert(context.Background(), m)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- FindAggregate ---

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
		id, "Campus 2025", "iVBORw0K...", 4096, 3072,
		nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 2) 子件数
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 3) 子一覧
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("map_201", "1F", false, 0).
		AddRow("map_202", "2F", false, 0)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(childRows)

	// Act
	agg, err := r.FindAggregate(context.Background(), id)

	// Assert
	if err != nil {
		t.Fatalf("FindAggregate returned err: %v", err)
	}
	if agg == nil || agg.Base == nil {
		t.Fatalf("FindAggregate returned nil aggregate")
	}
	if agg.Base.ID != id || agg.ChildrenCount != 2 || len(agg.Children) != 2 {
		t.Fatalf("unexpected aggregate: %#v", agg)
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
	// 0行を返す → nil
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
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
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
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

// --- FindIndexAggregates ---

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
		AddRow("p1", "Campus A", "BASE64A", 4096, 3072, nil, true, 3, now, now, nil).
		AddRow("p2", "Campus B", "BASE64B", 2048, 1536, nil, false, 0, now, now, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).
		WillReturnRows(parentRows)

	// 2) 子件数（親ごとに集約）
	mock.ExpectQuery(regexp.QuoteMeta(`
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

	// 3) 子の軽量一覧（親ID IN で一括）
	childCols := []string{"id", "name", "has_floors", "floor_count", "parent_map_id"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("c11", "1F", false, 0, "p1").
		AddRow("c12", "2F", false, 0, "p1").
		AddRow("c21", "展示エリア", false, 0, "p2")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count, parent_map_id
		  FROM maps
		 WHERE deleted_at IS NULL
		   AND parent_map_id IN (?,?)
		 ORDER BY name ASC
	`)).
		WithArgs("p1", "p2").
		WillReturnRows(childRows)

	// Act
	ags, err := r.FindIndexAggregates(context.Background())

	// Assert
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
	// 親0件
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).WillReturnRows(sqlmock.NewRows(parentCols))

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

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).WillReturnError(assertErr("parent select failed"))

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
		AddRow("p1", "Campus A", "BASE64A", 4096, 3072, nil, true, 3, now, now, nil).
		AddRow("p2", "Campus B", "BASE64B", 2048, 1536, nil, false, 0, now, now, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).WillReturnRows(parentRows)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
		AddRow("p1", "Campus A", "BASE64A", 4096, 3072, nil, true, 3, now, now, nil).
		AddRow("p2", "Campus B", "BASE64B", 2048, 1536, nil, false, 0, now, now, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).WillReturnRows(parentRows)

	mock.ExpectQuery(regexp.QuoteMeta(`
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

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count, parent_map_id
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

// --- FindMapResponseByID ---

func TestMapRepository_FindMapResponseByID_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	now := time.Now().UTC()

	// 1) 本体
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Campus 2025", "iVBORw0K...", 4096, 3072,
		nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 2) 子件数
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 3) 子一覧
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("map_201", "1F", false, 0).
		AddRow("map_202", "2F", false, 0)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(childRows)

	// Act
	res, err := r.FindMapResponseByID(context.Background(), id)

	// Assert
	if err != nil {
		t.Fatalf("FindMapResponseByID returned err: %v", err)
	}
	if res == nil {
		t.Fatalf("FindMapResponseByID returned nil")
	}
	if res.ID != id ||
		res.Name != "Campus 2025" ||
		res.ImageData != "iVBORw0K..." ||
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
	// 0行（= nil）
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
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

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
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

	// 1) 本体
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Campus 2025", "iVBORw0K...", 4096, 3072,
		nil, false, 0, now, now, nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 2) 子件数=0
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 3) 子一覧=0行
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(childCols))

	// Act
	res, err := r.FindMapResponseByID(context.Background(), id)

	// Assert
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
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- UpdatePartial (PATCH) ---

func TestMapRepository_UpdatePartial_UpdateSomeFields_OK(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	now := time.Now().UTC()

	// 現在値（更新前）
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Old Name", "BASE64_OLD", 1024, 768,
		nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 期待: name, natural_width, parent_map_id, has_floors, floor_count を更新 + modified_at
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE maps SET name = ?, natural_width = ?, parent_map_id = ?, has_floors = ?, floor_count = ?, modified_at = ? WHERE id = ? AND deleted_at IS NULL
	`)).
		WithArgs("New Campus", 2048, "parent_1", true, 2, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 更新後の再取得（FindMapResponseByID）
	afterRow := sqlmock.NewRows(mainCols).AddRow(
		id, "New Campus", "BASE64_OLD", 2048, 768,
		"parent_1", true, 2, now.Add(-time.Hour), now.Add(time.Minute), nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(afterRow)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	childCols := []string{"id", "name", "has_floors", "floor_count"}
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(childCols))

	// リクエスト
	req := &repository.MapUpdateRequest{
		Name:         repository.OptionalString{Set: true, Value: "New Campus"},
		NaturalWidth: repository.OptionalInt{Set: true, Value: 2048},
		// naturalHeight は未指定（据置）
		ParentMapID: repository.OptionalString{Set: true, Value: "parent_1"},
		HasFloors:   repository.OptionalBool{Set: true, Value: true},
		FloorCount:  repository.OptionalInt{Set: true, Value: 2},
	}

	// Act
	got, err := r.UpdatePartial(context.Background(), id, req)

	// Assert
	if err != nil {
		t.Fatalf("UpdatePartial returned err: %v", err)
	}
	if got == nil || got.ID != id || got.Name != "New Campus" || got.NaturalWidth != 2048 || !got.HasFloors || got.FloorCount != 2 {
		t.Fatalf("unexpected updated response: %+v", got)
	}
	if got.ParentMapID == nil || *got.ParentMapID != "parent_1" {
		t.Fatalf("expected parent_map_id=parent_1, got: %+v", got.ParentMapID)
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

	// 現在は親つき・階層あり
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Bldg", "BASE64", 3000, 2000,
		"parent_old", true, 5, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 期待: parent_map_id を NULL に、has_floors=false に伴い floor_count=0 へ正規化
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE maps SET parent_map_id = NULL, has_floors = ?, floor_count = ?, modified_at = ? WHERE id = ? AND deleted_at IS NULL
	`)).
		WithArgs(false, 0, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 更新後の再取得
	afterRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Bldg", "BASE64", 3000, 2000,
		nil, false, 0, now.Add(-time.Hour), now.Add(time.Minute), nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(afterRow)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	childCols := []string{"id", "name", "has_floors", "floor_count"}
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(childCols))

	req := &repository.MapUpdateRequest{
		// JSONの null を想定：Value="" で NULL に解釈
		ParentMapID: repository.OptionalString{Set: true, Value: ""},
		HasFloors:   repository.OptionalBool{Set: true, Value: false},
		// FloorCount 未指定でも false に正規化され 0 になる
	}

	got, err := r.UpdatePartial(context.Background(), id, req)
	if err != nil {
		t.Fatalf("UpdatePartial returned err: %v", err)
	}
	if got.ParentMapID != nil {
		t.Fatalf("expected parent_map_id=NULL, got: %+v", got.ParentMapID)
	}
	if got.HasFloors || got.FloorCount != 0 {
		t.Fatalf("expected hasFloors=false & floorCount=0; got hasFloors=%v floorCount=%d", got.HasFloors, got.FloorCount)
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

	// 現在値
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Same", "IMG", 1000, 800,
		nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// リクエストは全フィールド未指定（=変更なし）
	req := &repository.MapUpdateRequest{}

	// Act
	got, err := r.UpdatePartial(context.Background(), id, req)

	// Assert: UPDATE は発行されず、そのまま現在値を詰め替えて返る
	if err != nil {
		t.Fatalf("UpdatePartial returned err: %v", err)
	}
	if got == nil || got.ID != id || got.Name != "Same" || got.NaturalWidth != 1000 || got.NaturalHeight != 800 {
		t.Fatalf("unexpected response for no-change patch: %+v", got)
	}
	// ここでは FindMapResponseByID を呼ばない実装なので、追加のクエリ期待は無し
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
	// 0行 -> sql.ErrNoRows 想定で nil を返す
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
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

func TestMapRepository_UpdatePartial_ValidationError_Floors(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_v"
	now := time.Now().UTC()

	// 現在値（has_floors=false, floor_count=0）
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "X", "IMG", 100, 100, nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// リクエスト: hasFloors=true だが floorCount=0 -> バリデーションエラー
	req := &repository.MapUpdateRequest{
		HasFloors:  repository.OptionalBool{Set: true, Value: true},
		FloorCount: repository.OptionalInt{Set: true, Value: 0},
	}

	got, err := r.UpdatePartial(context.Background(), id, req)
	if err == nil {
		t.Fatalf("expected validation error, got nil; res=%#v", got)
	}
	// UPDATE は走らないのでここで終了
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_UpdatePartial_UpdateExecError(t *testing.T) {
	r, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_err"
	now := time.Now().UTC()

	// 現在値
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Old", "IMG", 100, 100, nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// UPDATE でエラー
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE maps SET name = ?, modified_at = ? WHERE id = ? AND deleted_at IS NULL
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

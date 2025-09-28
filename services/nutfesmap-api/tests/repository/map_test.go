package repository_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"nutfesmap/internal/model"
	repo "nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
)

func newMock(t *testing.T) (*repo.MapRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	repo := repo.NewMapRepository(db)
	cleanup := func() {
		_ = db.Close()
	}
	return repo, mock, cleanup
}

func TestMapRepository_Insert_OK(t *testing.T) {
	repo, mock, cleanup := newMock(t)
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
	err := repo.Insert(context.Background(), m)

	// Assert
	if err != nil {
		t.Fatalf("Insert returned err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unfulfilled expectations: %v", err)
	}
}

func TestMapRepository_Insert_ExecError(t *testing.T) {
	repo, mock, cleanup := newMock(t)
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

	err := repo.Insert(context.Background(), m)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindAggregate_OK(t *testing.T) {
	repo, mock, cleanup := newMock(t)
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
	agg, err := repo.FindAggregate(context.Background(), id)

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
	repo, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_not_found"
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	// 0行を返す → sql.ErrNoRows になる
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(mainCols))

	agg, err := repo.FindAggregate(context.Background(), id)
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
	repo, mock, cleanup := newMock(t)
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

	agg, err := repo.FindAggregate(context.Background(), id)
	if err == nil {
		t.Fatalf("expected error, got nil; agg=%#v", agg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindIndexAggregates_OK(t *testing.T) {
	repo, mock, cleanup := newMock(t)
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
	ags, err := repo.FindIndexAggregates(context.Background())

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
	repo, mock, cleanup := newMock(t)
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

	ags, err := repo.FindIndexAggregates(context.Background())
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
	repo, mock, cleanup := newMock(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).WillReturnError(assertErr("parent select failed"))

	ags, err := repo.FindIndexAggregates(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil; ags=%#v", ags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindIndexAggregates_CountQueryError(t *testing.T) {
	repo, mock, cleanup := newMock(t)
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

	ags, err := repo.FindIndexAggregates(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil; ags=%#v", ags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindIndexAggregates_ChildQueryError(t *testing.T) {
	repo, mock, cleanup := newMock(t)
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

	ags, err := repo.FindIndexAggregates(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil; ags=%#v", ags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- helpers ---

// assertErr はテスト内で分かりやすい固定エラーを作る小道具
type assertErr string

func (e assertErr) Error() string { return string(e) }

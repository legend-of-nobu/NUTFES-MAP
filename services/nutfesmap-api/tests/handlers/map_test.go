package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nutfesmap/internal/handlers"
	"nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// ========= 共通ユーティリティ =========

type customValidator struct{ v *validator.Validate }

func (cv *customValidator) Validate(i any) error { return cv.v.Struct(i) }

// 固定エラータイプ（比較しやすいように）
type assertErr string

func (e assertErr) Error() string { return string(e) }

// Echo + Handler + sqlmock をまとめて起動（クエリは完全一致マッチ）
func setupEchoWithMock(t *testing.T) (*echo.Echo, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	mapRepo := repository.NewMapRepository(db)
	mapH := handlers.NewMapHandler(mapRepo)

	e := echo.New()
	e.Validator = &customValidator{v: validator.New()}

	// ルーティング（本番と同じパス）
	e.POST("/maps", mapH.Create)
	e.GET("/maps/index", mapH.Index)
	e.GET("/maps/:mapId", mapH.Show)
	e.PATCH("/maps/:mapId", mapH.Update)
	e.DELETE("/maps/:mapId", mapH.Delete)

	cleanup := func() { _ = db.Close() }
	return e, mock, cleanup
}

// ========= リポジトリ実装と完全一致させた SQL 文字列 =========

const selectOneSQL = "SELECT id, COALESCE(name, ''), COALESCE(image_data, ''), COALESCE(natural_width, 0), COALESCE(natural_height, 0), parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at FROM maps WHERE id = ? AND deleted_at IS NULL LIMIT 1"

const selectParentsSQL = "SELECT id, COALESCE(name, ''), COALESCE(image_data, ''), COALESCE(natural_width, 0), COALESCE(natural_height, 0), parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at FROM maps WHERE parent_map_id IS NULL AND deleted_at IS NULL ORDER BY created_at DESC"

const countChildrenSQL = "SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL"

const childrenListSQL = "SELECT id, COALESCE(name, ''), has_floors, floor_count FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL ORDER BY name"

const indexCountByParentsSQL = "SELECT parent_map_id, COUNT(*) FROM maps WHERE deleted_at IS NULL AND parent_map_id IN (?,?) GROUP BY parent_map_id"

const indexChildrenByParentsSQL = "SELECT id, COALESCE(name, ''), has_floors, floor_count, parent_map_id FROM maps WHERE deleted_at IS NULL AND parent_map_id IN (?,?) ORDER BY name ASC"

const insertEmptyMapSQL = "INSERT INTO maps (id, name, image_data, natural_width, natural_height, parent_map_id, has_floors, floor_count, created_at, modified_at) VALUES (?,?,?,?,?,?,?,?,?,?)"

const parentCheckForUpdateSQL = "SELECT COUNT(*) FROM maps WHERE id = ? AND parent_map_id IS NULL AND deleted_at IS NULL FOR UPDATE"

const parentAggregateUpdateSQL = "UPDATE maps SET has_floors = TRUE, floor_count = floor_count + 1, modified_at = ? WHERE id = ? AND deleted_at IS NULL"

const selectExistForDeleteSQL = "SELECT COUNT(*) FROM maps WHERE id = ? AND deleted_at IS NULL LIMIT 1"

const deletePinsCascadeSQL = "WITH RECURSIVE submaps AS (SELECT id FROM maps WHERE id = ? AND deleted_at IS NULL UNION ALL SELECT m.id FROM maps m JOIN submaps s ON m.parent_map_id = s.id WHERE m.deleted_at IS NULL) DELETE p FROM pins p JOIN submaps sm ON p.map_id = sm.id"

const deleteMapsCascadeSQL = "WITH RECURSIVE submaps AS (SELECT id FROM maps WHERE id = ? AND deleted_at IS NULL UNION ALL SELECT m.id FROM maps m JOIN submaps s ON m.parent_map_id = s.id WHERE m.deleted_at IS NULL) DELETE m FROM maps m JOIN submaps sm ON m.id = sm.id"

// ========= テスト =========

func TestMapHandler_Create_OK_Minimal(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// トランザクション開始
	mock.ExpectBegin()

	// INSERT（has_floors=false, floor_count=0 を明示する10カラム版）image_data は NULL（LONGTEXT）
	mock.ExpectExec(insertEmptyMapSQL).
		WithArgs(
			sqlmock.AnyArg(), // id
			"",               // name
			nil,              // image_data (NULL 挿入)
			0,                // natural_width
			0,                // natural_height
			nil,              // parent_map_id
			false,            // has_floors
			0,                // floor_count
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // modified_at
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// コミット
	mock.ExpectCommit()

	// main select（空マップ直後）
	now := time.Now().UTC()
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		"map_dummy", "", "", 0, 0,
		nil, false, 0, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(sqlmock.AnyArg()). // newID
		WillReturnRows(mainRow)

	// children count
	mock.ExpectQuery(countChildrenSQL).
		WithArgs("map_dummy").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// children list（0行）
	mock.ExpectQuery(childrenListSQL).
		WithArgs("map_dummy").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// --- リクエスト（parent 未指定） ---
	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp repository.MapResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response json unmarshal error: %v, body=%s", err, rec.Body.String())
	}
	if resp.ID == "" || resp.ParentMapID != nil || resp.ChildrenCount != 0 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Fatalf("ETag header must be set")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Create_WithParent_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// トランザクション開始
	mock.ExpectBegin()

	// 親 root の存在チェック（FOR UPDATE）
	mock.ExpectQuery(parentCheckForUpdateSQL).
		WithArgs("parent_1").
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	// floor 行の INSERT（親ID設定, has_floors=false, floor_count=0）
	mock.ExpectExec(insertEmptyMapSQL).
		WithArgs(
			sqlmock.AnyArg(), // id
			"",               // name
			nil,              // image_data
			0, 0,
			"parent_1", // parent_map_id
			false,      // has_floors
			0,          // floor_count
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 親 root の集約値更新（has_floors=true, floor_count+1）
	mock.ExpectExec(parentAggregateUpdateSQL).
		WithArgs(sqlmock.AnyArg(), "parent_1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// コミット
	mock.ExpectCommit()

	// main select（親が付与されて返る）
	now := time.Now().UTC()
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		"map_dummy", "", "", 0, 0,
		"parent_1", false, 0, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(mainRow)

	mock.ExpectQuery(countChildrenSQL).
		WithArgs("map_dummy").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(childrenListSQL).
		WithArgs("map_dummy").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// リクエスト
	body := []byte(`{"parentMapId":"parent_1"}`)
	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp repository.MapResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.ParentMapID == nil || *resp.ParentMapID != "parent_1" {
		t.Fatalf("expected parent_1, got %+v", resp.ParentMapID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Create_BadJSON(t *testing.T) {
	e, _, cleanup := setupEchoWithMock(t)
	defer cleanup()

	body := []byte(`{"parentMapId":`) // 不正JSON
	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMapHandler_Create_InsertError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// トランザクション開始
	mock.ExpectBegin()

	// Insert（CreateEmpty(nil)）がエラー（10カラム版）
	mock.ExpectExec(insertEmptyMapSQL).
		WithArgs(
			sqlmock.AnyArg(),
			"",  // name
			nil, // image_data (NULL)
			0, 0,
			nil,
			false, 0,
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(assertErr("insert failed"))

	// ロールバック
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader([]byte(`{}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Index_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	now := time.Now().UTC()

	// 1) 親一覧（2件）
	parentCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	parentRows := sqlmock.NewRows(parentCols).
		AddRow("p1", "Campus A", "IMG_A", 4096, 3072, nil, true, 3, now, now, nil).
		AddRow("p2", "Campus B", "IMG_B", 2048, 1536, nil, false, 0, now.Add(-time.Minute), now.Add(-time.Minute), nil)

	mock.ExpectQuery(selectParentsSQL).
		WillReturnRows(parentRows)

	// 2) 子件数の集約
	mock.ExpectQuery(indexCountByParentsSQL).
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

	mock.ExpectQuery(indexChildrenByParentsSQL).
		WithArgs("p1", "p2").
		WillReturnRows(childRows)

	req := httptest.NewRequest(http.MethodGet, "/maps/index", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Fatalf("ETag header must be set")
	}

	var list struct {
		Items []repository.MapResponse `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("json unmarshal error: %v, body=%s", err, rec.Body.String())
	}
	if len(list.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(list.Items))
	}
	if list.Items[0].ID != "p1" || list.Items[0].ChildrenCount != 2 || len(list.Items[0].Children) != 2 {
		t.Fatalf("unexpected p1 aggregate: %+v", list.Items[0])
	}
	if list.Items[0].Children[0].Name != "1F" || list.Items[0].Children[1].Name != "2F" {
		t.Fatalf("children not sorted by name asc: %+v", list.Items[0].Children)
	}
	if list.Items[1].ID != "p2" || list.Items[1].ChildrenCount != 1 || len(list.Items[1].Children) != 1 {
		t.Fatalf("unexpected p2 aggregate: %+v", list.Items[1])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Index_SQL_Error(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	mock.ExpectQuery(selectParentsSQL).
		WillReturnError(assertErr("parent select failed"))

	req := httptest.NewRequest(http.MethodGet, "/maps/index", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Show_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_123"
	now := time.Now().UTC()

	// 1) 本体
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "キャンパスマップ2025", "IMG", 4096, 3072,
		nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 2) 子件数
	mock.ExpectQuery(countChildrenSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 3) 子一覧（順不同→昇順検証はハンドラ内で行う）
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("map_b", "B2F", false, 0).
		AddRow("map_a", "1F", false, 0)
	mock.ExpectQuery(childrenListSQL).
		WithArgs(id).
		WillReturnRows(childRows)

	req := httptest.NewRequest(http.MethodGet, "/maps/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Fatalf("ETag header must be set")
	}

	var resp repository.MapResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error: %v, body=%s", err, rec.Body.String())
	}
	if resp.ID != id || resp.Name != "キャンパスマップ2025" {
		t.Fatalf("unexpected base fields: %+v", resp)
	}
	if resp.ChildrenCount != 2 || len(resp.Children) != 2 {
		t.Fatalf("unexpected children meta: %+v", resp.Children)
	}
	if resp.Children[0].Name != "1F" || resp.Children[1].Name != "B2F" {
		t.Fatalf("children not sorted asc by name: %+v", resp.Children)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Show_NotFound(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_not_found"
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(mainCols))

	req := httptest.NewRequest(http.MethodGet, "/maps/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Show_QueryError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_123"
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnError(assertErr("select failed"))

	req := httptest.NewRequest(http.MethodGet, "/maps/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Show_NoChildren(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_123"
	now := time.Now().UTC()

	// 本体
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Empty Child Map", "IMG", 1024, 768,
		nil, false, 0, now, now, nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 子件数=0
	mock.ExpectQuery(countChildrenSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 子一覧=0
	mock.ExpectQuery(childrenListSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	req := httptest.NewRequest(http.MethodGet, "/maps/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp repository.MapResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error: %v, body=%s", err, rec.Body.String())
	}
	if resp.ChildrenCount != 0 || len(resp.Children) != 0 {
		t.Fatalf("expected no children, got: %+v", resp.Children)
	}
	if resp.HasFloors || resp.FloorCount != 0 {
		t.Fatalf("unexpected floor flags: hasFloors=%v floorCount=%d", resp.HasFloors, resp.FloorCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// --- PATCH /maps/:mapId ---

func TestMapHandler_Update_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_123"
	now := time.Now().UTC()

	// 1) 現在値
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	before := sqlmock.NewRows(mainCols).AddRow(
		id, "Old Name", "OLD", 1024, 768,
		nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(before)

	// 2) UPDATE（floors系は含めない）
	mock.ExpectExec("UPDATE maps SET name = ?, natural_width = ?, parent_map_id = ?, modified_at = ? WHERE id = ? AND deleted_at IS NULL").
		WithArgs("New Campus", 2048, "parent_1", sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 3) 更新後の再取得
	after := sqlmock.NewRows(mainCols).AddRow(
		id, "New Campus", "OLD", 2048, 768,
		"parent_1", false, 0, now.Add(-time.Hour), now.Add(time.Minute), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(after)

	// 子件数/一覧 = 0
	mock.ExpectQuery(countChildrenSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(childrenListSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// リクエスト
	body := map[string]any{
		"name":         "New Campus",
		"naturalWidth": 2048,
		"parentMapId":  "parent_1",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/maps/"+id, bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Fatalf("ETag header must be set")
	}
	var resp repository.MapResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json error: %v body=%s", err, rec.Body.String())
	}
	if resp.ID != id || resp.Name != "New Campus" || resp.NaturalWidth != 2048 {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	if resp.ParentMapID == nil || *resp.ParentMapID != "parent_1" {
		t.Fatalf("expected parent_1, got %+v", resp.ParentMapID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapHandler_Update_ClearParentToNULL_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_456"
	now := time.Now().UTC()

	// 現在は親あり
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	before := sqlmock.NewRows(mainCols).AddRow(
		id, "Bldg", "IMG", 3000, 2000,
		"parent_old", true, 5, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).WillReturnRows(before)

	// UPDATE: 親NULL のみ
	mock.ExpectExec("UPDATE maps SET parent_map_id = NULL, modified_at = ? WHERE id = ? AND deleted_at IS NULL").
		WithArgs(sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 更新後再取得（floorsは据え置き）
	after := sqlmock.NewRows(mainCols).AddRow(
		id, "Bldg", "IMG", 3000, 2000,
		nil, true, 5, now.Add(-time.Hour), now.Add(time.Minute), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).WillReturnRows(after)

	mock.ExpectQuery(countChildrenSQL).
		WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(childrenListSQL).
		WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// PATCH: parentMapId = null
	body := []byte(`{"parentMapId": null}`)
	req := httptest.NewRequest(http.MethodPatch, "/maps/"+id, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp repository.MapResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.ParentMapID != nil {
		t.Fatalf("expected parent_map_id NULL, got %+v", resp.ParentMapID)
	}
	// floorsは据え置き
	if !resp.HasFloors || resp.FloorCount != 5 {
		t.Fatalf("floors should be preserved; got hasFloors=%v floorCount=%d", resp.HasFloors, resp.FloorCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapHandler_Update_NotFound(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "missing"

	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(mainCols))

	req := httptest.NewRequest(http.MethodPatch, "/maps/"+id, bytes.NewReader([]byte(`{"name":"X"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapHandler_Update_ValidationError_EmptyName(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_v"
	now := time.Now().UTC()

	// 現在値
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	before := sqlmock.NewRows(mainCols).AddRow(
		id, "X", "IMG", 100, 100, nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).WillReturnRows(before)

	// name に空文字を指定 → リポジトリでバリデーションエラー（UPDATEは発行されない）
	req := httptest.NewRequest(http.MethodPatch, "/maps/"+id, bytes.NewReader([]byte(`{"name":""}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapHandler_Update_UpdateExecError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_err"
	now := time.Now().UTC()

	// 現在値
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	before := sqlmock.NewRows(mainCols).AddRow(
		id, "Old", "IMG", 100, 100, nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(selectOneSQL).
		WithArgs(id).WillReturnRows(before)

	// UPDATE でエラー
	mock.ExpectExec("UPDATE maps SET name = ?, modified_at = ? WHERE id = ? AND deleted_at IS NULL").
		WithArgs("New", sqlmock.AnyArg(), id).
		WillReturnError(assertErr("update failed"))

	req := httptest.NewRequest(http.MethodPatch, "/maps/"+id, bytes.NewReader([]byte(`{"name":"New"}`)))
	rec := httptest.NewRecorder()
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e.ServeHTTP(rec, req)

	// handler.Update は sql: no rows 以外のエラーは 400 に寄せる
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- DELETE /maps/:mapId ---

func TestMapHandler_Delete_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	rootID := "map_root"

	mock.ExpectBegin()

	// 存在確認
	mock.ExpectQuery(selectExistForDeleteSQL).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	// pins 削除
	mock.ExpectExec(deletePinsCascadeSQL).
		WithArgs(rootID).
		WillReturnResult(sqlmock.NewResult(0, 4))

	// maps 削除
	mock.ExpectExec(deleteMapsCascadeSQL).
		WithArgs(rootID).
		WillReturnResult(sqlmock.NewResult(0, 3))

	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodDelete, "/maps/"+rootID, nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMapHandler_Delete_NotFound(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	missing := "map_missing"

	mock.ExpectBegin()
	mock.ExpectQuery(selectExistForDeleteSQL).
		WithArgs(missing).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(0))
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodDelete, "/maps/"+missing, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMapHandler_Delete_PinsDeleteError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_err_pins"

	mock.ExpectBegin()
	mock.ExpectQuery(selectExistForDeleteSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	mock.ExpectExec(deletePinsCascadeSQL).
		WithArgs(id).
		WillReturnError(assertErr("delete pins failed"))

	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodDelete, "/maps/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// handler.Delete は no rows 以外のエラーはそのまま返す → Echo が 500
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMapHandler_Delete_CommitError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_commit_err"

	mock.ExpectBegin()
	mock.ExpectQuery(selectExistForDeleteSQL).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	mock.ExpectExec(deletePinsCascadeSQL).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(deleteMapsCascadeSQL).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit().WillReturnError(assertErr("commit failed"))

	req := httptest.NewRequest(http.MethodDelete, "/maps/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

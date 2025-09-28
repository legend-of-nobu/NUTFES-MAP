package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"nutfesmap/internal/handlers"
	"nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"

	"github.com/go-playground/validator/v10"
)

// EchoのValidator実装
type customValidator struct{ v *validator.Validate }

func (cv *customValidator) Validate(i any) error { return cv.v.Struct(i) }

// Echo + Handler + sqlmock をまとめて起動
func setupEchoWithMock(t *testing.T) (*echo.Echo, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
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

	cleanup := func() {
		_ = db.Close()
	}
	return e, mock, cleanup
}

func TestMapHandler_Create_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// --- 期待されるSQL（Insert → main select → count → children select） ---
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height,
			parent_map_id, has_floors, floor_count, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`)).
		WithArgs(
			sqlmock.AnyArg(), // id ("map_"+uuid) → ここはハンドラー内で生成済みの値を直接知らないので AnyArg
			"キャンパスマップ2025",
			sqlmock.AnyArg(), // image_data(base64)
			4096,
			3072,
			nil,              // parent_map_id
			true,             // has_floors
			3,                // floor_count
			sqlmock.AnyArg(), // created_at (time.Now().UTC())
			sqlmock.AnyArg(), // modified_at
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// main select
	now := time.Now().UTC()
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		"map_dummy", "キャンパスマップ2025", "iVBORw0K...", 4096, 3072,
		nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(sqlmock.AnyArg()). // ハンドラーで生成した newID
		WillReturnRows(mainRow)

	// children count
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs("map_dummy").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// children list
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs("map_dummy").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"})) // 0行

	// --- リクエスト作成 ---
	reqBody := handlers.MapCreateRequest{
		Name:          "キャンパスマップ2025",
		ImageData:     "iVBORw0K...",
		NaturalWidth:  4096,
		NaturalHeight: 3072,
		HasFloors:     true,
		FloorCount:    3,
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// --- 実行 ---
	e.ServeHTTP(rec, req)

	// --- 検証 ---
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	// レスポンスの最小検証（構造が正しいこと）
	var resp handlers.MapResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response json unmarshal error: %v, body=%s", err, rec.Body.String())
	}
	if resp.Name != reqBody.Name || resp.NaturalWidth != 4096 || resp.ChildrenCount != 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Create_ValidationError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// name が無い → バリデーション 400
	reqBody := map[string]any{
		"imageData":     "iVBORw0K...",
		"naturalWidth":  1024,
		"naturalHeight": 768,
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	// SQL は一切呼ばれない想定
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQLs were executed: %v", err)
	}
}

func TestMapHandler_Create_InsertError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// Insert がエラーを返すパス
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height,
			parent_map_id, has_floors, floor_count, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`)).
		WithArgs(
			sqlmock.AnyArg(),
			"ERR-MAP",
			sqlmock.AnyArg(),
			100, 100,
			nil,
			false, 0,
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(assertErr("insert failed"))

	reqBody := handlers.MapCreateRequest{
		Name:          "ERR-MAP",
		ImageData:     "AAAA",
		NaturalWidth:  100,
		NaturalHeight: 100,
		HasFloors:     false,
		FloorCount:    0,
	}
	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader(b))
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
	// created_at DESC を想定して p1 → p2 の順で返す
	parentRows := sqlmock.NewRows(parentCols).
		AddRow("p1", "Campus A", "BASE64A", 4096, 3072, nil, true, 3, now, now, nil).
		AddRow("p2", "Campus B", "BASE64B", 2048, 1536, nil, false, 0, now.Add(-time.Minute), now.Add(-time.Minute), nil)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).WillReturnRows(parentRows)

	// 2) 子件数の集約（IN (?,?)）
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

	// 3) 子の軽量一覧（IN (?,?)、名前昇順）
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

	// --- 実行 ---
	req := httptest.NewRequest(http.MethodGet, "/maps/index", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// --- 検証 ---
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Fatalf("ETag header must be set")
	}

	// レスポンスの最小検証
	var list struct {
		Items []handlers.MapResponse `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("json unmarshal error: %v, body=%s", err, rec.Body.String())
	}
	if len(list.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(list.Items))
	}
	// p1
	if list.Items[0].ID != "p1" || list.Items[0].ChildrenCount != 2 || len(list.Items[0].Children) != 2 {
		t.Fatalf("unexpected p1 aggregate: %+v", list.Items[0])
	}
	// 子は名前昇順（"1F","2F"）
	if list.Items[0].Children[0].Name != "1F" || list.Items[0].Children[1].Name != "2F" {
		t.Fatalf("children not sorted by name asc: %+v", list.Items[0].Children)
	}
	// p2
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

	// 親の最初のSELECTでエラーを返す
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).WillReturnError(assertErr("parent select failed"))

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
		id, "キャンパスマップ2025", "iVBORw0K...", 4096, 3072,
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

	// 3) 子一覧（順不同で返し、ハンドラー側の昇順ソートを検証）
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("map_b", "B2F", false, 0).
		AddRow("map_a", "1F", false, 0)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(childRows)

	// --- 実行 ---
	req := httptest.NewRequest(http.MethodGet, "/maps/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// --- 検証 ---
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Fatalf("ETag header must be set")
	}

	var resp handlers.MapResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error: %v, body=%s", err, rec.Body.String())
	}
	if resp.ID != id || resp.Name != "キャンパスマップ2025" {
		t.Fatalf("unexpected base fields: %+v", resp)
	}
	if resp.ChildrenCount != 2 || len(resp.Children) != 2 {
		t.Fatalf("unexpected children meta: %+v", resp.Children)
	}
	// ソート結果は "1F", "B2F"
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
	// 0行（= sql.ErrNoRows 相当）
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
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
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
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
		id, "Empty Child Map", "AAAA", 1024, 768,
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

	// 子件数=0
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 子一覧=0
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	req := httptest.NewRequest(http.MethodGet, "/maps/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp handlers.MapResponse
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

// --- ここから PATCH /maps/:mapId のテストを追加 ---

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
		WillReturnRows(before)

	// 2) UPDATE（複数フィールド）
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE maps SET name = ?, natural_width = ?, parent_map_id = ?, has_floors = ?, floor_count = ?, modified_at = ? WHERE id = ? AND deleted_at IS NULL
	`)).
		WithArgs("New Campus", 2048, "parent_1", true, 2, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 3) 更新後の再取得（FindMapResponseByID を内部で呼ぶ）
	after := sqlmock.NewRows(mainCols).AddRow(
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
		WillReturnRows(after)

	// 子件数=0
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	// 子一覧=0
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// リクエスト
	body := map[string]any{
		"name":         "New Campus",
		"naturalWidth": 2048,
		"parentMapId":  "parent_1",
		"hasFloors":    true,
		"floorCount":   2,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/maps/"+id, bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// 実行
	e.ServeHTTP(rec, req)

	// 検証
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Fatalf("ETag header must be set")
	}
	var resp handlers.MapResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json error: %v body=%s", err, rec.Body.String())
	}
	if resp.ID != id || resp.Name != "New Campus" || resp.NaturalWidth != 2048 || !resp.HasFloors || resp.FloorCount != 2 {
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

	// 現在は親あり・階あり
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	before := sqlmock.NewRows(mainCols).AddRow(
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
		WithArgs(id).WillReturnRows(before)

	// UPDATE: 親NULL + has_floors=false + floor_count=0
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE maps SET parent_map_id = NULL, has_floors = ?, floor_count = ?, modified_at = ? WHERE id = ? AND deleted_at IS NULL
	`)).
		WithArgs(false, 0, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 更新後再取得
	after := sqlmock.NewRows(mainCols).AddRow(
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
		WithArgs(id).WillReturnRows(after)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"}))

	// PATCH: parentMapId=null(=明示的にnullを送る) & hasFloors=false
	body := []byte(`{"parentMapId": null, "hasFloors": false}`)
	req := httptest.NewRequest(http.MethodPatch, "/maps/"+id, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp handlers.MapResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.ParentMapID != nil || resp.HasFloors || resp.FloorCount != 0 {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapHandler_Update_NotFound(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "missing"

	// 最初の SELECT が 0 行（= sql.ErrNoRows 相当）
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
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

func TestMapHandler_Update_ValidationError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	id := "map_v"
	now := time.Now().UTC()

	// 現在値（has_floors=false, floor_count=0）
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	before := sqlmock.NewRows(mainCols).AddRow(
		id, "X", "IMG", 100, 100, nil, false, 0, now.Add(-time.Hour), now.Add(-time.Hour), nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).WillReturnRows(before)

	// リクエスト: hasFloors=true かつ floorCount=0 → リポジトリでバリデーションエラー（UPDATEは走らない）
	req := httptest.NewRequest(http.MethodPatch, "/maps/"+id, bytes.NewReader([]byte(`{"hasFloors": true, "floorCount": 0}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Update は 400 を返す仕様（handler.Update は sql no rows 以外は 400 に寄せる）
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
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).WillReturnRows(before)

	// UPDATE でエラーを返す
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE maps SET name = ?, modified_at = ? WHERE id = ? AND deleted_at IS NULL
	`)).
		WithArgs("New", sqlmock.AnyArg(), id).
		WillReturnError(assertErr("update failed"))

	req := httptest.NewRequest(http.MethodPatch, "/maps/"+id, bytes.NewReader([]byte(`{"name":"New"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

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

	// ★ setupEchoWithMock に e.DELETE を追加済みであることが前提

	rootID := "map_root"

	// DeleteCascade の内部SQLに対応する期待値
	mock.ExpectBegin()

	// 存在確認
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*)
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(rootID).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	// pins 削除
	mock.ExpectExec(regexp.QuoteMeta(`
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
		WillReturnResult(sqlmock.NewResult(0, 4)) // 4件削除想定

	// maps 削除
	mock.ExpectExec(regexp.QuoteMeta(`
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
		WillReturnResult(sqlmock.NewResult(0, 3)) // 3件削除想定

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
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*)
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
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
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*)
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	mock.ExpectExec(regexp.QuoteMeta(`
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
		WithArgs(id).
		WillReturnError(assertErr("delete pins failed"))

	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodDelete, "/maps/"+id, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// handler.Delete は no rows 以外のエラーはそのまま返す → Echo が 500 にする
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
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*)
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	mock.ExpectExec(regexp.QuoteMeta(`
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
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(`
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

// 固定エラータイプ（比較しやすいように）
type assertErr string

func (e assertErr) Error() string { return string(e) }

package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"nutfesmap/internal/repository"
)

// ---- Request / Response DTO ----

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

// POST /maps（空マップ作成）リクエスト
type MapCreateRequest struct {
	ParentMapID *string `json:"parentMapId"`
}

// レスポンス全体ラッパ
type MapsIndexResponse struct {
	Items []MapResponse `json:"items"`
}

type MapHandler struct {
	Repo *repository.MapRepository
}

func NewMapHandler(r *repository.MapRepository) *MapHandler {
	return &MapHandler{Repo: r}
}

// POST /maps
// 空のマップのみ作成。name / imageData / natural* / floors は受け付けない。
func (h *MapHandler) Create(c echo.Context) error {
	var req MapCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid payload")
	}

	newID := uuid.NewString()
	ctx := c.Request().Context()

	// 空マップを作成（name/image/natural_* は未設定、親のみ任意）
	if err := h.Repo.CreateEmpty(ctx, newID, req.ParentMapID); err != nil {
		// DB起因の失敗はそのまま Echo が 500 にマップ
		return err
	}

	// 作成直後の状態を再取得
	agg, err := h.Repo.FindAggregate(ctx, newID)
	if err != nil {
		return err
	}

	// 正常系: Aggregate からレスポンス構築
	if agg != nil && agg.Base != nil {
		children := make([]MapChildRefDTO, 0, len(agg.Children))
		for _, ch := range agg.Children {
			children = append(children, MapChildRefDTO{
				ID:         ch.ID,
				Name:       ch.Name,
				HasFloors:  ch.HasFloors,
				FloorCount: ch.FloorCount,
			})
		}
		res := MapResponse{
			ID:            agg.Base.ID,
			Name:          agg.Base.Name,
			ImageData:     agg.Base.ImageData, // LONGTEXT(base64) をそのまま返す
			NaturalWidth:  agg.Base.NaturalWidth,
			NaturalHeight: agg.Base.NaturalHeight,
			ParentMapID:   agg.Base.ParentMapID,
			HasFloors:     agg.Base.HasFloors,
			FloorCount:    agg.Base.FloorCount,
			ChildrenCount: agg.ChildrenCount,
			Children:      children,
			CreatedAt:     agg.Base.CreatedAt,
			ModifiedAt:    agg.Base.ModifiedAt,
		}

		// ETag: ID + ModifiedAt
		hash := sha256.New()
		hash.Write([]byte(res.ID))
		hash.Write([]byte(res.ModifiedAt.UTC().Format(time.RFC3339Nano)))
		etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
		c.Response().Header().Set("ETag", etag)

		return c.JSON(http.StatusCreated, res)
	}

	// まれに直後の再読込に失敗した場合のフォールバック
	now := time.Now().UTC()
	fallback := MapResponse{
		ID:            newID,
		Name:          "",
		ImageData:     "",
		NaturalWidth:  0,
		NaturalHeight: 0,
		ParentMapID:   req.ParentMapID,
		HasFloors:     false,
		FloorCount:    0,
		ChildrenCount: 0,
		Children:      []MapChildRefDTO{},
		CreatedAt:     now,
		ModifiedAt:    now,
	}

	// フォールバックでも ETag を必ず付与
	hash := sha256.New()
	hash.Write([]byte(fallback.ID))
	hash.Write([]byte(fallback.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)

	return c.JSON(http.StatusCreated, fallback)
}

// GET /maps/index
func (h *MapHandler) Index(c echo.Context) error {
	ctx := c.Request().Context()

	ags, err := h.Repo.FindIndexAggregates(ctx)
	if err != nil {
		return err
	}

	items := make([]MapResponse, 0, len(ags))
	hash := sha256.New()

	for _, ag := range ags {
		// 念のため子を Name 昇順に
		sort.Slice(ag.Children, func(i, j int) bool {
			return ag.Children[i].Name < ag.Children[j].Name
		})

		children := make([]MapChildRefDTO, 0, len(ag.Children))
		for _, ch := range ag.Children {
			children = append(children, MapChildRefDTO{
				ID:         ch.ID,
				Name:       ch.Name,
				HasFloors:  ch.HasFloors,
				FloorCount: ch.FloorCount,
			})
		}

		base := ag.Base
		items = append(items, MapResponse{
			ID:            base.ID,
			Name:          base.Name,
			ImageData:     base.ImageData, // LONGTEXT(base64)
			NaturalWidth:  base.NaturalWidth,
			NaturalHeight: base.NaturalHeight,
			ParentMapID:   base.ParentMapID,
			HasFloors:     base.HasFloors,
			FloorCount:    base.FloorCount,
			ChildrenCount: ag.ChildrenCount,
			Children:      children,
			CreatedAt:     base.CreatedAt,
			ModifiedAt:    base.ModifiedAt,
		})

		// ETag 材料（ID+ModifiedAt）
		hash.Write([]byte(base.ID))
		hash.Write([]byte(base.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	}

	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)

	return c.JSON(http.StatusOK, MapsIndexResponse{Items: items})
}

// GET /maps/:mapId （地図メタ取得）
func (h *MapHandler) Show(c echo.Context) error {
	mapID := c.Param("mapId")
	if mapID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	ctx := c.Request().Context()
	res, err := h.Repo.FindMapResponseByID(ctx, mapID)
	if err != nil {
		return err
	}
	if res == nil {
		return echo.NewHTTPError(http.StatusNotFound, "map not found")
	}

	// 子は名前昇順で安定化
	sort.Slice(res.Children, func(i, j int) bool {
		return res.Children[i].Name < res.Children[j].Name
	})

	// 単体ETag: ID + ModifiedAt
	hash := sha256.New()
	hash.Write([]byte(res.ID))
	hash.Write([]byte(res.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)

	return c.JSON(http.StatusOK, res)
}

// PATCH /maps/:mapId
func (h *MapHandler) Update(c echo.Context) error {
	mapID := c.Param("mapId")
	if mapID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	// 受け取り: name / imageData / naturalWidth / naturalHeight / parentMapId のみ
	var req repository.MapUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}

	ctx := c.Request().Context()
	updated, err := h.Repo.UpdatePartial(ctx, mapID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "map not found")
		}
		// ドメイン検証エラーなどは 400 に寄せる（UpdatePartial は検証時に error を返す）
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if updated == nil {
		return echo.NewHTTPError(http.StatusNotFound, "map not found")
	}

	// 子は名前昇順で安定化
	sort.Slice(updated.Children, func(i, j int) bool {
		return updated.Children[i].Name < updated.Children[j].Name
	})

	// ETag: ID + ModifiedAt
	hash := sha256.New()
	hash.Write([]byte(updated.ID))
	hash.Write([]byte(updated.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)

	return c.JSON(http.StatusOK, updated)
}

// DELETE /maps/:mapId
func (h *MapHandler) Delete(c echo.Context) error {
	mapID := c.Param("mapId")
	if mapID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	ctx := c.Request().Context()
	_, _, err := h.Repo.DeleteCascade(ctx, mapID)
	if err != nil {
		// 対象なし
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "map not found")
		}
		// それ以外は内部エラーとして扱う（Echo 側で 500 にマップ）
		return err
	}

	// 本体・子マップ・ピンを再帰的に削除済み
	return c.NoContent(http.StatusNoContent)
}

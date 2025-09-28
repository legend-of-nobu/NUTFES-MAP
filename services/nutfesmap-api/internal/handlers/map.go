package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"nutfesmap/internal/model"
	"nutfesmap/internal/repository"
)

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
func (h *MapHandler) Create(c echo.Context) error {
	var req MapCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid payload")
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// 追加の軽い検証（ドメイン制約）
	if strings.TrimSpace(req.ImageData) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "imageData is required")
	}
	if req.HasFloors && req.FloorCount < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "floorCount must be >= 1 when hasFloors=true")
	}

	newID := "map_" + uuid.NewString()
	m := &model.Map{
		ID:            newID,
		Name:          req.Name,
		ImageData:     req.ImageData,
		NaturalWidth:  req.NaturalWidth,
		NaturalHeight: req.NaturalHeight,
		ParentMapID:   req.ParentMapID,
		HasFloors:     req.HasFloors,
		FloorCount:    req.FloorCount,
	}

	ctx := c.Request().Context()
	if err := h.Repo.Insert(ctx, m); err != nil {
		return err
	}

	agg, err := h.Repo.FindAggregate(ctx, newID)
	if err != nil {
		return err
	}

	// 正常系：Aggregate からレスポンスDTOを組み立て
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
			ImageData:     agg.Base.ImageData,
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
		return c.JSON(http.StatusCreated, res)
	}

	// まれに直後の再読込に失敗した場合のフォールバック
	fallback := MapResponse{
		ID:            m.ID,
		Name:          m.Name,
		ImageData:     m.ImageData,
		NaturalWidth:  m.NaturalWidth,
		NaturalHeight: m.NaturalHeight,
		ParentMapID:   m.ParentMapID,
		HasFloors:     m.HasFloors,
		FloorCount:    m.FloorCount,
		ChildrenCount: 0,
		Children:      []MapChildRefDTO{},
	}
	return c.JSON(http.StatusCreated, fallback)
}

// GET /maps/index
func (h *MapHandler) Index(c echo.Context) error {
	ctx := c.Request().Context()

	ags, err := h.Repo.FindIndexAggregates(ctx)
	if err != nil {
		return err
	}

	// 並びの安定化（子は名前昇順、親は作成日降順はリポジトリ側SQLで担保）
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
		})

		// ETag 材料（ID+ModifiedAt）
		hash.Write([]byte(base.ID))
		hash.Write([]byte(base.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	}

	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)

	return c.JSON(http.StatusOK, MapsIndexResponse{Items: items})
}

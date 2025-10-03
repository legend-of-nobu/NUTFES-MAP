package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ====== リクエスト/レスポンス DTO（Swagger準拠） ======

// POST /maps/{mapId}/pins
type PinCreateRequest struct {
	Name             string  `json:"name"`                           // required
	Description      *string `json:"description,omitempty"`          // optional
	DescriptionImage *string `json:"descriptionImageData,omitempty"` // optional (base64)
	Type             *string `json:"type,omitempty"`                 // enum: area_selector/exhibit/service/info (default: exhibit)
	LinkToMapID      *string `json:"linkToMapId,omitempty"`          // optional
	XNorm            float64 `json:"xNorm"`                          // 0..1
	YNorm            float64 `json:"yNorm"`                          // 0..1
	Category         string  `json:"category"`                       // enum: food/stage/exhibition/game/service/other
	Status           *string `json:"status,omitempty"`               // enum: open/paused/closed (default: open)
	WaitMinutes      *int    `json:"waitMinutes,omitempty"`          // default 0
}

// PATCH /pins/{pinId}
type PinUpdateRequest struct {
	Name             OptionalString         `json:"name"`
	Description      OptionalNullableString `json:"description"`
	DescriptionImage OptionalNullableString `json:"descriptionImageData"`
	Type             OptionalString         `json:"type"` // enum
	LinkToMapID      OptionalNullableString `json:"linkToMapId"`
	XNorm            OptionalFloat64        `json:"xNorm"`    // 0..1
	YNorm            OptionalFloat64        `json:"yNorm"`    // 0..1
	Category         OptionalString         `json:"category"` // enum
	Status           OptionalString         `json:"status"`   // enum
	WaitMinutes      OptionalInt            `json:"waitMinutes"`
}

type PinResponse struct {
	ID               string    `json:"id"`
	MapID            string    `json:"mapId"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	DescriptionImage *string   `json:"descriptionImageData,omitempty"`
	Type             string    `json:"type"`
	LinkToMapID      *string   `json:"linkToMapId,omitempty"`
	XNorm            float64   `json:"xNorm"`
	YNorm            float64   `json:"yNorm"`
	Category         string    `json:"category"`
	Status           string    `json:"status"`
	WaitMinutes      int       `json:"waitMinutes"`
	CreatedAt        time.Time `json:"createdAt"`
	ModifiedAt       time.Time `json:"modifiedAt"`
}

// ====== リポジトリ ======

type PinRepository struct{ DB *sql.DB }

func NewPinRepository(db *sql.DB) *PinRepository { return &PinRepository{DB: db} }

// GET /maps/{mapId}/pins
func (r *PinRepository) FindByMapID(ctx context.Context, mapID string) ([]*PinResponse, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT
		  id, map_id, name, description, description_image,
		  type, link_to_map_id, x_norm, y_norm,
		  category, status, wait_minutes, created_at, modified_at
		FROM pins
		WHERE map_id = ? 
		ORDER BY modified_at DESC, id ASC
	`, mapID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*PinResponse
	for rows.Next() {
		p, err := scanPin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GET /pins/{pinId}
func (r *PinRepository) FindByID(ctx context.Context, pinID string) (*PinResponse, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT
		  id, map_id, name, description, description_image,
		  type, link_to_map_id, x_norm, y_norm,
		  category, status, wait_minutes, created_at, modified_at
		FROM pins
		WHERE id = ?
		LIMIT 1
	`, pinID)
	return scanPin(row)
}

// POST /maps/{mapId}/pins
func (r *PinRepository) CreateOnMap(ctx context.Context, newID string, mapID string, req *PinCreateRequest) (*PinResponse, error) {
	// 軽い入力バリデーション（DB制約に沿う）
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("name must not be empty")
	}
	if req.XNorm < 0 || req.XNorm > 1 || req.YNorm < 0 || req.YNorm > 1 {
		return nil, fmt.Errorf("xNorm/yNorm must be in [0,1]")
	}
	if !isValidCategory(req.Category) {
		return nil, fmt.Errorf("invalid category")
	}
	typ := "exhibit"
	if req.Type != nil && *req.Type != "" {
		typ = *req.Type
	}
	if !isValidType(typ) {
		return nil, fmt.Errorf("invalid type")
	}
	status := "open"
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}
	if !isValidStatus(status) {
		return nil, fmt.Errorf("invalid status")
	}
	wait := 0
	if req.WaitMinutes != nil {
		if *req.WaitMinutes < 0 {
			return nil, fmt.Errorf("waitMinutes must be >= 0")
		}
		wait = *req.WaitMinutes
	}

	// map の存在は FK で担保されるが、404 区別したければ事前チェックも可
	now := time.Now().UTC()

	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO pins (
		  id, map_id, name, description, description_image,
		  type, link_to_map_id, x_norm, y_norm,
		  category, status, wait_minutes, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		newID, mapID, req.Name, req.Description, req.DescriptionImage,
		typ, req.LinkToMapID, req.XNorm, req.YNorm,
		req.Category, status, wait, now, now,
	)
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, newID)
}

// PATCH /pins/{pinId}
func (r *PinRepository) UpdatePartial(ctx context.Context, pinID string, req *PinUpdateRequest) (*PinResponse, error) {
	// 現在値
	cur, err := r.FindByID(ctx, pinID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, sql.ErrNoRows
	}

	set := make([]string, 0, 10)
	args := make([]any, 0, 12)

	if req.Name.Set {
		v := strings.TrimSpace(req.Name.Value)
		if v == "" {
			return nil, fmt.Errorf("name must not be empty")
		}
		if v != cur.Name {
			set = append(set, "name = ?")
			args = append(args, v)
		}
	}
	if req.Description.Set {
		// null 許容
		if req.Description.Null {
			set = append(set, "description = NULL")
		} else {
			set = append(set, "description = ?")
			args = append(args, req.Description.Value)
		}
	}
	if req.DescriptionImage.Set {
		if req.DescriptionImage.Null {
			set = append(set, "description_image = NULL")
		} else {
			set = append(set, "description_image = ?")
			args = append(args, req.DescriptionImage.Value)
		}
	}
	if req.Type.Set {
		if !isValidType(req.Type.Value) {
			return nil, fmt.Errorf("invalid type")
		}
		set = append(set, "type = ?")
		args = append(args, req.Type.Value)
	}
	if req.LinkToMapID.Set {
		if req.LinkToMapID.Null {
			set = append(set, "link_to_map_id = NULL")
		} else {
			set = append(set, "link_to_map_id = ?")
			args = append(args, req.LinkToMapID.Value)
		}
	}
	if req.XNorm.Set {
		if req.XNorm.Value < 0 || req.XNorm.Value > 1 {
			return nil, fmt.Errorf("xNorm must be in [0,1]")
		}
		set = append(set, "x_norm = ?")
		args = append(args, req.XNorm.Value)
	}
	if req.YNorm.Set {
		if req.YNorm.Value < 0 || req.YNorm.Value > 1 {
			return nil, fmt.Errorf("yNorm must be in [0,1]")
		}
		set = append(set, "y_norm = ?")
		args = append(args, req.YNorm.Value)
	}
	if req.Category.Set {
		if !isValidCategory(req.Category.Value) {
			return nil, fmt.Errorf("invalid category")
		}
		set = append(set, "category = ?")
		args = append(args, req.Category.Value)
	}
	if req.Status.Set {
		if !isValidStatus(req.Status.Value) {
			return nil, fmt.Errorf("invalid status")
		}
		set = append(set, "status = ?")
		args = append(args, req.Status.Value)
	}
	if req.WaitMinutes.Set {
		if req.WaitMinutes.Value < 0 {
			return nil, fmt.Errorf("waitMinutes must be >= 0")
		}
		set = append(set, "wait_minutes = ?")
		args = append(args, req.WaitMinutes.Value)
	}

	if len(set) == 0 {
		return cur, nil
	}

	set = append(set, "modified_at = ?")
	now := time.Now().UTC()
	args = append(args, now, pinID)

	q := "UPDATE pins SET " + strings.Join(set, ", ") + " WHERE id = ?"
	res, err := r.DB.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if aff, _ := res.RowsAffected(); aff == 0 {
		return nil, sql.ErrNoRows
	}
	return r.FindByID(ctx, pinID)
}

// DELETE /pins/{pinId}
func (r *PinRepository) Delete(ctx context.Context, pinID string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM pins WHERE id = ?`, pinID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ====== 内部：スキャン & バリデーション ======

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPin(s rowScanner) (*PinResponse, error) {
	var (
		id, mapID, name, typ, category, status string
		descNS, descImgNS, linkNS              sql.NullString
		xn, yn                                 float64
		wait                                   int
		createdAt, modifiedAt                  time.Time
	)
	if err := s.Scan(
		&id, &mapID, &name, &descNS, &descImgNS,
		&typ, &linkNS, &xn, &yn,
		&category, &status, &wait, &createdAt, &modifiedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var desc, descImg, link *string
	if descNS.Valid {
		v := descNS.String
		desc = &v
	}
	if descImgNS.Valid {
		v := descImgNS.String
		descImg = &v
	}
	if linkNS.Valid {
		v := linkNS.String
		link = &v
	}
	return &PinResponse{
		ID:               id,
		MapID:            mapID,
		Name:             name,
		Description:      desc,
		DescriptionImage: descImg,
		Type:             typ,
		LinkToMapID:      link,
		XNorm:            xn,
		YNorm:            yn,
		Category:         category,
		Status:           status,
		WaitMinutes:      wait,
		CreatedAt:        createdAt,
		ModifiedAt:       modifiedAt,
	}, nil
}

func isValidType(s string) bool {
	switch s {
	case "area_selector", "exhibit", "service", "info":
		return true
	default:
		return false
	}
}
func isValidCategory(s string) bool {
	switch s {
	case "food", "stage", "exhibition", "game", "service", "other":
		return true
	default:
		return false
	}
}
func isValidStatus(s string) bool {
	switch s {
	case "open", "paused", "closed":
		return true
	default:
		return false
	}
}

// ====== Optional ユーティリティ ======

// OptionalNullableString: null を区別したい場合（NULLに更新）
type OptionalNullableString struct {
	Set   bool
	Null  bool
	Value string
}

func (o *OptionalNullableString) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Null = true
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

type OptionalFloat64 struct {
	Set   bool
	Value float64
}

func (o *OptionalFloat64) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = 0
		return nil
	}
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	o.Value = f
	return nil
}

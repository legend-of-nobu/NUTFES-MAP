"use client";

import React from "react";
import { FaMapPin, FaHamburger } from "react-icons/fa";
import { MdFamilyRestroom } from "react-icons/md";
import { FaBuildingColumns } from "react-icons/fa6";
import { Category } from "@/types/enums";
import { useMapMetrics } from "@/components/map/MapImage";

// ---- BottomSheet が求める型 ----
export interface SpotData {
  title: string;
  category: Category;
  time: string;
  place: string;
  description: string;
  imageUrl: string;
}

// ---- APIのピンレスポンス型 ----
export type ApiPin = {
  id: string;
  mapId: string;
  name: string;
  description: string | null;
  descriptionImageData: string | null;
  type: "exhibit" | "area_selector" | "service" | "info" | string;
  linkToMapId: string | null;
  xNorm: number; // 0..1
  yNorm: number; // 0..1
  place: string | null;
  category: string;
  status: "open" | "paused" | "closed";
  waitMinutes: number;
  createdAt: string;
  modifiedAt: string;
};

const categoryIcons: Record<Category, React.ComponentType> = {
  [Category.Food]: FaHamburger,
  [Category.Child]: MdFamilyRestroom,
  [Category.Plan]: FaBuildingColumns,
};

const toCategoryEnum = (raw: string): Category => {
  const v = raw?.toLowerCase();
  if (v === "food") return Category.Food;
  if (v === "child" || v === "kids" || v === "family") return Category.Child;
  return Category.Plan;
};

const toPosStyle = (xNorm: number, yNorm: number) => ({
  left: `${xNorm * 100}%`,
  top: `${yNorm * 100}%`,
});

const toDataUrlOrEmpty = (b64: string | null): string =>
  b64 ? `data:image/png;base64,${b64}` : "";

// ---- props ----
type PlanPinProps = {
  pin: ApiPin;
  onSelect: (spot: SpotData) => void;
  ghost?: boolean; // ★ 追加：ゴースト表示（非クリック・半透明）
};

const BASE_PIN_SIZE = 144;
const BASE_WAIT_BADGE_SIZE = 36;
const BASE_WAIT_BADGE_FONT = 14;
const BASE_WAIT_BADGE_TOP = 20;
const BASE_WAIT_BADGE_RIGHT = 28;
const BASE_MAP_PIN_ICON = 64;
const BASE_CATEGORY_ICON = 32;
const BASE_LABEL_TOP = 92;
const BASE_LABEL_MIN_WIDTH = 80;
const BASE_LABEL_MAX_WIDTH = 136;
const BASE_LABEL_FONT = 14;
const BASE_REFERENCE_WIDTH = 720; // px

const PlanPin: React.FC<PlanPinProps> = ({ pin, onSelect, ghost = false }) => {
  const metrics = useMapMetrics();
  const rawScale =
    metrics.renderedWidth > 0 ? metrics.renderedWidth / BASE_REFERENCE_WIDTH : 1;
  const pinScale = Math.min(Math.max(rawScale, 0.5), 1.6);
  const scalePx = (value: number) => Math.max(value * pinScale, 1);

  const layout = {
    boxSize: scalePx(BASE_PIN_SIZE),
    waitBadge: {
      size: scalePx(BASE_WAIT_BADGE_SIZE),
      font: scalePx(BASE_WAIT_BADGE_FONT),
      top: scalePx(BASE_WAIT_BADGE_TOP),
      right: scalePx(BASE_WAIT_BADGE_RIGHT),
    },
    mapPinIcon: scalePx(BASE_MAP_PIN_ICON),
    categoryIcon: scalePx(BASE_CATEGORY_ICON),
    label: {
      top: scalePx(BASE_LABEL_TOP),
      minWidth: scalePx(BASE_LABEL_MIN_WIDTH),
      maxWidth: scalePx(BASE_LABEL_MAX_WIDTH),
      font: scalePx(BASE_LABEL_FONT),
    },
  };

  const catEnum = toCategoryEnum(pin.category);
  const CategoryIcon = categoryIcons[catEnum] ?? FaBuildingColumns;
  const placeLabel = pin.place?.trim() || `(${(pin.xNorm * 100).toFixed(1)}%, ${(pin.yNorm * 100).toFixed(1)}%)`;

  const spotData: SpotData = {
    title: pin.name,
    category: catEnum,
    time: pin.waitMinutes > 0 ? `${pin.waitMinutes}分待ち` : "待ちなし",
    place: placeLabel,
    description: pin.description ?? "",
    imageUrl: toDataUrlOrEmpty(pin.descriptionImageData),
  };

  const disabledStyle =
    pin.status === "closed"
      ? "opacity-50 grayscale"
      : pin.status === "paused"
      ? "opacity-80"
      : "";

  const interactiveProps = ghost
    ? {}
    : {
        onClick: (e: React.MouseEvent) => {
          e.stopPropagation();
          onSelect(spotData);
        },
        onKeyDown: (e: React.KeyboardEvent) => {
          if (e.key === "Enter" || e.key === " ") {
            e.stopPropagation();
            onSelect(spotData);
          }
        },
      };

  return (
    <div
      style={{ position: "absolute", ...toPosStyle(pin.xNorm, pin.yNorm) }}
      className={`absolute -translate-x-1/2 -translate-y-full ${
        ghost ? "pointer-events-none opacity-60" : "cursor-pointer"
      } ${disabledStyle}`}
      role={ghost ? undefined : "button"}
      tabIndex={ghost ? -1 : 0}
      aria-label={ghost ? undefined : `${spotData.title}（${spotData.time}）`}
      {...interactiveProps}
    >
      {/* 当たり判定は 60px（見た目は従来のまま） */}
      <div
        className="relative flex items-center justify-center"
        style={{ width: layout.boxSize, height: layout.boxSize }}
      >
        {/* 待ち時間バッジ（右上） */}
        <div
          className="absolute z-30 flex items-center justify-center rounded-full bg-[#DC143C] font-bold text-white"
          style={{
            width: layout.waitBadge.size,
            height: layout.waitBadge.size,
            top: layout.waitBadge.top,
            right: layout.waitBadge.right,
            fontSize: layout.waitBadge.font,
          }}
        >
          {pin.waitMinutes}分
        </div>

        {/* 画鋲（背面） */}
        <FaMapPin
          className="absolute z-10 text-[#9370DB] drop-shadow"
          style={{ fontSize: layout.mapPinIcon }}
        />

        {/* カテゴリアイコン（中央） */}
        <div
          className="absolute left-1/2 top-[46%] z-20 -translate-x-1/2 -translate-y-1/2 text-black"
          style={{ fontSize: layout.categoryIcon }}
        >
          <CategoryIcon />
        </div>

        {/* イベント名ラベル（下） */}
        <div
          className="absolute z-20 rounded-lg border border-black bg-white px-1"
          style={{
            top: layout.label.top,
            minWidth: layout.label.minWidth,
            maxWidth: layout.label.maxWidth,
          }}
        >
          <span
            className="block break-words text-center font-bold leading-tight"
            style={{ fontSize: layout.label.font }}
          >
            {pin.name}
          </span>
        </div>
      </div>
    </div>
  );
};

export default PlanPin;

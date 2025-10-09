"use client";

import React from "react";
import { FaMapPin, FaHamburger } from "react-icons/fa";
import { MdFamilyRestroom } from "react-icons/md";
import { FaBuildingColumns } from "react-icons/fa6";
import { Category } from "@/types/enums";

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

const PlanPin: React.FC<PlanPinProps> = ({ pin, onSelect, ghost = false }) => {
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
      <div className="relative flex h-[60px] w-[60px] items-center justify-center">
        {/* 待ち時間バッジ（右上） */}
        <div className="absolute right-[12px] top-[8px] z-30 flex h-4 w-4 items-center justify-center rounded-full bg-[#DC143C] text-[6px] font-bold text-white">
          {pin.waitMinutes}分
        </div>

        {/* 画鋲（背面） */}
        <FaMapPin className="absolute text-[28px] text-[#9370DB] z-10 drop-shadow" />

        {/* カテゴリアイコン（中央） */}
        <div className="absolute left-1/2 top-[46%] z-20 -translate-x-1/2 -translate-y-1/2 text-[13px] text-black">
          <CategoryIcon />
        </div>

        {/* イベント名ラベル（下） */}
        <div className="absolute top-[38px] z-20 px-1 rounded-lg border border-black bg-white min-w-[34px] max-w-[56px]">
          <span className="block break-words text-center text-[6px] font-bold line-clamp-2">
            {pin.name}
          </span>
        </div>
      </div>
    </div>
  );
};

export default PlanPin;

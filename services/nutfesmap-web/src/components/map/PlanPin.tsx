// ================================
// components/map/PlanPin.tsx
// ================================
"use client";

import React from "react";
import { FaMapPin, FaHamburger } from "react-icons/fa";
import { MdFamilyRestroom } from "react-icons/md";
import { FaBuildingColumns } from "react-icons/fa6";
import { Category } from "@/types/enums";

// ---- BottomSheet が求める型（そのまま流用）----
export interface SpotData {
  title: string;
  category: Category;
  time: string;        // ここでは「待ち時間」を見出し化
  location: string;    // 例: "(31.3%, 74.2%)"
  description: string;
  imageUrl: string;    // base64 が来たら data URL に変換、なければ空文字など
}

// ---- APIのピンレスポンス型（提示仕様に合わせる）----
export type ApiPin = {
  id: string; // "pin_01J0AABCDE" など
  mapId: string;
  name: string;                    // 表示名
  description: string | null;      // 説明
  descriptionImageData: string | null; // base64（先頭に data: は付与されない想定）
  type: "exhibit" | "area_selector" | "service" | "info" | string;
  linkToMapId: string | null;
  xNorm: number; // 0..1
  yNorm: number; // 0..1
  category: string; // "food" など（enum 文字列）
  status: "open" | "paused" | "closed";
  waitMinutes: number;
  createdAt: string;
  modifiedAt: string;
};

// ---- カテゴリアイコン（Category enumに合わせて適宜拡張）----
const categoryIcons: Record<Category, React.ComponentType> = {
  [Category.Food]: FaHamburger,
  [Category.Child]: MdFamilyRestroom,
  [Category.Plan]: FaBuildingColumns,
};

// ---- APIの category(string) → Category(enum) の簡易マッパ ----
const toCategoryEnum = (raw: string): Category => {
  const v = raw?.toLowerCase();
  if (v === "food") return Category.Food;
  if (v === "child" || v === "kids" || v === "family") return Category.Child;
  return Category.Plan; // 既知以外は Plan にフォールバック
};

// ---- xNorm / yNorm を % 文字列に変換 ----
const toPosStyle = (xNorm: number, yNorm: number) => ({
  left: `${xNorm * 100}%`,
  top: `${yNorm * 100}%`,
});

// ---- base64文字列を <img> 用の data URL に ----
const toDataUrlOrEmpty = (b64: string | null): string =>
  b64 ? `data:image/png;base64,${b64}` : "";

// ---- PlanPin props ----
type PlanPinProps = {
  pin: ApiPin;
  onSelect: (spot: SpotData) => void; // クリックで BottomSheet に渡すデータを上位へ
};

const PlanPin: React.FC<PlanPinProps> = ({ pin, onSelect }) => {
  const catEnum = toCategoryEnum(pin.category);
  const CategoryIcon = categoryIcons[catEnum] ?? FaBuildingColumns;

  // BottomSheet へ渡すデータをここで作る
  const spotData: SpotData = {
    title: pin.name,
    category: catEnum,
    time: pin.waitMinutes > 0 ? `${pin.waitMinutes}分待ち` : "待ちなし",
    location: `(${(pin.xNorm * 100).toFixed(1)}%, ${(pin.yNorm * 100).toFixed(1)}%)`,
    description: pin.description ?? "",
    imageUrl: toDataUrlOrEmpty(pin.descriptionImageData),
  };

  // 状態ごとの見た目（必要に応じて拡張）
  const disabledStyle =
    pin.status === "closed"
      ? "opacity-50 grayscale"
      : pin.status === "paused"
      ? "opacity-80"
      : "";

  return (
    <div
      style={{ position: "absolute", ...toPosStyle(pin.xNorm, pin.yNorm) }}
      onClick={(e) => {
        e.stopPropagation();
        onSelect(spotData);
      }}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.stopPropagation();
          onSelect(spotData);
        }
      }}
      className={`cursor-pointer absolute -translate-x-1/2 -translate-y-full ${disabledStyle}`}
      role="button"
      tabIndex={0}
      aria-label={`${spotData.title}（${spotData.time}）`}
    >
      {/* ← 当たり判定（全体ボックス）を 60x60 に縮小 */}
      <div className="relative flex h-[60px] w-[60px] items-center justify-center">
        {/* 待ち時間バッジ（右上） - 位置・サイズを縮小調整 */}
        <div className="absolute right-[14px] top-[8px] z-30 flex h-4 w-4 items-center justify-center rounded-full bg-[#DC143C] text-[6px] font-bold text-white">
          {pin.waitMinutes}分
        </div>

        {/* 画鋲（背面） - サイズ縮小 */}
        <FaMapPin className="absolute text-[28px] text-[#9370DB] z-10 drop-shadow" />

        {/* カテゴリアイコン（中央） - ほんの少し小さく */}
        <div className="absolute left-1/2 top-[43%] z-20 -translate-x-1/2 -translate-y-1/2 text-[12px] text-black">
          <CategoryIcon />
        </div>

        {/* イベント名ラベル（下） - 位置・幅を60px用に */}
        <div className="absolute top-[32px] z-20 px-1 rounded-lg border border-black bg-white min-w-[36px] max-w-[56px]">
          <span className="block break-words text-center text-[4px] font-bold line-clamp-2">
            {pin.name}
          </span>
        </div>
      </div>
    </div>
  );
};

export default PlanPin;

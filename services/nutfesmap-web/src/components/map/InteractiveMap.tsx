"use client";

import React from "react";
import { useRouter } from "next/navigation";
import PlanPin from "@/components/map/PlanPin"; // PlanPinコンポーネントをインポート

// --- ここにマップに関するすべての情報を集約 ---

// ピン１つ分のデータの型を定義
type Pin = {
  id: string;
  position: {
    top: string;
    left: string;
  };
  name: string;
};

// ピンのデータ
const pins: Pin[] = [
  {
    id: "building-1",
    position: { top: "45%", left: "60%" }, // マップ上のどこに置くか適当に指定
    name: "原子力・安全\nシステム等",
  },
  {
    id: "building-2",
    position: { top: "85%", left: "60%" },
    name: "講義棟",
  },
];

// 背景のマップ画像
const mapImageUrl = "/エリア.png";

// --- コンポーネントの定義 ---

export const InteractiveMap = () => {
  const router = useRouter();

  // ピンがクリックされたときの処理
  const handlePinClick = (buildingId: string) => {
    router.push(`/building/${buildingId}`);
  };

  return (
    <div className="relative w-full h-screen">
      {/* 背景マップ */}
      <img
        src={mapImageUrl}
        alt="屋外マップ"
        className="w-full h-full object-cover"
      />

      {/* ピンをマップ上に配置 */}
      {pins.map((pin) => (
        <div
          key={pin.id}
          className="absolute"
          style={{ top: pin.position.top, left: pin.position.left }}
        >
          <PlanPin
            name={pin.name} // pin.name(文字列)を直接渡す
            onClick={() => handlePinClick(pin.id)}
          />
        </div>
      ))}
    </div>
  );
};

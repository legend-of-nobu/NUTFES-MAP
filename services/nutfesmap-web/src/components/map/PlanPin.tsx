"use client";

import React, { useState } from "react";
import { FaHamburger } from "react-icons/fa";
import { FaMapPin, FaBuildingColumns } from "react-icons/fa6";
import { MdFamilyRestroom } from "react-icons/md";
import BottomSheet from "@/components/plans/BottomSheet";
import { Category } from "@/types/enums";

// --- SpotData型（BottomSheetに渡す用） ---
type SpotData = {
  title: string;
  category: Category;
  time: string;
  location: string;
  description: string;
  imageUrl: string;
};

// --- Pin型 ---
type Pin = {
  id: number;
  top: string;
  left: string;
  category: Category;
  eventName: string;
  waitTime: number;
};

// --- ピンのデータ ---
const pinData: Pin[] = [
  {
    id: 1,
    top: "50px",
    left: "150px",
    category: Category.Food,
    eventName: "スイートポテトコンテスト",
    waitTime: 15,
  },
  {
    id: 2,
    top: "250px",
    left: "80px",
    category: Category.Child,
    eventName: "こども広場",
    waitTime: 30,
  },
  {
    id: 3,
    top: "200px",
    left: "280px",
    category: Category.Plan,
    eventName: "特別企画展",
    waitTime: 5,
  },
];

// --- カテゴリアイコン ---
const categoryIcons: Record<Category, React.ComponentType> = {
  [Category.Food]: FaHamburger,
  [Category.Child]: MdFamilyRestroom,
  [Category.Plan]: FaBuildingColumns,
};

// --- 個々のピン ---
const PlanPin: React.FC<{ pin: Pin; onClick: (pin: Pin) => void }> = ({
  pin,
  onClick,
}) => {
  const CategoryIcon = categoryIcons[pin.category];

  return (
    <div
      style={{ position: "absolute", top: pin.top, left: pin.left }}
      onClick={() => onClick(pin)}
      className="cursor-pointer"
    >
      <div className="relative flex h-[120px] w-[120px] items-center justify-center">
        <div className="absolute right-[40px] top-[30px] z-30 flex h-5 w-5 items-center justify-center rounded-full bg-[#DC143C] text-[6px] font-bold text-white">
          {pin.waitTime}分
        </div>
        <FaMapPin className="absolute text-[40px] text-[#9370DB] z-10" />
        <div className="absolute left-1/2 top-[43%] z-20 -translate-x-1/2 -translate-y-1/2 text-[15px] text-black">
          <CategoryIcon />
        </div>
        <div className="absolute top-[60px] z-20 px-1 rounded-lg border border-black bg-white min-w-[40px]">
          <span className="block break-words text-center text-[4px] font-bold">
            {pin.eventName}
          </span>
        </div>
      </div>
    </div>
  );
};

// --- 全ピン表示 + BottomSheet制御 ---
const PlanPinsOnMap = () => {
  const [selectedPin, setSelectedPin] = useState<Pin | null>(null);
  const [isOpen, setIsOpen] = useState(false);

  const handlePinClick = (pin: Pin) => {
    setSelectedPin(pin);
    setIsOpen(true);
  };

  const handleClose = () => setIsOpen(false);

  // ピン情報 → BottomSheet 用データに変換
  const spotData: SpotData | null = selectedPin
    ? {
        title: selectedPin.eventName,
        category: selectedPin.category,
        time: `${selectedPin.waitTime}分`,
        location: "未設定",
        description: "このイベントの詳細情報です。",
        imageUrl: "/example.png",
      }
    : null;

  return (
    <div className="relative w-full h-screen bg-gray-100">
      {pinData.map((pin) => (
        <PlanPin key={pin.id} pin={pin} onClick={handlePinClick} />
      ))}

      {spotData && (
        <BottomSheet isOpen={isOpen} onClose={handleClose} spotData={spotData} />
      )}
    </div>
  );
};

export default PlanPinsOnMap;

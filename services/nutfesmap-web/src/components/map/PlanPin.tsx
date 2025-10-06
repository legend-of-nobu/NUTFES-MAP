"use client";

import React from "react";
import { FaMapPin } from "react-icons/fa6";
import { MdFamilyRestroom } from "react-icons/md";
import { FaHamburger } from "react-icons/fa";
import { FaBuildingColumns } from "react-icons/fa6";
import { Category } from "@/types/enums";

// --- Pin型 ---
export type Pin = {
  id: number;
  top: string;
  left: string;
  category: Category;
  eventName: string;
  waitTime: number;
};

// --- カテゴリアイコン ---
const categoryIcons: Record<Category, React.ComponentType> = {
  [Category.Food]: FaHamburger,
  [Category.Child]: MdFamilyRestroom,
  [Category.Plan]: FaBuildingColumns,
};

type PlanPinProps = {
  pin: Pin;
  onClick: (pin: Pin) => void;
};

const PlanPin: React.FC<PlanPinProps> = ({ pin, onClick }) => {
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

export default PlanPin;

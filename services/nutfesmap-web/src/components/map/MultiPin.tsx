"use client";

import React, { useState } from "react";
import PlanPin, { Pin } from "./PlanPin";
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

const MultiPin = () => {
  const [selectedPin, setSelectedPin] = useState<Pin | null>(null);
  const [isOpen, setIsOpen] = useState(false);

  const handlePinClick = (pin: Pin) => {
    setSelectedPin(pin);
    setIsOpen(true);
  };

  const handleClose = () => setIsOpen(false);

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

export default MultiPin;

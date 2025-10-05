"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { IoCloseCircle, IoFastFood } from "react-icons/io5";
import { VscLocation } from "react-icons/vsc";
import { MdAccessTime } from "react-icons/md";

// ====== enum定義 ======
enum Category {
  Food = "飲食",
  Child = "子供向け",
  Plan = "企画",
}

// ====== データ型定義 ======
interface SpotData {
  title: string;
  category: Category;
  time: string;
  location: string;
  description: string;
  imageUrl: string;
}

export default function BottomSheet() {
  const [isOpen, setIsOpen] = useState(false);

  const spotData: SpotData = {
    title: "お化け屋敷",
    category: Category.Food,
    time: "15分",
    location: "講義棟1F 101",
    description:
      "怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い怖い",
    imageUrl: "お化け屋敷.png",
  };

  return (
    <div className="relative h-screen w-full font-sans-custom">
      {/* ====== 仮のマップエリア ====== */}
      {/* <div className="h-full w-full bg-gray-200 flex items-center justify-center">
        <button
          onClick={() => setIsOpen(true)}
          className="px-4 py-2 bg-blue-500 text-white rounded-lg shadow-md transition-transform hover:scale-105"
        >
          お化け屋敷のピン
        </button>
      </div> */}

      <AnimatePresence>
        {isOpen && (
          <>
            {/* オーバーレイ */}
            <motion.div
              className="fixed inset-0  z-40"
              onClick={() => setIsOpen(false)}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
            />

            {/* ボトムシート */}
            <motion.div
              className="fixed bottom-0 left-0 right-0 z-50 flex flex-col overflow-hidden bg-main shadow-2xl rounded-t-2xl "
              style={{ maxHeight: "50vh" }}
              initial={{ y: "100%" }}
              animate={{ y: 0 }}
              exit={{ y: "100%" }}
              transition={{ type: "spring", stiffness: 300, damping: 30 }}
            >
              {/* --- ヘッダー --- */}
              <div className="p-3">
                <div className="flex items-center justify-between ">
                  <h2 className="text-xl font-bold  pt-2 pl-2">
                    {spotData.title}
                  </h2>
                  <div className="flex flex-col items-center ">
                    <IoFastFood size={36} />
                    <span className="bg-planning-details rounded-md text-[12px] px-2">
                      {spotData.category}
                    </span>
                  </div>

                  <button
                    onClick={() => setIsOpen(false)}
                    className="p-2 rounded-full "
                  >
                    <IoCloseCircle size={28} />
                  </button>
                </div>
              </div>

              {/* --- コンテンツエリア --- */}
              <div className="flex-1 p-4 overflow-y-auto">
                {/* 時間・場所 */}
                <div className="flex w-full gap-4 mb-4">
                  <div className="w-1/2">
                    <span className="text-black">所要時間</span>
                    <div className="flex items-center justify-center w-full gap-2 py-3 mt-1 pr-2 rounded-lg bg-planning-details ">
                      <MdAccessTime size={28} />
                      <span>{spotData.time}</span>
                    </div>
                  </div>

                  <div className="w-1/2">
                    <span >場所</span>
                    <div className="flex items-center justify-center w-full gap-2 py-3 mt-1 rounded-lg bg-planning-details">
                      <VscLocation size={28} />
                      <span>{spotData.location}</span>
                    </div>
                  </div>
                </div>

                {/* 説明文 */}
                <div className="mb-4">
                  <span>企画詳細</span>
                  <div className="p-4 mt-1 rounded-lg bg-planning-details">
                    <p className="leading-relaxed ">{spotData.description}</p>
                  </div>
                </div>

                {/* 画像 */}
                <div className="p-2 rounded-lg bg-planning-details">
                  <img
                    src={spotData.imageUrl}
                    alt={spotData.title}
                    className="object-cover w-full "
                  />
                </div>
              </div>
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </div>
  );
}

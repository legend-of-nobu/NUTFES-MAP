"use client";

import { motion, AnimatePresence } from "framer-motion";
import { IoCloseCircle, IoFastFood } from "react-icons/io5";
import { VscLocation } from "react-icons/vsc";
import { MdAccessTime,MdFamilyRestroom } from "react-icons/md";
import { FaBuildingColumns } from "react-icons/fa6";
import type { SpotData } from "./PlanPin";
import { Category } from "@/types/enums";
import { NextJsHotReloaderInterface } from "next/dist/server/dev/hot-reloader-types";

/** ご指定のデザイン・Props に合わせた BottomSheet */
type Props = {
  isOpen: boolean;
  onClose: () => void;
  spotData: SpotData | null;
};

export default function PlanSpotBottomSheet({ isOpen, onClose, spotData }: Props) {
  if (!spotData) return null;

 // ✅ カテゴリごとのアイコンマップ
  const categoryIconMap: Record<string, React.ReactNode> = {
     [Category.Food]: <IoFastFood size={36} />,       // 飲食
   [Category.Plan] : <FaBuildingColumns size={36} />,   // 企画
    [Category.Child]: <MdFamilyRestroom size={36} />,      // 子供向け
  };

  // ✅ 一致しない場合はデフォルトを設定
  const categoryIcon = categoryIconMap[spotData.category] || <IoFastFood size={36} />;

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* オーバーレイ */}
          <motion.div
            className="fixed inset-0 z-40 bg-black/30"
            onClick={onClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          />

          {/* ボトムシート */}
          <motion.div
            className="fixed bottom-0 left-0 right-0 z-50 flex flex-col overflow-hidden bg-main shadow-2xl rounded-t-2xl"
            style={{ maxHeight: "50vh" }}
            initial={{ y: "100%" }}
            animate={{ y: 0 }}
            exit={{ y: "100%" }}
            transition={{ type: "spring", stiffness: 300, damping: 30 }}
            role="dialog"
            aria-hidden={!isOpen}
          >
            {/* --- ヘッダー --- */}
            <div className="p-3">
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-bold pt-2 pl-2">{spotData.title}</h2>

                  {/* ✅ アイコン＋カテゴリ名（切り替え対応） */}
                <div className="flex flex-col items-center">
                  {categoryIcon}
                  <span className="bg-planning-details rounded-md text-[12px] px-2">
                    {spotData.category}
                  </span>
                </div>

                <button onClick={onClose} className="p-2 rounded-full" aria-label="閉じる">
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
                  <div className="flex items-center justify-center w-full gap-2 py-3 mt-1 rounded-lg bg-planning-details">
                    <MdAccessTime size={28} />
                    <span>{spotData.time}</span>
                  </div>
                </div>

                <div className="w-1/2">
                  <span>場所</span>
              <div className="flex items-center justify-center w-full gap-2 py-3 mt-1 rounded-lg bg-planning-details">
                <VscLocation size={28} />
                <span>{spotData.place || "場所未設定"}</span>
              </div>
            </div>
              </div>

              {/* 説明文 */}
              <div className="mb-4">
                <span>企画詳細</span>
                <div className="p-4 mt-1 rounded-lg bg-planning-details">
                  <p className="leading-relaxed">{spotData.description}</p>
                </div>
              </div>

              {/* 画像（あれば） */}
              {spotData.imageUrl ? (
                <div className="p-2 rounded-lg bg-planning-details">
                  <img
                    src={spotData.imageUrl}
                    alt={spotData.title}
                    className="object-cover w-full"
                  />
                </div>
              ) : null}
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}

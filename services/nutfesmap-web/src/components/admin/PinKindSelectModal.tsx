"use client";

import React, { useEffect } from "react";
import { FaTimes } from "react-icons/fa";

type PinKind = "area" | "plan";

type Props = {
  visible: boolean;
  selected: PinKind | null;
  onSelect: (value: PinKind) => void;
  onConfirm: () => void;
  onCancel: () => void;
};

export default function PinKindSelectModal({
  visible,
  selected,
  onSelect,
  onConfirm,
  onCancel,
}: Props) {
  // モーダルを開いた時に未選択なら "area" をデフォルトにする
  useEffect(() => {
    if (visible && !selected) onSelect("area");
  }, [visible, selected, onSelect]);

  if (!visible) return null;

  const handleBackdropClick: React.MouseEventHandler<HTMLDivElement> = (e) => {
    // ダイアログ外クリックで閉じる（中クリックは伝播抑止）
    if (e.target === e.currentTarget) onCancel();
  };

  const handleConfirm = () => {
    onConfirm(); // 親側で選択結果を処理
    onCancel();  // モーダルを閉じる
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      role="dialog"
      aria-modal="true"
      aria-labelledby="pin-kind-title"
      onClick={handleBackdropClick}
    >
      <div className="relative w-[280px] rounded-lg bg-[#FAF9F1] px-6 py-5 shadow-lg">
        {/* 閉じるボタン */}
        <div className="mb-2 flex justify-end">
          <button
            type="button"
            onClick={onCancel}
            className="flex h-6 w-6 items-center justify-center rounded-full bg-black text-xs text-white hover:opacity-85"
            aria-label="閉じる"
          >
            <FaTimes />
          </button>
        </div>

        {/* タイトル */}
        <h2 id="pin-kind-title" className="mb-3 text-center text-base font-semibold text-black">
          ピンの種類を選択
        </h2>

        {/* ラジオボタンエリア */}
        <div className="mb-4 flex flex-col gap-2 rounded border-2 border-[#a08702] p-3">
          <label className="flex cursor-pointer items-center gap-2">
            <input
              type="radio"
              name="pinKind"
              value="area"
              checked={selected === "area"}
              onChange={() => onSelect("area")}
              className="text-[#a08702] focus:ring-[#a08702]"
            />
            <span className="text-sm text-black">エリア</span>
          </label>

          <label className="flex cursor-pointer items-center gap-2">
            <input
              type="radio"
              name="pinKind"
              value="plan"
              checked={selected === "plan"}
              onChange={() => onSelect("plan")}
              className="text-[#a08702] focus:ring-[#a08702]"
            />
            <span className="text-sm text-black">企画</span>
          </label>
        </div>

        {/* 決定ボタン */}
        <button
          type="button"
          onClick={handleConfirm}
          className="w-full rounded-full bg-[#a08702] py-2 font-medium text-white transition hover:opacity-90"
        >
          決定
        </button>
      </div>
    </div>
  );
}

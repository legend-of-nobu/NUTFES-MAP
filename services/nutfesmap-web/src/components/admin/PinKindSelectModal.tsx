import { FaTimes } from "react-icons/fa";
import { useEffect } from "react";

type Props = {
  visible: boolean;
  selected: "area" | "plan" | null;
  onSelect: (value: "area" | "plan") => void;
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
    if (visible && !selected) {
      onSelect("area");
    }
  }, [visible, selected, onSelect]);

  if (!visible) return null;

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black/40 z-50">
      <div className="relative bg-[#FAF9F1] rounded-lg shadow-lg px-6 py-5 w-[280px]">
        {/* 閉じるボタン */}
        <div className="flex justify-end mb-2">
          <button
            onClick={onCancel}
            className="w-6 h-6 flex items-center justify-center rounded-full bg-black text-white text-xs"
          >
            <FaTimes />
          </button>
        </div>

        {/* ラジオボタンエリア */}
        <div className="border-2 border-[#a08702] rounded p-3 flex flex-col gap-2 mb-4">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name="pinKind"
              value="area"
              checked={selected === "area"}
              onChange={() => onSelect("area")}
              className="text-[#a08702] focus:ring-[#a08702]"
            />
            <span className="text-black text-sm">エリア</span>
          </label>

          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name="pinKind"
              value="plan"
              checked={selected === "plan"}
              onChange={() => onSelect("plan")}
              className="text-[#a08702] focus:ring-[#a08702]"
            />
            <span className="text-black text-sm">企画</span>
          </label>
        </div>

        {/* 決定ボタン */}
        <button
          onClick={() => {
            onConfirm();   // 親側で選択結果を処理
            onCancel();    // モーダルを閉じる
          }}
          className="w-full bg-[#a08702] text-white py-2 rounded-full font-medium hover:opacity-90 transition"
        >
          決定
        </button>
      </div>
    </div>
  );
}

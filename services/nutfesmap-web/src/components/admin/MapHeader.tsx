import { FaArrowLeft } from "react-icons/fa6";

export default function MapHeader({
  mapName,
  onBack,
}: {
  mapName: string;
  onBack: () => void;
}) {
  return (
    <div className="flex items-center justify-between bg-[#e9e1c8] px-4 py-2 border-b">
      {/* 左矢印ボタン */}
      <button
        onClick={onBack}
        className="flex items-center justify-center w-8 h-8 rounded-md bg-black text-white hover:opacity-80 transition"
      >
        <FaArrowLeft size={14} />
      </button>

      {/* 中央にマップ名 */}
      <span className="font-medium text-sm text-gray-800">{mapName}</span>

      {/* 右側余白 */}
      <div className="w-8" />
    </div>
  );
}

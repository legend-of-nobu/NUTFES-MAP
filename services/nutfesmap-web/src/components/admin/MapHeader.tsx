import { FaArrowLeft } from "react-icons/fa6";

export default function MapHeader({
  mapName,
  parentMapId,
  onBack,
}: {
  mapName: string;
  parentMapId?: string | null;
  onBack?: () => void;
}) {
  const canGoBack = !!parentMapId;

  return (
    <div className="flex items-center justify-between bg-[#e9e1c8] px-4 py-2 border-b">
      {/* 左矢印ボタン：親があるときだけ表示。無いときはプレースホルダで左右バランス維持 */}
      {canGoBack ? (
        <button
          onClick={onBack}
          className="flex items-center justify-center w-8 h-8 rounded-md bg-black text-white hover:opacity-80 transition"
          aria-label="ひとつ上のマップへ戻る"
        >
          <FaArrowLeft size={14} />
        </button>
      ) : (
        <div className="w-8 h-8" />
      )}

      {/* 中央にマップ名 */}
      <span className="font-medium text-sm text-gray-800">{mapName}</span>

      {/* 右側余白 */}
      <div className="w-8 h-8" />
    </div>
  );
}

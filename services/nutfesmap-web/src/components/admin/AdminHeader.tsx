import ModeChanger from "../ui/ModeChanger";
import { Button } from "../ui/Button";
import { FaCirclePlus, FaCircleXmark } from "react-icons/fa6";

type Props = {
  currentMapName: string;
  mode: "edit" | "user";
  onModeChange: (m: "edit" | "user") => void;
  onMapEdit: () => void;
  onStairAdd: () => void;
  onStairDelete: () => void;
};

export default function AdminHeader({
  currentMapName,
  mode,
  onModeChange,
  onMapEdit,
  onStairAdd,
  onStairDelete,
}: Props) {
  return (
    <header className="flex flex-col items-start gap-3 px-4 py-3 bg-transparent">
      {/* マップ名（黒背景＋白文字） */}
      <div className="bg-black text-white text-sm font-semibold px-4 py-1 rounded">
        {currentMapName}
      </div>

      {/* ボタン群（半透明カード） */}
      <div className="bg-white rounded-lg shadow px-4 py-3">
        <div className="grid grid-cols-2 gap-3">
          {/* 左上：Edit/View */}
          <ModeChanger mode={mode} onModeChange={onModeChange} />

          {/* 右上：階を追加 */}
          <Button
            label="階を追加"
            color="green"
            onClick={onStairAdd}
            fullWidth={false}
            icon={<FaCirclePlus />}
          />

          {/* 左下：マップ編集 */}
          <Button
            label="マップ編集"
            color="yellow"
            onClick={onMapEdit}
            fullWidth={false}
          />

          {/* 右下：階を削除 */}
          <Button
            label="階を削除"
            color="red"
            onClick={onStairDelete}
            fullWidth={false}
            icon={<FaCircleXmark />}
          />
        </div>
      </div>
    </header>
  );
}

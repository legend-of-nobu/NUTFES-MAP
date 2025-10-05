import Pin from '../map/Pin';
import StairSelector from '../map/StairSelector';
import AddPinButton from '../map/AddPinButton';

export default function Map({ pins, onPinClick, onAddPin, header, mode }: any) {
  return (
    <div className="relative flex-1 bg-[#e2d7b5] overflow-hidden">
      {/* 背景マップ上にピン配置 */}
      {pins.map((p: any) => (
        <Pin key={p.id} pin={p} onClick={() => onPinClick(p)} />
      ))}

      {/* 左下：階層セレクタ */}
      <div className="absolute bottom-4 left-4">
        <StairSelector
          floors={["3F", "2F", "1F"]}
          selectedFloor={"2F"}
          onSelect={(floor) => console.log("Selected floor:", floor)}
        />
      </div>

      {/* 右下：ピン追加ボタン（editモードのときのみ表示） */}
      {mode === "edit" && (
        <div className="absolute bottom-6 right-6">
          <AddPinButton onClick={onAddPin} />
        </div>
      )}

      {/* 上部：AdminHeader */}
      <div className="absolute top-0 left-0 right-0 p-4">
        {header}
      </div>
    </div>
  );
}

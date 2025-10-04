import Pin from '../map/Pin';
import StairSelector from '../map/StairSelector';
import AddPinButton from '../map/AddPinButton';

export default function Map({ pins, onPinClick, onAddPin }: any) {
  return (
    <div className="relative flex-1 bg-[#e2d7b5]">
      {/* ピン配置 */}
      {pins.map((p: any) => (
        <Pin key={p.id} pin={p} onClick={() => onPinClick(p)} />
      ))}

      {/* 左下に固定配置 */}
      <div className="absolute bottom-8 left-16">
        <StairSelector
          floors={["3F", "2F", "1F"]}
          selectedFloor={"2F"}
          onSelect={(floor) => {
            console.log("Selected floor:", floor);
          }}
        />
      </div>

      {/* 右下に追加ボタン */}
      <div className="absolute bottom-6 right-6">
        <AddPinButton onClick={onAddPin} />
      </div>
    </div>
  );
}

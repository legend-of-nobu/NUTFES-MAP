// src/components/admin/Map.tsx
import Pin from '@/components/map/Pin';
import StairSelector from '@/components/map/StairSelector';
import AddPinButton from '@/components/map/AddPinButton';

type PinType = {
  id: string | number;
  xNorm: number;
  yNorm: number;
  // 必要ならメタ情報をここに追加（UIは変えない）
};

type MapProps = {
  pins?: PinType[];
  onPinClick?: (p: PinType) => void;
  onAddPin?: () => void;
  header?: React.ReactNode;
  mode?: 'edit' | 'user';
  // 階層セレクタの入出力を明示的に
  floors?: string[];
  selectedFloor?: string;
  onSelectFloor?: (floor: string) => void;
};

export default function Map({
  pins = [],
  onPinClick = () => {},
  onAddPin = () => {},
  header = null,
  mode = 'user',
  floors = ['3F', '2F', '1F'],
  selectedFloor = '2F',
  onSelectFloor = () => {},
}: MapProps) {
  return (
    <div className="relative flex-1 bg-[#e2d7b5] overflow-hidden">
      {/* 背景マップ上にピン配置（見た目は同一） */}
      {pins.map((p) => (
        <Pin key={p.id} pin={p} onClick={() => onPinClick(p)} />
      ))}

      {/* 左下：階層セレクタ（見た目は同一） */}
      <div className="absolute bottom-4 left-4">
        <StairSelector
          floors={floors}
          selectedFloor={selectedFloor}
          onSelect={onSelectFloor}
        />
      </div>

      {/* 右下：ピン追加ボタン（edit時のみ） */}
      {mode === 'edit' && (
        <div className="absolute bottom-6 right-6">
          <AddPinButton onClick={onAddPin} />
        </div>
      )}

      {/* 上部：AdminHeader */}
      <div className="absolute top-0 left-0 right-0 p-4">{header}</div>
    </div>
  );
}

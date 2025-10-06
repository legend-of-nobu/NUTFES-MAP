type Props = {
  floors: string[];                // ["3F", "2F", "1F"] みたいに渡す
  selectedFloor: string;           // 現在選択中のフロア
  onSelect: (floor: string) => void;
};

export default function StairSelector({ floors, selectedFloor, onSelect }: Props) {
  return (
    <div className="flex flex-col items-center gap-1">
      {floors.map((floor, idx) => {
        const isSelected = selectedFloor === floor;
        return (
          <button
            key={floor}
            onClick={() => onSelect(floor)}
            className={`w-12 h-8 rounded-md text-white text-sm font-bold transition
              ${isSelected ? "bg-[#9a9f4f]" : "bg-[#492b04] hover:opacity-80"}
            `}
          >
            {floor}
          </button>
        );
      })}
    </div>
  );
}

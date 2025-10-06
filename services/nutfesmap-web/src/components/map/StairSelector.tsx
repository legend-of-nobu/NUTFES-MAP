// src/components/map/StairSelector.tsx
type Props = {
  floors?: string[];
  selectedFloor?: string;
  onSelect: (floor: string) => void;
};

export default function StairSelector({
  floors = [],
  selectedFloor = '',
  onSelect,
}: Props) {
  return (
    <div className="flex flex-col items-center gap-1">
      {floors.map((floor) => {
        const isSelected = selectedFloor === floor;
        return (
          <button
            key={floor}
            onClick={() => onSelect(floor)}
            className={`w-12 h-8 rounded-md text-white text-sm font-bold transition
              ${isSelected ? 'bg-[#a0a852]' : 'bg-[#4a2d0a] hover:opacity-80'}
            `}
          >
            {floor}
          </button>
        );
      })}
    </div>
  );
}

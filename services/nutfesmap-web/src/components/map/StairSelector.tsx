type FloorOption = { id: string; label: string };

type Props = {
  floors?: FloorOption[];
  selectedFloorId?: string | null;
  onSelect: (floorId: string) => void;
};

export default function StairSelector({
  floors = [],
  selectedFloorId = null,
  onSelect,
}: Props) {
  if (!floors.length) return null;

  return (
    <div className="flex flex-col items-center gap-1">
      {floors.map((floor) => {
        const isSelected = selectedFloorId === floor.id;
        return (
          <button
            key={floor.id}
            onClick={() => onSelect(floor.id)}
            className={`w-12 h-8 rounded-md text-white text-sm font-bold transition ${
              isSelected ? "bg-[#a0a852]" : "bg-[#4a2d0a] hover:opacity-80"
            }`}
          >
            {floor.label}
          </button>
        );
      })}
    </div>
  );
}

import { FaUser, FaPen } from "react-icons/fa6";

export default function ModeChanger({
  mode,
  onModeChange,
}: {
  mode: "edit" | "user";
  onModeChange: (m: "edit" | "user") => void;
}) {
  return (
    <div className="w-[120px] h-[34px] flex rounded-[12px] overflow-hidden border-[2px] border-[#A08702] text-sm font-medium">
      {/* Edit */}
      <button
        onClick={() => onModeChange("edit")}
        className={`flex-1 flex items-center justify-center gap-1 transition ${
          mode === "edit"
            ? "bg-[#A08702] text-white"
            : "bg-white text-[#A08702]"
        }`}
      >
        <FaPen className="text-xs" />
        Edit
      </button>

      {/* View */}
      <button
        onClick={() => onModeChange("user")}
        className={`flex-1 flex items-center justify-center gap-1 transition ${
          mode === "user"
            ? "bg-[#A08702] text-white"
            : "bg-white text-[#A08702]"
        }`}
      >
        <FaUser className="text-xs" />
        View
      </button>
    </div>
  );
}

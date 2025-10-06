import { FaUser, FaPen } from "react-icons/fa6";

export default function ModeChanger({
  mode,
  onModeChange,
}: {
  mode: "edit" | "user";
  onModeChange: (m: "edit" | "user") => void;
}) {
  return (
    <div className="w-[120px] h-[34px] flex rounded-full overflow-hidden border border-yellow-700 text-sm font-medium">
      {/* Edit */}
      <button
        onClick={() => onModeChange("edit")}
        className={`flex-1 flex items-center justify-center gap-1 transition ${
          mode === "edit"
            ? "bg-yellow-700 text-white"
            : "bg-white text-yellow-700"
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
            ? "bg-yellow-700 text-white"
            : "bg-white text-yellow-700"
        }`}
      >
        <FaUser className="text-xs" />
        View
      </button>
    </div>
  );
}

import type { ReactNode } from "react";

type Props = {
  label: string;
  onClick: () => void;
  color?: "yellow" | "green" | "red";
  icon?: ReactNode;
  fullWidth?: boolean;
  disabled?: boolean;
};

export function Button({
  label,
  onClick,
  color = "yellow",
  icon,
  fullWidth = false,
  disabled = false,
}: Props) {
  const base =
    "py-1.5 rounded-full border text-sm font-medium transition flex items-center justify-center gap-1";
  const colors = {
    yellow:
      "border-yellow-700 text-yellow-700 hover:bg-yellow-50 bg-white",
    green:
      "border-green-500 text-green-500 hover:bg-green-50 bg-white",
    red: "border-red-500 text-red-500 hover:bg-red-50 bg-white",
  };
  const widthClass = fullWidth ? "w-full" : "w-[120px]";

  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={`${base} ${widthClass} ${colors[color]} ${
        disabled ? "opacity-50 cursor-not-allowed hover:bg-transparent" : ""
      }`}
    >
      {icon && <span className="text-sm">{icon}</span>}
      {label}
    </button>
  );
}

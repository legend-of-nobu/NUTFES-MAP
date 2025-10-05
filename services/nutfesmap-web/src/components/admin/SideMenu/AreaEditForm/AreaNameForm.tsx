import React from "react";
import "../Style.css";
import "../FormStyle.css";

export const AreaNameForm: React.FC<{
  value: string;
  onChange: (value: string) => void;
}> = ({ value, onChange }) => (
  <div className="mb-4">
    <label className="Label">エリア名</label>
    <input
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder="エリア名を入力"
      className="Form"
    />
  </div>
);

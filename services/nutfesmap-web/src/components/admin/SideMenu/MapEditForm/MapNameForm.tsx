import React from "react";
import "../Style.css";

export const MapNameForm: React.FC<{
  value: string;
  onChange: (value: string) => void;
}> = ({ value, onChange }) => (
  <div className="mapNameFormContainer">
    <label className="Label">マップ名</label>
    <input
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder="マップ名を入力"
      className="Form"
    />
  </div>
);
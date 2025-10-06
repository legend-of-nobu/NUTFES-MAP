"use client";
import React from "react";
import  "../FormStyle.css";
import  "../Style.css";

type PlanClosedFormProps = {
  value: boolean;
  onChange: (val: boolean) => void;
};

export default function PlanClosedForm({ value, onChange }: PlanClosedFormProps) {
  return (
    <div className="planClosedForm">
        <label className="Label">営業状況</label>
    <div className="checkboxContainer">
        
      <input
        type="checkbox"
        checked={value}
        onChange={(e) => onChange(e.target.checked)}
        className="checkbox"
      />
      <label className="checkboxName">営業終了</label>
    </div>
    </div>
  );
}

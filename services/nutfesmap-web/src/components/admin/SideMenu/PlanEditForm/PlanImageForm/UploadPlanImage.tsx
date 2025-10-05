"use client";
import React, { ChangeEvent } from "react";
import { Upload } from "lucide-react";
import  "../../Image.css"; 
import "../../Style.css"

type UploadPlanImageProps = {
  onUpload: (imageUrl: string) => void;
};

export default function UploadPlanImage({ onUpload }: UploadPlanImageProps) {
  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onloadend = () => {
      if (reader.result) onUpload(reader.result as string);
    };
    reader.readAsDataURL(file);
  };

  return (
    <div>
     <label htmlFor="mapUpload" className="mapUploadLabel">
        <Upload size={20} />
        <span>マップをアップロード</span>
      </label>
    <input
    id="mapUpload"
      type="file"
      accept="image/*"
      onChange={handleFileChange}
      className="fileInput"
    /></div>
  );
}

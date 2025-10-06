"use client";
import React, { useState } from "react";
import UploadPlanImage from "./UploadPlanImage";
import PreviewPlanImage from "./PreviewPlanImage";
import "../../Image.css"
import "../../FormStyle.css";
import "../../Style.css"

interface PlanImageFormProps {
  value: string | null;
  onChange: (image: string | null) => void;
}

export default function PlanImageForm({ value, onChange }: PlanImageFormProps) {
  const [image, setImage] = useState<string | null>(value);

  const handleUpload = (uploadedImage: string | null) => {
    setImage(uploadedImage);
    onChange(uploadedImage);
  };

  return (
    <div className="mapContainer">
      <label className="Label">マップをアップロード</label>
      <div className="fileinputContainer">
         <UploadPlanImage onUpload={handleUpload} />
         {image && (
            <div>
          <PreviewPlanImage image={image} />
            </div>
        
             )}
        </div>
    </div>
  );
}


import React from "react";
import  "../Image.css";
import { Upload } from "lucide-react"; 

export const MapImageUploadForm: React.FC<{
  value: File | null;
  onChange: (file: File | null) => void;
}> = ({ value, onChange }) => (
  <div>
     <label htmlFor="mapUpload" className="mapUploadLabel">
        <Upload size={20} />
        <span>マップをアップロード</span>
      </label>
    <input
    id="mapUpload"
      type="file"
      accept="image/*"
      onChange={(e) => onChange(e.target.files?.[0] || null)}
      className="fileInput"
    />
  </div>
);

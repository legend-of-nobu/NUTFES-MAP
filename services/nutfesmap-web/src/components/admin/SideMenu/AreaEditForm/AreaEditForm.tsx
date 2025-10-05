import React, { useState } from "react";
import { AreaNameForm } from "./AreaNameForm";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import  "../Style.css";

export const AreaEditForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const [areaName, setAreaName] = useState("");

  return (
    <div className="container">
      <CloseButton onClick={onClose} />
      <h2 className="title"> ピンを編集</h2>
      <AreaNameForm value={areaName} onChange={setAreaName} />
      <div className="buttonRow">
      <SaveButton />
      <DeleteButton />
      </div>
    </div>
  );
};



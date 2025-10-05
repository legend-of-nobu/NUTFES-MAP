"use client";
import React, { useState } from "react";
import  PlanNameForm  from "./PlanNameForm";
import  PlanCategoryForm  from "./PlanCategoryForm";
import  WaitTimeForm  from "./WaitTimeForm";
import  PlanImageForm  from "./PlanImageForm/PlanImageForm";
import  PlanExplainForm  from "./PlanExplainForm";
import PlanClosedForm from "./PlanClosedForm";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import styles from "./PlanEditForm.module.css";

export const PlanEditForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const [planName, setPlanName] = useState("");
  const [category, setCategory] = useState("");
  const [waitTime, setWaitTime] = useState("");
  const [image, setImage] = useState<string | null>(null);
  const [description, setDescription] = useState("");
  const [closed, setClosed] = useState(false); 

  return (
    <div className={styles.container}>
      <CloseButton onClick={onClose} />
      <h2 className={styles.title}> ピンを編集</h2>

      <PlanNameForm value={planName} onChange={setPlanName} />
      <PlanCategoryForm value={category} onChange={setCategory} />
      <WaitTimeForm value={waitTime} onChange={setWaitTime} />
      <PlanClosedForm value={closed} onChange={setClosed} />
      <PlanImageForm value={image} onChange={setImage} />
      <PlanExplainForm value={description} onChange={setDescription} />
      <div className={styles.buttonRow}>
      <SaveButton />
      <DeleteButton />
      </div>
    </div>
  );
};

import type { FormEvent, ReactNode } from "react";

export type FormContainerProps = {
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  children: ReactNode;
  className?: string;
};

export function FormContainer({ onSubmit, children, className = "" }: FormContainerProps) {
  return (
    <div
      className={`mx-auto mt-6 max-w-6xl rounded-xl border border-slate-200/80 bg-white p-8 shadow-sm ${className}`.trim()}
    >
      <form onSubmit={onSubmit}>{children}</form>
    </div>
  );
}

export default FormContainer;

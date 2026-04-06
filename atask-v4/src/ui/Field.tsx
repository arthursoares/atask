import { forwardRef, type InputHTMLAttributes } from "react";

export interface FieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  hint?: string;
  error?: string;
}

const Field = forwardRef<HTMLInputElement, FieldProps>(function Field(
  { label, hint, error, className, id, ...props },
  ref,
) {
  const inputId = id ?? props.name;

  return (
    <label className="ui-field-block" htmlFor={inputId}>
      {label && <span className="ui-field-label">{label}</span>}
      <input
        ref={ref}
        id={inputId}
        className={["ui-field", className ?? ""].filter(Boolean).join(" ")}
        {...props}
      />
      {(error || hint) && (
        <span className={`ui-field-meta${error ? " is-error" : ""}`}>
          {error ?? hint}
        </span>
      )}
    </label>
  );
});

export default Field;

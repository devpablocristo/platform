import type { ChangeEventHandler } from "react";

export type SearchInputProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  ariaLabel?: string;
  autoComplete?: string;
  rootClassName?: string;
  inputClassName?: string;
  clearLabel?: string;
};

/**
 * Input canónico de búsqueda para consola.
 * La app host aporta el CSS del tema sobre las clases `page-search*`.
 */
export function SearchInput({
  value,
  onChange,
  placeholder,
  ariaLabel,
  autoComplete = "off",
  rootClassName,
  inputClassName,
  clearLabel = "Limpiar búsqueda",
}: SearchInputProps) {
  const handleChange: ChangeEventHandler<HTMLInputElement> = (event) => {
    onChange(event.target.value);
  };

  return (
    <div className={["page-search", rootClassName].filter(Boolean).join(" ").trim()}>
      <input
        type="search"
        className={["page-search__input", inputClassName].filter(Boolean).join(" ").trim()}
        placeholder={placeholder}
        autoComplete={autoComplete}
        value={value}
        onChange={handleChange}
        aria-label={ariaLabel ?? placeholder}
      />
      {value.length > 0 ? (
        <button
          className="page-search__clear"
          onClick={() => onChange("")}
          aria-label={clearLabel}
          type="button"
        >
          ×
        </button>
      ) : null}
    </div>
  );
}

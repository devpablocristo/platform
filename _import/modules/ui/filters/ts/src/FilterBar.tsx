import React, { useCallback, useEffect, useRef, useState, type ReactNode } from "react";

export interface FilterOption {
  id: number | string;
  name: string;
}

export interface FilterItem {
  type: "select" | "search";
  ref?: string;
  name: string;
  label: string;
  total?: number;
  placeholder?: string;
  options?: FilterOption[];
  value?: string | number | null;
  disabled?: boolean;
  onChange: (value: string) => void;
  setData: (value: unknown) => void;
}

export interface ActionButton {
  label: string;
  variant?:
    | "primary"
    | "success"
    | "secondary"
    | "outlineGreen"
    | "danger"
    | "warning"
    | "light"
    | "dark"
    | "outlineGray"
    | "outlinePonti";
  icon?: ReactNode;
  disabled?: boolean;
  onClick?: () => void;
  accept?: string;
  onFileChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;
  href?: string;
  isPrimary?: boolean;
}

export type FilterBarProps = {
  filters: FilterItem[];
  actions?: ActionButton[];
  children?: ReactNode;
  className?: string;
  inputSize?: "sm" | "md";
};

function sizeClass(size: "sm" | "md") {
  return size === "sm" ? "px-3.5 py-2 text-sm" : "px-3.5 py-2.5 text-sm";
}

function buttonClass(variant?: ActionButton["variant"], isPrimary?: boolean) {
  const resolved = variant ?? (isPrimary ? "success" : "outlineGreen");
  const base = "inline-flex items-center justify-center gap-2 rounded-lg font-medium transition-all duration-200 active:scale-[0.97]";

  switch (resolved) {
    case "secondary":
      return `${base} bg-slate-100 text-slate-700 shadow-sm hover:bg-slate-200`;
    case "danger":
      return `${base} bg-red-600 text-white shadow-sm hover:bg-red-700`;
    case "warning":
      return `${base} bg-amber-500 text-white shadow-sm hover:bg-amber-600`;
    case "light":
      return `${base} border border-slate-200 bg-white text-slate-700 shadow-sm hover:bg-slate-50`;
    case "dark":
      return `${base} bg-slate-800 text-white shadow-sm hover:bg-slate-900`;
    case "outlineGray":
      return `${base} border border-slate-200 bg-white text-slate-700 hover:bg-slate-50`;
    case "outlineGreen":
    case "outlinePonti":
      return `${base} border border-custom-btn bg-transparent text-custom-btn hover:bg-primary-50`;
    case "success":
      return `${base} bg-custom-btn text-white shadow-sm hover:bg-custom-btn/85`;
    default:
      return `${base} bg-primary-700 text-white shadow-sm hover:bg-primary-800`;
  }
}

function ActionButtonView({
  action,
  size,
}: {
  action: ActionButton;
  size: "sm" | "md";
}) {
  const classes = `${buttonClass(action.variant, action.isPrimary)} ${sizeClass(size)} whitespace-nowrap ${
    action.disabled ? "cursor-not-allowed opacity-50 active:scale-100" : ""
  }`;

  if (action.onFileChange) {
    return (
      <label className={`${classes} relative overflow-hidden ${action.disabled ? "pointer-events-none" : ""}`.trim()}>
        <input
          type="file"
          className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
          accept={action.accept}
          disabled={action.disabled}
          onClick={(event) => {
            event.currentTarget.value = "";
          }}
          onChange={(event) => {
            action.onFileChange?.(event);
            requestAnimationFrame(() => {
              event.target.value = "";
            });
          }}
        />
        {action.icon ? <span>{action.icon}</span> : null}
        <span>{action.label}</span>
      </label>
    );
  }

  if (action.href) {
    return (
      <a href={action.href} className={classes} aria-disabled={action.disabled ? "true" : undefined}>
        {action.icon ? <span>{action.icon}</span> : null}
        <span>{action.label}</span>
      </a>
    );
  }

  return (
    <button type="button" className={classes} disabled={action.disabled} onClick={action.disabled ? undefined : action.onClick}>
      {action.icon ? <span>{action.icon}</span> : null}
      <span>{action.label}</span>
    </button>
  );
}

function SearchInput({
  label,
  name,
  value,
  disabled,
  onClick,
  onChange,
  onFocus,
  onKeyDown,
  placeholder,
  size,
}: {
  label: string;
  name: string;
  value: string;
  disabled?: boolean;
  onClick?: React.MouseEventHandler<HTMLInputElement>;
  onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
  onKeyDown?: (event: React.KeyboardEvent<HTMLInputElement>) => void;
  placeholder?: string;
  size: "sm" | "md";
}) {
  return (
    <div className="w-full">
      {label ? <label className="mb-1.5 block text-xs font-medium text-slate-600">{label}</label> : null}
      <div className="relative">
        <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">⌕</span>
        <input
          type="text"
          autoComplete="off"
          name={name}
          value={value}
          disabled={disabled}
          onClick={onClick}
          onChange={onChange}
          onKeyDown={onKeyDown}
          onFocus={onFocus}
          placeholder={placeholder}
          className={`input-base block w-full pl-9 pr-8 ${sizeClass(size)} ${
            disabled ? "cursor-not-allowed border-slate-200 bg-slate-50 text-slate-400" : ""
          }`}
        />
        <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400">⌄</span>
      </div>
    </div>
  );
}

function SelectInput({
  label,
  name,
  placeholder,
  options,
  value,
  onChange,
  size,
  disabled,
}: {
  label: string;
  name: string;
  placeholder?: string;
  options: FilterOption[];
  value: string;
  onChange: (event: React.ChangeEvent<HTMLSelectElement>) => void;
  size: "sm" | "md";
  disabled?: boolean;
}) {
  return (
    <div className="w-full">
      {label ? <label className="mb-1.5 block text-xs font-medium text-slate-600">{label}</label> : null}
      <div className="relative">
        <select
          name={name}
          value={value}
          disabled={disabled}
          onChange={onChange}
          className={`input-base block w-full appearance-none px-3.5 ${sizeClass(size)} ${
            disabled ? "cursor-not-allowed border-slate-200 bg-slate-50 text-slate-400" : ""
          }`}
        >
          <option value="" disabled>
            {placeholder || "Seleccionar..."}
          </option>
          {options.map((option) => (
            <option key={String(option.id)} value={String(option.id)}>
              {option.name}
            </option>
          ))}
        </select>
        <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-slate-400">⌄</span>
      </div>
    </div>
  );
}

function ResponsiveActionContainer({ actions, size }: { actions: ActionButton[]; size: "sm" | "md" }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="fixed bottom-6 right-6 z-50">
      {open ? (
        <div className="mb-3 flex flex-col items-end gap-2">
          {actions.map((action) => (
            <ActionButtonView key={`floating-action-${action.label}`} action={action} size={size} />
          ))}
        </div>
      ) : null}

      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        className="flex h-12 w-12 items-center justify-center rounded-full bg-custom-btn text-2xl text-white shadow-lg transition-all duration-200 hover:bg-custom-btn/85 active:scale-95"
        aria-label="Acciones"
      >
        +
      </button>
    </div>
  );
}

export function FilterBar({
  filters,
  actions = [],
  children,
  className = "",
  inputSize = "sm",
}: FilterBarProps) {
  const [suggestionsVisible, setSuggestionsVisible] = useState<Record<string, boolean>>({});
  const [highlightedIndex, setHighlightedIndex] = useState<Record<string, number>>({});

  const refs = useRef<Record<string, HTMLDivElement | null>>({});

  const hideSuggestions = useCallback((name: string) => {
    setSuggestionsVisible((prev) => ({ ...prev, [name]: false }));
  }, []);

  const showSuggestions = useCallback((name: string) => {
    setSuggestionsVisible((prev) => ({ ...prev, [name]: true }));
  }, []);

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      Object.entries(refs.current).forEach(([name, element]) => {
        if (suggestionsVisible[name] && element && !element.contains(event.target as Node)) {
          hideSuggestions(name);
        }
      });
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        Object.keys(suggestionsVisible).forEach((name) => hideSuggestions(name));
      }
    };

    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleEscape);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [hideSuggestions, suggestionsVisible]);

  const handleSuggestionClick = useCallback(
    (
      name: string,
      option: FilterOption,
      onChange: (value: string) => void,
      setData: (value: unknown) => void,
    ) => {
      setData(option);
      onChange(option.name);
      hideSuggestions(name);
    },
    [hideSuggestions],
  );

  const createHandleKeyDown = useCallback(
    (
      name: string,
      options: FilterOption[],
      onChange: (value: string) => void,
      setData: (value: unknown) => void,
    ) =>
      (event: React.KeyboardEvent<HTMLInputElement>) => {
        const currentIndex = highlightedIndex[name] || 0;
        let nextIndex = currentIndex;

        if (event.key === "ArrowDown") {
          event.preventDefault();
          nextIndex = Math.min(options.length - 1, currentIndex + 1);
        } else if (event.key === "ArrowUp") {
          event.preventDefault();
          nextIndex = Math.max(0, currentIndex - 1);
        } else if (event.key === "Enter") {
          event.preventDefault();
          if (options.length > 0) {
            handleSuggestionClick(name, options[currentIndex], onChange, setData);
          }
        } else if (event.key === "Escape") {
          event.preventDefault();
          hideSuggestions(name);
        }

        setHighlightedIndex((prev) => ({ ...prev, [name]: nextIndex }));
      },
    [handleSuggestionClick, hideSuggestions, highlightedIndex],
  );

  return (
    <div className={`w-full ${className}`.trim()}>
      <div className="flex flex-col items-start justify-between gap-3 px-1 py-2 sm:flex-row sm:items-end sm:gap-4">
        <div className="flex w-full flex-col gap-4 sm:flex-1 sm:flex-row">
          {filters.map((filter) => (
            <div key={`filter-${filter.name}`} className="w-full flex-1 sm:w-auto sm:max-w-60">
              {filter.type === "select" ? (
                <SelectInput
                  label={filter.label}
                  name={filter.name}
                  placeholder={filter.placeholder || `Seleccione ${filter.label.toLowerCase()}`}
                  value={filter.value !== undefined && filter.value !== null ? String(filter.value) : ""}
                  onChange={(event) => {
                    const selectedId = event.target.value;
                    const selectedItem = filter.options?.find((option) => String(option.id) === selectedId);
                    filter.setData(selectedItem);
                    filter.onChange(selectedItem?.name ?? "");
                  }}
                  options={filter.options || []}
                  disabled={filter.disabled}
                  size={inputSize}
                />
              ) : (
                <div
                  ref={(element) => {
                    refs.current[filter.name] = element;
                  }}
                  className="relative"
                >
                  <SearchInput
                    label={filter.label}
                    name={filter.name}
                    placeholder={filter.placeholder || `Buscar ${filter.label.toLowerCase()}`}
                    value={String(filter.value ?? "")}
                    size={inputSize}
                    onClick={() => showSuggestions(filter.name)}
                    onChange={(event) => filter.onChange(event.target.value)}
                    onFocus={() => showSuggestions(filter.name)}
                    onKeyDown={createHandleKeyDown(
                      filter.name,
                      filter.options || [],
                      filter.onChange,
                      filter.setData,
                    )}
                    disabled={filter.disabled}
                  />

                  {suggestionsVisible[filter.name] && filter.options ? (
                    <div className="flex items-center justify-between">
                      <ul className="absolute top-full z-10 max-h-[200px] w-full overflow-y-auto rounded-xl border border-slate-200 bg-white shadow-lg">
                        <li
                          className="cursor-pointer px-3.5 py-2.5 text-sm font-medium text-slate-600 transition-colors duration-150 hover:bg-slate-50"
                          onClick={() => {
                            filter.setData({ id: 0, name: "Todos los registros" });
                            filter.onChange("Todos los registros");
                            hideSuggestions(filter.name);
                          }}
                        >
                          Todos los registros
                        </li>

                        {filter.options.map((option, index) => (
                          <li
                            key={String(option.id)}
                            onClick={() =>
                              handleSuggestionClick(filter.name, option, filter.onChange, filter.setData)
                            }
                            className={`cursor-pointer px-3.5 py-2.5 text-sm transition-colors duration-150 ${
                              highlightedIndex[filter.name] === index
                                ? "bg-primary-50 font-medium text-primary-700"
                                : "hover:bg-slate-50"
                            }`}
                          >
                            {option.name}
                          </li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                </div>
              )}
            </div>
          ))}

          {children ? <div className="mx-2">{children}</div> : null}
        </div>

        {actions.length > 0 ? (
          <>
            <div className="hidden items-center justify-end gap-2 sm:flex">
              {actions.map((action) => (
                <ActionButtonView key={`action-${action.label}`} action={action} size={inputSize} />
              ))}
            </div>
            <div className="sm:hidden">
              <ResponsiveActionContainer actions={actions} size={inputSize} />
            </div>
          </>
        ) : null}
      </div>
    </div>
  );
}

export default FilterBar;

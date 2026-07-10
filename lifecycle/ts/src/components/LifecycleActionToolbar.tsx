export type LifecycleView = "active" | "archived" | "trash";
export type LifecycleBulkAction = "archive" | "trash" | "restore" | "purge";

export type LifecycleActionToolbarLabels = {
  newButton?: string;
  editButton?: string;
  clearButton?: string;
  archiveButton?: string;
  trashButton?: string;
  restoreButton?: string;
  deleteButton?: string;
  selectedSuffix?: string;
};

export type LifecycleActionToolbarClassNames = {
  root?: string;
  buttons?: string;
  group?: string;
  lifecycleGroup?: string;
  buttonBase?: string;
  primaryButton?: string;
  secondaryButton?: string;
  dangerButton?: string;
  newButton?: string;
  selectedCount?: string;
  inlineNote?: string;
};

export type LifecycleActionToolbarProps = {
  selectedCount: number;
  view: LifecycleView;
  createOpen: boolean;
  editOpen?: boolean;
  busy: boolean;
  blockedMessage?: string;
  onCreate: () => void;
  onEdit?: () => void;
  onClear: () => void;
  onBulkAction: (action: LifecycleBulkAction) => void;
  labels?: LifecycleActionToolbarLabels;
  classNames?: LifecycleActionToolbarClassNames;
};

const defaultLabels: Required<LifecycleActionToolbarLabels> = {
  newButton: "New",
  editButton: "Edit",
  clearButton: "Clear",
  archiveButton: "Archive",
  trashButton: "Trash",
  restoreButton: "Restore",
  deleteButton: "Delete",
  selectedSuffix: "selected",
};

const defaultClassNames: Required<LifecycleActionToolbarClassNames> = {
  root: "",
  buttons: "",
  group: "",
  lifecycleGroup: "",
  buttonBase: "",
  primaryButton: "",
  secondaryButton: "",
  dangerButton: "",
  newButton: "",
  selectedCount: "",
  inlineNote: "",
};

export function LifecycleActionToolbar(props: LifecycleActionToolbarProps) {
  const labels = { ...defaultLabels, ...props.labels };
  const classNames = { ...defaultClassNames, ...props.classNames };
  const actionsDisabled = props.busy || props.selectedCount === 0;
  const editDisabled = props.busy || props.selectedCount !== 1;

  const buttonClass = (variant: "primary" | "secondary" | "danger", extra = "") =>
    [
      classNames.buttonBase,
      variant === "primary" ? classNames.primaryButton : "",
      variant === "secondary" ? classNames.secondaryButton : "",
      variant === "danger" ? classNames.dangerButton : "",
      extra,
    ]
      .filter(Boolean)
      .join(" ");

  return (
    <div className={classNames.root}>
      <div className={classNames.buttons}>
        <div className={classNames.group}>
          <button
            type="button"
            className={buttonClass(props.createOpen ? "primary" : "secondary", classNames.newButton)}
            onClick={props.onCreate}
          >
            {labels.newButton}
          </button>
        </div>

        {props.view === "active" ? (
          <>
            <div className={classNames.group}>
              {props.onEdit ? (
                <button
                  type="button"
                  className={buttonClass(props.editOpen ? "primary" : "secondary")}
                  disabled={editDisabled}
                  onClick={props.onEdit}
                >
                  {labels.editButton}
                </button>
              ) : null}
              <button
                type="button"
                className={buttonClass("secondary")}
                disabled={actionsDisabled}
                onClick={props.onClear}
              >
                {labels.clearButton}
              </button>
            </div>
            <div className={[classNames.group, classNames.lifecycleGroup].filter(Boolean).join(" ")}>
              <button
                type="button"
                className={buttonClass("secondary")}
                disabled={actionsDisabled}
                onClick={() => props.onBulkAction("archive")}
              >
                {labels.archiveButton}
              </button>
              <button
                type="button"
                className={buttonClass("danger")}
                disabled={actionsDisabled}
                onClick={() => props.onBulkAction("trash")}
              >
                {labels.trashButton}
              </button>
            </div>
          </>
        ) : null}

        {props.view === "archived" ? (
          <>
            <div className={classNames.group}>
              <button
                type="button"
                className={buttonClass("secondary")}
                disabled={actionsDisabled}
                onClick={props.onClear}
              >
                {labels.clearButton}
              </button>
            </div>
            <div className={[classNames.group, classNames.lifecycleGroup].filter(Boolean).join(" ")}>
              <button
                type="button"
                className={buttonClass("primary")}
                disabled={actionsDisabled}
                onClick={() => props.onBulkAction("restore")}
              >
                {labels.restoreButton}
              </button>
            </div>
          </>
        ) : null}

        {props.view === "trash" ? (
          <>
            <div className={classNames.group}>
              <button
                type="button"
                className={buttonClass("secondary")}
                disabled={actionsDisabled}
                onClick={props.onClear}
              >
                {labels.clearButton}
              </button>
            </div>
            <div className={[classNames.group, classNames.lifecycleGroup].filter(Boolean).join(" ")}>
              <button
                type="button"
                className={buttonClass("primary")}
                disabled={actionsDisabled}
                onClick={() => props.onBulkAction("restore")}
              >
                {labels.restoreButton}
              </button>
              <button
                type="button"
                className={buttonClass("danger")}
                disabled={actionsDisabled}
                onClick={() => props.onBulkAction("purge")}
              >
                {labels.deleteButton}
              </button>
            </div>
          </>
        ) : null}
      </div>
      <span className={classNames.selectedCount}>
        {props.selectedCount} {labels.selectedSuffix}
      </span>
      {props.blockedMessage ? <span className={classNames.inlineNote}>{props.blockedMessage}</span> : null}
    </div>
  );
}

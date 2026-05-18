import type { ReactNode } from "react";

export type CopilotRelatedInsightItem = {
  id: string;
  title: string;
  href?: string;
};

export type CopilotResponsePanelProps = {
  answer: string;
  data: unknown;
  sources: unknown;
  warnings: unknown;
  relatedInsightsCount: number;
  relatedInsights: CopilotRelatedInsightItem[];
  relatedInsightsTitle?: string;
  relatedInsightsAction?: ReactNode;
  emptyRelatedInsightsMessage?: string;
  renderRelatedInsight?: (item: CopilotRelatedInsightItem) => ReactNode;
};

function JsonSection({ title, value }: { title: string; value: unknown }) {
  return (
    <div className="border rounded-md p-4">
      <h3 className="font-semibold">{title}</h3>
      <pre className="text-xs text-slate-700 whitespace-pre-wrap">{JSON.stringify(value, null, 2)}</pre>
    </div>
  );
}

export function CopilotResponsePanel({
  answer,
  data,
  sources,
  warnings,
  relatedInsightsCount,
  relatedInsights,
  relatedInsightsTitle = "Insights Relacionados",
  relatedInsightsAction,
  emptyRelatedInsightsMessage = "No hay insights activos.",
  renderRelatedInsight,
}: CopilotResponsePanelProps) {
  return (
    <div className="grid grid-cols-1 gap-4">
      <div className="border rounded-md p-4">
        <h3 className="font-semibold">Respuesta</h3>
        <p className="text-sm text-slate-700">{answer}</p>
      </div>
      <JsonSection title="Datos" value={data} />
      <JsonSection title="Fuentes" value={sources} />
      <JsonSection title="Advertencias" value={warnings} />
      <div className="border rounded-md p-4">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold">{relatedInsightsTitle}</h3>
          {relatedInsightsAction}
        </div>
        <div className="mt-2 text-sm text-slate-700">Relacionados: {relatedInsightsCount}</div>
        {relatedInsights.length === 0 ? (
          <div className="mt-2 text-sm text-slate-500">{emptyRelatedInsightsMessage}</div>
        ) : (
          <div className="mt-3 grid grid-cols-1 gap-2">
            {relatedInsights.map((item) =>
              renderRelatedInsight ? (
                renderRelatedInsight(item)
              ) : (
                <a
                  key={item.id}
                  className="rounded-md border px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
                  href={item.href || "#"}
                >
                  {item.title}
                </a>
              ),
            )}
          </div>
        )}
      </div>
    </div>
  );
}

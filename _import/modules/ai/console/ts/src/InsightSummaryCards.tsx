export type InsightSummaryCard = {
  label: string;
  value: string | number;
};

export type InsightSummaryCardsProps = {
  cards: InsightSummaryCard[];
};

export function InsightSummaryCards({ cards }: InsightSummaryCardsProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      {cards.map((card) => (
        <div key={card.label} className="border rounded-md p-4">
          <div className="text-sm text-slate-500">{card.label}</div>
          <div className="text-2xl font-semibold">{card.value}</div>
        </div>
      ))}
    </div>
  );
}

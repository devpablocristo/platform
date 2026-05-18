import React from 'react';

export type InsightCandidateItem = {
  id: string;
  title: string;
  body: string;
  status: string;
  severity: string;
  event_type: string;
  occurrence_count: number;
  last_seen_at: string;
};

export type InsightCandidatesCardProps = {
  items: InsightCandidateItem[];
  formatDateTime: (iso: string) => string;
  title?: string;
  emptyText?: string;
  activeLabel?: (count: number) => string;
  maxRows?: number;
};

export function InsightCandidatesCard({
  items,
  formatDateTime,
  title = 'Insight candidates',
  emptyText = 'Sin insight candidates persistidos',
  activeLabel = (count) => `${count} activos`,
  maxRows = 50,
}: InsightCandidatesCardProps) {
  const rows = items.slice(0, Math.max(1, maxRows));

  return (
    <div className="card">
      <div className="card-header admin-card-header--wrap">
        <h2>{title}</h2>
        <div className="admin-audit-header-actions">
          <span className="badge badge-neutral">{activeLabel(items.length)}</span>
        </div>
      </div>
      {items.length === 0 ? (
        <div className="empty-state">
          <p>{emptyText}</p>
        </div>
      ) : (
        <div className="admin-activity-wrap">
          <table className="admin-activity-table">
            <thead>
              <tr>
                <th>Última vez</th>
                <th>Título</th>
                <th>Estado</th>
                <th>Severidad</th>
                <th>Evento</th>
                <th>Ocurrencias</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id}>
                  <td>{formatDateTime(row.last_seen_at)}</td>
                  <td>
                    <div>{row.title}</div>
                    <div className="admin-row-subtext">{row.body}</div>
                  </td>
                  <td>
                    <code className="admin-code">{row.status}</code>
                  </td>
                  <td>
                    <code className="admin-code">{row.severity}</code>
                  </td>
                  <td>
                    <code className="admin-code">{row.event_type}</code>
                  </td>
                  <td>{row.occurrence_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

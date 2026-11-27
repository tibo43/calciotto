// Display helpers for the Agoranking tab (RankingsV2.vue). The ranking itself
// is computed by the backend (GET /standings/agoranking): rows arrive already
// sorted, with the unrounded rating.

export function formatRating(rating) {
  const value = Number(rating);
  return Number.isFinite(value) ? value.toFixed(2) : '–';
}

export function pointsLabel(points) {
  return `${points} ${points === 1 ? 'pt' : 'pts'}`;
}

export function recordLine(row) {
  return `${row.Won}W ${row.Drawn}D ${row.Lost}L · ${pointsLabel(row.Points)}`;
}

// Mobile card line of a ranked player: "8 M · 4W 2D 2L · 14 pts".
export function rankedLine(row) {
  return `${row.Played} M · ${recordLine(row)}`;
}

export function emptyAgoranking() {
  return { Matches: 0, N: 0, GroupAverage: 0, Ranked: [], Unranked: [] };
}

// Accepts whatever the API returned and always gives back the full shape.
export function normalizeAgoranking(data) {
  const base = emptyAgoranking();
  if (!data || typeof data !== 'object') return base;
  return {
    ...base,
    ...data,
    Ranked: Array.isArray(data.Ranked) ? data.Ranked : [],
    Unranked: Array.isArray(data.Unranked) ? data.Unranked : [],
  };
}

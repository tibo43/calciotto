// Display helpers for the Agoranking tab (RankingsV2.vue). The ranking itself
// is computed by the backend (GET /standings/agoranking): rows arrive already
// sorted, with the unrounded rating.

import { t } from '@/i18n';
import { formatDecimal } from '@/services/intl';

// Hidden feature (GET /players/me/features), matching models.FeatureAgoranking.
export const AGORANKING_FEATURE = 'agoranking';

export function formatRating(rating) {
  const value = Number(rating);
  return Number.isFinite(value) ? formatDecimal(value) : '–';
}

export function pointsLabel(points) {
  return t('unit.points', points);
}

export function recordLine(row) {
  return t('rankings.agr.record', { won: row.Won, drawn: row.Drawn, lost: row.Lost, points: pointsLabel(row.Points) });
}

// Mobile card line of a ranked player: "8 M · 4W 2D 2L · 14 pts".
export function rankedLine(row) {
  return t('rankings.agr.rankedLine', { played: row.Played, record: recordLine(row) });
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

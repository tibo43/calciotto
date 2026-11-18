import { getMyFeatureStatus, getSessionKey } from './api';

const EMPTY_STATUS = { features: [] };
let session = null;

// Router guards and the shell share one features request per session.
export function loadFeatureStatus() {
  const key = getSessionKey();
  if (!key) {
    return Promise.resolve(EMPTY_STATUS);
  }
  if (session?.key === key) {
    return session.promise;
  }
  const entry = { key, promise: null, status: null };
  entry.promise = getMyFeatureStatus()
    .then((status) => {
      entry.status = status;
      return status;
    })
    .catch((error) => {
      console.error('Error fetching features:', error);
      if (session === entry) {
        session = null;
      }
      return EMPTY_STATUS;
    });
  session = entry;
  return entry.promise;
}

export function knownFeatureStatus() {
  return session && session.key === getSessionKey() ? session.status : null;
}

export function rememberGuideCompletion() {
  const status = knownFeatureStatus();
  if (status) status.guideCompleted = true;
}

export function resetFeatureStatus() {
  session = null;
}

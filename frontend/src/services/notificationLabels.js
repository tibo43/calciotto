// Labels of push notification types in the settings that are not written
// inline in the page listing them. The notification texts themselves are in
// public/sw.js.

// The motm_announced notification, listed on the Profile notifications page
// right under "Final score". Its own wording, not MVP / MVP of the Match:
// "MVP announced". The stored preference key stays motm_announced; the text
// is the message under this key (en.json).
export const MOTM_ANNOUNCED_KEY = 'notifications.motmAnnounced';

// Agoran's service worker. Deliberately empty of any caching: it exists so
// the app is installable and so it can receive push notifications — not to
// work offline. A worker that caches the app shell is the classic cause of
// "the app never updates", and nothing here needs offline use.
//
// No `fetch` handler either: without one, every request goes straight to the
// network exactly as it would with no worker at all.

// Take over as soon as a new version is installed, instead of waiting for
// every tab of the old one to close: there is no cached state to migrate.
self.addEventListener('install', () => {
  self.skipWaiting();
});

// The legacy frontend host only redirects to the new domain now, except for
// this file (scripts/vercel-domains.mjs), so that a worker installed there
// picks up this version on its next update check (after a push, a click on a
// notification, or a visit) and retires: it unsubscribes from push and
// unregisters, so a player who turns notifications on again on the new
// domain does not get every push twice.
const LEGACY_HOST = 'calcio-otto.vercel.app';

async function retireLegacyWorker(registration) {
  const subscription = await registration.pushManager.getSubscription();
  if (subscription) await subscription.unsubscribe();
  await registration.unregister();
}

self.addEventListener('activate', (event) => {
  if (self.location.hostname === LEGACY_HOST) {
    event.waitUntil(retireLegacyWorker(self.registration).catch(() => {}));
    return;
  }
  event.waitUntil(self.clients.claim());
});

// ---------------------------------------------------------------------------
// Push notifications. The backend (services/notifications.go) sends data, not
// text: the wording lives here so that a kick-off time is written in the
// phone's own time zone, the way the rest of the app shows it
// (src/services/datetime.js), and in the player's language.
//
// The language: the one the player chose in the app (the payload's `locale`,
// sent only when they chose one), else the first of the phone's languages the
// app speaks, else English. This worker is served as is from public/, outside
// the bundle, so it cannot import the message bundles: TEXT below is a copy of
// the push wording, and tests/unit/public/sw.spec.js keeps its languages in
// step with src/locales and every language to the same keys and placeholders.

const LANGUAGES = ['en', 'fr', 'it', 'de', 'es'];
const DEFAULT_LANGUAGE = 'en';
// English formats with en-US, as the rest of the app does.
const INTL_TAG = { en: 'en-US', fr: 'fr-FR', it: 'it-IT', de: 'de-DE', es: 'es-ES' };

// "fr", "fr-CA", "FR_ca" -> "fr" when the app speaks it, else null.
function matchLanguage(tag) {
  const match = /^([a-z]{2,3})(?:[-_]|$)/i.exec(String(tag || '').trim());
  const code = match ? match[1].toLowerCase() : '';
  return LANGUAGES.includes(code) ? code : null;
}

// The language of a notification: the player's own choice, else the phone's.
function resolveLanguage(chosen, phoneLanguages) {
  const own = matchLanguage(chosen);
  if (own) return own;
  for (const tag of Array.isArray(phoneLanguages) ? phoneLanguages : []) {
    const code = matchLanguage(tag);
    if (code) return code;
  }
  return DEFAULT_LANGUAGE;
}

// The short on-screen name of the award (MVP), the same in every language:
// `name` is a copy of src/services/v2Home.js's MVP, and
// tests/unit/public/sw.spec.js fails if the two differ. There is no plural: a
// tie still says "MVP: Luca & Marco".
const MVP_NAME = 'MVP';
const MOTM_TEXT = { name: MVP_NAME };

// {name} placeholders; `\u00a0` keeps a French « ! » or « : » with its word.
const TEXT = {
  en: {
    registrationOpen: { title: 'Sign-ups are open', body: '{prefix}match on {kickoff}. Grab your spot.' },
    teamAssigned: { title: "You're playing for {team}", body: '{prefix}match on {kickoff}.' },
    predictionsOpen: { title: 'Predictions are open', body: '{prefix}match on {kickoff}. Make your pick.' },
    kickoffReminder: { title: 'Match starting soon', body: 'Kick-off at {time}.', bodyTeam: "Kick-off at {time}, you're with {team}." },
    resultRecorded: { title: 'Final score: {score}', titleNoScore: 'The result is in', body: 'Vote for your ' + MVP_NAME + '.' },
    predictionsSettled: {
      title: 'Predictions settled',
      won: '{team} won! ',
      draw: "It's a draw! ",
      earned: { one: 'You earned {n} point.', other: 'You earned {n} points.' },
      lost: { one: 'You lost {n} point.', other: 'You lost {n} points.' },
    },
    motm: { winnerTitle: "You're the " + MVP_NAME + '!', voterTitle: MVP_NAME + ': {names}', body: 'See the game.' },
    memberJoined: {
      title: '{player} joined {group}',
      titleNoGroup: '{player} joined your group',
      someone: 'Someone',
      body: 'Not someone you know? Remove them and generate a new invite code.',
    },
    fallbackBody: 'Something new in your group.',
    colon: ': ',
    listAnd: ' & ',
  },
  fr: {
    registrationOpen: { title: 'Les inscriptions sont ouvertes', body: '{prefix}match le {kickoff}. Réserve ta place.' },
    teamAssigned: { title: 'Tu joues pour {team}', body: '{prefix}match le {kickoff}.' },
    predictionsOpen: { title: 'Les pronos sont ouverts', body: '{prefix}match le {kickoff}. Fais ton prono.' },
    kickoffReminder: { title: 'Le match va commencer', body: 'Coup d’envoi à {time}.', bodyTeam: 'Coup d’envoi à {time}, tu es avec {team}.' },
    resultRecorded: { title: 'Score final\u00a0: {score}', titleNoScore: 'Le résultat est tombé', body: 'Vote pour ton ' + MVP_NAME + '.' },
    predictionsSettled: {
      title: 'Résultats des pronos',
      won: '{team} a gagné\u00a0! ',
      draw: 'Match nul\u00a0! ',
      earned: { one: 'Tu marques {n}\u00a0point.', other: 'Tu marques {n}\u00a0points.' },
      lost: { one: 'Tu as perdu {n}\u00a0point.', other: 'Tu as perdu {n}\u00a0points.' },
    },
    motm: { winnerTitle: 'Tu es le ' + MVP_NAME + '\u00a0!', voterTitle: MVP_NAME + '\u00a0: {names}', body: 'Voir le match.' },
    memberJoined: {
      title: '{player} a rejoint {group}',
      titleNoGroup: '{player} a rejoint ton groupe',
      someone: 'Quelqu’un',
      body: 'Tu ne connais pas cette personne\u00a0? Retire-la et génère un nouveau code d’invitation.',
    },
    fallbackBody: 'Du nouveau dans ton groupe.',
    colon: '\u00a0: ',
    listAnd: ' & ',
  },
  it: {
    registrationOpen: { title: 'Le iscrizioni sono aperte', body: '{prefix}partita il {kickoff}. Prenota il tuo posto.' },
    teamAssigned: { title: 'Giochi per {team}', body: '{prefix}partita il {kickoff}.' },
    predictionsOpen: { title: 'I pronostici sono aperti', body: '{prefix}partita il {kickoff}. Fai il tuo pronostico.' },
    kickoffReminder: { title: 'La partita sta per iniziare', body: 'Calcio d’inizio alle {time}.', bodyTeam: 'Calcio d’inizio alle {time}, sei con {team}.' },
    resultRecorded: { title: 'Risultato finale: {score}', titleNoScore: 'Il risultato è arrivato', body: 'Vota il tuo ' + MVP_NAME + '.' },
    predictionsSettled: {
      title: 'Esiti dei pronostici',
      won: 'Ha vinto {team}! ',
      draw: 'Pareggio! ',
      earned: { one: 'Hai guadagnato {n}\u00a0punto.', other: 'Hai guadagnato {n}\u00a0punti.' },
      lost: { one: 'Hai perso {n}\u00a0punto.', other: 'Hai perso {n}\u00a0punti.' },
    },
    motm: { winnerTitle: 'Sei tu l’' + MVP_NAME + '!', voterTitle: MVP_NAME + ': {names}', body: 'Vai alla partita.' },
    memberJoined: {
      title: '{player} è entrato in {group}',
      titleNoGroup: '{player} è entrato nel tuo gruppo',
      someone: 'Qualcuno',
      body: 'Non conosci questa persona? Rimuovila e genera un nuovo codice d’invito.',
    },
    fallbackBody: 'Novità nel tuo gruppo.',
    colon: ': ',
    listAnd: ' & ',
  },
  de: {
    registrationOpen: { title: 'Die Anmeldung ist offen', body: '{prefix}Spiel am {kickoff}. Sichere dir deinen Platz.' },
    teamAssigned: { title: 'Du spielst für {team}', body: '{prefix}Spiel am {kickoff}.' },
    predictionsOpen: { title: 'Die Tipprunde ist offen', body: '{prefix}Spiel am {kickoff}. Gib deinen Tipp ab.' },
    kickoffReminder: { title: 'Das Spiel beginnt gleich', body: 'Anstoß um {time}.', bodyTeam: 'Anstoß um {time}, du bist bei {team}.' },
    resultRecorded: { title: 'Endergebnis: {score}', titleNoScore: 'Das Ergebnis steht fest', body: 'Stimme für deinen ' + MVP_NAME + ' ab.' },
    predictionsSettled: {
      title: 'Tipps ausgewertet',
      won: '{team} hat gewonnen! ',
      draw: 'Unentschieden! ',
      earned: { one: 'Du bekommst {n}\u00a0Punkt.', other: 'Du bekommst {n}\u00a0Punkte.' },
      lost: { one: 'Du hast {n}\u00a0Punkt verloren.', other: 'Du hast {n}\u00a0Punkte verloren.' },
    },
    motm: { winnerTitle: 'Du bist der ' + MVP_NAME + '!', voterTitle: MVP_NAME + ': {names}', body: 'Zum Spiel.' },
    memberJoined: {
      title: '{player} ist {group} beigetreten',
      titleNoGroup: '{player} ist deiner Gruppe beigetreten',
      someone: 'Jemand',
      body: 'Kennst du die Person nicht? Entferne sie und erzeuge einen neuen Einladungscode.',
    },
    fallbackBody: 'Neues in deiner Gruppe.',
    colon: ': ',
    listAnd: ' & ',
  },
  es: {
    registrationOpen: { title: 'Las inscripciones están abiertas', body: '{prefix}partido el {kickoff}. Asegura tu plaza.' },
    teamAssigned: { title: 'Juegas con {team}', body: '{prefix}partido el {kickoff}.' },
    predictionsOpen: { title: 'Los pronósticos están abiertos', body: '{prefix}partido el {kickoff}. Haz tu pronóstico.' },
    kickoffReminder: { title: 'El partido empieza pronto', body: 'Inicio a las {time}.', bodyTeam: 'Inicio a las {time}, vas con {team}.' },
    resultRecorded: { title: 'Resultado final: {score}', titleNoScore: 'Ya hay resultado', body: 'Vota a tu ' + MVP_NAME + '.' },
    predictionsSettled: {
      title: 'Pronósticos resueltos',
      won: '¡Ganó {team}! ',
      draw: '¡Empate! ',
      earned: { one: 'Has ganado {n}\u00a0punto.', other: 'Has ganado {n}\u00a0puntos.' },
      lost: { one: 'Has perdido {n}\u00a0punto.', other: 'Has perdido {n}\u00a0puntos.' },
    },
    motm: { winnerTitle: '¡Eres el ' + MVP_NAME + '!', voterTitle: MVP_NAME + ': {names}', body: 'Ver el partido.' },
    memberJoined: {
      title: '{player} se ha unido a {group}',
      titleNoGroup: '{player} se ha unido a tu grupo',
      someone: 'Alguien',
      body: '¿No conoces a esta persona? Quítala y genera un nuevo código de invitación.',
    },
    fallbackBody: 'Novedades en tu grupo.',
    colon: ': ',
    listAnd: ' & ',
  },
};

function fill(template, values) {
  // A missing value (undefined, null) prints nothing rather than "undefined".
  return template.replace(/\{(\w+)\}/g, (_, key) => (values[key] == null ? '' : String(values[key])));
}

// Intl's own text for the language, in the phone's time zone.
function formatKickoff(iso, language) {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString(INTL_TAG[language], {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

function formatTime(iso, language) {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleTimeString(INTL_TAG[language], { hour: 'numeric', minute: '2-digit' });
}

function formatScore(score) {
  return (score || []).map((line) => `${line.team} ${line.goals}`).join(' – ');
}

// The result half of a settled prediction, without repeating the score the
// result notification already showed: "Bianco won! " with the real name of
// the team that scored more, "It's a draw! " when the top scores are level,
// and nothing when the score is missing or unusable.
function formatOutcome(score, text) {
  const lines = (Array.isArray(score) ? score : [])
    .filter((line) => line && Number.isFinite(Number(line.goals)));
  if (lines.length < 2) return '';
  const top = Math.max(...lines.map((line) => Number(line.goals)));
  const leaders = lines.filter((line) => Number(line.goals) === top);
  if (leaders.length > 1) return text.draw;
  return leaders[0].team ? fill(text.won, { team: leaders[0].team }) : '';
}

// "Luca", "Luca & Marco", "Luca, Marco & Sara": the winners of the vote,
// several on a tie.
function formatNames(names, and) {
  const list = (Array.isArray(names) ? names : []).filter(Boolean);
  if (list.length <= 1) return list[0] || '';
  return `${list.slice(0, -1).join(', ')}${and}${list[list.length - 1]}`;
}

// The points half of a settled prediction. A gain or a zero is "You earned
// 2.33 points."; a loss is "You lost 1 point." — the absolute value, no
// minus. A whole number drops the decimals; anything else keeps two, with the
// language's decimal separator. The noun follows the language's plural rule
// for that amount (English: singular only when it is exactly 1).
function formatPointsSentence(points, language, text) {
  const value = Number(points);
  const finite = Number.isFinite(value) ? value : 0;
  const rounded = Math.abs(finite).toFixed(2);
  const amount = rounded.endsWith('.00') ? rounded.slice(0, -3) : rounded;
  // Intl may be missing or partial in a worker: fall back to "." and the
  // English plural rather than throw and lose the push.
  let separator = '.';
  try {
    const decimal = new Intl.NumberFormat(INTL_TAG[language]).formatToParts(1.5).find((part) => part.type === 'decimal');
    if (decimal) separator = decimal.value;
  } catch {
    separator = '.';
  }
  const shown = amount.replace('.', separator);
  let category;
  try {
    category = new Intl.PluralRules(INTL_TAG[language]).select(Number(amount)) === 'one' ? 'one' : 'other';
  } catch {
    category = Number(amount) === 1 ? 'one' : 'other';
  }
  return fill((finite < 0 ? text.lost : text.earned)[category], { n: shown });
}

// Title and body for one push payload, in `language`. Unknown types still show
// something rather than nothing, since a push the user can't see is a wasted
// one.
function buildNotification(data, language = DEFAULT_LANGUAGE) {
  const text = TEXT[language] || TEXT[DEFAULT_LANGUAGE];
  const kickoff = data.kickoffAt ? formatKickoff(data.kickoffAt, language) : '';
  const prefix = data.groupName ? `${data.groupName}${text.colon}` : '';
  switch (data.type) {
    case 'registration_open':
      return {
        title: text.registrationOpen.title,
        body: fill(text.registrationOpen.body, { prefix, kickoff }),
      };
    case 'team_assigned':
      return {
        title: fill(text.teamAssigned.title, { team: data.teamName }),
        body: fill(text.teamAssigned.body, { prefix, kickoff }),
      };
    case 'predictions_open':
      return {
        title: text.predictionsOpen.title,
        body: fill(text.predictionsOpen.body, { prefix, kickoff }),
      };
    case 'kickoff_reminder': {
      const time = formatTime(data.kickoffAt, language);
      return {
        title: text.kickoffReminder.title,
        body: data.teamName
          ? fill(text.kickoffReminder.bodyTeam, { time, team: data.teamName })
          : fill(text.kickoffReminder.body, { time }),
      };
    }
    case 'result_recorded': {
      const score = formatScore(data.score);
      return {
        title: score ? fill(text.resultRecorded.title, { score }) : text.resultRecorded.titleNoScore,
        body: text.resultRecorded.body,
      };
    }
    case 'predictions_settled':
      return {
        title: text.predictionsSettled.title,
        body: formatOutcome(data.score, text.predictionsSettled) + formatPointsSentence(data.points, language, text.predictionsSettled),
      };
    case 'motm_announced': {
      if (data.winner) {
        return { title: text.motm.winnerTitle, body: text.motm.body };
      }
      const names = formatNames(data.winners, text.listAnd);
      return {
        title: names ? fill(text.motm.voterTitle, { names }) : MVP_NAME,
        body: text.motm.body,
      };
    }
    case 'member_joined': {
      const player = data.playerName || text.memberJoined.someone;
      return {
        title: data.groupName
          ? fill(text.memberJoined.title, { player, group: data.groupName })
          : fill(text.memberJoined.titleNoGroup, { player }),
        body: text.memberJoined.body,
      };
    }
    default:
      return { title: 'Agoran', body: data.groupName || text.fallbackBody };
  }
}

// The phone's languages in order of preference, like browserLanguages() in
// src/services/locale.js: navigator.languages when it lists any, else
// navigator.language.
function navigatorLanguages(nav) {
  if (!nav) return [];
  if (Array.isArray(nav.languages) && nav.languages.length) return nav.languages;
  return nav.language ? [nav.language] : [];
}

self.addEventListener('push', (event) => {
  let data;
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = {};
  }
  const language = resolveLanguage(data.locale, navigatorLanguages(self.navigator));
  const { title, body } = buildNotification(data, language);
  event.waitUntil(self.registration.showNotification(title, {
    body,
    icon: '/icons/icon-192.png',
    // The same tag replaces the previous notification of that kind for that
    // match (e.g. a team change after a first assignment) instead of stacking.
    tag: data.tag,
    data: { url: data.url || '/' },
  }));
});

// Tapping a notification focuses an open Agoran window and sends it to the
// match, or opens one.
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const url = new URL((event.notification.data && event.notification.data.url) || '/', self.location.origin).href;
  event.waitUntil((async () => {
    const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
    for (const client of windows) {
      if (new URL(client.url).origin === self.location.origin) {
        await client.focus();
        // navigate() only works on a window this worker controls; any
        // other gets a fresh window instead.
        try {
          return await client.navigate(url);
        } catch {
          break;
        }
      }
    }
    return self.clients.openWindow(url);
  })());
});

// Lets the unit tests import the wording without a service worker around.
if (typeof module === 'object' && module.exports) {
  module.exports = { buildNotification, resolveLanguage, navigatorLanguages, matchLanguage, LANGUAGES, TEXT, MOTM_TEXT, LEGACY_HOST, retireLegacyWorker };
}

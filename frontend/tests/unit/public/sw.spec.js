import { createRequire } from 'node:module';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { MVP } from '@/services/v2Home';
import { LOCALES } from '@/locales';

// The notification wording lives in public/sw.js (see its comment); the file
// exports it for exactly this test.
const { buildNotification, resolveLanguage, navigatorLanguages, matchLanguage, LANGUAGES, TEXT, MOTM_TEXT, LEGACY_HOST, retireLegacyWorker } = createRequire(import.meta.url)('../../../public/sw.js');

const KICKOFF = '2026-09-06T19:00:00Z';
const kickoffText = new Date(KICKOFF).toLocaleString('en-US', {
  weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit'
});
const kickoffTime = new Date(KICKOFF).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });

describe('buildNotification', () => {
  it("names the award like the v2 screens (its copy of MVP)", () => {
    expect(MOTM_TEXT.name).toBe(MVP);
  });

  it('announces open sign-ups with the group and kick-off', () => {
    expect(buildNotification({ type: 'registration_open', groupName: 'Calciotto Milano', kickoffAt: KICKOFF })).toEqual({
      title: 'Sign-ups are open',
      body: `Calciotto Milano: match on ${kickoffText}. Grab your spot.`
    });
  });

  it("names the player's team", () => {
    expect(buildNotification({ type: 'team_assigned', teamName: 'Black', groupName: 'Calciotto Milano', kickoffAt: KICKOFF })).toEqual({
      title: "You're playing for Black",
      body: `Calciotto Milano: match on ${kickoffText}.`
    });
  });

  it('reminds of kick-off, with the team once composed', () => {
    expect(buildNotification({ type: 'kickoff_reminder', kickoffAt: KICKOFF, teamName: 'White' }).body)
      .toBe(`Kick-off at ${kickoffTime}, you're with White.`);
    expect(buildNotification({ type: 'kickoff_reminder', kickoffAt: KICKOFF }).body)
      .toBe(`Kick-off at ${kickoffTime}.`);
  });

  it('gives the final score and asks for an MVP vote', () => {
    expect(buildNotification({
      type: 'result_recorded',
      score: [{ team: 'Black', goals: 3 }, { team: 'White', goals: 1 }]
    })).toEqual({ title: 'Final score: Black 3 – White 1', body: 'Vote for your MVP.' });
  });

  it('announces that predictions are open for the match', () => {
    expect(buildNotification({ type: 'predictions_open', groupName: 'Calciotto Milano', kickoffAt: KICKOFF })).toEqual({
      title: 'Predictions are open',
      body: `Calciotto Milano: match on ${kickoffText}. Make your pick.`
    });
  });

  it('settles a prediction with the winner and the points earned or lost', () => {
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 5 }, { team: 'Nero', goals: 3 }],
      points: (1 / 0.3) - 1
    })).toEqual({
      title: 'Predictions settled',
      body: 'Bianco won! You earned 2.33 points.'
    });
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 5 }, { team: 'Nero', goals: 3 }],
      points: -1
    }).body).toBe('Bianco won! You lost 1 point.');
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 0 }, { team: 'Nero', goals: 2 }],
      points: 1.2
    }).body).toBe('Nero won! You earned 1.20 points.');
  });

  it('says it is a draw when the goals are level', () => {
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 1 }, { team: 'Nero', goals: 1 }],
      points: 1.5
    }).body).toBe("It's a draw! You earned 1.50 points.");
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 0 }, { team: 'Nero', goals: 0 }],
      points: -1
    }).body).toBe("It's a draw! You lost 1 point.");
  });

  it('never repeats the score in the settled notification', () => {
    const { body } = buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 5 }, { team: 'Nero', goals: 3 }],
      points: 2
    });
    expect(body).not.toContain('Final score');
    expect(body).not.toMatch(/\d+ – \d+|5|3/);
  });

  it('shows the points alone without a usable score', () => {
    expect(buildNotification({ type: 'predictions_settled', points: 1.5 }).body)
      .toBe('You earned 1.50 points.');
    expect(buildNotification({ type: 'predictions_settled', score: [], points: -1.5 }).body)
      .toBe('You lost 1.50 points.');
    expect(buildNotification({ type: 'predictions_settled', score: [{ team: 'Bianco', goals: 2 }], points: 1 }).body)
      .toBe('You earned 1 point.');
    expect(buildNotification({ type: 'predictions_settled', points: 2 }).body)
      .toBe('You earned 2 points.');
    expect(buildNotification({ type: 'predictions_settled', points: 0 }).body)
      .toBe('You earned 0 points.');
  });

  it('tells an MVP winner they won', () => {
    expect(buildNotification({ type: 'motm_announced', winner: true, winners: ['Luca'] })).toEqual({
      title: "You're the MVP!",
      body: 'See the game.'
    });
    expect(buildNotification({ type: 'motm_announced', winner: true, winners: ['Luca', 'Marco'] }).title)
      .toBe("You're the MVP!");
  });

  it('tells a voter who the MVP is, every name on a tie', () => {
    expect(buildNotification({ type: 'motm_announced', winners: ['Luca'] })).toEqual({
      title: 'MVP: Luca',
      body: 'See the game.'
    });
    expect(buildNotification({ type: 'motm_announced', winners: ['Luca', 'Marco'] }).title)
      .toBe('MVP: Luca & Marco');
    expect(buildNotification({ type: 'motm_announced', winners: ['Luca', 'Marco', 'Sara'] }).title)
      .toBe('MVP: Luca, Marco & Sara');
    expect(buildNotification({ type: 'motm_announced', winners: ['Luca', 'Marco', 'Sara', 'Tom'] }).title)
      .toBe('MVP: Luca, Marco, Sara & Tom');
    expect(buildNotification({ type: 'motm_announced' })).toEqual({ title: 'MVP', body: 'See the game.' });
  });

  it('still shows something for a type it does not know', () => {
    expect(buildNotification({ type: 'something_new' }).title).toBe('Agoran');
  });

  it('names who joined which group for an admin', () => {
    const notification = buildNotification({ type: 'member_joined', playerName: 'Sam', groupName: 'Thursday Football' });
    expect(notification.title).toBe('Sam joined Thursday Football');
    expect(notification.body).toContain('generate a new invite code');
  });
});

describe('push wording in each language', () => {
  const KICK = '2026-09-06T19:00:00Z';
  const when = (tag) => new Date(KICK).toLocaleString(tag, { weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
  const clock = (tag) => new Date(KICK).toLocaleTimeString(tag, { hour: 'numeric', minute: '2-digit' });
  const SCORE = [{ team: 'Bianco', goals: 5 }, { team: 'Nero', goals: 3 }];

  it('speaks exactly the languages of the app', () => {
    expect([...LANGUAGES].sort()).toEqual(LOCALES.map((locale) => locale.code).sort());
  });

  it('has the same wording keys and placeholders in every language', () => {
    const shape = (node) => (typeof node === 'string'
      ? [...node.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',')
      : Object.fromEntries(Object.entries(node).map(([key, value]) => [key, shape(value)])));
    for (const language of LANGUAGES) {
      expect(Object.keys(TEXT[language]).sort(), language).toEqual(Object.keys(TEXT.en).sort());
      // Wording differs, the slots must not.
      expect(shape(TEXT[language]), language).toEqual(shape(TEXT.en));
    }
  });

  it('never translates the award name', () => {
    for (const language of LANGUAGES) {
      const { motm, resultRecorded } = TEXT[language];
      expect(motm.winnerTitle, language).toContain(MVP);
      expect(motm.voterTitle, language).toContain(MVP);
      expect(resultRecorded.body, language).toContain(MVP);
    }
  });

  it('writes German push texts with the kick-off in German', () => {
    expect(buildNotification({ type: 'registration_open', groupName: 'Calciotto', kickoffAt: KICK }, 'de')).toEqual({
      title: 'Die Anmeldung ist offen',
      body: `Calciotto: Spiel am ${when('de-DE')}. Sichere dir deinen Platz.`
    });
    expect(buildNotification({ type: 'kickoff_reminder', kickoffAt: KICK, teamName: 'Weiß' }, 'de').body)
      .toBe(`Anstoß um ${clock('de-DE')}, du bist bei Weiß.`);
    expect(buildNotification({ type: 'result_recorded', score: SCORE }, 'de'))
      .toEqual({ title: 'Endergebnis: Bianco 5 – Nero 3', body: 'Stimme für deinen MVP ab.' });
  });

  it('writes French push texts with the usual spacing before « : » and « ! »', () => {
    expect(buildNotification({ type: 'predictions_open', groupName: 'Calciotto', kickoffAt: KICK }, 'fr')).toEqual({
      title: 'Les pronos sont ouverts',
      body: `Calciotto\u00a0: match le ${when('fr-FR')}. Fais ton prono.`
    });
    expect(buildNotification({ type: 'motm_announced', winner: true }, 'fr').title).toBe('Tu es le MVP\u00a0!');
  });

  it('uses each language’s own wording and plural for settled predictions', () => {
    const settled = (points, language) => buildNotification({ type: 'predictions_settled', score: SCORE, points }, language);
    expect(settled(1.33, 'fr')).toEqual({ title: 'Résultats des pronos', body: 'Bianco a gagné\u00a0! Tu marques 1,33\u00a0point.' });
    expect(settled(-1, 'de').body).toBe('Bianco hat gewonnen! Du hast 1\u00a0Punkt verloren.');
    expect(settled(1.33, 'de').body).toBe('Bianco hat gewonnen! Du bekommst 1,33\u00a0Punkte.');
    expect(settled(1, 'de').body).toBe('Bianco hat gewonnen! Du bekommst 1\u00a0Punkt.');
    expect(settled(2.5, 'it')).toEqual({ title: 'Esiti dei pronostici', body: 'Ha vinto Bianco! Hai guadagnato 2,50\u00a0punti.' });
    expect(settled(0, 'es').body).toBe('¡Ganó Bianco! Has ganado 0\u00a0puntos.');
    expect(buildNotification({ type: 'predictions_settled', score: [{ team: 'A', goals: 2 }, { team: 'B', goals: 2 }], points: 1 }, 'it').body)
      .toBe('Pareggio! Hai guadagnato 1\u00a0punto.');
  });

  it('keeps English as the default when no language is given', () => {
    expect(buildNotification({ type: 'motm_announced', winner: true }).title).toBe("You're the MVP!");
    expect(buildNotification({ type: 'motm_announced', winner: true }, 'xx').title).toBe("You're the MVP!");
  });

  it('names who joined with and without a group, in each language', () => {
    expect(buildNotification({ type: 'member_joined', playerName: 'Sam', groupName: 'Calciotto' }, 'it').title).toBe('Sam è entrato in Calciotto');
    expect(buildNotification({ type: 'member_joined', playerName: 'Sam' }, 'it').title).toBe('Sam è entrato nel tuo gruppo');
    expect(buildNotification({ type: 'member_joined' }, 'de').title).toBe('Jemand ist deiner Gruppe beigetreten');
    expect(buildNotification({ type: 'member_joined', playerName: 'Sam' }, 'es').title).toBe('Sam se ha unido a tu grupo');
  });

  it('lists the MVP winners the same way in each language', () => {
    expect(buildNotification({ type: 'motm_announced', winners: ['Luca', 'Marco', 'Sara'] }, 'de').title).toBe('MVP: Luca, Marco & Sara');
  });

  it('gives every language a complete text for every type, even an unknown one', () => {
    for (const language of LANGUAGES) {
      for (const type of ['registration_open', 'team_assigned', 'predictions_open', 'kickoff_reminder', 'result_recorded', 'predictions_settled', 'motm_announced', 'member_joined', 'other']) {
        const { title, body } = buildNotification({ type, groupName: 'G', teamName: 'T', playerName: 'P', kickoffAt: KICK, score: SCORE, points: 1, winners: ['W'] }, language);
        expect(title, `${language} ${type}`).toBeTruthy();
        expect(body, `${language} ${type}`).toBeTruthy();
        expect(`${title} ${body}`, `${language} ${type}`).not.toMatch(/\{|undefined|NaN/);
      }
    }
  });
});

describe('resolveLanguage', () => {
  it('prefers the language the player chose in the app', () => {
    expect(resolveLanguage('de', ['fr-FR', 'en-US'])).toBe('de');
  });

  it("follows the phone's first supported language when the player chose none", () => {
    expect(resolveLanguage(undefined, ['pt-BR', 'it-IT', 'en'])).toBe('it');
    expect(resolveLanguage(null, ['fr-CA'])).toBe('fr');
    expect(resolveLanguage('', ['ES'])).toBe('es');
  });

  it('falls back to English', () => {
    expect(resolveLanguage(undefined, ['pt-BR', 'ja'])).toBe('en');
    expect(resolveLanguage('xx', [])).toBe('en');
    expect(resolveLanguage(undefined, undefined)).toBe('en');
  });

  it('reads language tags loosely', () => {
    expect(matchLanguage('de_AT')).toBe('de');
    expect(matchLanguage('de-AT')).toBe('de');
    expect(matchLanguage('deutsch')).toBeNull();
    expect(matchLanguage(null)).toBeNull();
  });
});

describe('navigatorLanguages', () => {
  it("lists the phone's languages, falling back on navigator.language when the list is empty", () => {
    expect(navigatorLanguages({ languages: ['it-IT', 'en'], language: 'fr' })).toEqual(['it-IT', 'en']);
    expect(navigatorLanguages({ languages: [], language: 'fr-FR' })).toEqual(['fr-FR']);
    expect(navigatorLanguages({ language: 'de' })).toEqual(['de']);
    expect(navigatorLanguages({ languages: [] })).toEqual([]);
    expect(navigatorLanguages(undefined)).toEqual([]);
    expect(resolveLanguage(undefined, navigatorLanguages({ languages: [], language: 'es-ES' }))).toBe('es');
  });
});

describe('robustness of the push text', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('still words a settled prediction when Intl is missing', () => {
    vi.stubGlobal('Intl', {});
    const SCORE = [{ team: 'Bianco', goals: 5 }, { team: 'Nero', goals: 3 }];
    expect(buildNotification({ type: 'predictions_settled', score: SCORE, points: 1.33 }, 'fr').body)
      .toBe('Bianco a gagné\u00a0! Tu marques 1.33\u00a0points.');
    expect(buildNotification({ type: 'predictions_settled', score: SCORE, points: -1 }).body)
      .toBe('Bianco won! You lost 1 point.');
    expect(buildNotification({ type: 'predictions_settled', points: 2 }).body)
      .toBe('You earned 2 points.');
  });

  it('never prints "undefined" for a value the payload lacks', () => {
    const { title, body } = buildNotification({ type: 'team_assigned', kickoffAt: KICKOFF });
    expect(title).toBe("You're playing for ");
    expect(`${title} ${body}`).not.toMatch(/undefined|null/);
    expect(buildNotification({ type: 'motm_announced', winners: [null, undefined] }).title).toBe('MVP');
  });
});

describe('retireLegacyWorker', () => {
  it('names the legacy frontend host from devops/env/domains.env', () => {
    const domainsEnv = readFileSync(resolve(import.meta.dirname, '../../../../devops/env/domains.env'), 'utf8');
    const origin = domainsEnv.match(/^LEGACY_FRONTEND_ORIGIN=(.+)$/m)[1].trim();
    expect(LEGACY_HOST).toBe(new URL(origin).hostname);
  });

  it('unsubscribes from push, then unregisters', async () => {
    const calls = [];
    const subscription = { unsubscribe: vi.fn(async () => { calls.push('unsubscribe'); return true; }) };
    const registration = {
      pushManager: { getSubscription: vi.fn(async () => subscription) },
      unregister: vi.fn(async () => { calls.push('unregister'); return true; }),
    };
    await retireLegacyWorker(registration);
    expect(calls).toEqual(['unsubscribe', 'unregister']);
  });

  it('still unregisters when there is no push subscription', async () => {
    const registration = {
      pushManager: { getSubscription: vi.fn(async () => null) },
      unregister: vi.fn(async () => true),
    };
    await retireLegacyWorker(registration);
    expect(registration.unregister).toHaveBeenCalledOnce();
  });
});

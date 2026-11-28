// Every English message that puts a count next to a noun, at 0, 1 and 2 (and
// at a decimal where a count can have one): « 1 match », never « 1 matches ».
import { afterEach, describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import { i18n } from '@/i18n';
import { pointsLabel } from '@/services/agoranking';

const t = (key, count) => i18n.global.t(key, count);
const tn = (key, named, count) => i18n.global.t(key, named, { plural: count });

afterEach(() => { i18n.global.locale.value = 'en'; });

describe('English count messages', () => {
  it.each([
    ['unit.points', 0, '0 pts'], ['unit.points', 1, '1 pt'], ['unit.points', 2, '2 pts'],
    ['match.goals', 0, '0 goals'], ['match.goals', 1, '1 goal'], ['match.goals', 2, '2 goals'],
    ['rankings.matches', 0, '0 matches'], ['rankings.matches', 1, '1 match'], ['rankings.matches', 2, '2 matches'],
    ['rankings.goals', 1, '1 goal'], ['rankings.goals', 2, '2 goals'],
    ['rankings.awards', 0, '0 awards'], ['rankings.awards', 1, '1 award'], ['rankings.awards', 2, '2 awards'],
    ['match.votesAria', 1, '1 vote'], ['match.votesAria', 2, '2 votes'],
    ['passkeys.devices', 0, '0 devices'], ['passkeys.devices', 1, '1 device'], ['passkeys.devices', 2, '2 devices'],
    ['home.placesLeft', 0, 'places left'], ['home.placesLeft', 1, 'place left'], ['home.placesLeft', 2, 'places left'],
    ['profile.groupMatches', 0, '0 matches'], ['profile.groupMatches', 1, '1 match'], ['profile.groupMatches', 2, '2 matches'],
    ['profile.groupPoints', 0, '0 pts'], ['profile.groupPoints', 1, '1 pt'], ['profile.groupPoints', 2, '2 pts'],
    ['profile.groupsCombined', 0, '0 groups combined'], ['profile.groupsCombined', 1, '1 group combined'], ['profile.groupsCombined', 2, '2 groups combined'],
    ['edit.confirmedCount', 0, '0 confirmed'], ['edit.confirmedCount', 1, '1 confirmed'], ['edit.confirmedCount', 2, '2 confirmed'],
    ['edit.placesCount', 0, '0 places'], ['edit.placesCount', 1, '1 place'], ['edit.placesCount', 2, '2 places'],
  ])('%s with %s reads « %s »', (key, count, text) => {
    expect(t(key, count)).toMatch(new RegExp(`${text}$`));
  });

  it('puts a decimal count in the plural, as English does', () => {
    expect(pointsLabel(1)).toBe('1 pt');
    expect(pointsLabel(1.5)).toBe('1.5 pts');
    expect(pointsLabel(0.5)).toBe('0.5 pts');
    expect(pointsLabel(2.25)).toBe('2.25 pts');
    expect(t('profile.groupPoints', 1.5)).toBe('1.5 pts');
  });

  it('writes the count the way the language writes numbers', () => {
    expect(t('rankings.matches', 1234)).toBe('1,234 matches');
    expect(t('unit.points', 1.5)).toBe('1.5 pts');
    expect(tn('unit.points', { n: 2.25 }, 2.25)).toBe('2.25 pts');
    expect(i18n.global.t('home.position', { n: 7 })).toBe('#7');
    // The plural form comes from the number, not from its formatted text.
    expect(t('unit.points', 1)).toBe('1 pt');
    expect(t('unit.points', 0)).toBe('0 pts');
  });

  it('does the same for $t in a template', () => {
    const wrapper = mount({ template: '<p>{{ $t("unit.points", 1.5) }} / {{ $t("rankings.agr.playedShort", { n: 1234 }) }}</p>' });
    expect(wrapper.text()).toBe('1.5 pts / 1,234 M');
  });

  it('agrees the reserve note with the number of confirmed players', () => {
    const note = (n) => tn('edit.signups.notComposed', { confirmed: n, bench: '' }, n);
    expect(note(0)).toBe('Not composed yet. 0 confirmed players are in the reserve. The waiting list is not split automatically.');
    expect(note(1)).toBe('Not composed yet. 1 confirmed player is in the reserve. The waiting list is not split automatically.');
    expect(note(2)).toBe('Not composed yet. 2 confirmed players are in the reserve. The waiting list is not split automatically.');
  });

  it('writes the two counts of the profile row and of the sign-up line each with its own noun', () => {
    const meta = (matches, points) => i18n.global.t('profile.groupMeta', {
      matches: t('profile.groupMatches', matches), points: t('profile.groupPoints', points),
    });
    expect(meta(1, 1)).toBe('1 match · 1 pt');
    expect(meta(0, 0)).toBe('0 matches · 0 pts');
    expect(meta(3, 6)).toBe('3 matches · 6 pts');
    const places = (confirmed, cap) => i18n.global.t('edit.confirmedPlaces', {
      confirmed: t('edit.confirmedCount', confirmed), cap: t('edit.placesCount', cap),
    });
    expect(places(1, 1)).toBe('1 confirmed / 1 place');
    expect(places(0, 12)).toBe('0 confirmed / 12 places');
    expect(places(11, 12)).toBe('11 confirmed / 12 places');
  });
});

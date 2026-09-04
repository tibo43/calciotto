import { mount, flushPromises } from '@vue/test-utils';
import MatchDetails from '@/components/MatchDetails.vue';
import {
  getMatchDetailsByID,
  updateMatch,
  getMatchRegistrations,
  registerForMatch,
  unregisterFromMatch,
  closeMatchRegistrations,
  reopenMatchRegistrations,
  getGroupMembers,
  getMatchVotes,
  voteForMotm,
  removeMotmVote,
  getToken
} from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';
import { decodeMatchId } from '@/services/shortLink';

jest.mock('@/services/api', () => ({
  getMatchDetailsByID: jest.fn(),
  updateMatch: jest.fn(),
  deleteMatch: jest.fn(),
  getGroupMembers: jest.fn(),
  createPlayer: jest.fn(),
  getMatchRegistrations: jest.fn(),
  registerForMatch: jest.fn(),
  unregisterFromMatch: jest.fn(),
  closeMatchRegistrations: jest.fn(),
  reopenMatchRegistrations: jest.fn(),
  getMatchVotes: jest.fn(),
  voteForMotm: jest.fn(),
  removeMotmVote: jest.fn(),
  getToken: jest.fn()
}));

jest.mock('@/services/activeGroup', () => ({
  resolveActiveGroup: jest.fn()
}));

const ME = '11111111-1111-1111-1111-111111111111';
const SOMEONE_ELSE = '22222222-2222-2222-2222-222222222222';
// A real-looking UUID rather than a placeholder string: whatsappShareUrl
// (see the "Share on WhatsApp" describe block) base62-encodes match.ID, which
// only works on an actual UUID.
const MATCH_ID = 'cccccccc-0000-4000-8000-000000000001';

// A JWT is only ever read here for its player_id claim, so an unsigned token
// with a real base64url payload is enough — currentPlayerIdFromToken never
// verifies anything.
const tokenFor = (playerId) => {
  const payload = Buffer.from(JSON.stringify({ player_id: playerId })).toString('base64');
  return `header.${payload}.signature`;
};

const OPENS_AT = '2026-09-01T12:00:00+02:00';
const KICKOFF = '2026-09-06T20:30:00+02:00';
const DURING_SIGNUPS = Date.parse('2026-09-04T09:00:00+02:00');
const BEFORE_SIGNUPS = Date.parse('2026-08-30T09:00:00+02:00');
const AFTER_KICKOFF = Date.parse('2026-09-06T22:00:00+02:00');

const teams = () => [
  { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0, Players: [] },
  { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
];

const scheduledMatch = (overrides = {}) => ({
  ID: MATCH_ID,
  GroupID: 'group-uuid',
  Date: '2026-09-06',
  ScheduledAt: KICKOFF,
  RegistrationOpensAt: OPENS_AT,
  MaxPlayers: 3,
  RegistrationCount: 0,
  Teams: teams(),
  ...overrides
});

const unscheduledMatch = () => ({
  ID: MATCH_ID,
  GroupID: 'group-uuid',
  Date: '2026-09-06',
  Teams: teams()
});

// A cap of 3 with 4 sign-ups: the server marks the 4th IsWaiting, and nothing on
// the client is allowed to recompute that.
const registrationList = () => [
  { PlayerID: SOMEONE_ELSE, Name: 'marco', Position: 1, IsWaiting: false, RegisteredAt: OPENS_AT },
  { PlayerID: 'p3', Name: 'luca', Position: 2, IsWaiting: false, RegisteredAt: OPENS_AT },
  { PlayerID: 'p4', Name: 'gigi', Position: 3, IsWaiting: false, RegisteredAt: OPENS_AT },
  { PlayerID: 'p5', Name: 'nico', Position: 4, IsWaiting: true, RegisteredAt: OPENS_AT }
];

let nowSpy;

const mountDetails = async ({ match, registrations = [], isAdmin = false, now = DURING_SIGNUPS } = {}) => {
  nowSpy = jest.spyOn(Date, 'now').mockReturnValue(now);
  getMatchDetailsByID.mockResolvedValue(match);
  getMatchRegistrations.mockResolvedValue(registrations);
  resolveActiveGroup.mockResolvedValue({
    groups: [{ id: 'group-uuid', role: isAdmin ? 'admin' : 'member' }],
    activeGroupId: 'group-uuid'
  });

  const wrapper = mount(MatchDetails, {
    global: {
      mocks: {
        $route: { params: { id: MATCH_ID } },
        $router: { go: jest.fn(), push: jest.fn() }
      }
    }
  });
  await flushPromises();
  return wrapper;
};

beforeEach(() => {
  jest.clearAllMocks();
  getToken.mockReturnValue(tokenFor(ME));
  registerForMatch.mockResolvedValue({ PlayerID: ME, Name: 'me', Position: 1, IsWaiting: false });
  unregisterFromMatch.mockResolvedValue({ unregistered: true });
  closeMatchRegistrations.mockResolvedValue({ closed: true });
  reopenMatchRegistrations.mockResolvedValue({ reopened: true });
  // A composed roster (see many fixtures below) makes MatchDetails.vue load
  // the Man of the Match tally in created() regardless of what a given test
  // is actually about — every test needs this mocked or loadMotmVotes' own
  // catch block logs a spurious "getMatchVotes is not a function" error.
  getMatchVotes.mockResolvedValue({ Tally: [], MyVoteFor: null });
  voteForMotm.mockResolvedValue({ Tally: [], MyVoteFor: null });
  removeMotmVote.mockResolvedValue({ unvoted: true });
});

afterEach(() => {
  if (nowSpy) nowSpy.mockRestore();
});

describe('MatchDetails.vue sign-up panel visibility', () => {
  it('shows no panel at all on an ordinary, unscheduled match', async () => {
    const wrapper = await mountDetails({ match: unscheduledMatch() });

    expect(wrapper.find('.signup-panel').exists()).toBe(false);
    // And no sign-up list is requested for a match that has none.
    expect(getMatchRegistrations).not.toHaveBeenCalled();
  });

  it('shows the panel with the kick-off date and time on a scheduled match', async () => {
    const wrapper = await mountDetails({ match: scheduledMatch() });

    expect(wrapper.find('.signup-panel').exists()).toBe(true);
    const kickoff = wrapper.find('.signup-kickoff-value').text();
    expect(kickoff).toContain('Sep 6, 2026');
    expect(kickoff).toMatch(/\d:\d\d\s?(AM|PM)/);
  });

  // The state badge, the count and Participate/Withdraw sit on one line
  // (.signup-status-group) rather than the count living only in the
  // "Confirmed X/Y" heading further down — a member glancing at the panel
  // sees state, room left, and the one button they might click, together.
  it('puts the state badge, the sign-up count and Participate on the same row', async () => {
    const wrapper = await mountDetails({ match: scheduledMatch(), registrations: registrationList() });

    const group = wrapper.find('.signup-status-group');
    expect(group.find('.signup-state-badge').text()).toBe('Sign-ups open');
    // registrationList() has 4 entries against MaxPlayers: 3.
    expect(group.find('.signup-count-inline').text()).toBe('4 / 3 signed up');
    expect(group.find('.participate-btn, .withdraw-btn').exists()).toBe(true);
  });

  it('loads the sign-up list for a scheduled match, without a group_id', async () => {
    await mountDetails({ match: scheduledMatch(), registrations: registrationList() });

    expect(getMatchRegistrations).toHaveBeenCalledWith(MATCH_ID);
    expect(getMatchRegistrations).toHaveBeenCalledTimes(1);
  });
});

// The requirement most easily got backwards: every other control on this page
// is admin-gated, but signing yourself up is what an ordinary member comes here
// to do.
describe('MatchDetails.vue Participate/Withdraw for a NON-admin', () => {
  it('offers Participate to a plain member who has not signed up', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: false
    });

    expect(wrapper.vm.isAdmin).toBe(false);
    expect(wrapper.find('.participate-btn').exists()).toBe(true);
    expect(wrapper.find('.withdraw-btn').exists()).toBe(false);
  });

  it('offers Withdraw, and not Participate, to a plain member already signed up', async () => {
    const registrations = registrationList();
    registrations[0] = { ...registrations[0], PlayerID: ME };
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations,
      isAdmin: false
    });

    expect(wrapper.find('.withdraw-btn').exists()).toBe(true);
    expect(wrapper.find('.participate-btn').exists()).toBe(false);
  });

  it('offers neither Close nor Reopen to a plain member', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: false
    });

    const labels = wrapper.findAll('.signup-actions button').map(button => button.text());
    expect(labels.join(' ')).not.toMatch(/sign-ups/i);
  });
});

describe('MatchDetails.vue sign-up window states', () => {
  it('hides both buttons before sign-ups open, and says when they do', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: [],
      now: BEFORE_SIGNUPS
    });

    expect(wrapper.vm.registrationState).toBe('not-open-yet');
    expect(wrapper.find('.participate-btn').exists()).toBe(false);
    expect(wrapper.find('.withdraw-btn').exists()).toBe(false);
    expect(wrapper.find('.signup-state-detail').text()).toContain('Sep 1, 2026');
  });

  // Copy check with teeth: there is no notification anywhere in this feature,
  // so the panel must not imply one.
  it('never promises to notify anyone when sign-ups open', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: [],
      now: BEFORE_SIGNUPS
    });

    const detail = wrapper.find('.signup-state-detail').text();
    expect(detail).toMatch(/check back/i);
    expect(detail).not.toMatch(/notify you when|we'll let you know|email/i);
  });

  // Withdrawing is gated on the same window as signing up, so the button has to
  // disappear rather than fail on click once an admin closes the list.
  it('hides Withdraw from a registered member once an admin has closed sign-ups', async () => {
    const registrations = registrationList();
    registrations[0] = { ...registrations[0], PlayerID: ME };
    const wrapper = await mountDetails({
      match: scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }),
      registrations
    });

    expect(wrapper.vm.registrationState).toBe('closed-by-admin');
    expect(wrapper.vm.isRegistered).toBe(true);
    expect(wrapper.find('.withdraw-btn').exists()).toBe(false);
    expect(wrapper.find('.participate-btn').exists()).toBe(false);
  });

  it('hides both buttons once kick-off has passed, even with sign-ups never closed', async () => {
    const registrations = registrationList();
    registrations[0] = { ...registrations[0], PlayerID: ME };
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations,
      now: AFTER_KICKOFF
    });

    expect(wrapper.vm.registrationState).toBe('closed-at-kickoff');
    expect(wrapper.find('.withdraw-btn').exists()).toBe(false);
    expect(wrapper.find('.participate-btn').exists()).toBe(false);
    expect(wrapper.find('.signup-state-detail').text()).toMatch(/kick-off has passed/i);
  });
});

describe('MatchDetails.vue admin close/reopen', () => {
  it('offers Close, not Reopen, to an admin while sign-ups are open', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: true
    });

    const labels = wrapper.findAll('.signup-actions button').map(button => button.text());
    expect(labels).toContain('Close sign-ups');
    expect(labels).not.toContain('Reopen sign-ups');
  });

  it('offers Reopen, not Close, to an admin on a list they closed', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }),
      registrations: registrationList(),
      isAdmin: true
    });

    const labels = wrapper.findAll('.signup-actions button').map(button => button.text());
    expect(labels).toContain('Reopen sign-ups');
    expect(labels).not.toContain('Close sign-ups');
  });

  it('offers neither once kick-off has passed — nobody can un-pass kick-off', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: true,
      now: AFTER_KICKOFF
    });

    expect(wrapper.vm.canCloseRegistrations).toBe(false);
    expect(wrapper.vm.canReopenRegistrations).toBe(false);
  });

  it('flips the state locally on close without touching the match, so no unsaved-changes prompt appears', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: true
    });

    await wrapper.vm.closeRegistrations();

    expect(closeMatchRegistrations).toHaveBeenCalledWith(MATCH_ID);
    expect(wrapper.vm.registrationState).toBe('closed-by-admin');
    expect(wrapper.vm.match.RegistrationsClosedAt).toBeUndefined();
    expect(wrapper.vm.hasUnsavedChanges()).toBe(false);
  });

  it('clears the state again on reopen', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }),
      registrations: registrationList(),
      isAdmin: true
    });

    await wrapper.vm.reopenRegistrations();

    expect(reopenMatchRegistrations).toHaveBeenCalledWith(MATCH_ID);
    expect(wrapper.vm.registrationState).toBe('open');
  });
});

describe('MatchDetails.vue "Share on WhatsApp"', () => {
  it('offers it to an admin while sign-ups are open', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: true
    });

    const link = wrapper.find('.whatsapp-share-btn');
    expect(link.exists()).toBe(true);
    expect(link.attributes('href')).toContain('https://wa.me/?text=');
    expect(link.attributes('target')).toBe('_blank');
  });

  it('hides it from a non-admin', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: false
    });

    expect(wrapper.find('.whatsapp-share-btn').exists()).toBe(false);
  });

  it('hides it once sign-ups are closed — nothing left to invite people to', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }),
      registrations: registrationList(),
      isAdmin: true
    });

    expect(wrapper.find('.whatsapp-share-btn').exists()).toBe(false);
  });

  it('hides it before sign-ups have opened yet', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: [],
      isAdmin: true,
      now: BEFORE_SIGNUPS
    });

    expect(wrapper.find('.whatsapp-share-btn').exists()).toBe(false);
  });

  it('includes the kick-off and a short /m/:code link, not the full /matches/:id/edit URL', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: true
    });

    const href = wrapper.find('.whatsapp-share-btn').attributes('href');
    const text = decodeURIComponent(href.replace('https://wa.me/?text=', ''));
    expect(text).toContain(wrapper.vm.kickoffLabel);

    const shortUrl = text.split('\n').pop();
    expect(shortUrl).toMatch(/^http:\/\/localhost\/m\/[0-9A-Za-z]+$/);
    expect(decodeMatchId(shortUrl.split('/m/')[1])).toBe(MATCH_ID);
  });
});

describe('MatchDetails.vue confirmed/waiting split', () => {
  it('splits the list on the server-sent IsWaiting, keeping each entry position', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList()
    });

    const lists = wrapper.findAll('.signup-list');
    expect(lists).toHaveLength(2);

    const confirmed = lists[0].findAll('.signup-entry');
    expect(confirmed).toHaveLength(3);
    expect(confirmed[0].text()).toContain('1');
    expect(confirmed[0].text()).toContain('Marco');
    expect(confirmed[2].text()).toContain('Gigi');

    const waiting = lists[1].findAll('.signup-entry');
    expect(waiting).toHaveLength(1);
    expect(waiting[0].text()).toContain('4');
    expect(waiting[0].text()).toContain('Nico');
    expect(lists[1].find('.signup-list-title').text()).toContain('Waiting list');
  });

  // The cap the confirmed heading shows comes from MaxPlayers, but which rows
  // are confirmed does not — a stale cap must never move a row between lists.
  it('trusts IsWaiting even when it disagrees with MaxPlayers', async () => {
    const registrations = [
      { PlayerID: 'p1', Name: 'marco', Position: 1, IsWaiting: false, RegisteredAt: OPENS_AT },
      { PlayerID: 'p2', Name: 'luca', Position: 2, IsWaiting: true, RegisteredAt: OPENS_AT }
    ];
    const wrapper = await mountDetails({
      match: scheduledMatch({ MaxPlayers: 16 }),
      registrations
    });

    expect(wrapper.vm.confirmedRegistrations).toHaveLength(1);
    expect(wrapper.vm.waitingRegistrations).toHaveLength(1);
  });

  it('hides the waiting list entirely when nobody is on it', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList().slice(0, 2)
    });

    expect(wrapper.findAll('.signup-list')).toHaveLength(1);
  });

  it('shows an empty-state row when nobody has signed up', async () => {
    const wrapper = await mountDetails({ match: scheduledMatch(), registrations: [] });

    expect(wrapper.find('.signup-empty').text()).toMatch(/nobody has signed up/i);
  });
});

describe('MatchDetails.vue signing up and withdrawing', () => {
  it('re-fetches the list after signing up rather than patching it locally', async () => {
    const wrapper = await mountDetails({ match: scheduledMatch(), registrations: [] });
    getMatchRegistrations.mockResolvedValue(registrationList());

    await wrapper.vm.participate();

    expect(registerForMatch).toHaveBeenCalledWith(MATCH_ID);
    // Once on load, once after the sign-up.
    expect(getMatchRegistrations).toHaveBeenCalledTimes(2);
    expect(wrapper.vm.registrations).toHaveLength(4);
  });

  it('says explicitly that the caller landed on the waiting list, with their number', async () => {
    registerForMatch.mockResolvedValue({ PlayerID: ME, Name: 'me', Position: 17, IsWaiting: true });
    const wrapper = await mountDetails({ match: scheduledMatch(), registrations: [] });

    await wrapper.vm.participate();

    expect(wrapper.vm.message).toContain('#17');
    expect(wrapper.vm.message).toMatch(/waiting list/i);
    expect(wrapper.vm.messageType).toBe('success');
  });

  it('does not claim a waiting-list place for a confirmed sign-up', async () => {
    registerForMatch.mockResolvedValue({ PlayerID: ME, Name: 'me', Position: 5, IsWaiting: false });
    const wrapper = await mountDetails({ match: scheduledMatch(), registrations: [] });

    await wrapper.vm.participate();

    expect(wrapper.vm.message).not.toMatch(/waiting list/i);
    expect(wrapper.vm.message).toContain('#5');
  });

  it('surfaces the backend 409 message verbatim when a sign-up is refused', async () => {
    registerForMatch.mockRejectedValue({
      response: { data: { error: 'registrations for this match are closed' } }
    });
    const wrapper = await mountDetails({ match: scheduledMatch(), registrations: [] });

    await wrapper.vm.participate();

    expect(wrapper.vm.message).toBe('registrations for this match are closed');
    expect(wrapper.vm.messageType).toBe('error');
  });

  it('confirms before withdrawing, and re-fetches the list so the promoted reserve shows', async () => {
    const registrations = registrationList();
    registrations[0] = { ...registrations[0], PlayerID: ME };
    const wrapper = await mountDetails({ match: scheduledMatch(), registrations });

    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);
    // The reserve is promoted server-side; the re-fetch is how it becomes visible.
    getMatchRegistrations.mockResolvedValue([
      { PlayerID: 'p3', Name: 'luca', Position: 1, IsWaiting: false, RegisteredAt: OPENS_AT },
      { PlayerID: 'p4', Name: 'gigi', Position: 2, IsWaiting: false, RegisteredAt: OPENS_AT },
      { PlayerID: 'p5', Name: 'nico', Position: 3, IsWaiting: false, RegisteredAt: OPENS_AT }
    ]);

    wrapper.vm.confirmWithdraw();
    await flushPromises();

    expect(unregisterFromMatch).toHaveBeenCalledWith(MATCH_ID);
    expect(wrapper.vm.waitingRegistrations).toHaveLength(0);
    expect(wrapper.vm.confirmedRegistrations).toHaveLength(3);
    confirmSpy.mockRestore();
  });

  it('does nothing at all when the withdrawal confirm is dismissed', async () => {
    const registrations = registrationList();
    registrations[0] = { ...registrations[0], PlayerID: ME };
    const wrapper = await mountDetails({ match: scheduledMatch(), registrations });

    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(false);

    wrapper.vm.confirmWithdraw();
    await flushPromises();

    expect(unregisterFromMatch).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });
});

// The gating is the whole risk here: the action edits the two team rosters, so
// it must never be reachable by a plain member, and it must not be offered while
// the sign-up list is still moving.
describe('MatchDetails.vue "Fill teams from sign-ups" gating', () => {
  const CLOSED = '2026-09-05T18:00:00+02:00';

  it('offers it to an admin once sign-ups are closed', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch({ RegistrationsClosedAt: CLOSED }),
      registrations: registrationList(),
      isAdmin: true
    });

    expect(wrapper.vm.registrationState).toBe('closed-by-admin');
    expect(wrapper.find('.fill-teams-btn').exists()).toBe(true);
  });

  it('offers it to an admin once kick-off has closed the list on its own', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: true,
      now: AFTER_KICKOFF
    });

    expect(wrapper.vm.registrationState).toBe('closed-at-kickoff');
    expect(wrapper.find('.fill-teams-btn').exists()).toBe(true);
  });

  it('does not offer it to a plain member, closed list or not', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch({ RegistrationsClosedAt: CLOSED }),
      registrations: registrationList(),
      isAdmin: false
    });

    expect(wrapper.vm.canFillTeamsFromSignups).toBe(false);
    expect(wrapper.find('.fill-teams-btn').exists()).toBe(false);
  });

  it('does not offer it to an admin while sign-ups are still open', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: true
    });

    expect(wrapper.vm.registrationState).toBe('open');
    expect(wrapper.find('.fill-teams-btn').exists()).toBe(false);
  });

  it('does not offer it before sign-ups have opened either', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: [],
      isAdmin: true,
      now: BEFORE_SIGNUPS
    });

    expect(wrapper.vm.registrationState).toBe('not-open-yet');
    expect(wrapper.find('.fill-teams-btn').exists()).toBe(false);
  });

  it('does not offer it on an ordinary, unscheduled match', async () => {
    const wrapper = await mountDetails({ match: unscheduledMatch(), isAdmin: true });

    expect(wrapper.find('.fill-teams-btn').exists()).toBe(false);
  });

  // "Fill teams from sign-ups" opens a chooser (Auto-split vs Build
  // manually) rather than acting immediately — see the describe block below.
  // The button itself stays offered regardless of composition state, since
  // "build manually" is meaningful even once everyone confirmed is already
  // placed; what changes is which option the chooser offers.
  //
  // registrationList()'s 3 confirmed entries are marco (SOMEONE_ELSE), luca
  // (p3) and gigi (p4) — see the fixture above.
  it('hides only the Auto-split option once every confirmed sign-up already has a team', async () => {
    const match = scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' });
    match.Teams[0].Players = [
      { ID: SOMEONE_ELSE, Name: 'marco', GoalNumber: 0 },
      { ID: 'p3', Name: 'luca', GoalNumber: 0 }
    ];
    match.Teams[1].Players = [{ ID: 'p4', Name: 'gigi', GoalNumber: 0 }];
    const wrapper = await mountDetails({ match, registrations: registrationList(), isAdmin: true });

    expect(wrapper.find('.fill-teams-btn').exists()).toBe(true);
    await wrapper.find('.fill-teams-btn').trigger('click');

    expect(wrapper.find('.compose-choice-auto').exists()).toBe(false);
    expect(wrapper.find('.compose-choice-manual').exists()).toBe(true);
  });

  it('offers both options while at least one confirmed sign-up has no team yet', async () => {
    const match = scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' });
    // Only marco placed — luca and gigi are still unplaced.
    match.Teams[0].Players = [{ ID: SOMEONE_ELSE, Name: 'marco', GoalNumber: 0 }];
    const wrapper = await mountDetails({ match, registrations: registrationList(), isAdmin: true });
    await wrapper.find('.fill-teams-btn').trigger('click');

    expect(wrapper.find('.compose-choice-auto').exists()).toBe(true);
    expect(wrapper.find('.compose-choice-manual').exists()).toBe(true);
  });

  // The empty-confirmed-list case is deliberately left alone: hiding
  // Auto-split there too would leave an admin who closed an empty sign-up
  // list with no explanation for why choosing it does nothing. See "refuses
  // with an error when nobody is on the confirmed list" below, which still
  // exercises Auto-split being chosen in that state.
  it('still offers Auto-split when the confirmed list is empty, unlike the all-placed case', async () => {
    const match = scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' });
    const wrapper = await mountDetails({
      match,
      registrations: [{ PlayerID: 'p5', Name: 'nico', Position: 1, IsWaiting: true, RegisteredAt: OPENS_AT }],
      isAdmin: true
    });
    await wrapper.find('.fill-teams-btn').trigger('click');

    expect(wrapper.find('.compose-choice-auto').exists()).toBe(true);
  });
});

describe('MatchDetails.vue "Fill teams from sign-ups" behaviour', () => {
  const CLOSED = '2026-09-05T18:00:00+02:00';

  const mountClosedAsAdmin = (registrations, matchOverrides = {}) => mountDetails({
    match: scheduledMatch({ RegistrationsClosedAt: CLOSED, ...matchOverrides }),
    registrations,
    isAdmin: true
  });

  // "Fill teams from sign-ups" now opens the compose-choice modal rather than
  // acting immediately — this is the two-click path every case below drives
  // through to reach the same auto-split behaviour the old single click did.
  const clickAutoFill = async (wrapper) => {
    await wrapper.find('.fill-teams-btn').trigger('click');
    await wrapper.find('.compose-choice-auto').trigger('click');
  };

  it('alternates the confirmed roster across the two teams and ignores the waiting list', async () => {
    const wrapper = await mountClosedAsAdmin(registrationList());

    await clickAutoFill(wrapper);

    const [teamA, teamB] = wrapper.vm.match.Teams;
    expect(teamA.Players.map(p => p.Name)).toEqual(['marco', 'gigi']);
    expect(teamB.Players.map(p => p.Name)).toEqual(['luca']);
    // 'nico' is on the waiting list and must not be placed.
    expect(JSON.stringify(wrapper.vm.match.Teams)).not.toContain('nico');
  });

  it('saves nothing — it only marks the match dirty for the existing Save Changes', async () => {
    const wrapper = await mountClosedAsAdmin(registrationList());

    expect(wrapper.vm.hasUnsavedChanges()).toBe(false);
    await clickAutoFill(wrapper);

    expect(updateMatch).not.toHaveBeenCalled();
    expect(wrapper.vm.hasUnsavedChanges()).toBe(true);
  });

  it('says in the confirmation that nothing is saved yet', async () => {
    const wrapper = await mountClosedAsAdmin(registrationList());

    await clickAutoFill(wrapper);

    expect(wrapper.vm.messageType).toBe('success');
    expect(wrapper.vm.message).toMatch(/nothing is saved yet/i);
    expect(wrapper.vm.message).toMatch(/save changes/i);
  });

  // The copy has to be visible before the admin commits, not only afterwards:
  // the failure mode is an admin who fills the teams and walks away.
  it('warns in the Auto-split option, before it is chosen, that the split is not saved', async () => {
    const wrapper = await mountClosedAsAdmin(registrationList());
    await wrapper.find('.fill-teams-btn').trigger('click');

    const option = wrapper.find('.compose-choice-auto');
    expect(option.exists()).toBe(true);
    expect(option.text()).toMatch(/nothing is saved/i);
    // And it states the already-in-a-team policy, since that is the choice a
    // reader cannot otherwise guess.
    expect(option.text()).toMatch(/already in a team/i);
  });

  it('leaves a player already in a team alone rather than duplicating them', async () => {
    const match = scheduledMatch({ RegistrationsClosedAt: CLOSED });
    match.Teams[1].Players = [{ ID: SOMEONE_ELSE, Name: 'marco', GoalNumber: 2 }];
    const wrapper = await mountDetails({ match, registrations: registrationList(), isAdmin: true });

    await clickAutoFill(wrapper);

    const [teamA, teamB] = wrapper.vm.match.Teams;
    const everyone = [...teamA.Players, ...teamB.Players].map(p => p.Name);
    expect(everyone.filter(name => name === 'marco')).toHaveLength(1);
    // Their goals survive, and the message accounts for them.
    expect(teamB.Players[0].GoalNumber).toBe(2);
    expect(wrapper.vm.message).toMatch(/1 already in a team was left where they are/i);
  });

  it('refuses with an error when nobody is on the confirmed list', async () => {
    const wrapper = await mountClosedAsAdmin([
      { PlayerID: 'p5', Name: 'nico', Position: 1, IsWaiting: true, RegisteredAt: OPENS_AT }
    ]);

    await clickAutoFill(wrapper);

    expect(wrapper.vm.messageType).toBe('error');
    expect(wrapper.vm.message).toMatch(/nothing to fill the teams with/i);
    expect(wrapper.vm.hasUnsavedChanges()).toBe(false);
  });

  it('does nothing when called on a state that does not allow it, even directly', async () => {
    // The button is hidden while sign-ups are open; the method must not be the
    // weaker of the two checks.
    const wrapper = await mountDetails({
      match: scheduledMatch(),
      registrations: registrationList(),
      isAdmin: true
    });

    wrapper.vm.fillTeamsFromSignups();

    expect(wrapper.vm.match.Teams[0].Players).toHaveLength(0);
    expect(wrapper.vm.hasUnsavedChanges()).toBe(false);
  });
});

// This guard predates scheduled matches: it was written for the old
// always-populated-immediately match flow, where an empty team really was a
// mistake worth catching before saving a game record. A scheduled match's
// normal state is exactly the opposite — both teams start empty and stay that
// way through the whole sign-up window — so the same guard applied there made
// it impossible to ever save one.
describe('MatchDetails.vue Save Changes and empty teams', () => {
  it('blocks saving an unscheduled match with an empty team, unchanged from before', async () => {
    const wrapper = await mountDetails({ match: unscheduledMatch(), isAdmin: true });

    await wrapper.vm.saveChanges();

    expect(updateMatch).not.toHaveBeenCalled();
    expect(wrapper.vm.messageType).toBe('error');
    expect(wrapper.vm.message).toMatch(/each team requires at least 1 player/i);
  });

  it('allows saving a scheduled match with both teams still empty', async () => {
    updateMatch.mockResolvedValue({});
    const wrapper = await mountDetails({ match: scheduledMatch(), isAdmin: true });

    await wrapper.vm.saveChanges();

    expect(updateMatch).toHaveBeenCalledWith(MATCH_ID, expect.objectContaining({ ID: MATCH_ID }));
  });

  it('allows saving a scheduled match with only one team filled', async () => {
    updateMatch.mockResolvedValue({});
    const match = scheduledMatch();
    match.Teams[0].Players = [{ ID: SOMEONE_ELSE, Name: 'marco', GoalNumber: 0 }];
    const wrapper = await mountDetails({ match, isAdmin: true });

    await wrapper.vm.saveChanges();

    expect(updateMatch).toHaveBeenCalled();
  });
});

// The score header and the whole roster/team-management block (tabs, Add
// Player, the player list) hide until a scheduled match's teams are actually
// composed — an empty "0 vs 0" / two "No players yet" columns is noise for
// the whole sign-up window, not something worth rendering. Delete Match is
// deliberately left out of this gate (see the template): it stays reachable
// regardless, since cancelling a still-empty scheduled match is a legitimate
// action. Save Changes stays visible too, but disabled — there's nothing to
// persist before the roster exists, so a live button there was confusing.
describe('MatchDetails.vue team roster visibility on a scheduled match', () => {
  it('hides the score header and the roster while both teams are still empty', async () => {
    const wrapper = await mountDetails({ match: scheduledMatch(), isAdmin: true });

    expect(wrapper.find('.teams-score').exists()).toBe(false);
    expect(wrapper.find('.tabs-buttons').exists()).toBe(false);
    expect(wrapper.find('.players-grid').exists()).toBe(false);
    expect(wrapper.find('.no-roster-hint').exists()).toBe(true);
  });

  it('shows the roster once at least one team has a player', async () => {
    const match = scheduledMatch();
    match.Teams[0].Players = [{ ID: SOMEONE_ELSE, Name: 'marco', GoalNumber: 0 }];
    const wrapper = await mountDetails({ match, isAdmin: true });

    expect(wrapper.find('.teams-score').exists()).toBe(true);
    expect(wrapper.find('.tabs-buttons').exists()).toBe(true);
    expect(wrapper.find('.players-grid').exists()).toBe(true);
    expect(wrapper.find('.no-roster-hint').exists()).toBe(false);
  });

  it('still offers Save Changes and Delete Match to an admin while the roster is hidden, but Save is disabled', async () => {
    const wrapper = await mountDetails({ match: scheduledMatch(), isAdmin: true });

    expect(wrapper.html()).toContain('Save Changes');
    expect(wrapper.find('.team-management .btn-primary').attributes('disabled')).toBeDefined();
    expect(wrapper.html()).toContain('Delete Match');
    expect(wrapper.find('.delete-match-btn').attributes('disabled')).toBeUndefined();
  });

  it('enables Save Changes once the roster is composed', async () => {
    const match = scheduledMatch();
    match.Teams[0].Players = [{ ID: SOMEONE_ELSE, Name: 'marco', GoalNumber: 0 }];
    const wrapper = await mountDetails({ match, isAdmin: true });

    expect(wrapper.find('.team-management .btn-primary').attributes('disabled')).toBeUndefined();
  });

  it('leaves an unscheduled match exactly as it was, roster and all', async () => {
    const wrapper = await mountDetails({ match: unscheduledMatch(), isAdmin: true });

    expect(wrapper.find('.teams-score').exists()).toBe(true);
    expect(wrapper.find('.tabs-buttons').exists()).toBe(true);
    expect(wrapper.find('.no-roster-hint').exists()).toBe(false);
  });
});

// getMatchStatus() predates scheduled matches: "any goal recorded" was a
// reasonable proxy for "has this been played" back when a match was always
// created and scored immediately after the fact. These pin the fix for a
// scheduled match with a composed, scored roster entered *before* kick-off —
// it must read as upcoming regardless of the goal count, and only fall back
// to the goal-based heuristic once kick-off has actually passed.
describe('MatchDetails.vue match-status badge', () => {
  const scoredTeams = () => [
    { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 3, Players: [{ ID: 'p1', Name: 'marco', GoalNumber: 3 }] },
    { ID: 'team-b', Name: 'White', Colour: 'white', Score: 1, Players: [{ ID: 'p2', Name: 'luca', GoalNumber: 1 }] }
  ];

  it('reads "Upcoming" for a scheduled, already-scored match before kick-off', async () => {
    const match = scheduledMatch({ Teams: scoredTeams() });
    const wrapper = await mountDetails({ match, now: DURING_SIGNUPS }); // before KICKOFF

    expect(wrapper.find('.match-status-badge').classes()).toContain('upcoming');
    expect(wrapper.find('.match-status-badge').text()).toBe('Upcoming');
  });

  it('falls back to the goal-based read once kick-off has passed', async () => {
    const match = scheduledMatch({ Teams: scoredTeams() });
    const wrapper = await mountDetails({ match, now: AFTER_KICKOFF });

    expect(wrapper.find('.match-status-badge').classes()).toContain('completed');
  });

  it('reads "Upcoming" for a scheduled match with no goals yet, even past kick-off', async () => {
    const match = scheduledMatch(); // teams() defaults to Score: 0, Players: []
    const wrapper = await mountDetails({ match, now: AFTER_KICKOFF });

    expect(wrapper.find('.match-status-badge').classes()).toContain('upcoming');
  });

  it('leaves an unscheduled match on the original goal-based heuristic', async () => {
    const withGoals = { ...unscheduledMatch(), Teams: scoredTeams() };
    const wrapper = await mountDetails({ match: withGoals });

    expect(wrapper.find('.match-status-badge').classes()).toContain('completed');

    const noGoals = unscheduledMatch();
    const wrapper2 = await mountDetails({ match: noGoals });

    expect(wrapper2.find('.match-status-badge').classes()).toContain('upcoming');
  });
});

// "Fill teams from sign-ups" opens a choice between two ways to compose the
// roster from the sign-up list, rather than acting immediately.
describe('MatchDetails.vue compose-choice modal', () => {
  it('opens on click, with both options offered by default', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }),
      registrations: registrationList(),
      isAdmin: true
    });

    expect(wrapper.find('.compose-choice-modal').exists()).toBe(false);
    await wrapper.find('.fill-teams-btn').trigger('click');

    expect(wrapper.find('.compose-choice-modal').exists()).toBe(true);
    expect(wrapper.find('.compose-choice-auto').exists()).toBe(true);
    expect(wrapper.find('.compose-choice-manual').exists()).toBe(true);
  });

  it('closes without choosing anything via the close button', async () => {
    const wrapper = await mountDetails({
      match: scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }),
      registrations: registrationList(),
      isAdmin: true
    });
    await wrapper.find('.fill-teams-btn').trigger('click');

    await wrapper.find('.compose-choice-modal .modal-close').trigger('click');

    expect(wrapper.find('.compose-choice-modal').exists()).toBe(false);
    expect(wrapper.vm.match.Teams[0].Players).toHaveLength(0);
  });

  it('"Build manually" closes the chooser and opens the Add Player modal', async () => {
    getGroupMembers.mockResolvedValue([
      { id: SOMEONE_ELSE, name: 'marco', role: 'member' }
    ]);
    const wrapper = await mountDetails({
      match: scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }),
      registrations: registrationList(),
      isAdmin: true
    });
    await wrapper.find('.fill-teams-btn').trigger('click');

    await wrapper.find('.compose-choice-manual').trigger('click');
    await flushPromises();

    expect(wrapper.find('.compose-choice-modal').exists()).toBe(false);
    expect(wrapper.find('.enhanced-multi-player-modal').exists()).toBe(true);
    expect(getGroupMembers).toHaveBeenCalledWith('group-uuid');
  });
});

// The gap "Build manually" bridges: before this feature, Add Player (and the
// tabs it lives next to) stayed hidden by showTeamRoster until composition
// had already started somehow — there was no way to reach the very first
// player by hand on a freshly scheduled match. This describe block also
// covers the pre-existing "+" icon once a roster exists, since
// filterAvailablePlayers() drives both the same way.
describe('MatchDetails.vue Add Player list tiers sign-ups first', () => {
  // marco/luca/gigi confirmed (Position 1-3), nico waiting (Position 4) —
  // see registrationList(). 'unrelated' is a group member who never signed
  // up for this match at all.
  const groupMembers = () => [
    { id: SOMEONE_ELSE, name: 'marco', role: 'member' },
    { id: 'p3', name: 'luca', role: 'member' },
    { id: 'p4', name: 'gigi', role: 'member' },
    { id: 'p5', name: 'nico', role: 'member' },
    { id: 'p6', name: 'unrelated', role: 'member' }
  ];

  const openManually = async (match, registrations = registrationList()) => {
    getGroupMembers.mockResolvedValue(groupMembers());
    const wrapper = await mountDetails({ match, registrations, isAdmin: true });
    await wrapper.find('.fill-teams-btn').trigger('click');
    await wrapper.find('.compose-choice-manual').trigger('click');
    await flushPromises();
    return wrapper;
  };

  it('lists confirmed sign-ups first, in Position order, before the waiting list', async () => {
    const wrapper = await openManually(scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }));

    // formatPlayerNameForDisplay title-cases every name shown here.
    const names = wrapper.findAll('.available-player-item .player-name').map(n => n.text());
    expect(names).toEqual(['Marco', 'Luca', 'Gigi', 'Nico']);
  });

  it('leaves the rest of the group out of the default view entirely', async () => {
    const wrapper = await openManually(scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }));

    const names = wrapper.findAll('.available-player-item .player-name').map(n => n.text());
    expect(names).not.toContain('unrelated');
  });

  it('badges confirmed and waiting differently, with their sign-up position', async () => {
    const wrapper = await openManually(scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }));

    const items = wrapper.findAll('.available-player-item');
    const marco = items.find(item => item.text().includes('Marco'));
    const nico = items.find(item => item.text().includes('Nico'));

    expect(marco.find('.registration-badge').text()).toBe('#1');
    expect(marco.find('.registration-badge').classes()).not.toContain('waiting');
    expect(nico.find('.registration-badge').text()).toBe('#4 waiting');
    expect(nico.find('.registration-badge').classes()).toContain('waiting');
  });

  // onPlayerSearch debounces filterAvailablePlayers by 300ms (a real
  // setTimeout) — calling the filter directly is what the debounce is
  // shorthand for, once the input's own value has actually changed.
  it('reaches an unregistered group member the moment a search term matches them', async () => {
    const wrapper = await openManually(scheduledMatch({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' }));

    await wrapper.find('#playerSearch').setValue('unrel');
    wrapper.vm.filterAvailablePlayers();
    await wrapper.vm.$nextTick();

    const names = wrapper.findAll('.available-player-item .player-name').map(n => n.text());
    expect(names).toEqual(['Unrelated']);
  });

  it('leaves an unscheduled match on the plain, untiered group list', async () => {
    // Fill teams from sign-ups (and its chooser) only exist for a scheduled
    // match — the pre-existing "+" icon is the only trigger here, matching
    // how an ordinary match has always reached this modal.
    getGroupMembers.mockResolvedValue(groupMembers());
    const wrapper = await mountDetails({ match: unscheduledMatch(), registrations: [], isAdmin: true });
    await wrapper.find('.add-player-icon-btn').trigger('click');
    await flushPromises();

    const names = wrapper.findAll('.available-player-item .player-name').map(n => n.text());
    // allPlayers' own order, not sign-up order — and no badges anywhere.
    expect(names).toEqual(['Marco', 'Luca', 'Gigi', 'Nico', 'Unrelated']);
    expect(wrapper.find('.registration-badge').exists()).toBe(false);
  });
});

// Man of the Match voting: gated on teamsAreComposed exactly like the score
// header/roster itself (see "team roster visibility" above), and — unlike
// every other control on this page — not admin-gated at all.
describe('MatchDetails.vue Man of the Match voting', () => {
  const composedMatch = () => {
    const match = unscheduledMatch();
    match.Teams[0].Players = [
      { ID: ME, Name: 'me', GoalNumber: 0 },
      { ID: SOMEONE_ELSE, Name: 'marco', GoalNumber: 0 }
    ];
    return match;
  };

  it('hides the panel while no player has been placed on either team', async () => {
    const wrapper = await mountDetails({ match: scheduledMatch() });

    expect(wrapper.find('.motm-panel').exists()).toBe(false);
    expect(getMatchVotes).not.toHaveBeenCalled();
  });

  it('shows the panel and loads the tally, without a group_id, once the roster is composed', async () => {
    const wrapper = await mountDetails({ match: composedMatch() });

    expect(wrapper.find('.motm-panel').exists()).toBe(true);
    expect(getMatchVotes).toHaveBeenCalledWith(MATCH_ID);
    expect(getMatchVotes).toHaveBeenCalledTimes(1);
  });

  it('is not admin-gated: a plain member sees the same voting form', async () => {
    const wrapper = await mountDetails({ match: composedMatch(), isAdmin: false });

    expect(wrapper.find('.motm-panel').exists()).toBe(true);
    expect(wrapper.find('.motm-candidate-select').exists()).toBe(true);
  });

  it('excludes the caller from the candidate dropdown, but offers other roster players', async () => {
    const wrapper = await mountDetails({ match: composedMatch() });

    const values = wrapper.findAll('.motm-candidate-select option').map(o => o.attributes('value'));
    expect(values).not.toContain(ME);
    expect(values).toContain(SOMEONE_ELSE);
  });

  it('shows "Nobody has voted yet" and no tally list when nobody has voted', async () => {
    const wrapper = await mountDetails({ match: composedMatch() });

    expect(wrapper.find('.motm-empty').exists()).toBe(true);
    expect(wrapper.find('.motm-tally').exists()).toBe(false);
  });

  it('disables the vote button until a candidate is chosen', async () => {
    const wrapper = await mountDetails({ match: composedMatch() });

    expect(wrapper.find('.motm-vote-form .btn-primary').attributes('disabled')).toBeDefined();
  });

  it('preselects the dropdown on the caller\'s existing vote and offers "Change vote"/"Remove vote"', async () => {
    getMatchVotes.mockResolvedValue({
      Tally: [{ PlayerID: SOMEONE_ELSE, Name: 'marco', Votes: 1 }],
      MyVoteFor: SOMEONE_ELSE
    });
    const wrapper = await mountDetails({ match: composedMatch() });

    expect(wrapper.find('.motm-candidate-select').element.value).toBe(SOMEONE_ELSE);
    expect(wrapper.find('.motm-vote-form .btn-primary').text()).toBe('Change vote');
    expect(wrapper.find('.motm-vote-form .btn-secondary').exists()).toBe(true);
    expect(wrapper.find('.motm-my-vote').text()).toContain('Marco');
  });

  it('casts a vote for the selected candidate and reloads the tally', async () => {
    const wrapper = await mountDetails({ match: composedMatch() });

    await wrapper.find('.motm-candidate-select').setValue(SOMEONE_ELSE);
    await wrapper.find('.motm-vote-form .btn-primary').trigger('click');
    await flushPromises();

    expect(voteForMotm).toHaveBeenCalledWith(MATCH_ID, SOMEONE_ELSE);
    // Once on created(), once more after the vote is cast — always a full
    // reload rather than a locally-patched tally, same as loadRegistrations.
    expect(getMatchVotes).toHaveBeenCalledTimes(2);
  });

  it('removes the caller\'s own vote via "Remove vote"', async () => {
    getMatchVotes.mockResolvedValue({
      Tally: [{ PlayerID: SOMEONE_ELSE, Name: 'marco', Votes: 1 }],
      MyVoteFor: SOMEONE_ELSE
    });
    const wrapper = await mountDetails({ match: composedMatch() });

    await wrapper.find('.motm-vote-form .btn-secondary').trigger('click');
    await flushPromises();

    expect(removeMotmVote).toHaveBeenCalledWith(MATCH_ID);
  });
});

import { mount, flushPromises } from '@vue/test-utils';
import MatchesPanel from '@/components/MatchesPanel.vue';
import {
  getMatchesDetails,
  createMatch,
  getMatchRegistrations,
  registerForMatch,
  unregisterFromMatch,
  getMatchVotes,
  voteForMotm,
  removeMotmVote,
  getToken
} from '@/services/api';

jest.mock('@/services/api', () => ({
  getMatchesDetails: jest.fn(),
  createMatch: jest.fn(),
  getMatchRegistrations: jest.fn(),
  registerForMatch: jest.fn(),
  unregisterFromMatch: jest.fn(),
  getMatchVotes: jest.fn(),
  voteForMotm: jest.fn(),
  removeMotmVote: jest.fn(),
  getToken: jest.fn()
}));

// Europe/Paris in September, regardless of the machine running the suite.
const OFFSET_MINUTES = -120;

const ME = '11111111-1111-1111-1111-111111111111';
const SOMEONE_ELSE = '22222222-2222-2222-2222-222222222222';

// Same shape as MatchDetails.spec.js's own helper — an unsigned token with a
// real base64url payload is enough, since currentPlayerIdFromToken never
// verifies anything.
const tokenFor = (playerId) => {
  const payload = Buffer.from(JSON.stringify({ player_id: playerId })).toString('base64');
  return `header.${payload}.signature`;
};

let offsetSpy;
let push;

const mountPanel = async (propsOverrides = {}) => {
  const wrapper = mount(MatchesPanel, {
    props: { activeGroupId: 'group-uuid', isAdmin: true, season: '', ...propsOverrides },
    global: {
      stubs: { 'router-link': true },
      mocks: { $router: { push } }
    }
  });
  // created() kicks off loadMatches(), which now also awaits
  // loadSelectedRegistrations() and loadSelectedMotmVotes() for whichever
  // match got auto-selected — flushPromises is what reliably drains all of
  // it, where a fixed number of nextTick()s would be guessing at the chain's
  // depth.
  await flushPromises();
  return wrapper;
};

// Opens the create-match modal with a day already picked, which is the state
// every scheduling assertion below starts from.
const openModalWithDate = async (wrapper) => {
  await wrapper.setData({ showCreateModal: true, selectedDate: '2026-09-06' });
};

beforeEach(() => {
  getMatchesDetails.mockReset();
  getMatchesDetails.mockResolvedValue([]);
  createMatch.mockReset();
  createMatch.mockResolvedValue('new-match-uuid');
  getMatchRegistrations.mockReset();
  getMatchRegistrations.mockResolvedValue([]);
  registerForMatch.mockReset();
  registerForMatch.mockResolvedValue({ PlayerID: ME, Name: 'me', Position: 1, IsWaiting: false });
  unregisterFromMatch.mockReset();
  unregisterFromMatch.mockResolvedValue({ unregistered: true });
  getMatchVotes.mockReset();
  getMatchVotes.mockResolvedValue({ Tally: [], MyVoteFor: null });
  voteForMotm.mockReset();
  voteForMotm.mockResolvedValue({ Tally: [], MyVoteFor: null });
  removeMotmVote.mockReset();
  removeMotmVote.mockResolvedValue({ unvoted: true });
  getToken.mockReset();
  getToken.mockReturnValue(tokenFor(ME));
  push = jest.fn();
  offsetSpy = jest
    .spyOn(Date.prototype, 'getTimezoneOffset')
    .mockReturnValue(OFFSET_MINUTES);
});

afterEach(() => {
  offsetSpy.mockRestore();
});

describe('MatchesPanel.vue create-match modal', () => {
  it('hides the scheduling fields until the toggle is checked', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);

    expect(wrapper.find('.schedule-checkbox').exists()).toBe(true);
    expect(wrapper.find('.schedule-fields').exists()).toBe(false);

    await wrapper.find('.schedule-checkbox').setValue(true);

    expect(wrapper.find('.schedule-fields').exists()).toBe(true);
    expect(wrapper.find('#schedule-kickoff-time').exists()).toBe(true);
    expect(wrapper.find('#schedule-registration-opens').exists()).toBe(true);
    expect(wrapper.find('#schedule-max-players').exists()).toBe(true);
  });

  it('defaults the maximum to 16 players', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);
    await wrapper.find('.schedule-checkbox').setValue(true);

    expect(wrapper.find('#schedule-max-players').element.value).toBe('16');
  });

  it('creates an unscheduled match with no scheduling argument at all', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);

    await wrapper.vm.creatingMatch();

    expect(createMatch).toHaveBeenCalledWith({ Date: '2026-09-06' }, 'group-uuid', null);
    expect(push).toHaveBeenCalledWith('/matches/new-match-uuid/edit');
  });

  it('sends both timestamps with the browser local offset, never UTC', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);
    await wrapper.setData({
      isScheduled: true,
      kickoffTime: '20:30',
      registrationOpensAt: '2026-09-01T12:00',
      maxPlayers: 16
    });

    await wrapper.vm.creatingMatch();

    expect(createMatch).toHaveBeenCalledWith({ Date: '2026-09-06' }, 'group-uuid', {
      scheduledAt: '2026-09-06T20:30:00+02:00',
      registrationOpensAt: '2026-09-01T12:00:00+02:00',
      maxPlayers: 16
    });
  });

  it('coerces the max-players input from a string to a number', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);
    await wrapper.setData({
      isScheduled: true,
      kickoffTime: '20:30',
      registrationOpensAt: '2026-09-01T12:00',
      // v-model on a number input still hands back a string.
      maxPlayers: '12'
    });

    await wrapper.vm.creatingMatch();

    expect(createMatch.mock.calls[0][2].maxPlayers).toBe(12);
  });

  it('refuses to submit with a missing kick-off time', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);
    await wrapper.setData({ isScheduled: true, registrationOpensAt: '2026-09-01T12:00' });

    await wrapper.vm.creatingMatch();

    expect(createMatch).not.toHaveBeenCalled();
    expect(wrapper.find('.schedule-error').text()).toMatch(/kick-off time/i);
  });

  it('refuses to submit with a missing sign-up opening', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);
    await wrapper.setData({ isScheduled: true, kickoffTime: '20:30' });

    await wrapper.vm.creatingMatch();

    expect(createMatch).not.toHaveBeenCalled();
    expect(wrapper.find('.schedule-error').text()).toMatch(/sign-ups open/i);
  });

  it('refuses a maximum below one player', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);
    await wrapper.setData({
      isScheduled: true,
      kickoffTime: '20:30',
      registrationOpensAt: '2026-09-01T12:00',
      maxPlayers: 0
    });

    await wrapper.vm.creatingMatch();

    expect(createMatch).not.toHaveBeenCalled();
    expect(wrapper.find('.schedule-error').text()).toMatch(/at least 1/i);
  });

  it('refuses sign-ups that open at or after kick-off', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);
    await wrapper.setData({
      isScheduled: true,
      kickoffTime: '20:30',
      registrationOpensAt: '2026-09-06T20:30'
    });

    await wrapper.vm.creatingMatch();

    expect(createMatch).not.toHaveBeenCalled();
    expect(wrapper.find('.schedule-error').text()).toMatch(/before kick-off/i);
  });

  it('surfaces the backend message when it rejects the schedule anyway', async () => {
    createMatch.mockRejectedValue({
      response: { data: { error: 'registration_opens_at must be before scheduled_at' } }
    });
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);
    await wrapper.setData({
      isScheduled: true,
      kickoffTime: '20:30',
      registrationOpensAt: '2026-09-01T12:00'
    });

    await wrapper.vm.creatingMatch();

    expect(wrapper.vm.dateError).toBe('registration_opens_at must be before scheduled_at');
    expect(push).not.toHaveBeenCalled();
  });

  it('falls back to the generic failure message when the error carries none', async () => {
    createMatch.mockRejectedValue(new Error('network down'));
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);

    await wrapper.vm.creatingMatch();

    expect(wrapper.vm.dateError).toBe('Failed to create match. Please try again.');
  });

  it('resets the scheduling fields when the modal closes', async () => {
    const wrapper = await mountPanel();
    await openModalWithDate(wrapper);
    await wrapper.setData({
      isScheduled: true,
      kickoffTime: '20:30',
      registrationOpensAt: '2026-09-01T12:00',
      maxPlayers: 22,
      scheduleError: 'something'
    });

    wrapper.vm.closeModal();

    expect(wrapper.vm.isScheduled).toBe(false);
    expect(wrapper.vm.kickoffTime).toBe('');
    expect(wrapper.vm.registrationOpensAt).toBe('');
    expect(wrapper.vm.maxPlayers).toBe(16);
    expect(wrapper.vm.scheduleError).toBe('');
  });
});

// The card side of the sign-up feature. Everything about the list itself lives
// on the match page — the card only has to be recognisably a scheduled match,
// show its kick-off and show how full it is.
describe('MatchesPanel.vue scheduled match cards', () => {
  const scheduled = {
    ID: 'scheduled-uuid',
    GroupID: 'group-uuid',
    Date: '2026-09-06',
    ScheduledAt: '2026-09-06T20:30:00+02:00',
    RegistrationOpensAt: '2026-09-01T12:00:00+02:00',
    MaxPlayers: 16,
    RegistrationCount: 7,
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0, Players: [] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
    ]
  };

  const played = {
    ID: 'played-uuid',
    GroupID: 'group-uuid',
    Date: '2026-08-30',
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 3, Players: [] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 2, Players: [] }
    ]
  };

  it('marks a scheduled card, and shows its kick-off date and time', async () => {
    getMatchesDetails.mockResolvedValue([scheduled]);
    const wrapper = await mountPanel();

    const cards = wrapper.findAll('.match-card-horizontal:not(.add-match-card)');
    expect(cards).toHaveLength(1);
    expect(cards[0].classes()).toContain('scheduled');
    const dateLine = cards[0].find('.match-date-horizontal').text();
    expect(dateLine).toContain('Sep 6, 2026');
    expect(dateLine).toMatch(/\d:\d\d\s?(AM|PM)/);
  });

  it('shows how full the roster is, from the server-sent count', async () => {
    getMatchesDetails.mockResolvedValue([scheduled]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-count').text()).toBe('7 / 16 signed up');
    // Existence only, not the label's text: which state it derives to depends
    // on the real clock at test time (this fixture's window spans real dates),
    // and the point here is just that a scheduled card always carries a state
    // badge, not what it says right now — see the dedicated describe block
    // below for the label content itself, pinned against a frozen clock.
    expect(wrapper.find('.signup-state-badge').exists()).toBe(true);
  });

  // RegistrationCount is a *int with omitempty on the Go side precisely so that
  // "scheduled, nobody signed up" (0, key present) stays distinguishable from
  // "not a scheduled match" (key absent) — "0 / 16" is a real state to render.
  it('renders a zero sign-up count rather than treating it as missing', async () => {
    getMatchesDetails.mockResolvedValue([{ ...scheduled, RegistrationCount: 0 }]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-count').text()).toBe('0 / 16 signed up');
  });

  it('leaves an ordinary card exactly as it was: no badge, no count, plain date', async () => {
    getMatchesDetails.mockResolvedValue([played]);
    const wrapper = await mountPanel();

    const card = wrapper.find('.match-card-horizontal:not(.add-match-card)');
    expect(card.classes()).not.toContain('scheduled');
    expect(wrapper.find('.signup-state-badge').exists()).toBe(false);
    expect(wrapper.find('.signup-count').exists()).toBe(false);
    expect(card.find('.match-date-horizontal').text()).toBe('Aug 30, 2026');
  });

  // loadMatches() discards the *entire* list if any match fails its shape check,
  // so requiring the new scheduling keys there would blank the whole page as
  // soon as one unscheduled match showed up. This pins that it doesn't.
  it('keeps both kinds of match in one list, the new fields being purely additive', async () => {
    getMatchesDetails.mockResolvedValue([scheduled, played]);
    const wrapper = await mountPanel();

    expect(wrapper.vm.matches).toHaveLength(2);
    expect(wrapper.findAll('.match-card-horizontal:not(.add-match-card)')).toHaveLength(2);
  });
});

// The inline sign-up affordance in the "Selected Match Details" preview —
// added so a member can sign up without leaving the Matches tab. Deliberately
// light (state badge, count, Participate/Withdraw only); the full
// confirmed/waiting roster with names stays on the match page, which is why
// these tests never assert on a roster list here.
describe('MatchesPanel.vue inline sign-up panel', () => {
  // Factories, not shared consts: participate()/withdraw() mutate
  // selectedMatch.RegistrationCount in place (see the component), and a
  // shared object literal would carry that mutation from one test into the
  // next — matching MatchDetails.spec.js's own scheduledMatch()/teams()
  // pattern for exactly this reason.
  const scheduled = (overrides = {}) => ({
    ID: 'scheduled-uuid',
    GroupID: 'group-uuid',
    Date: '2026-09-06',
    ScheduledAt: '2026-09-06T20:30:00+02:00',
    RegistrationOpensAt: '2026-09-01T12:00:00+02:00',
    MaxPlayers: 16,
    RegistrationCount: 7,
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0, Players: [] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
    ],
    ...overrides
  });

  const played = () => ({
    ID: 'played-uuid',
    GroupID: 'group-uuid',
    Date: '2026-08-30',
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 3, Players: [] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 2, Players: [] }
    ]
  });

  it('shows no sign-up section at all for an ordinary, unscheduled match', async () => {
    getMatchesDetails.mockResolvedValue([played()]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-inline').exists()).toBe(false);
    expect(getMatchRegistrations).not.toHaveBeenCalled();
  });

  it('loads the sign-up list for the selected scheduled match alone, not on every card', async () => {
    getMatchesDetails.mockResolvedValue([scheduled(), played()]);
    await mountPanel();

    // scheduled is matches[0] — the one auto-selected — so this is the only
    // registrations call, never one for `played`.
    expect(getMatchRegistrations).toHaveBeenCalledWith('scheduled-uuid');
    expect(getMatchRegistrations).toHaveBeenCalledTimes(1);
  });

  it('re-fetches the sign-up list when a different scheduled match is selected', async () => {
    const otherScheduled = { ...scheduled(), ID: 'other-scheduled-uuid' };
    getMatchesDetails.mockResolvedValue([scheduled(), otherScheduled]);
    const wrapper = await mountPanel();
    getMatchRegistrations.mockClear();

    await wrapper.vm.selectMatch(otherScheduled);
    await flushPromises();

    expect(getMatchRegistrations).toHaveBeenCalledWith('other-scheduled-uuid');
  });

  it('offers Participate to a member who has not signed up, alongside the live count', async () => {
    getMatchesDetails.mockResolvedValue([scheduled()]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-state-badge').text()).toBe('Sign-ups open');
    expect(wrapper.find('.signup-count-inline').text()).toBe('7 / 16 signed up');
    expect(wrapper.find('.signup-inline-actions button').text()).toContain('Participate');
  });

  it('signs the caller up, bumps the count immediately, and switches to Withdraw', async () => {
    getMatchesDetails.mockResolvedValue([scheduled()]);
    registerForMatch.mockResolvedValue({ PlayerID: ME, Name: 'me', Position: 8, IsWaiting: false });
    const wrapper = await mountPanel();
    // The re-fetch loadSelectedRegistrations() runs right after a successful
    // register() must see the caller's own new row — a real backend would;
    // the default empty-list mock from beforeEach otherwise makes isRegistered
    // stay false and the button never flips.
    getMatchRegistrations.mockResolvedValue([
      { PlayerID: ME, Name: 'me', Position: 8, IsWaiting: false, RegisteredAt: '2026-09-02T10:00:00+02:00' }
    ]);

    await wrapper.find('.signup-inline-actions button').trigger('click');
    await flushPromises();

    expect(registerForMatch).toHaveBeenCalledWith('scheduled-uuid');
    // Bumped locally, not from a full matches reload — the point of the fix.
    expect(wrapper.find('.signup-count-inline').text()).toBe('8 / 16 signed up');
    // signupCountLabel reads the same match object the card renders from, so
    // the card's own badge reflects it too — "the list updates as we go".
    expect(wrapper.find('.match-card-horizontal.scheduled .signup-count').text()).toBe('8 / 16 signed up');
    expect(wrapper.find('.signup-inline-message').text()).toMatch(/you are #8/i);
    expect(wrapper.find('.signup-inline-actions button').text()).toContain('Withdraw');
  });

  it('says explicitly when the sign-up lands on the waiting list', async () => {
    getMatchesDetails.mockResolvedValue([scheduled()]);
    registerForMatch.mockResolvedValue({ PlayerID: ME, Name: 'me', Position: 17, IsWaiting: true });
    const wrapper = await mountPanel();

    await wrapper.find('.signup-inline-actions button').trigger('click');
    await flushPromises();

    expect(wrapper.find('.signup-inline-message').text()).toMatch(/waiting list/i);
  });

  it('withdraws, decrements the count, and switches back to Participate — after confirming', async () => {
    getMatchesDetails.mockResolvedValue([scheduled()]);
    // Once for the initial load (I'm registered, hence "Withdraw" showing
    // already), then empty for the re-fetch loadSelectedRegistrations() runs
    // right after a successful unregister() — a real backend would no longer
    // list my row either.
    getMatchRegistrations.mockResolvedValueOnce([
      { PlayerID: ME, Name: 'me', Position: 3, IsWaiting: false, RegisteredAt: '2026-09-01T12:00:00+02:00' }
    ]).mockResolvedValue([]);
    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-inline-actions button').text()).toContain('Withdraw');

    await wrapper.find('.signup-inline-actions button').trigger('click');
    await flushPromises();

    expect(confirmSpy).toHaveBeenCalled();
    expect(unregisterFromMatch).toHaveBeenCalledWith('scheduled-uuid');
    expect(wrapper.find('.signup-count-inline').text()).toBe('6 / 16 signed up');
    expect(wrapper.find('.signup-inline-actions button').text()).toContain('Participate');
    confirmSpy.mockRestore();
  });

  it('does nothing when the withdrawal confirm is dismissed', async () => {
    getMatchesDetails.mockResolvedValue([scheduled()]);
    getMatchRegistrations.mockResolvedValue([
      { PlayerID: ME, Name: 'me', Position: 3, IsWaiting: false, RegisteredAt: '2026-09-01T12:00:00+02:00' }
    ]);
    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(false);
    const wrapper = await mountPanel();

    await wrapper.find('.signup-inline-actions button').trigger('click');
    await flushPromises();

    expect(unregisterFromMatch).not.toHaveBeenCalled();
    expect(wrapper.find('.signup-inline-actions button').text()).toContain('Withdraw');
    confirmSpy.mockRestore();
  });

  it('surfaces the backend 409 message verbatim and reloads the list', async () => {
    getMatchesDetails.mockResolvedValue([scheduled()]);
    registerForMatch.mockRejectedValue({ response: { data: { error: 'registrations for this match are closed' } } });
    const wrapper = await mountPanel();
    getMatchRegistrations.mockClear();

    await wrapper.find('.signup-inline-actions button').trigger('click');
    await flushPromises();

    expect(wrapper.find('.signup-inline-message').text()).toBe('registrations for this match are closed');
    expect(wrapper.find('.signup-inline-message').classes()).toContain('error');
    // The count is left untouched on failure — only a successful call bumps it.
    expect(wrapper.find('.signup-count-inline').text()).toBe('7 / 16 signed up');
    expect(getMatchRegistrations).toHaveBeenCalledTimes(1);
  });

  it('hides both actions once sign-ups are closed', async () => {
    getMatchesDetails.mockResolvedValue([scheduled({ RegistrationsClosedAt: '2026-09-02T09:00:00+02:00' })]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-state-badge').text()).toBe('Sign-ups closed');
    expect(wrapper.find('.signup-inline-actions button').exists()).toBe(false);
  });

  it("says who is already registered as 'me', not a stranger's entry", async () => {
    getMatchesDetails.mockResolvedValue([scheduled()]);
    getMatchRegistrations.mockResolvedValue([
      { PlayerID: SOMEONE_ELSE, Name: 'someone else', Position: 1, IsWaiting: false, RegisteredAt: '2026-09-01T12:00:00+02:00' }
    ]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-inline-actions button').text()).toContain('Participate');
  });
});

// The full confirmed/waiting lists, with names — moved here from
// MatchDetails.vue's own .signup-panel now that a plain member can no longer
// reach that page at all (see router/index.js's canEditMatch). Reuses its
// exact markup/classes rather than inventing new ones.
describe('MatchesPanel.vue inline sign-up panel named lists', () => {
  const scheduled = (overrides = {}) => ({
    ID: 'scheduled-uuid',
    GroupID: 'group-uuid',
    Date: '2026-09-06',
    ScheduledAt: '2026-09-06T20:30:00+02:00',
    RegistrationOpensAt: '2026-09-01T12:00:00+02:00',
    MaxPlayers: 2,
    RegistrationCount: 3,
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0, Players: [] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
    ],
    ...overrides
  });

  const registrations = () => [
    { PlayerID: ME, Name: 'me', Position: 1, IsWaiting: false, RegisteredAt: '2026-09-01T12:00:00+02:00' },
    { PlayerID: SOMEONE_ELSE, Name: 'marco', Position: 2, IsWaiting: false, RegisteredAt: '2026-09-01T13:00:00+02:00' },
    { PlayerID: 'p3', Name: 'luca', Position: 3, IsWaiting: true, RegisteredAt: '2026-09-01T14:00:00+02:00' }
  ];

  it('renders the confirmed and waiting lists with names, marking the caller as "you"', async () => {
    getMatchesDetails.mockResolvedValue([scheduled()]);
    getMatchRegistrations.mockResolvedValue(registrations());
    const wrapper = await mountPanel();

    const confirmedNames = wrapper.findAll('.signup-list:not(.signup-list-waiting) .signup-name').map(n => n.text());
    expect(confirmedNames).toEqual(['Me', 'Marco']);
    expect(wrapper.find('.signup-list:not(.signup-list-waiting) .count-badge').text()).toBe('2 / 2');

    const waitingNames = wrapper.findAll('.signup-list-waiting .signup-name').map(n => n.text());
    expect(waitingNames).toEqual(['Luca']);
    expect(wrapper.find('.signup-entry.is-me .signup-you').exists()).toBe(true);
    expect(wrapper.find('.signup-entry.is-me .signup-name').text()).toBe('Me');
  });

  it('shows "Nobody has signed up yet" for an empty confirmed list, and no waiting section at all', async () => {
    getMatchesDetails.mockResolvedValue([scheduled({ RegistrationCount: 0 })]);
    getMatchRegistrations.mockResolvedValue([]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-empty').text()).toBe('Nobody has signed up yet');
    expect(wrapper.find('.signup-list-waiting').exists()).toBe(false);
  });
});

// The "Edit Match" link is admin-only now that MatchDetails.vue's route
// itself is (see router/index.js's beforeEnter/canEditMatch) — a plain
// member following it would only bounce straight back here, so it is hidden
// rather than shown-and-relabelled the way it used to be.
describe('MatchesPanel.vue edit-match link visibility', () => {
  const played = () => ({
    ID: 'played-uuid',
    GroupID: 'group-uuid',
    Date: '2026-08-30',
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 3, Players: [{ ID: 'p1', Name: 'marco', GoalNumber: 1 }] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 2, Players: [] }
    ]
  });

  it('shows the link to an admin', async () => {
    getMatchesDetails.mockResolvedValue([played()]);
    const wrapper = await mountPanel({ isAdmin: true });

    expect(wrapper.find('.edit-match-btn').exists()).toBe(true);
  });

  it('hides the link entirely from a plain member', async () => {
    getMatchesDetails.mockResolvedValue([played()]);
    const wrapper = await mountPanel({ isAdmin: false });

    expect(wrapper.find('.edit-match-btn').exists()).toBe(false);
  });
});

// Man of the Match voting — moved here from MatchDetails.vue entirely (see
// CLAUDE.md): a star per roster player instead of a dropdown, with a small
// muted vote-count pill next to it. Needs a frozen clock, same reasoning as
// the registration-state badge describe below: the star's own disabled state
// depends on where "now" falls relative to the match's 24h voting window.
describe('MatchesPanel.vue Man of the Match voting', () => {
  afterEach(() => {
    jest.useRealTimers();
  });

  const composedMatch = (overrides = {}) => ({
    ID: 'played-uuid',
    GroupID: 'group-uuid',
    Date: '2026-08-30',
    CreatedAt: '2026-08-30T18:00:00+02:00',
    Teams: [
      {
        ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0,
        Players: [
          { ID: ME, Name: 'me', GoalNumber: 0 },
          { ID: SOMEONE_ELSE, Name: 'marco', GoalNumber: 0 }
        ]
      },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
    ],
    ...overrides
  });

  // Well within 24h of composedMatch()'s default CreatedAt — the "window
  // open" instant every test below starts from unless it says otherwise.
  const withinWindow = () => jest.useFakeTimers().setSystemTime(new Date('2026-08-30T19:00:00+02:00'));

  // The caller's own star still renders (disabled, visibility:hidden via
  // .is-self) rather than being omitted — omitting it would shift every
  // other row's star out of its column, since .player-motm's alignment
  // depends on every row having the same internal structure.
  it('renders a star for every roster player, hiding and disabling the caller\'s own', async () => {
    withinWindow();
    getMatchesDetails.mockResolvedValue([composedMatch()]);
    const wrapper = await mountPanel();

    const stars = wrapper.findAll('.motm-star-btn');
    expect(stars.length).toBe(2);

    const myStar = stars.find(star => star.classes().includes('is-self'));
    const otherStar = stars.find(star => !star.classes().includes('is-self'));
    expect(myStar.attributes('disabled')).toBeDefined();
    expect(otherStar.attributes('aria-label')).toMatch(/marco/i);
  });

  it('shows a muted vote-count pill only for a candidate with at least one vote', async () => {
    withinWindow();
    getMatchesDetails.mockResolvedValue([composedMatch()]);
    getMatchVotes.mockResolvedValue({
      Tally: [{ PlayerID: SOMEONE_ELSE, Name: 'marco', Votes: 3 }],
      MyVoteFor: null
    });
    const wrapper = await mountPanel();

    expect(wrapper.find('.motm-vote-count').text()).toBe('3 votes');
  });

  it('casts a vote when the star is clicked, then reloads the tally', async () => {
    withinWindow();
    getMatchesDetails.mockResolvedValue([composedMatch()]);
    const wrapper = await mountPanel();
    getMatchVotes.mockClear();

    await wrapper.find('.motm-star-btn:not(.is-self)').trigger('click');
    await flushPromises();

    expect(voteForMotm).toHaveBeenCalledWith('played-uuid', SOMEONE_ELSE);
    expect(getMatchVotes).toHaveBeenCalledTimes(1);
  });

  it('removes the vote (toggles off) when clicking the star already voted for', async () => {
    withinWindow();
    getMatchesDetails.mockResolvedValue([composedMatch()]);
    getMatchVotes.mockResolvedValue({
      Tally: [{ PlayerID: SOMEONE_ELSE, Name: 'marco', Votes: 1 }],
      MyVoteFor: SOMEONE_ELSE
    });
    const wrapper = await mountPanel();

    const marcoStar = wrapper.find('.motm-star-btn:not(.is-self)');
    expect(marcoStar.classes()).toContain('is-voted');

    await marcoStar.trigger('click');
    await flushPromises();

    expect(removeMotmVote).toHaveBeenCalledWith('played-uuid');
    expect(voteForMotm).not.toHaveBeenCalled();
  });

  it('disables the star and explains why once the 24h window has passed', async () => {
    // CreatedAt is the anchor for this unscheduled match; "now" set well
    // more than 24h after it. Targets marco's star specifically — the
    // caller's own star is disabled unconditionally (see the self-vote
    // test above) and would pass this assertion for the wrong reason.
    jest.useFakeTimers().setSystemTime(new Date('2026-09-02T09:00:00+02:00'));
    getMatchesDetails.mockResolvedValue([composedMatch({ CreatedAt: '2026-08-01T18:00:00+02:00' })]);
    const wrapper = await mountPanel();

    const star = wrapper.find('.motm-star-btn:not(.is-self)');
    expect(star.attributes('disabled')).toBeDefined();
    expect(star.attributes('title')).toBe('Vote fermé 24h après le match');

    await star.trigger('click');
    await flushPromises();
    expect(voteForMotm).not.toHaveBeenCalled();
  });

  it('offers no star at all before any player has been placed on a team', async () => {
    withinWindow();
    const scheduledEmpty = {
      ID: 'scheduled-uuid',
      GroupID: 'group-uuid',
      Date: '2026-09-06',
      ScheduledAt: '2026-09-06T20:30:00+02:00',
      RegistrationOpensAt: '2026-09-01T12:00:00+02:00',
      MaxPlayers: 16,
      RegistrationCount: 0,
      Teams: [
        { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0, Players: [] },
        { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
      ]
    };
    getMatchesDetails.mockResolvedValue([scheduledEmpty]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.motm-star-btn').exists()).toBe(false);
    expect(getMatchVotes).not.toHaveBeenCalled();
  });
});

// Same hidden-until-composed rule as MatchDetails.vue, applied to the card's
// own team/score row and the "Selected Match Details" preview's team columns —
// an empty "0 vs 0" and two "No players yet" columns are noise for the whole
// sign-up window, where the badge/count and the inline sign-up panel are what
// actually matter.
describe('MatchesPanel.vue team roster visibility on a scheduled match', () => {
  const scheduled = (overrides = {}) => ({
    ID: 'scheduled-uuid',
    GroupID: 'group-uuid',
    Date: '2026-09-06',
    ScheduledAt: '2026-09-06T20:30:00+02:00',
    RegistrationOpensAt: '2026-09-01T12:00:00+02:00',
    MaxPlayers: 16,
    RegistrationCount: 7,
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0, Players: [] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
    ],
    ...overrides
  });

  it('hides the card\'s team/score row and the preview\'s team columns while both teams are empty', async () => {
    getMatchesDetails.mockResolvedValue([scheduled()]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.match-card-horizontal.scheduled .teams-horizontal').exists()).toBe(false);
    expect(wrapper.find('.players-section').exists()).toBe(false);
    expect(wrapper.find('.no-roster-hint').exists()).toBe(true);
  });

  it('shows both once at least one team has a player', async () => {
    const withPlayer = scheduled();
    withPlayer.Teams[0].Players = [{ ID: 'p1', Name: 'marco', GoalNumber: 0 }];
    getMatchesDetails.mockResolvedValue([withPlayer]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.match-card-horizontal.scheduled .teams-horizontal').exists()).toBe(true);
    expect(wrapper.find('.players-section').exists()).toBe(true);
    expect(wrapper.find('.no-roster-hint').exists()).toBe(false);
  });

  it('leaves an ordinary, unscheduled match exactly as it was', async () => {
    const played = {
      ID: 'played-uuid',
      GroupID: 'group-uuid',
      Date: '2026-08-30',
      Teams: [
        { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 3, Players: [] },
        { ID: 'team-b', Name: 'White', Colour: 'white', Score: 2, Players: [] }
      ]
    };
    getMatchesDetails.mockResolvedValue([played]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.teams-horizontal').exists()).toBe(true);
    expect(wrapper.find('.players-section').exists()).toBe(true);
    expect(wrapper.find('.no-roster-hint').exists()).toBe(false);
  });
});

// The card's own registration-state badge — a per-card equivalent of the
// inline preview's badge above, computed straight from each match's own
// scheduling fields (no extra request, unlike isRegistered). Needs a frozen
// clock since the label depends on where "now" falls in the window.
describe("MatchesPanel.vue card's registration state badge", () => {
  afterEach(() => {
    jest.useRealTimers();
  });

  const scheduled = (overrides = {}) => ({
    ID: 'scheduled-uuid',
    GroupID: 'group-uuid',
    Date: '2026-09-06',
    ScheduledAt: '2026-09-06T20:30:00+02:00',
    RegistrationOpensAt: '2026-09-01T12:00:00+02:00',
    MaxPlayers: 16,
    RegistrationCount: 7,
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0, Players: [] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
    ],
    ...overrides
  });

  it('labels it "Sign-ups open" during the window', async () => {
    jest.useFakeTimers().setSystemTime(new Date('2026-09-03T09:00:00+02:00'));
    getMatchesDetails.mockResolvedValue([scheduled()]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-state-badge').text()).toBe('Sign-ups open');
    expect(wrapper.find('.signup-state-badge').classes()).toContain('open');
  });

  it('labels it "Sign-ups not open yet" before the opening instant', async () => {
    jest.useFakeTimers().setSystemTime(new Date('2026-08-30T09:00:00+02:00'));
    getMatchesDetails.mockResolvedValue([scheduled()]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-state-badge').text()).toBe('Sign-ups not open yet');
  });

  it('labels it "Sign-ups closed" once an admin has closed the list', async () => {
    jest.useFakeTimers().setSystemTime(new Date('2026-09-03T09:00:00+02:00'));
    getMatchesDetails.mockResolvedValue([scheduled({ RegistrationsClosedAt: '2026-09-02T18:00:00+02:00' })]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-state-badge').text()).toBe('Sign-ups closed');
  });

  it('labels it "Sign-ups closed" once kick-off has passed, with no admin close at all', async () => {
    jest.useFakeTimers().setSystemTime(new Date('2026-09-07T09:00:00+02:00'));
    getMatchesDetails.mockResolvedValue([scheduled()]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-state-badge').text()).toBe('Sign-ups closed');
  });
});

// The compact/expanded toggle for the "Selected Match Details" preview, once
// a scheduled match's roster is actually composed. Before that (still signing
// up) or for an ordinary unscheduled match, the sign-up chrome *is* the main
// content, so there is no toggle at all and .signup-inline renders exactly as
// it always has. None of these assertions depend on the real clock — the
// toggle's own presence/state is independent of registrationState/the 24h
// MOTM window, unlike the describe blocks above that do mock the clock.
describe('MatchesPanel.vue collapsible team view once composed', () => {
  const composedScheduled = (overrides = {}) => ({
    ID: 'scheduled-uuid',
    GroupID: 'group-uuid',
    Date: '2026-09-06',
    ScheduledAt: '2026-09-06T20:30:00+02:00',
    RegistrationOpensAt: '2026-09-01T12:00:00+02:00',
    MaxPlayers: 16,
    RegistrationCount: 7,
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0, Players: [{ ID: 'p1', Name: 'marco', GoalNumber: 0 }] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
    ],
    ...overrides
  });

  const notYetComposedScheduled = () => ({
    ID: 'scheduled-empty-uuid',
    GroupID: 'group-uuid',
    Date: '2026-09-06',
    ScheduledAt: '2026-09-06T20:30:00+02:00',
    RegistrationOpensAt: '2026-09-01T12:00:00+02:00',
    MaxPlayers: 16,
    RegistrationCount: 7,
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 0, Players: [] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 0, Players: [] }
    ]
  });

  const playedWithPlayers = () => ({
    ID: 'played-uuid',
    GroupID: 'group-uuid',
    Date: '2026-08-30',
    Teams: [
      { ID: 'team-a', Name: 'Black', Colour: 'black', Score: 3, Players: [{ ID: 'p1', Name: 'marco', GoalNumber: 1 }] },
      { ID: 'team-b', Name: 'White', Colour: 'white', Score: 2, Players: [] }
    ]
  });

  it('shows no toggle for a scheduled match whose roster is not yet composed', async () => {
    getMatchesDetails.mockResolvedValue([notYetComposedScheduled()]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-toggle-btn').exists()).toBe(false);
    // Nothing to collapse yet, so the full sign-up chrome is what renders,
    // exactly as before this feature existed.
    expect(wrapper.find('.signup-inline').exists()).toBe(true);
  });

  it('shows no toggle for an ordinary, unscheduled match, composed or not', async () => {
    getMatchesDetails.mockResolvedValue([playedWithPlayers()]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-toggle-btn').exists()).toBe(false);
    expect(wrapper.find('.signup-inline').exists()).toBe(false);
  });

  it('appears and defaults to collapsed once a scheduled match is composed', async () => {
    getMatchesDetails.mockResolvedValue([composedScheduled()]);
    const wrapper = await mountPanel();

    expect(wrapper.find('.signup-toggle-btn').exists()).toBe(true);
    expect(wrapper.find('.signup-toggle-btn').attributes('aria-expanded')).toBe('false');
    expect(wrapper.find('.signup-inline').exists()).toBe(false);
    // The teams themselves are what a composed match leads with instead.
    expect(wrapper.find('.players-section').exists()).toBe(true);
  });

  it('reveals the sign-up sections when clicked, and collapses again on a second click', async () => {
    getMatchesDetails.mockResolvedValue([composedScheduled()]);
    const wrapper = await mountPanel();

    await wrapper.find('.signup-toggle-btn').trigger('click');

    expect(wrapper.find('.signup-inline').exists()).toBe(true);
    expect(wrapper.find('.signup-toggle-btn').text()).toMatch(/hide sign-up details/i);
    expect(wrapper.find('.signup-toggle-btn').attributes('aria-expanded')).toBe('true');

    await wrapper.find('.signup-toggle-btn').trigger('click');

    expect(wrapper.find('.signup-inline').exists()).toBe(false);
    expect(wrapper.find('.signup-toggle-btn').text()).toMatch(/show sign-up details/i);
  });

  it('resets to the collapsed default when a different match is selected', async () => {
    const other = composedScheduled({ ID: 'other-scheduled-uuid' });
    getMatchesDetails.mockResolvedValue([composedScheduled(), other]);
    const wrapper = await mountPanel();

    await wrapper.find('.signup-toggle-btn').trigger('click');
    expect(wrapper.find('.signup-inline').exists()).toBe(true);

    await wrapper.vm.selectMatch(other);
    await flushPromises();

    expect(wrapper.find('.signup-inline').exists()).toBe(false);
    expect(wrapper.find('.signup-toggle-btn').attributes('aria-expanded')).toBe('false');
  });
});

import { mount, flushPromises } from '@vue/test-utils';
import MatchesPanel from '@/components/MatchesPanel.vue';
import {
  getMatchesDetails,
  createMatch,
  getMatchRegistrations,
  registerForMatch,
  unregisterFromMatch,
  getToken
} from '@/services/api';

jest.mock('@/services/api', () => ({
  getMatchesDetails: jest.fn(),
  createMatch: jest.fn(),
  getMatchRegistrations: jest.fn(),
  registerForMatch: jest.fn(),
  unregisterFromMatch: jest.fn(),
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

const mountPanel = async () => {
  const wrapper = mount(MatchesPanel, {
    props: { activeGroupId: 'group-uuid', isAdmin: true, season: '' },
    global: {
      stubs: { 'router-link': true },
      mocks: { $router: { push } }
    }
  });
  // created() kicks off loadMatches(), which now also awaits
  // loadSelectedRegistrations() for whichever match got auto-selected —
  // flushPromises is what reliably drains both, where a fixed number of
  // nextTick()s would be guessing at the chain's depth.
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

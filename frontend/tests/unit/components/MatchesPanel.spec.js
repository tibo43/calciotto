import { mount } from '@vue/test-utils';
import MatchesPanel from '@/components/MatchesPanel.vue';
import { getMatchesDetails, createMatch } from '@/services/api';

jest.mock('@/services/api', () => ({
  getMatchesDetails: jest.fn(),
  createMatch: jest.fn()
}));

// Europe/Paris in September, regardless of the machine running the suite.
const OFFSET_MINUTES = -120;

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
  // created() kicks off loadMatches(); let it settle before touching state.
  await wrapper.vm.$nextTick();
  await wrapper.vm.$nextTick();
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
    expect(wrapper.find('.scheduled-pill').exists()).toBe(true);
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
    expect(wrapper.find('.scheduled-pill').exists()).toBe(false);
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

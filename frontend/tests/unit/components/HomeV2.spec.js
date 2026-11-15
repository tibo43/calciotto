import { mount, flushPromises } from '@vue/test-utils';
import HomeV2 from '@/components/v2/HomeV2.vue';
import { getMatchesDetails, getMatchVotes } from '@/services/api';

jest.mock('@/services/api', () => ({
  getMatchesDetails: jest.fn(),
  getMatchRegistrations: jest.fn(() => Promise.resolve([])),
  getMatchVotes: jest.fn(),
  registerForMatch: jest.fn(),
  unregisterFromMatch: jest.fn(),
  getToken: jest.fn(() => '')
}));

const DAY = 24 * 60 * 60 * 1000;
const dayAgo = (n) => new Date(Date.now() - n * DAY).toISOString().slice(0, 10);
const played = (id, daysAgo) => ({
  ID: id, Date: dayAgo(daysAgo),
  Teams: [{ ID: 'a', Name: 'Black', Score: 1, Players: [{ ID: 'p' }] }, { ID: 'b', Name: 'White', Score: 0, Players: [{ ID: 'q' }] }]
});

async function mountHome(matches, isAdmin = false) {
  getMatchesDetails.mockResolvedValue(matches);
  getMatchVotes.mockResolvedValue(null);
  const wrapper = mount(HomeV2, {
    props: { activeGroupId: 'g', groupName: 'Group', season: '2026-2027', isAdmin },
    global: {
      provide: { v2Context: { confirming: false } },
      stubs: { 'router-link': { template: '<a :data-to="JSON.stringify(to)"><slot /></a>', props: ['to'] } }
    }
  });
  await flushPromises();
  return wrapper;
}

describe('HomeV2.vue with nothing scheduled', () => {
  it('shows the last match once: in the hero, not again at the top of Latest matches', async () => {
    const wrapper = await mountHome([played('last', 3), played('before', 10), played('oldest', 17)]);
    expect(wrapper.find('.hero-last').exists()).toBe(true);
    const listed = wrapper.findAll('.list .row').map((row) => JSON.parse(row.attributes('data-to')).params.id);
    expect(listed).toEqual(['before', 'oldest']);
  });

  it('hides Latest matches when the hero is the only match', async () => {
    const wrapper = await mountHome([played('only', 3)]);
    expect(wrapper.find('.hero-last').exists()).toBe(true);
    expect(wrapper.text()).not.toContain('Latest matches');
  });
});

describe('HomeV2.vue New match button', () => {
  it('floats for an admin, whatever the page shows', async () => {
    for (const matches of [[], [played('last', 3)], [{ ...played('next', -3), ScheduledAt: new Date(Date.now() + 3 * DAY).toISOString(), MaxPlayers: 16 }]]) {
      const admin = await mountHome(matches, true);
      expect(admin.find('.fab').attributes('aria-label')).toBe('New match');
    }
  });

  it('is absent for a member, who keeps the usual texts', async () => {
    const member = await mountHome([], false);
    expect(member.find('.fab').exists()).toBe(false);
    expect(member.text()).toContain('No matches yet');
    expect((await mountHome([played('last', 3)], false)).text()).toContain('No upcoming match yet');
  });
});

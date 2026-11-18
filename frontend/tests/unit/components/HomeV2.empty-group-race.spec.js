import { mount, flushPromises } from '@vue/test-utils';
import HomeV2 from '@/components/v2/HomeV2.vue';
import { getMatchesDetails, getMatchRegistrations, getToken } from '@/services/api';

vi.mock('@/services/api', () => ({
  getMatchesDetails: vi.fn(),
  getMatchRegistrations: vi.fn(() => Promise.resolve([])),
  registerForMatch: vi.fn(),
  unregisterFromMatch: vi.fn(),
  getToken: vi.fn(() => '')
}));

const giovediMatch = {
  ID: 'gio-1',
  Date: '2026-10-01',
  Teams: [
    { ID: 'a', Name: 'Bianco', Score: 2, Players: [{ ID: 'p' }] },
    { ID: 'b', Name: 'Nero', Score: 1, Players: [{ ID: 'q' }] }
  ]
};

function mountHome(props) {
  return mount(HomeV2, {
    props: { activeGroupId: '', groupName: '', season: '', isAdmin: true, ...props },
    global: {
      provide: { v2Context: { confirming: false } },
      stubs: { 'router-link': { template: '<a><slot /></a>', props: ['to'] } }
    }
  });
}

describe('HomeV2.vue empty group vs first-group fallback race', () => {
  beforeEach(() => {
    getMatchesDetails.mockReset();
    getMatchRegistrations.mockResolvedValue([]);
    getToken.mockReturnValue('');
  });

  it('does not call GET /matches/details until an active group id is set', async () => {
    getMatchesDetails.mockResolvedValue([]);
    const wrapper = mountHome({ activeGroupId: '' });
    await flushPromises();
    expect(getMatchesDetails).not.toHaveBeenCalled();
    expect(wrapper.find('.skel').exists()).toBe(true);

    await wrapper.setProps({ activeGroupId: 'test-group-id' });
    await flushPromises();
    expect(getMatchesDetails).toHaveBeenCalledWith('test-group-id', '');
    wrapper.unmount();
  });

  it('must not let a slower unscoped fetch replace the empty group with another group\'s matches', async () => {
    let releaseGiovedi;
    const giovediWait = new Promise((resolve) => {
      releaseGiovedi = resolve;
    });
    getMatchesDetails.mockImplementation((groupId) => {
      if (!groupId) {
        return giovediWait.then(() => [giovediMatch]);
      }
      return Promise.resolve([]);
    });

    const wrapper = mountHome({ activeGroupId: '' });
    await flushPromises();
    expect(getMatchesDetails).not.toHaveBeenCalled();

    await wrapper.setProps({ activeGroupId: 'test-group-id' });
    await flushPromises();
    expect(wrapper.text()).toContain('No matches yet');
    expect(wrapper.text()).toContain('Start the season');

    releaseGiovedi();
    await flushPromises();

    expect(wrapper.text()).toContain('No matches yet');
    expect(wrapper.text()).not.toContain('Bianco');
    expect(wrapper.text()).not.toContain('Nero');
    wrapper.unmount();
  });

  it('ignores a slower response from a previous group after the active group changes', async () => {
    let releaseGiovedi;
    const giovediWait = new Promise((resolve) => {
      releaseGiovedi = resolve;
    });
    getMatchesDetails.mockImplementation((groupId) => {
      if (groupId === 'giovedi-id') {
        return giovediWait.then(() => [giovediMatch]);
      }
      return Promise.resolve([]);
    });

    const wrapper = mountHome({ activeGroupId: 'giovedi-id' });
    await flushPromises();
    await wrapper.setProps({ activeGroupId: 'test-group-id' });
    await flushPromises();
    expect(wrapper.text()).toContain('No matches yet');

    releaseGiovedi();
    await flushPromises();

    expect(wrapper.text()).toContain('No matches yet');
    expect(wrapper.text()).not.toContain('Bianco');
    expect(wrapper.text()).not.toContain('Nero');
    wrapper.unmount();
  });
});

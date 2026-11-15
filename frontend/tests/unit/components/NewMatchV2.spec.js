import { mount, flushPromises } from '@vue/test-utils';
import NewMatchV2 from '@/components/v2/NewMatchV2.vue';
import { createMatch } from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';

jest.mock('@/services/api', () => ({ createMatch: jest.fn() }));
jest.mock('@/services/activeGroup', () => ({ resolveActiveGroup: jest.fn() }));

async function mountPage(role = 'admin') {
  resolveActiveGroup.mockResolvedValue({ groups: [{ id: 'g', role }], activeGroupId: 'g' });
  const router = { replace: jest.fn(), push: jest.fn() };
  const v2Context = {};
  const wrapper = mount(NewMatchV2, { global: { provide: { v2Context }, mocks: { $router: router } } });
  await flushPromises();
  return { wrapper, router, v2Context };
}

// A kick-off safely in the future, whatever day the tests run.
function futureDay() {
  const date = new Date(Date.now() + 10 * 24 * 60 * 60 * 1000);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

beforeEach(() => jest.clearAllMocks());

describe('NewMatchV2.vue', () => {
  it('sends a non-admin home', async () => {
    const { router } = await mountPage('member');
    expect(router.replace).toHaveBeenCalledWith('/');
  });

  it('titles the page and offers a back link home', async () => {
    const { v2Context } = await mountPage();
    expect(v2Context.pageTitle).toBe('New match');
    expect(v2Context.backTo).toBe('/');
  });

  it('shows the reason next to the field and sends nothing', async () => {
    const { wrapper } = await mountPage();
    await wrapper.find('form').trigger('submit');
    expect(wrapper.find('.field-error').text()).toBe('Pick the match day.');
    expect(createMatch).not.toHaveBeenCalled();
  });

  it('creates the match in the active group and opens it', async () => {
    const { wrapper, router } = await mountPage();
    createMatch.mockResolvedValue('new-id');
    await wrapper.find('#new-match-day').setValue(futureDay());
    await wrapper.find('#new-match-time').setValue('20:30');
    await wrapper.find('[role="switch"]').trigger('click');
    await wrapper.find('form').trigger('submit');
    await flushPromises();

    const [matchData, groupId, scheduling] = createMatch.mock.calls[0];
    expect(matchData).toEqual({ Date: futureDay() });
    expect(groupId).toBe('g');
    expect(scheduling).toMatchObject({ maxPlayers: 16, registerCreator: true });
    expect(router.push).toHaveBeenCalledWith({ name: 'MatchView', params: { id: 'new-id' } });
  });

  it("shows the backend's refusal", async () => {
    const { wrapper } = await mountPage();
    createMatch.mockRejectedValue({ response: { data: { error: 'only an admin can create a match' } } });
    await wrapper.find('#new-match-day').setValue(futureDay());
    await wrapper.find('#new-match-time').setValue('20:30');
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(wrapper.text()).toContain('only an admin can create a match');
  });
});

import { shallowMount, flushPromises } from '@vue/test-utils';
import Signup from '@/components/Signup.vue';
import { signup } from '@/services/api';

jest.mock('@/services/api', () => ({
  signup: jest.fn()
}));

const mountPage = (routeQuery = {}) => {
  return shallowMount(Signup, {
    global: {
      mocks: {
        $route: { query: routeQuery },
        $router: { push: jest.fn() }
      }
    }
  });
};

describe('Signup.vue invite code', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('prefills the invite code from a ?invite=CODE query param', () => {
    const wrapper = mountPage({ invite: 'AB2N7TQR' });
    expect(wrapper.vm.inviteCode).toBe('AB2N7TQR');
  });

  it('leaves the invite code empty when no query param is present', () => {
    const wrapper = mountPage();
    expect(wrapper.vm.inviteCode).toBe('');
  });

  it('blocks submit and shows an error when the invite code is empty, without calling the API', async () => {
    const wrapper = mountPage();
    wrapper.vm.name = 'Zzz Test Player';
    wrapper.vm.email = 'zzz@example.com';
    wrapper.vm.password = 's3cret-pass';

    await wrapper.vm.submit();
    await flushPromises();

    expect(wrapper.vm.error).toBe('Invite code is required.');
    expect(signup).not.toHaveBeenCalled();
  });

  it('submits with a non-empty invite code', async () => {
    signup.mockResolvedValue({ id: 'player-uuid' });
    const wrapper = mountPage({ invite: 'AB2N7TQR' });
    wrapper.vm.name = 'Zzz Test Player';
    wrapper.vm.email = 'zzz@example.com';
    wrapper.vm.password = 's3cret-pass';

    await wrapper.vm.submit();
    await flushPromises();

    expect(signup).toHaveBeenCalledWith('Zzz Test Player', 'zzz@example.com', 's3cret-pass', 'AB2N7TQR');
  });
});

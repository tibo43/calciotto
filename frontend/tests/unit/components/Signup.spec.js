import { shallowMount, flushPromises } from '@vue/test-utils';
import Signup from '@/components/Signup.vue';
import { signup, setToken } from '@/services/api';

jest.mock('@/services/api', () => ({
  signup: jest.fn(),
  setToken: jest.fn()
}));

const mountPage = (routeQuery = {}) => {
  const push = jest.fn();
  const wrapper = shallowMount(Signup, {
    global: {
      mocks: {
        $route: { query: routeQuery },
        $router: { push }
      }
    }
  });
  return { wrapper, push };
};

describe('Signup.vue invite code', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('prefills the invite code from a ?invite=CODE query param', () => {
    const { wrapper } = mountPage({ invite: 'AB2N7TQR' });
    expect(wrapper.vm.inviteCode).toBe('AB2N7TQR');
  });

  it('leaves the invite code empty when no query param is present', () => {
    const { wrapper } = mountPage();
    expect(wrapper.vm.inviteCode).toBe('');
  });

  it('blocks submit and shows an error when the invite code is empty, without calling the API', async () => {
    const { wrapper } = mountPage();
    wrapper.vm.name = 'Zzz Test Player';
    wrapper.vm.email = 'zzz@example.com';
    wrapper.vm.password = 's3cret-pass';

    await wrapper.vm.submit();
    await flushPromises();

    expect(wrapper.vm.error).toBe('Invite code is required.');
    expect(signup).not.toHaveBeenCalled();
  });

  it('submits with a non-empty invite code', async () => {
    signup.mockResolvedValue({ id: 'player-uuid', token: 'jwt-token' });
    const { wrapper } = mountPage({ invite: 'AB2N7TQR' });
    wrapper.vm.name = 'Zzz Test Player';
    wrapper.vm.email = 'zzz@example.com';
    wrapper.vm.password = 's3cret-pass';

    await wrapper.vm.submit();
    await flushPromises();

    expect(signup).toHaveBeenCalledWith('Zzz Test Player', 'zzz@example.com', 's3cret-pass', 'AB2N7TQR');
  });
});

// Signup now logs the player straight in (POST /auth/signup returns a usable
// token, same shape as Login) instead of bouncing them to /login to retype
// the password they just entered — mirrors Login.vue's own setToken()-then-
// push('/') pattern exactly.
describe('Signup.vue auto-login', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('stores the token from the signup response and navigates to the home page', async () => {
    signup.mockResolvedValue({ id: 'player-uuid', token: 'jwt-token' });
    const { wrapper, push } = mountPage({ invite: 'AB2N7TQR' });
    wrapper.vm.name = 'Zzz Test Player';
    wrapper.vm.email = 'zzz@example.com';
    wrapper.vm.password = 's3cret-pass';

    await wrapper.vm.submit();
    await flushPromises();

    expect(setToken).toHaveBeenCalledWith('jwt-token');
    expect(push).toHaveBeenCalledWith('/');
  });

  it('does not store a token or navigate when signup fails', async () => {
    signup.mockRejectedValue({ response: { data: { error: 'email already in use' } } });
    const { wrapper, push } = mountPage({ invite: 'AB2N7TQR' });
    wrapper.vm.name = 'Zzz Test Player';
    wrapper.vm.email = 'zzz@example.com';
    wrapper.vm.password = 's3cret-pass';

    await wrapper.vm.submit();
    await flushPromises();

    expect(setToken).not.toHaveBeenCalled();
    expect(push).not.toHaveBeenCalled();
    expect(wrapper.vm.error).toBe('email already in use');
  });
});

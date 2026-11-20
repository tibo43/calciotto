import { mount, flushPromises } from '@vue/test-utils';
import AppToast from '@/components/AppToast.vue';
import LoginV2 from '@/components/v2/LoginV2.vue';
import { login, setToken } from '@/services/api';
import { signInWithPasskey, supportsPasskeys } from '@/services/passkeys';
import { shouldOfferPasskeyLogin } from '@/services/passkeyAccess';

vi.mock('@/services/api', () => ({ login: vi.fn(), setToken: vi.fn() }));
vi.mock('@/services/passkeys', () => ({ signInWithPasskey: vi.fn(), supportsPasskeys: vi.fn(() => true) }));
vi.mock('@/services/passkeyAccess', () => ({ shouldOfferPasskeyLogin: vi.fn(() => true) }));

const PASSKEY_ERROR = 'Could not log in with a passkey. Try again or use your password.';

function mountLogin(query = {}) {
  const push = vi.fn();
  const wrapper = mount(LoginV2, {
    global: {
      mocks: { $route: { query }, $router: { push } },
      stubs: { 'router-link': { props: ['to'], template: '<a :href="to"><slot /></a>' } },
    },
  });
  return { wrapper, push };
}

const toast = (wrapper) => {
  const { message, type } = wrapper.findComponent(AppToast).vm;
  return { message, type };
};
const buttonByText = (wrapper, text) => wrapper.findAll('button').find((b) => b.text().trim() === text);

async function fillAndSubmit(wrapper) {
  await wrapper.find('#email').setValue('thibaut@calciotto.test');
  await wrapper.find('#password').setValue('secret');
  await wrapper.find('form').trigger('submit');
}

beforeEach(() => {
  vi.clearAllMocks();
  supportsPasskeys.mockReturnValue(true);
  shouldOfferPasskeyLogin.mockReturnValue(true);
});

describe('LoginV2.vue', () => {
  it('shows the title, the lead, the passkey button, OR, the form and the two links', () => {
    const { wrapper } = mountLogin();
    expect(wrapper.find('h1').text()).toBe('Log in');
    expect(wrapper.find('.auth-lead').text()).toBe('Find your groups and your matches.');
    expect(wrapper.find('.pk-btn').text()).toBe('Log in with a passkey');
    expect(wrapper.find('.pk-btn').classes()).toEqual(expect.arrayContaining(['btn', 'btn-ghost', 'btn-block']));
    expect(wrapper.find('.or').text()).toBe('or');
    expect(wrapper.find('label[for="email"]').text()).toBe('Email');
    expect(wrapper.find('label[for="password"]').text()).toBe('Password');
    expect(wrapper.find('#email').attributes('autocomplete')).toBe('username webauthn');
    expect(wrapper.find('#password').attributes('autocomplete')).toBe('current-password');
    expect(wrapper.find('form button[type="submit"]').text()).toBe('Log in');
    expect(wrapper.findAll('.auth-links a').map((a) => [a.text(), a.attributes('href')])).toEqual([
      ['Forgot password?', '/forgot-password'],
      ['No account yet? Sign up with an invite code', '/signup'],
    ]);
    expect(wrapper.find('.theme-toggle').exists()).toBe(false);
  });

  it.each([
    ['no WebAuthn', false, true],
    ['no passkey mark', true, false],
  ])('hides the passkey button and OR with %s', (_, supported, offered) => {
    supportsPasskeys.mockReturnValue(supported);
    shouldOfferPasskeyLogin.mockReturnValue(offered);
    const { wrapper } = mountLogin();
    expect(wrapper.find('.pk-btn').exists()).toBe(false);
    expect(wrapper.find('.or').exists()).toBe(false);
    expect(wrapper.find('form').exists()).toBe(true);
  });

  it('logs in with the password, opens the session and goes home', async () => {
    login.mockResolvedValue({ token: 'tok' });
    const { wrapper, push } = mountLogin();
    await fillAndSubmit(wrapper);
    await flushPromises();
    expect(login).toHaveBeenCalledWith('thibaut@calciotto.test', 'secret');
    expect(setToken).toHaveBeenCalledWith('tok');
    expect(push).toHaveBeenCalledWith('/');
  });

  it('puts a refused password in the error toast, with the server reason', async () => {
    login.mockRejectedValue({ response: { data: { error: 'invalid email or password' } } });
    const { wrapper, push } = mountLogin();
    await fillAndSubmit(wrapper);
    await flushPromises();
    expect(toast(wrapper)).toEqual({ message: 'Invalid email or password.', type: 'error' });
    expect(wrapper.find('[role="alert"]').exists()).toBe(false);
    expect(push).not.toHaveBeenCalled();
  });

  it('falls back to the v1 wording when the server gives no reason', async () => {
    login.mockRejectedValue(new Error('network'));
    const { wrapper } = mountLogin();
    await fillAndSubmit(wrapper);
    await flushPromises();
    expect(toast(wrapper)).toEqual({ message: 'Login failed. Please check your credentials.', type: 'error' });
  });

  it('reads Waiting for your device… while the passkey prompt is open, then logs in', async () => {
    let resolve;
    signInWithPasskey.mockReturnValue(new Promise((r) => { resolve = r; }));
    const { wrapper, push } = mountLogin();
    await wrapper.find('.pk-btn').trigger('click');
    expect(wrapper.find('.pk-btn').text()).toBe('Waiting for your device…');
    expect(wrapper.find('.pk-btn').attributes('disabled')).toBeDefined();
    expect(wrapper.find('main').classes()).toContain('is-waiting');
    expect(buttonByText(wrapper, 'Log in').attributes('disabled')).toBeDefined();
    resolve({ token: 'pk-tok' });
    await flushPromises();
    expect(setToken).toHaveBeenCalledWith('pk-tok');
    expect(push).toHaveBeenCalledWith('/');
  });

  it('shows a passkey failure only in the toast, not under the button', async () => {
    signInWithPasskey.mockRejectedValue(new Error('NotAllowedError'));
    const { wrapper, push } = mountLogin();
    await wrapper.find('.pk-btn').trigger('click');
    await flushPromises();
    expect(toast(wrapper)).toEqual({ message: PASSKEY_ERROR, type: 'error' });
    expect(wrapper.find('.toast-message').text()).toContain(PASSKEY_ERROR);
    expect(wrapper.find('.toast-close').exists()).toBe(true);
    expect(wrapper.find('[role="alert"]').exists()).toBe(false);
    expect(wrapper.find('.pk-btn').text()).toBe('Log in with a passkey');
    expect(push).not.toHaveBeenCalled();
  });

  it('keeps Login.vue\'s notices after a reset or an account deletion', () => {
    expect(mountLogin({ reset: 'success' }).wrapper.find('.auth-notice').text())
      .toBe('Your password has been updated. Log in with it below.');
    expect(mountLogin({ accountDeleted: 'success' }).wrapper.find('.auth-notice').text())
      .toBe('Your account has been deleted.');
    expect(mountLogin().wrapper.find('.auth-notice').exists()).toBe(false);
  });
});

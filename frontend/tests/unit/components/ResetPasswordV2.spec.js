import { mount, flushPromises } from '@vue/test-utils';
import AppToast from '@/components/AppToast.vue';
import ResetPasswordV2 from '@/components/v2/ResetPasswordV2.vue';
import { resetPassword } from '@/services/api';

vi.mock('@/services/api', () => ({ resetPassword: vi.fn() }));

const EXPIRED = 'This link has expired. Request a new one.';

function mountPage(query = { token: 'raw-token' }) {
  const push = vi.fn();
  const wrapper = mount(ResetPasswordV2, {
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

async function save(wrapper) {
  await wrapper.find('#password').setValue('new-password');
  await wrapper.find('form').trigger('submit');
  await flushPromises();
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('ResetPasswordV2.vue', () => {
  it('shows the title, the lead, one password field, the button and the link', () => {
    const { wrapper } = mountPage();
    expect(wrapper.find('main').classes()).toEqual(expect.arrayContaining(['page', 'is-auth', 'auth-v2', 'has-toast-room']));
    expect(wrapper.find('h1').text()).toBe('New password');
    expect(wrapper.find('.auth-lead').text()).toBe('Choose a new password for your account.');
    expect(wrapper.findAll('input')).toHaveLength(1);
    expect(wrapper.find('label[for="password"]').text()).toBe('New password');
    expect(wrapper.find('#password').attributes('type')).toBe('password');
    expect(wrapper.find('#password').attributes('autocomplete')).toBe('new-password');
    expect(wrapper.find('form button[type="submit"]').text()).toBe('Save password');
    expect(wrapper.findAll('.auth-links a').map((a) => [a.text(), a.attributes('href')])).toEqual([
      ['Request a new link', '/forgot-password'],
    ]);
  });

  it('sends the token from the link with the new password, then opens the login page with the notice flag', async () => {
    resetPassword.mockResolvedValue({ message: 'password updated' });
    const { wrapper, push } = mountPage();
    await save(wrapper);
    expect(resetPassword).toHaveBeenCalledWith('raw-token', 'new-password');
    expect(push).toHaveBeenCalledWith({ path: '/login', query: { reset: 'success' } });
    expect(toast(wrapper).message).toBe('');
  });

  it('shows "Saving…" and ignores a second submit while the request runs', async () => {
    let resolve;
    resetPassword.mockReturnValue(new Promise((res) => { resolve = res; }));
    const { wrapper } = mountPage();
    await save(wrapper);
    expect(wrapper.find('form button[type="submit"]').text()).toBe('Saving…');
    await wrapper.find('form').trigger('submit');
    expect(resetPassword).toHaveBeenCalledTimes(1);
    resolve({});
    await flushPromises();
  });

  it('says the link has expired when it carries no token, without calling the API', async () => {
    const { wrapper, push } = mountPage({});
    await save(wrapper);
    expect(resetPassword).not.toHaveBeenCalled();
    expect(toast(wrapper)).toEqual({ message: EXPIRED, type: 'error' });
    expect(push).not.toHaveBeenCalled();
  });

  it('says the link has expired when the backend no longer accepts the token', async () => {
    resetPassword.mockRejectedValue({ response: { status: 400, data: { error: 'invalid or expired reset token' } } });
    const { wrapper, push } = mountPage();
    await save(wrapper);
    expect(toast(wrapper)).toEqual({ message: EXPIRED, type: 'error' });
    expect(push).not.toHaveBeenCalled();
  });

  it("puts any other reason in the toast as a sentence", async () => {
    resetPassword.mockRejectedValue({ response: { status: 400, data: { error: 'password must be at least 8 characters long' } } });
    const { wrapper } = mountPage();
    await save(wrapper);
    expect(toast(wrapper)).toEqual({ message: 'Password must be at least 8 characters long.', type: 'error' });
  });

  it('falls back to a generic toast when the backend gives no reason', async () => {
    resetPassword.mockRejectedValue(new Error('Network Error'));
    const { wrapper } = mountPage();
    await save(wrapper);
    expect(toast(wrapper)).toEqual({ message: 'Could not reset the password. Please try again.', type: 'error' });
  });
});

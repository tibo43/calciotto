import { mount, flushPromises } from '@vue/test-utils';
import AppToast from '@/components/AppToast.vue';
import SignupV2 from '@/components/v2/SignupV2.vue';
import { signup, setToken } from '@/services/api';

vi.mock('@/services/api', () => ({ signup: vi.fn(), setToken: vi.fn() }));

function mountSignup(query = {}) {
  const push = vi.fn();
  const wrapper = mount(SignupV2, {
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

async function fill(wrapper, { code } = {}) {
  if (code !== undefined) await wrapper.find('#invite-code').setValue(code);
  await wrapper.find('#name').setValue('Damien Rossi');
  await wrapper.find('#email').setValue('damien@calciotto.test');
  await wrapper.find('#password').setValue('password');
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('SignupV2.vue', () => {
  it('shows the title, the lead, the four fields with the invite code first, the button and the link', () => {
    const { wrapper } = mountSignup();
    expect(wrapper.find('main').classes()).toEqual(expect.arrayContaining(['page', 'is-auth', 'auth-v2', 'has-toast-room']));
    expect(wrapper.find('h1').text()).toBe('Sign up');
    expect(wrapper.find('.auth-lead').text()).toBe('You need an invite code from your group admin.');
    expect(wrapper.findAll('form label').map((label) => [label.attributes('for'), label.text()])).toEqual([
      ['invite-code', 'Invite code'],
      ['name', 'Name'],
      ['email', 'Email'],
      ['password', 'Password'],
    ]);
    const code = wrapper.find('#invite-code');
    expect(code.classes()).toContain('code-input');
    expect(code.attributes('placeholder')).toBe('AB2N7TQR');
    expect(code.attributes('autocapitalize')).toBe('characters');
    expect(wrapper.find('#name').attributes('autocomplete')).toBe('name');
    expect(wrapper.find('#email').attributes('autocomplete')).toBe('email');
    expect(wrapper.find('#password').attributes('autocomplete')).toBe('new-password');
    expect(wrapper.find('form button[type="submit"]').text()).toBe('Sign up');
    expect(wrapper.find('form button[type="submit"]').classes()).toEqual(expect.arrayContaining(['btn', 'btn-block']));
    expect(wrapper.findAll('.auth-links a').map((a) => [a.text(), a.attributes('href')])).toEqual([
      ['Already have an account? Log in', '/login'],
    ]);
    expect(wrapper.find('.theme-toggle').exists()).toBe(false);
  });

  it('asks for the code when there is no invite link', () => {
    const { wrapper } = mountSignup();
    expect(wrapper.find('#invite-code').element.value).toBe('');
    expect(wrapper.find('.hint').text()).toBe('Ask your group admin for it.');
  });

  it('fills the code in from ?invite= and says so while the field holds it', async () => {
    const { wrapper } = mountSignup({ invite: 'A7FFYDMN' });
    expect(wrapper.find('#invite-code').element.value).toBe('A7FFYDMN');
    expect(wrapper.find('.hint').text()).toBe('Filled in from your invite link.');
    await wrapper.find('#invite-code').setValue('OTHERCOD');
    expect(wrapper.find('.hint').text()).toBe('Ask your group admin for it.');
  });

  it.each([[''], ['   ']])('refuses an empty code (%j) in the toast, without calling the API', async (code) => {
    const { wrapper, push } = mountSignup();
    await fill(wrapper, { code });
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(signup).not.toHaveBeenCalled();
    expect(toast(wrapper)).toEqual({ message: 'Invite code is required.', type: 'error' });
    expect(push).not.toHaveBeenCalled();
  });

  it('sends the same fields as before, stores the token and goes home', async () => {
    signup.mockResolvedValue({ id: 'player-uuid', token: 'jwt-token' });
    const { wrapper, push } = mountSignup({ invite: 'A7FFYDMN' });
    await fill(wrapper);
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(signup).toHaveBeenCalledWith('Damien Rossi', 'damien@calciotto.test', 'password', 'A7FFYDMN');
    expect(setToken).toHaveBeenCalledWith('jwt-token');
    expect(push).toHaveBeenCalledWith('/');
    expect(toast(wrapper).message).toBe('');
  });

  it('shows "Signing up…" and ignores a second submit while the request runs', async () => {
    let resolve;
    signup.mockReturnValue(new Promise((res) => { resolve = res; }));
    const { wrapper } = mountSignup({ invite: 'A7FFYDMN' });
    await fill(wrapper);
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    const button = wrapper.find('form button[type="submit"]');
    expect(button.text()).toBe('Signing up…');
    expect(button.attributes('disabled')).toBeDefined();
    await wrapper.find('form').trigger('submit');
    expect(signup).toHaveBeenCalledTimes(1);
    resolve({ token: 'jwt-token' });
    await flushPromises();
  });

  it("puts the backend's reason in the toast as a sentence, and stays on the page", async () => {
    signup.mockRejectedValue({ response: { status: 400, data: { error: 'no group matches this invite code' } } });
    const { wrapper, push } = mountSignup({ invite: 'A7FFYDMN' });
    await fill(wrapper);
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(toast(wrapper)).toEqual({ message: 'No group matches this invite code.', type: 'error' });
    expect(setToken).not.toHaveBeenCalled();
    expect(push).not.toHaveBeenCalled();
    expect(wrapper.find('main [role="alert"]').exists()).toBe(false);
    expect(wrapper.find('form button[type="submit"]').text()).toBe('Sign up');
  });

  it('falls back to a generic toast when the backend gives no reason', async () => {
    signup.mockRejectedValue(new Error('Network Error'));
    const { wrapper } = mountSignup({ invite: 'A7FFYDMN' });
    await fill(wrapper);
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(toast(wrapper)).toEqual({ message: 'Signup failed. Please try again.', type: 'error' });
  });
});

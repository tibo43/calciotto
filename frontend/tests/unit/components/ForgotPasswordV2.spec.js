import { mount, flushPromises } from '@vue/test-utils';
import AppToast from '@/components/AppToast.vue';
import ForgotPasswordV2 from '@/components/v2/ForgotPasswordV2.vue';
import { forgotPassword } from '@/services/api';

vi.mock('@/services/api', () => ({ forgotPassword: vi.fn() }));

const SENT = 'If an account exists for this email, a reset link has been sent.';

function mountPage() {
  return mount(ForgotPasswordV2, {
    global: {
      mocks: { $route: { query: {} }, $router: { push: vi.fn() } },
      stubs: { 'router-link': { props: ['to'], template: '<a :href="to"><slot /></a>' } },
    },
  });
}

const toast = (wrapper) => {
  const { message, type } = wrapper.findComponent(AppToast).vm;
  return { message, type };
};
const button = (wrapper) => wrapper.find('form button[type="submit"]');

async function send(wrapper) {
  await wrapper.find('#email').setValue('thibaut@calciotto.test');
  await wrapper.find('form').trigger('submit');
  await flushPromises();
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('ForgotPasswordV2.vue', () => {
  it('shows the title, the lead, the email field, the button and the link', () => {
    const wrapper = mountPage();
    expect(wrapper.find('main').classes()).toEqual(expect.arrayContaining(['page', 'is-auth', 'auth-v2', 'has-toast-room']));
    expect(wrapper.find('h1').text()).toBe('Forgot password');
    expect(wrapper.find('.auth-lead').text()).toBe("Enter your email. We'll send you a link to set a new password.");
    expect(wrapper.find('label[for="email"]').text()).toBe('Email');
    expect(wrapper.find('#email').attributes('type')).toBe('email');
    expect(wrapper.find('#email').attributes('autocomplete')).toBe('email');
    expect(button(wrapper).text()).toBe('Send the link');
    expect(wrapper.find('.auth-notice').exists()).toBe(false);
    expect(wrapper.findAll('.auth-links a').map((a) => [a.text(), a.attributes('href')])).toEqual([
      ['Back to log in', '/login'],
    ]);
  });

  it("shows the backend's answer as a grey notice above the form, and offers to send again", async () => {
    forgotPassword.mockResolvedValue({ message: 'if an account exists for this email, a reset link has been sent' });
    const wrapper = mountPage();
    await send(wrapper);
    expect(forgotPassword).toHaveBeenCalledWith('thibaut@calciotto.test');
    const notice = wrapper.find('.auth-notice');
    expect(notice.text()).toBe(SENT);
    expect(notice.attributes('role')).toBe('status');
    expect(notice.element.nextElementSibling.tagName).toBe('FORM');
    expect(button(wrapper).text()).toBe('Send again');
    expect(toast(wrapper).message).toBe('');
  });

  it('shows "Sending…" while the request runs, and sends again from the notice', async () => {
    forgotPassword.mockResolvedValue({ message: 'if an account exists for this email, a reset link has been sent' });
    const wrapper = mountPage();
    await send(wrapper);
    let resolve;
    forgotPassword.mockReturnValue(new Promise((res) => { resolve = res; }));
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(button(wrapper).text()).toBe('Sending…');
    expect(button(wrapper).attributes('disabled')).toBeDefined();
    await wrapper.find('form').trigger('submit');
    expect(forgotPassword).toHaveBeenCalledTimes(2);
    resolve({ message: 'if an account exists for this email, a reset link has been sent' });
    await flushPromises();
    expect(button(wrapper).text()).toBe('Send again');
    expect(wrapper.find('.auth-notice').text()).toBe(SENT);
  });

  it("puts a failure in the toast, as a sentence, with no notice", async () => {
    forgotPassword.mockRejectedValue({ response: { status: 503, data: { error: 'password reset is temporarily unavailable' } } });
    const wrapper = mountPage();
    await send(wrapper);
    expect(toast(wrapper)).toEqual({ message: 'Password reset is temporarily unavailable.', type: 'error' });
    expect(wrapper.find('.auth-notice').exists()).toBe(false);
    expect(button(wrapper).text()).toBe('Send the link');
  });

  it('falls back to a generic toast when the backend gives no reason', async () => {
    forgotPassword.mockRejectedValue(new Error('Network Error'));
    const wrapper = mountPage();
    await send(wrapper);
    expect(toast(wrapper)).toEqual({ message: 'Could not send the reset link. Please try again.', type: 'error' });
  });
});

import { mount, flushPromises } from '@vue/test-utils';
import AppToast from '@/components/AppToast.vue';
import PasskeysV2, { formatPasskeyDate } from '@/components/v2/PasskeysV2.vue';
import { getMyPasskeys, removeMyPasskey } from '@/services/api';
import { addPasskey, supportsPasskeys } from '@/services/passkeys';

vi.mock('@/services/api', () => ({
  getMyPasskeys: vi.fn(),
  removeMyPasskey: vi.fn(),
}));

vi.mock('@/services/passkeys', () => ({
  addPasskey: vi.fn(),
  supportsPasskeys: vi.fn(() => true),
}));

const PHONE = { id: 'k1', name: 'My phone', created_at: '2026-11-18T10:00:00Z', last_used_at: '2026-11-19T08:30:00Z' };
const LAPTOP = { id: 'k2', name: 'Laptop', created_at: '2026-11-10T10:00:00Z', last_used_at: null };

const buttonByText = (root, text) => root.findAll('button').find((b) => b.text().trim() === text);
const summary = (wrapper) => wrapper.find('.cc-summary').text();
const toast = (wrapper) => {
  const { message, type } = wrapper.findComponent(AppToast).vm;
  return { message, type };
};

async function mountPasskeys(keys = [PHONE, LAPTOP], { open = true } = {}) {
  getMyPasskeys.mockResolvedValue(keys);
  const wrapper = mount(PasskeysV2);
  await flushPromises();
  if (open) await wrapper.find('.cc-head').trigger('click');
  return wrapper;
}

beforeEach(() => {
  vi.clearAllMocks();
  supportsPasskeys.mockReturnValue(true);
});

describe('formatPasskeyDate', () => {
  it('reads day, short month and year in the local time zone', () => {
    const local = new Date(2026, 10, 18, 23, 30);
    expect(formatPasskeyDate(local.toISOString())).toBe('18 Nov 2026');
    expect(formatPasskeyDate('nope')).toBe('');
  });
});

describe('PasskeysV2.vue', () => {
  it('is a closed card whose summary reads Loading… then the device count', async () => {
    let resolve;
    getMyPasskeys.mockReturnValue(new Promise((r) => { resolve = r; }));
    const wrapper = mount(PasskeysV2);
    expect(wrapper.find('details').exists()).toBe(false);
    expect(wrapper.find('h2').text()).toBe('Passkeys');
    expect(wrapper.find('.cc-head').attributes('aria-expanded')).toBe('false');
    expect(wrapper.find('.cc-title').text()).toBe('Passkeys');
    expect(summary(wrapper)).toBe('Loading…');
    expect(wrapper.find('.pk-row').exists()).toBe(false);
    resolve([PHONE, LAPTOP]);
    await flushPromises();
    expect(summary(wrapper)).toBe('2 devices');
    expect(getMyPasskeys).toHaveBeenCalledTimes(1);
  });

  it('summarises one device and none', async () => {
    expect(summary(await mountPasskeys([PHONE], { open: false }))).toBe('1 device');
    expect(summary(await mountPasskeys([], { open: false }))).toBe('None yet');
  });

  it('shows skeleton rows when opened while loading', async () => {
    getMyPasskeys.mockReturnValue(new Promise(() => {}));
    const wrapper = mount(PasskeysV2);
    await wrapper.find('.cc-head').trigger('click');
    expect(wrapper.findAll('.pk-skel')).toHaveLength(2);
    expect(getMyPasskeys).toHaveBeenCalledTimes(1);
  });

  it('shows one row per passkey with one date: last used, else added', async () => {
    const wrapper = await mountPasskeys();
    const rows = wrapper.findAll('.pk-row');
    expect(rows.map((row) => row.find('.pk-name').text())).toEqual(['My phone', 'Laptop']);
    expect(rows[0].find('.pk-meta').text()).toBe(`Last used ${formatPasskeyDate(PHONE.last_used_at)}`);
    expect(rows[1].find('.pk-meta').text()).toBe(`Added ${formatPasskeyDate(LAPTOP.created_at)}`);
    expect(rows[0].find('.pk-trash').attributes('aria-label')).toBe('Remove My phone');
    expect(wrapper.find('.pk-empty').exists()).toBe(false);
    expect(wrapper.find('.pk-add').text()).toBe('Add a passkey');
  });

  it('explains passkeys above the add row when there is none', async () => {
    const wrapper = await mountPasskeys([]);
    expect(wrapper.find('.pk-empty').text()).toBe('Sign in with Face ID, your fingerprint or your screen lock.');
    expect(wrapper.find('.pk-add').exists()).toBe(true);
  });

  it('retries on open, then offers Try again when the list fails to load', async () => {
    getMyPasskeys.mockRejectedValue(new Error('down'));
    const wrapper = mount(PasskeysV2);
    await flushPromises();
    expect(summary(wrapper)).toBe("Couldn't load");
    await wrapper.find('.cc-head').trigger('click');
    await flushPromises();
    expect(getMyPasskeys).toHaveBeenCalledTimes(2);
    expect(wrapper.find('[role="alert"]').text()).toBe('Could not load your passkeys.');
    getMyPasskeys.mockResolvedValueOnce([PHONE]);
    await buttonByText(wrapper, 'Try again').trigger('click');
    await flushPromises();
    expect(wrapper.findAll('.pk-row')).toHaveLength(1);
    expect(summary(wrapper)).toBe('1 device');
  });

  it('asks before removing, and confirms with a success toast', async () => {
    removeMyPasskey.mockResolvedValue();
    const wrapper = await mountPasskeys();
    await wrapper.findAll('.pk-trash')[0].trigger('click');
    expect(wrapper.emitted('confirming').at(-1)).toEqual([true]);
    const sheet = wrapper.find('.confirm-sheet');
    expect(sheet.find('h3').text()).toBe('Remove this passkey?');
    expect(sheet.text()).toContain("You won't be able to sign in with it anymore.");
    expect(wrapper.findAll('.pk-row')[0].find('.confirm-pop').exists()).toBe(true);
    await sheet.find('.btn-danger').trigger('click');
    await flushPromises();
    expect(removeMyPasskey).toHaveBeenCalledWith('k1');
    expect(wrapper.findAll('.pk-row')).toHaveLength(1);
    expect(summary(wrapper)).toBe('1 device');
    expect(toast(wrapper)).toEqual({ message: 'Passkey removed.', type: 'success' });
    expect(wrapper.find('.toast-message').text()).toContain('Passkey removed.');
    expect(wrapper.find('.pk-feedback').exists()).toBe(false);
    expect(wrapper.find('.confirm-sheet').exists()).toBe(false);
    expect(wrapper.emitted('confirming').at(-1)).toEqual([false]);
  });

  it('Cancel keeps the passkey', async () => {
    const wrapper = await mountPasskeys();
    await wrapper.findAll('.pk-trash')[1].trigger('click');
    await buttonByText(wrapper.find('.confirm-pop'), 'Cancel').trigger('click');
    expect(wrapper.find('.confirm-pop').exists()).toBe(false);
    expect(removeMyPasskey).not.toHaveBeenCalled();
  });

  it('shows a failed removal in an error toast', async () => {
    removeMyPasskey.mockRejectedValue(new Error('no'));
    const wrapper = await mountPasskeys();
    await wrapper.findAll('.pk-trash')[0].trigger('click');
    await wrapper.find('.confirm-sheet .btn-danger').trigger('click');
    await flushPromises();
    expect(wrapper.findAll('.pk-row')).toHaveLength(2);
    expect(toast(wrapper)).toEqual({ message: 'Could not remove the passkey.', type: 'error' });
    expect(wrapper.find('.cc-panel [role="alert"]').exists()).toBe(false);
  });

  it('opens the add form in place and adds a passkey', async () => {
    let resolve;
    addPasskey.mockReturnValue(new Promise((r) => { resolve = r; }));
    const wrapper = await mountPasskeys([]);
    await wrapper.find('.pk-add').trigger('click');
    const form = wrapper.find('form.pk-form');
    expect(form.exists()).toBe(true);
    expect(wrapper.find('.pk-add').exists()).toBe(false);
    expect(form.find('label[for="passkey-name"]').text()).toBe('Name (optional)');
    expect(form.find('#passkey-name').attributes('placeholder')).toBe('My phone');
    expect(form.find('label[for="passkey-password"]').text()).toBe('Current password');
    expect(buttonByText(form, 'Create passkey').attributes('disabled')).toBeDefined();
    await form.find('#passkey-name').setValue('Work laptop');
    await form.find('#passkey-password').setValue('secret');
    await form.trigger('submit');
    expect(addPasskey).toHaveBeenCalledWith('Work laptop', 'secret');
    expect(buttonByText(wrapper, 'Waiting for your device…')).toBeTruthy();
    resolve({ id: 'k3', name: 'Work laptop', created_at: '2026-11-19T09:00:00Z', last_used_at: null });
    await flushPromises();
    expect(wrapper.find('form').exists()).toBe(false);
    expect(wrapper.find('.pk-name').text()).toBe('Work laptop');
    expect(toast(wrapper)).toEqual({ message: 'Passkey added.', type: 'success' });
    expect(summary(wrapper)).toBe('1 device');
  });

  it("shows the server's refusal in an error toast and keeps the form open", async () => {
    addPasskey.mockRejectedValue({ response: { data: { error: 'Wrong password' } } });
    const wrapper = await mountPasskeys([]);
    await wrapper.find('.pk-add').trigger('click');
    await wrapper.find('#passkey-password').setValue('bad');
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(toast(wrapper)).toEqual({ message: 'Wrong password', type: 'error' });
    expect(wrapper.find('.cc-panel [role="alert"]').exists()).toBe(false);
    expect(wrapper.find('form').exists()).toBe(true);
    expect(wrapper.find('#passkey-password').element.value).toBe('');
  });

  it('says it could not add the passkey when the device prompt is cancelled', async () => {
    addPasskey.mockRejectedValue(Object.assign(new Error('cancelled'), { name: 'NotAllowedError' }));
    const wrapper = await mountPasskeys([]);
    await wrapper.find('.pk-add').trigger('click');
    await wrapper.find('#passkey-password').setValue('secret');
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(toast(wrapper)).toEqual({ message: 'Could not add the passkey.', type: 'error' });
    expect(wrapper.findAll('.pk-row')).toHaveLength(0);
  });

  it('Cancel, or folding the card, closes the form and clears it', async () => {
    const wrapper = await mountPasskeys([]);
    await wrapper.find('.pk-add').trigger('click');
    await wrapper.find('#passkey-name').setValue('x');
    await buttonByText(wrapper.find('form'), 'Cancel').trigger('click');
    expect(wrapper.find('form').exists()).toBe(false);
    await wrapper.find('.pk-add').trigger('click');
    expect(wrapper.find('#passkey-name').element.value).toBe('');
    await wrapper.find('#passkey-name').setValue('y');
    await wrapper.find('.cc-head').trigger('click');
    await wrapper.find('.cc-head').trigger('click');
    expect(wrapper.find('form').exists()).toBe(false);
  });

  it('keeps the list but replaces the add row in a browser without WebAuthn', async () => {
    supportsPasskeys.mockReturnValue(false);
    const wrapper = await mountPasskeys();
    expect(wrapper.findAll('.pk-row')).toHaveLength(2);
    expect(wrapper.find('.pk-add').exists()).toBe(false);
    expect(wrapper.find('.pk-unsupported').text()).toBe('Open Agoran in a browser that supports passkeys to add one.');
  });
});

describe('CollapsibleCardV2 wiring', () => {
  it('links the header to its panel', async () => {
    const wrapper = await mountPasskeys([PHONE], { open: false });
    const head = wrapper.find('.cc-head');
    const panel = wrapper.find(`#${head.attributes('aria-controls')}`);
    expect(panel.exists()).toBe(true);
    expect(panel.attributes('aria-labelledby')).toBe(head.attributes('id'));
    expect(panel.attributes('hidden')).toBeDefined();
    await head.trigger('click');
    expect(head.attributes('aria-expanded')).toBe('true');
    expect(panel.find('.cc-rule').exists()).toBe(true);
  });
});

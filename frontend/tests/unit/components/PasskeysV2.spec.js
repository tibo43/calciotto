import { mount, flushPromises } from '@vue/test-utils';
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

async function mountPasskeys(keys = [PHONE, LAPTOP]) {
  getMyPasskeys.mockResolvedValue(keys);
  const wrapper = mount(PasskeysV2);
  await flushPromises();
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
  it('is an open section that loads its list on display, with skeleton rows meanwhile', async () => {
    let resolve;
    getMyPasskeys.mockReturnValue(new Promise((r) => { resolve = r; }));
    const wrapper = mount(PasskeysV2);
    expect(wrapper.find('details').exists()).toBe(false);
    expect(wrapper.find('h2').text()).toBe('Passkeys');
    expect(wrapper.text()).toContain('Sign in with Face ID, your fingerprint or your screen lock.');
    expect(getMyPasskeys).toHaveBeenCalledTimes(1);
    expect(wrapper.findAll('.pk-skel')).toHaveLength(2);
    expect(wrapper.text()).not.toContain('Loading');
    resolve([PHONE]);
    await flushPromises();
    expect(wrapper.findAll('.pk-skel')).toHaveLength(0);
    expect(wrapper.findAll('.pk-row')).toHaveLength(1);
  });

  it('shows one row per passkey, with Last used only once it has been used', async () => {
    const wrapper = await mountPasskeys();
    const rows = wrapper.findAll('.pk-row');
    expect(rows.map((row) => row.find('.pk-name').text())).toEqual(['My phone', 'Laptop']);
    expect(rows[0].find('.pk-meta').text()).toBe(`Added ${formatPasskeyDate(PHONE.created_at)} · Last used ${formatPasskeyDate(PHONE.last_used_at)}`);
    expect(rows[1].find('.pk-meta').text()).toBe(`Added ${formatPasskeyDate(LAPTOP.created_at)}`);
    expect(rows[0].find('.pk-remove button').attributes('aria-label')).toBe('Remove My phone');
    // Separate pieces so a phone can stack them and drop the separator.
    expect(rows[0].findAll('.pk-date').map((d) => d.text())).toEqual([
      `Added ${formatPasskeyDate(PHONE.created_at)}`, `Last used ${formatPasskeyDate(PHONE.last_used_at)}`,
    ]);
    expect(rows[0].findAll('.pk-sep')).toHaveLength(1);
    expect(rows[1].findAll('.pk-date')).toHaveLength(1);
    expect(rows[1].find('.pk-sep').exists()).toBe(false);
  });

  it('says so when there is no passkey', async () => {
    const wrapper = await mountPasskeys([]);
    expect(wrapper.find('.pk-empty').text()).toBe('No passkeys yet.');
    expect(buttonByText(wrapper, 'Add a passkey')).toBeTruthy();
  });

  it('offers Try again when the list fails to load', async () => {
    getMyPasskeys.mockRejectedValueOnce(new Error('down'));
    const wrapper = mount(PasskeysV2);
    await flushPromises();
    expect(wrapper.find('[role="alert"]').text()).toBe('Could not load your passkeys.');
    getMyPasskeys.mockResolvedValueOnce([PHONE]);
    await buttonByText(wrapper, 'Try again').trigger('click');
    await flushPromises();
    expect(wrapper.findAll('.pk-row')).toHaveLength(1);
  });

  it('asks before removing, and confirms with the removed notice', async () => {
    removeMyPasskey.mockResolvedValue();
    const wrapper = await mountPasskeys();
    await wrapper.findAll('.pk-remove > button')[0].trigger('click');
    expect(wrapper.emitted('confirming').at(-1)).toEqual([true]);
    const sheet = wrapper.find('.confirm-sheet');
    expect(sheet.find('h3').text()).toBe('Remove this passkey?');
    expect(sheet.text()).toContain("You won't be able to sign in with it anymore.");
    expect(wrapper.findAll('.pk-row')[0].find('.confirm-pop').exists()).toBe(true);
    await sheet.find('.btn-danger').trigger('click');
    await flushPromises();
    expect(removeMyPasskey).toHaveBeenCalledWith('k1');
    expect(wrapper.findAll('.pk-row')).toHaveLength(1);
    expect(wrapper.find('.pk-feedback').text()).toBe('Passkey removed.');
    expect(wrapper.find('.confirm-sheet').exists()).toBe(false);
    expect(wrapper.emitted('confirming').at(-1)).toEqual([false]);
  });

  it('Cancel keeps the passkey', async () => {
    const wrapper = await mountPasskeys();
    await wrapper.findAll('.pk-remove > button')[1].trigger('click');
    await buttonByText(wrapper.find('.confirm-pop'), 'Cancel').trigger('click');
    expect(wrapper.find('.confirm-pop').exists()).toBe(false);
    expect(removeMyPasskey).not.toHaveBeenCalled();
  });

  it('shows a failed removal', async () => {
    removeMyPasskey.mockRejectedValue(new Error('no'));
    const wrapper = await mountPasskeys();
    await wrapper.findAll('.pk-remove > button')[0].trigger('click');
    await wrapper.find('.confirm-sheet .btn-danger').trigger('click');
    await flushPromises();
    expect(wrapper.findAll('.pk-row')).toHaveLength(2);
    expect(wrapper.find('[role="alert"]').text()).toBe('Could not remove the passkey. Try again.');
  });

  it('opens the add form in place and adds a passkey', async () => {
    let resolve;
    addPasskey.mockReturnValue(new Promise((r) => { resolve = r; }));
    const wrapper = await mountPasskeys([]);
    await buttonByText(wrapper, 'Add a passkey').trigger('click');
    const form = wrapper.find('form.pk-form');
    expect(form.exists()).toBe(true);
    expect(buttonByText(wrapper, 'Add a passkey')).toBeUndefined();
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
    expect(wrapper.find('.pk-feedback').text()).toBe('Passkey added.');
  });

  it("shows the server's refusal and keeps the form open", async () => {
    addPasskey.mockRejectedValue({ response: { data: { error: 'Wrong password' } } });
    const wrapper = await mountPasskeys([]);
    await buttonByText(wrapper, 'Add a passkey').trigger('click');
    await wrapper.find('#passkey-password').setValue('bad');
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(wrapper.find('[role="alert"]').text()).toBe('Wrong password');
    expect(wrapper.find('form').exists()).toBe(true);
    expect(wrapper.find('#passkey-password').element.value).toBe('');
  });

  it('Cancel closes the form and clears it', async () => {
    const wrapper = await mountPasskeys([]);
    await buttonByText(wrapper, 'Add a passkey').trigger('click');
    await wrapper.find('#passkey-name').setValue('x');
    await buttonByText(wrapper.find('form'), 'Cancel').trigger('click');
    expect(wrapper.find('form').exists()).toBe(false);
    await buttonByText(wrapper, 'Add a passkey').trigger('click');
    expect(wrapper.find('#passkey-name').element.value).toBe('');
  });

  it('keeps the list but replaces the button in a browser without WebAuthn', async () => {
    supportsPasskeys.mockReturnValue(false);
    const wrapper = await mountPasskeys();
    expect(wrapper.findAll('.pk-row')).toHaveLength(2);
    expect(buttonByText(wrapper, 'Add a passkey')).toBeUndefined();
    expect(wrapper.find('.pk-unsupported').text()).toBe('Open Calciotto in a browser that supports passkeys to add one.');
  });
});

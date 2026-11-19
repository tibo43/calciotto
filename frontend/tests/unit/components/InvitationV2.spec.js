import { mount, flushPromises } from '@vue/test-utils';
import InvitationV2 from '@/components/v2/InvitationV2.vue';
import { rotateGroupInvitation } from '@/services/api';

vi.mock('@/services/api', () => ({
  rotateGroupInvitation: vi.fn(),
}));

function mountInvitation(props = {}) {
  return mount(InvitationV2, {
    props: { groupId: 'group-1', code: 'ABC23XYZ', shareText: 'Join us', ...props },
    global: { stubs: { ShareButton: { template: '<button class="share-stub">{{ label }}</button>', props: ['label', 'text', 'icon'] } } },
  });
}

async function mountOpen(props = {}) {
  const wrapper = mountInvitation(props);
  await wrapper.find('.cc-head').trigger('click');
  return wrapper;
}

const tile = (wrapper, label) => wrapper.findAll('.invite-tiles button').find((b) => b.text().trim() === label);

const buttonByText = (wrapper, text) => wrapper.findAll('button').find((b) => b.text().trim() === text);

beforeEach(() => {
  vi.clearAllMocks();
  vi.useRealTimers();
});

describe('InvitationV2.vue', () => {
  it('is a closed card whose header summarises the code', async () => {
    const wrapper = mountInvitation();
    expect(wrapper.find('details').exists()).toBe(false);
    expect(wrapper.find('h2').text()).toBe('Invitation');
    const head = wrapper.find('.cc-head');
    expect(head.attributes('aria-expanded')).toBe('false');
    expect(head.text()).toContain('Invite players');
    expect(wrapper.find('.cc-summary').text()).toBe('Code active');
    expect(wrapper.find('.invite-code').exists()).toBe(false);
    const panel = wrapper.find(`#${head.attributes('aria-controls')}`);
    expect(panel.attributes('hidden')).toBeDefined();
    await head.trigger('click');
    expect(head.attributes('aria-expanded')).toBe('true');
    expect(panel.attributes('hidden')).toBeUndefined();
  });

  it('open, shows the code and the four tiles', async () => {
    const wrapper = await mountOpen();
    expect(wrapper.find('.invite-code').text()).toBe('ABC23XYZ');
    expect(wrapper.text()).toContain('Works until you renew it');
    expect(wrapper.findAll('.invite-tiles button').map((b) => b.text())).toEqual(['Share', 'Copy', 'Renew']);
    expect(wrapper.find('.btn-base').exists()).toBe(false);
  });

  it('without a code offers only Make a code, and makes one straight away', async () => {
    rotateGroupInvitation.mockResolvedValue({ invite_code: 'NEWCODE1' });
    const wrapper = mountInvitation({ code: '' });
    expect(wrapper.find('.cc-summary').text()).toBe('No active code');
    await wrapper.find('.cc-head').trigger('click');
    expect(wrapper.text()).toContain('Make a code to let new players join.');
    expect(wrapper.text()).not.toContain('—');
    expect(wrapper.find('.invite-tiles').exists()).toBe(false);
    await buttonByText(wrapper, 'Make a code').trigger('click');
    await flushPromises();
    expect(rotateGroupInvitation).toHaveBeenCalledWith('group-1');
    expect(wrapper.emitted('updated')[0]).toEqual([{ invite_code: 'NEWCODE1' }]);
  });

  it('copies the code and reads Copied for two seconds', async () => {
    vi.useFakeTimers();
    const writeText = vi.fn().mockResolvedValue();
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
    const wrapper = await mountOpen();
    await tile(wrapper, 'Copy').trigger('click');
    await flushPromises();
    expect(writeText).toHaveBeenCalledWith('ABC23XYZ');
    expect(tile(wrapper, 'Copied')).toBeTruthy();
    vi.advanceTimersByTime(1999);
    await wrapper.vm.$nextTick();
    expect(tile(wrapper, 'Copied')).toBeTruthy();
    vi.advanceTimersByTime(1);
    await wrapper.vm.$nextTick();
    expect(tile(wrapper, 'Copy')).toBeTruthy();
  });

  it('shows a copy failure inside the card', async () => {
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: vi.fn().mockRejectedValue(new Error('no')) }, configurable: true });
    const wrapper = await mountOpen();
    await tile(wrapper, 'Copy').trigger('click');
    await flushPromises();
    expect(wrapper.find('.cc-panel [role="alert"]').text()).toContain("Couldn't copy");
  });

  it('asks before renewing the code, and Cancel changes nothing', async () => {
    const wrapper = await mountOpen();
    await tile(wrapper, 'Renew').trigger('click');
    expect(wrapper.emitted('confirming').at(-1)).toEqual([true]);
    const sheet = wrapper.find('.confirm-sheet');
    expect(sheet.find('h3').text()).toBe('Renew the code?');
    expect(sheet.text()).toContain('The old code and invite link stop working.');
    expect(wrapper.find('.confirm-pop').exists()).toBe(true);
    await sheet.findAll('button').find((b) => b.text() === 'Cancel').trigger('click');
    expect(wrapper.find('.confirm-sheet').exists()).toBe(false);
    expect(wrapper.emitted('confirming').at(-1)).toEqual([false]);
    expect(rotateGroupInvitation).not.toHaveBeenCalled();
  });

  it('renews the code once confirmed', async () => {
    rotateGroupInvitation.mockResolvedValue({ invite_code: 'NEWCODE1' });
    const wrapper = await mountOpen();
    await tile(wrapper, 'Renew').trigger('click');
    const confirm = wrapper.find('.confirm-sheet').findAll('button').find((b) => b.text() === 'Renew');
    expect(confirm.classes()).not.toContain('btn-danger');
    await confirm.trigger('click');
    await flushPromises();
    expect(rotateGroupInvitation).toHaveBeenCalledWith('group-1');
    expect(wrapper.emitted('updated')[0]).toEqual([{ invite_code: 'NEWCODE1' }]);
    expect(wrapper.find('.confirm-sheet').exists()).toBe(false);
  });

  it('offers no way to revoke the code in v2', async () => {
    const wrapper = await mountOpen();
    expect(wrapper.text()).not.toMatch(/revoke/i);
    expect(wrapper.find('.btn-danger').exists()).toBe(false);
    await tile(wrapper, 'Renew').trigger('click');
    expect(wrapper.find('.btn-danger').exists()).toBe(false);
  });

  it('shows a failed update inside the card and closes the confirmation', async () => {
    rotateGroupInvitation.mockRejectedValue(new Error('boom'));
    const wrapper = await mountOpen();
    await tile(wrapper, 'Renew').trigger('click');
    await wrapper.find('.confirm-sheet').findAll('button').find((b) => b.text() === 'Renew').trigger('click');
    await flushPromises();
    expect(wrapper.find('.invite-error').text()).toBe('Could not update the invitation.');
    expect(wrapper.find('.confirm-sheet').exists()).toBe(false);
    expect(wrapper.emitted('updated')).toBeUndefined();
  });

  it('drops a late answer for a group the admin already left', async () => {
    let resolve;
    rotateGroupInvitation.mockReturnValue(new Promise((r) => { resolve = r; }));
    const wrapper = await mountOpen({ code: '' });
    await buttonByText(wrapper, 'Make a code').trigger('click');
    await wrapper.setProps({ groupId: 'group-2' });
    resolve({ invite_code: 'LATE' });
    await flushPromises();
    expect(wrapper.emitted('updated')).toBeUndefined();
  });
});

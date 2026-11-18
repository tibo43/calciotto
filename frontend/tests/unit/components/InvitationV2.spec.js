import { mount, flushPromises } from '@vue/test-utils';
import InvitationV2 from '@/components/v2/InvitationV2.vue';
import { rotateGroupInvitation, revokeGroupInvitation } from '@/services/api';

vi.mock('@/services/api', () => ({
  rotateGroupInvitation: vi.fn(),
  revokeGroupInvitation: vi.fn(),
}));

function mountInvitation(props = {}) {
  return mount(InvitationV2, {
    props: { groupId: 'group-1', code: 'ABC23XYZ', shareText: 'Join us', ...props },
    global: { stubs: { ShareButton: { template: '<button class="share-stub">{{ label }}</button>', props: ['label', 'text'] } } },
  });
}

const buttonByText = (wrapper, text) => wrapper.findAll('button').find((b) => b.text().trim() === text);

beforeEach(() => {
  vi.clearAllMocks();
  vi.useRealTimers();
});

describe('InvitationV2.vue', () => {
  it('shows the code card with Share invite, Copy and the two text actions', () => {
    const wrapper = mountInvitation();
    expect(wrapper.find('details').exists()).toBe(false);
    expect(wrapper.find('h2').text()).toBe('Invitation');
    expect(wrapper.find('.invite-code').text()).toBe('ABC23XYZ');
    expect(wrapper.text()).toContain('Invite code');
    expect(wrapper.text()).toContain('Works until you make a new one or revoke it');
    expect(wrapper.find('.share-stub').text()).toBe('Share invite');
    expect(buttonByText(wrapper, 'Copy')).toBeTruthy();
    expect(wrapper.findAll('.invite-link').map((b) => b.text())).toEqual(['↻ New code', 'Revoke']);
    expect(wrapper.find('.btn-base').exists()).toBe(false);
  });

  it('without a code offers only Make a code, and makes one straight away', async () => {
    rotateGroupInvitation.mockResolvedValue({ invite_code: 'NEWCODE1' });
    const wrapper = mountInvitation({ code: '' });
    expect(wrapper.text()).toContain('No active invite');
    expect(wrapper.text()).toContain('Make a code to invite players.');
    expect(wrapper.text()).not.toContain('—');
    expect(buttonByText(wrapper, 'Copy')).toBeUndefined();
    expect(wrapper.find('.invite-links').exists()).toBe(false);
    await buttonByText(wrapper, 'Make a code').trigger('click');
    await flushPromises();
    expect(rotateGroupInvitation).toHaveBeenCalledWith('group-1');
    expect(wrapper.emitted('updated')[0]).toEqual([{ invite_code: 'NEWCODE1' }]);
  });

  it('copies the code and reads Copied for two seconds', async () => {
    vi.useFakeTimers();
    const writeText = vi.fn().mockResolvedValue();
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
    const wrapper = mountInvitation();
    await buttonByText(wrapper, 'Copy').trigger('click');
    await flushPromises();
    expect(writeText).toHaveBeenCalledWith('ABC23XYZ');
    expect(buttonByText(wrapper, 'Copied')).toBeTruthy();
    vi.advanceTimersByTime(1999);
    await wrapper.vm.$nextTick();
    expect(buttonByText(wrapper, 'Copied')).toBeTruthy();
    vi.advanceTimersByTime(1);
    await wrapper.vm.$nextTick();
    expect(buttonByText(wrapper, 'Copy')).toBeTruthy();
  });

  it('shows a copy failure inside the card', async () => {
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: vi.fn().mockRejectedValue(new Error('no')) }, configurable: true });
    const wrapper = mountInvitation();
    await buttonByText(wrapper, 'Copy').trigger('click');
    await flushPromises();
    expect(wrapper.find('.invite-card [role="alert"]').text()).toContain("Couldn't copy");
  });

  it('asks before making a new code, and Cancel changes nothing', async () => {
    const wrapper = mountInvitation();
    await wrapper.findAll('.invite-link')[0].trigger('click');
    expect(wrapper.emitted('confirming').at(-1)).toEqual([true]);
    const sheet = wrapper.find('.confirm-sheet');
    expect(sheet.find('h3').text()).toBe('Make a new code?');
    expect(sheet.text()).toContain('The current code stops working.');
    expect(wrapper.find('.confirm-pop').exists()).toBe(true);
    await sheet.findAll('button').find((b) => b.text() === 'Cancel').trigger('click');
    expect(wrapper.find('.confirm-sheet').exists()).toBe(false);
    expect(wrapper.emitted('confirming').at(-1)).toEqual([false]);
    expect(rotateGroupInvitation).not.toHaveBeenCalled();
  });

  it('makes a new code once confirmed', async () => {
    rotateGroupInvitation.mockResolvedValue({ invite_code: 'NEWCODE1' });
    const wrapper = mountInvitation();
    await wrapper.findAll('.invite-link')[0].trigger('click');
    const confirm = wrapper.find('.confirm-sheet').findAll('button').find((b) => b.text() === 'New code');
    expect(confirm.classes()).not.toContain('btn-danger');
    await confirm.trigger('click');
    await flushPromises();
    expect(rotateGroupInvitation).toHaveBeenCalledWith('group-1');
    expect(wrapper.emitted('updated')[0]).toEqual([{ invite_code: 'NEWCODE1' }]);
    expect(wrapper.find('.confirm-sheet').exists()).toBe(false);
  });

  it('revokes once confirmed with a danger button', async () => {
    revokeGroupInvitation.mockResolvedValue({ invite_code: '' });
    const wrapper = mountInvitation();
    await wrapper.findAll('.invite-link')[1].trigger('click');
    const pop = wrapper.find('.confirm-pop');
    expect(pop.find('h3').text()).toBe('Revoke the invite?');
    expect(pop.text()).toContain('Nobody can join with this code anymore.');
    const confirm = pop.findAll('button').find((b) => b.text() === 'Revoke');
    expect(confirm.classes()).toContain('btn-danger');
    await confirm.trigger('click');
    await flushPromises();
    expect(revokeGroupInvitation).toHaveBeenCalledWith('group-1');
    expect(wrapper.emitted('updated')[0]).toEqual([{ invite_code: '' }]);
  });

  it('shows a failed update inside the card and closes the confirmation', async () => {
    revokeGroupInvitation.mockRejectedValue(new Error('boom'));
    const wrapper = mountInvitation();
    await wrapper.findAll('.invite-link')[1].trigger('click');
    await wrapper.find('.confirm-sheet .btn-danger').trigger('click');
    await flushPromises();
    expect(wrapper.find('.invite-card .scope').text()).toBe('Could not update the invitation.');
    expect(wrapper.find('.confirm-sheet').exists()).toBe(false);
    expect(wrapper.emitted('updated')).toBeUndefined();
  });

  it('drops a late answer for a group the admin already left', async () => {
    let resolve;
    rotateGroupInvitation.mockReturnValue(new Promise((r) => { resolve = r; }));
    const wrapper = mountInvitation({ code: '' });
    await buttonByText(wrapper, 'Make a code').trigger('click');
    await wrapper.setProps({ groupId: 'group-2' });
    resolve({ invite_code: 'LATE' });
    await flushPromises();
    expect(wrapper.emitted('updated')).toBeUndefined();
  });
});

import { shallowMount, flushPromises } from '@vue/test-utils';
import DeleteAccountModal from '@/components/DeleteAccountModal.vue';
import { deleteMyAccount, clearToken } from '@/services/api';

jest.mock('@/services/api', () => ({
  deleteMyAccount: jest.fn(),
  clearToken: jest.fn()
}));

jest.mock('@/services/theme', () => ({
  isDarkModeActive: () => false
}));

describe('DeleteAccountModal.vue', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('does not call the API when submitted with an empty password', async () => {
    const wrapper = shallowMount(DeleteAccountModal);

    await wrapper.vm.submitDelete();
    await flushPromises();

    expect(deleteMyAccount).not.toHaveBeenCalled();
  });

  it('clears the token once the account is deleted', async () => {
    deleteMyAccount.mockResolvedValue({ deleted: true });
    const wrapper = shallowMount(DeleteAccountModal);
    wrapper.vm.password = 's3cret-pass';

    await wrapper.vm.submitDelete();
    await flushPromises();

    expect(deleteMyAccount).toHaveBeenCalledWith('s3cret-pass');
    expect(clearToken).toHaveBeenCalled();
  });

  it('surfaces the backend error and leaves the modal open (and the session intact) on failure', async () => {
    deleteMyAccount.mockRejectedValue({ response: { data: { error: 'invalid email or password' } } });
    const wrapper = shallowMount(DeleteAccountModal);
    wrapper.vm.password = 'wrong-pass';

    await wrapper.vm.submitDelete();
    await flushPromises();

    expect(wrapper.vm.deleteError).toBe('invalid email or password');
    expect(wrapper.vm.isDeleting).toBe(false);
    expect(clearToken).not.toHaveBeenCalled();
  });

  it('does not close while a deletion is in flight', async () => {
    let resolveDelete;
    deleteMyAccount.mockReturnValue(new Promise((resolve) => { resolveDelete = resolve; }));
    const wrapper = shallowMount(DeleteAccountModal);
    wrapper.vm.password = 's3cret-pass';

    const submitPromise = wrapper.vm.submitDelete();
    wrapper.vm.close();
    expect(wrapper.emitted('close')).toBeFalsy();

    resolveDelete({ deleted: true });
    await submitPromise;
  });
});

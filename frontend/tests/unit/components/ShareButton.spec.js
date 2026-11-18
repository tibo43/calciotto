import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils';
import ShareButton from '@/components/ShareButton.vue';

const text = 'Join our group\nhttps://example.test/invite';

enableAutoUnmount(afterEach);
afterEach(() => vi.unstubAllGlobals());

async function shareWith(browser) {
  vi.stubGlobal('navigator', browser);
  const wrapper = mount(ShareButton, { props: { label: 'Share invitation', text } });
  wrapper.get('dialog').element.showModal = vi.fn();
  await wrapper.get('button[aria-label="Share invitation"]').trigger('click');
  await flushPromises();
  return wrapper;
}

it('opens the device share menu with the complete message', async () => {
  const share = vi.fn().mockResolvedValue(undefined);
  const writeText = vi.fn();
  await shareWith({ share, clipboard: { writeText } });
  expect(share).toHaveBeenCalledWith({ text });
  expect(writeText).not.toHaveBeenCalled();
});

it('leaves the clipboard alone when the user cancels the menu', async () => {
  const share = vi.fn().mockRejectedValue(new DOMException('User canceled', 'AbortError'));
  const writeText = vi.fn();
  await shareWith({ share, clipboard: { writeText } });
  expect(writeText).not.toHaveBeenCalled();
});

it('copies the message when native sharing is unavailable', async () => {
  const writeText = vi.fn().mockResolvedValue(undefined);
  const wrapper = await shareWith({ clipboard: { writeText } });
  expect(writeText).toHaveBeenCalledWith(text);
  expect(wrapper.get('[role="status"]').text()).toContain('Message copied');
});

it('offers selectable text when the browser blocks clipboard access', async () => {
  const writeText = vi.fn().mockRejectedValue(new DOMException('Clipboard unavailable', 'NotAllowedError'));
  const wrapper = await shareWith({ clipboard: { writeText } });
  expect(wrapper.get('dialog').element.showModal).toHaveBeenCalledOnce();
  expect(wrapper.get('textarea').element.value).toBe(text);
});

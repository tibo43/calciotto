import { mount } from '@vue/test-utils';
import AppToast, { TOAST_DURATION_MS } from '@/components/AppToast.vue';

describe('AppToast.vue', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it('renders nothing until shown', () => {
    const wrapper = mount(AppToast);
    expect(wrapper.find('.toast-message').exists()).toBe(false);
  });

  it('shows the message with its type, bottom right', async () => {
    const wrapper = mount(AppToast);
    wrapper.vm.show('Saved.', 'success');
    await wrapper.vm.$nextTick();
    expect(wrapper.find('.toast-container.toast-bottom .toast-message.success span').text()).toBe('Saved.');
  });

  it('closes from its cross', async () => {
    const wrapper = mount(AppToast);
    wrapper.vm.show('Oops', 'error');
    await wrapper.vm.$nextTick();
    await wrapper.find('.toast-close').trigger('click');
    expect(wrapper.find('.toast-message').exists()).toBe(false);
  });

  it('closes on its own, and a new message restarts the timer', async () => {
    const wrapper = mount(AppToast);
    wrapper.vm.show('First', 'success');
    jest.advanceTimersByTime(TOAST_DURATION_MS - 1000);
    wrapper.vm.show('Second', 'error');
    jest.advanceTimersByTime(TOAST_DURATION_MS - 1000);
    await wrapper.vm.$nextTick();
    expect(wrapper.find('.toast-message span').text()).toBe('Second');

    jest.advanceTimersByTime(1000);
    await wrapper.vm.$nextTick();
    expect(wrapper.find('.toast-message').exists()).toBe(false);
  });
});

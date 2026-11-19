import { shallowMount } from '@vue/test-utils';
import { computed } from 'vue';
import LoginGate from '@/components/LoginGate.vue';
import LoginPage from '@/components/Login.vue';
import LoginV2 from '@/components/v2/LoginV2.vue';

const mountGate = (on) => shallowMount(LoginGate, { global: { provide: { designV2On: computed(() => on) } } });

describe('LoginGate.vue', () => {
  it('keeps today\'s login page while design v2 is off', () => {
    const wrapper = mountGate(false);
    expect(wrapper.findComponent(LoginPage).exists()).toBe(true);
    expect(wrapper.findComponent(LoginV2).exists()).toBe(false);
  });

  it('renders the v2 login when the shell turned design v2 on', () => {
    const wrapper = mountGate(true);
    expect(wrapper.findComponent(LoginV2).exists()).toBe(true);
    expect(wrapper.findComponent(LoginPage).exists()).toBe(false);
  });

  it('defaults to today\'s page without a provider', () => {
    expect(shallowMount(LoginGate).findComponent(LoginPage).exists()).toBe(true);
  });
});

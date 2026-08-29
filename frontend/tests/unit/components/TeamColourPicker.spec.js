import { mount } from '@vue/test-utils';
import TeamColourPicker from '@/components/TeamColourPicker.vue';

describe('TeamColourPicker.vue', () => {
  it('renders a single swatch wrapping a native colour input', () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: '#ef4444' } });

    expect(wrapper.findAll('.colour-swatch')).toHaveLength(1);
    expect(wrapper.find('.colour-swatch-native-input').exists()).toBe(true);
  });

  it('sets the native input value to the current modelValue', () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: '#3b82f6' } });

    expect(wrapper.find('.colour-swatch-native-input').element.value).toBe('#3b82f6');
  });

  it('reflects modelValue as the swatch background when set', () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: '#3b82f6' } });

    expect(wrapper.find('.colour-swatch').attributes('style')).toContain('background: rgb(59, 130, 246)');
  });

  it('falls back to the rainbow placeholder (no inline background) when modelValue is empty', () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: '' } });

    expect(wrapper.find('.colour-swatch').attributes('style')).toBeUndefined();
  });

  it('emits update:modelValue with the picked hex value when the native input changes', async () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: '#ef4444' } });

    const input = wrapper.find('.colour-swatch-native-input');
    await input.setValue('#123456');

    expect(wrapper.emitted('update:modelValue')).toEqual([['#123456']]);
  });

  it('disables the native input while disabled, so it cannot emit', async () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: '#ef4444', disabled: true } });

    const input = wrapper.find('.colour-swatch-native-input');
    expect(input.attributes('disabled')).toBeDefined();
    expect(wrapper.find('.colour-swatch').classes()).toContain('colour-swatch--disabled');
  });
});

import { mount } from '@vue/test-utils';
import TeamColourPicker from '@/components/TeamColourPicker.vue';

// Mirrors PRESET_COLOURS in the component itself (kept in sync deliberately,
// not imported, so a change to the component's presets is visible as a
// failing assertion here rather than silently following along).
const PRESET_COLOURS = [
  '#ef4444', '#3b82f6', '#10b981', '#f59e0b', '#8b5cf6',
  '#f97316', '#ec4899', '#06b6d4', '#f8fafc', '#1f2937'
];

const CUSTOM_COLOUR = '#123456';

describe('TeamColourPicker.vue', () => {
  it('renders one swatch per preset colour plus the custom-colour escape hatch', () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: PRESET_COLOURS[0] } });

    const presetSwatches = wrapper.findAll('.colour-swatch:not(.colour-swatch--custom)');
    expect(presetSwatches).toHaveLength(PRESET_COLOURS.length);
    expect(wrapper.find('.colour-swatch--custom').exists()).toBe(true);
  });

  it('clicking a preset swatch emits update:modelValue with that swatch\'s exact hex value', async () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: PRESET_COLOURS[0] } });

    const targetIndex = 3;
    const swatches = wrapper.findAll('.colour-swatch:not(.colour-swatch--custom)');
    await swatches[targetIndex].trigger('click');

    expect(wrapper.emitted('update:modelValue')).toEqual([[PRESET_COLOURS[targetIndex]]]);
  });

  it('shows the "selected" state only on the swatch matching the current modelValue', () => {
    const selectedIndex = 2;
    const wrapper = mount(TeamColourPicker, { props: { modelValue: PRESET_COLOURS[selectedIndex] } });

    const swatches = wrapper.findAll('.colour-swatch:not(.colour-swatch--custom)');
    swatches.forEach((swatch, index) => {
      expect(swatch.classes('colour-swatch--selected')).toBe(index === selectedIndex);
    });
    expect(wrapper.find('.colour-swatch--custom').classes()).not.toContain('colour-swatch--selected');
  });

  it('shows the custom-colour control (not a preset) as selected when modelValue is outside the 10 presets', () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: CUSTOM_COLOUR } });

    const presetSwatches = wrapper.findAll('.colour-swatch:not(.colour-swatch--custom)');
    presetSwatches.forEach((swatch) => {
      expect(swatch.classes('colour-swatch--selected')).toBe(false);
    });
    expect(wrapper.find('.colour-swatch--custom').classes()).toContain('colour-swatch--selected');
  });

  it('does not emit when a preset swatch is clicked while disabled', async () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: PRESET_COLOURS[0], disabled: true } });

    const swatches = wrapper.findAll('.colour-swatch:not(.colour-swatch--custom)');
    expect(swatches[1].attributes('disabled')).toBeDefined();

    await swatches[1].trigger('click');

    expect(wrapper.emitted('update:modelValue')).toBeUndefined();
  });

  it('disables the custom-colour native input too, so it cannot emit while disabled', () => {
    const wrapper = mount(TeamColourPicker, { props: { modelValue: PRESET_COLOURS[0], disabled: true } });

    const nativeInput = wrapper.find('.colour-swatch-native-input');
    expect(nativeInput.attributes('disabled')).toBeDefined();
    expect(wrapper.find('.colour-swatch--custom').classes()).toContain('colour-swatch--disabled');
  });
});

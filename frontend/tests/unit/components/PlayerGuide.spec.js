import { mount } from '@vue/test-utils';
import PlayerGuide from '@/components/PlayerGuide.vue';

const toLastStep = async (wrapper) => {
  for (let step = 1; step < 5; step += 1) {
    await wrapper.find('.guide-actions .btn:not(.btn-ghost)').trigger('click');
  }
};

describe('PlayerGuide.vue', () => {
  it('walks through the steps and finishes on the last one', async () => {
    const wrapper = mount(PlayerGuide, { props: { required: true } });
    expect(wrapper.find('.label').text()).toBe('Step 1 of 5');
    expect(wrapper.find('img').attributes('src')).toBe('/guide/light/welcome.jpg');

    await toLastStep(wrapper);
    expect(wrapper.find('.label').text()).toBe('Step 5 of 5');
    await wrapper.find('.guide-actions .btn:not(.btn-ghost)').trigger('click');
    expect(wrapper.emitted('finish')).toHaveLength(1);
  });

  it('can be closed when replayed, not when required', async () => {
    const replay = mount(PlayerGuide, { props: { dark: true } });
    expect(replay.find('img').attributes('src')).toBe('/guide/dark/welcome.jpg');
    await replay.find('.guide-close').trigger('click');
    expect(replay.emitted('dismiss')).toHaveLength(1);

    const required = mount(PlayerGuide, { props: { required: true } });
    await required.find('.player-guide').trigger('keydown', { key: 'Escape' });
    expect(required.emitted('dismiss')).toBeUndefined();
    expect(required.find('.guide-close').exists()).toBe(false);
  });

  it('offers to continue anyway once saving the required guide failed', async () => {
    const wrapper = mount(PlayerGuide, { props: { required: true } });
    await toLastStep(wrapper);
    expect(wrapper.text()).not.toContain('Continue anyway');

    await wrapper.setProps({ error: 'Could not save your progress. Try again.' });
    expect(wrapper.find('[role="alert"]').text()).toContain('Could not save');
    const skip = wrapper.findAll('.guide-actions .btn').find((button) => button.text() === 'Continue anyway');
    await skip.trigger('click');
    expect(wrapper.emitted('skip')).toHaveLength(1);
  });

  it('never offers to continue anyway in a replay', async () => {
    const wrapper = mount(PlayerGuide, { props: { error: 'Could not save your progress. Try again.' } });
    expect(wrapper.text()).not.toContain('Continue anyway');
  });
});

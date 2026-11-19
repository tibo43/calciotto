import { mount } from '@vue/test-utils';
import GearLink from '@/components/v2/GearLink.vue';
import ManageMatchLink from '@/components/v2/ManageMatchLink.vue';

const stubs = { 'router-link': { template: '<a :data-to="to"><slot /></a>', props: ['to'] } };

describe('GearLink', () => {
  it('renders an icon-only link with identical aria-label and title', () => {
    const wrapper = mount(GearLink, { props: { to: '/x', label: 'Manage thing' }, global: { stubs } });
    const link = wrapper.get('a');
    expect(link.attributes('data-to')).toBe('/x');
    expect(link.attributes('aria-label')).toBe('Manage thing');
    expect(link.attributes('title')).toBe('Manage thing');
    expect(link.text()).toBe('');
    expect(link.get('svg').attributes('aria-hidden')).toBe('true');
  });

  it('keeps the caller class for sizing', () => {
    const wrapper = mount(GearLink, { props: { to: '/x', label: 'L' }, attrs: { class: 'group-gear' }, global: { stubs } });
    expect(wrapper.get('a').classes()).toContain('group-gear');
  });

  it('is what ManageMatchLink renders, with its classes and label', () => {
    const wrapper = mount(ManageMatchLink, { props: { id: 7 }, global: { stubs } });
    const link = wrapper.get('a');
    expect(link.classes()).toEqual(expect.arrayContaining(['btn', 'btn-ghost', 'hero-manage']));
    expect(link.attributes('data-to')).toBe('/matches/7/edit');
    expect(link.attributes('aria-label')).toBe('Manage match');
    expect(link.attributes('title')).toBe('Manage match');
  });
});

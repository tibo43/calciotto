import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import LanguageSelectorV2 from '@/components/v2/LanguageSelectorV2.vue';
import LocaleSaveErrorToast from '@/components/LocaleSaveErrorToast.vue';
import { startLocaleSync } from '@/services/playerLocale';
import { followPhone, getLocale, getLocaleSource, initLocale, onLocaleChange, SOURCE_EXPLICIT, SOURCE_PHONE } from '@/services/locale';

const api = vi.hoisted(() => ({
  getMyLocale: vi.fn(),
  updateMyLocale: vi.fn(),
  getSessionKey: vi.fn(() => 'session-1'),
}));
vi.mock('@/services/api', () => api);

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

const setLanguages = (languages) => {
  Object.defineProperty(window.navigator, 'languages', { value: languages, configurable: true });
};

const names = (wrapper) => wrapper.findAll('[role="radio"]').map((radio) => radio.text());
const checked = (wrapper) => wrapper.findAll('[role="radio"]').filter((radio) => radio.attributes('aria-checked') === 'true').map((radio) => radio.text());

const mountOpen = async () => {
  const wrapper = mount(LanguageSelectorV2, { attachTo: document.body });
  await wrapper.find('button.lang-toggle').trigger('click');
  return wrapper;
};

describe('LanguageSelectorV2', () => {
  let changes;
  let stop;

  beforeEach(() => {
    setLanguages(['de-DE', 'en']);
    initLocale();
    changes = [];
    stop = onLocaleChange((change) => changes.push(change));
  });

  afterEach(() => {
    stop();
    delete window.navigator.languages;
    initLocale();
    document.body.innerHTML = '';
  });

  it('shows the language row (in the active language) with the active language in its own language, closed at first', () => {
    const wrapper = mount(LanguageSelectorV2);
    const toggle = wrapper.find('button.lang-toggle');
    expect(toggle.text()).toBe('SpracheDeutsch');
    expect(toggle.attributes('aria-expanded')).toBe('false');
    expect(toggle.find('[lang="de"]').text()).toBe('Deutsch');
    expect(wrapper.find('[role="radiogroup"]').exists()).toBe(false);
  });

  it('lists the phone\'s language first, a rule, then the four others in the usual order', async () => {
    const wrapper = await mountOpen();
    expect(wrapper.find('button.lang-toggle').attributes('aria-expanded')).toBe('true');
    expect(wrapper.find('[role="radiogroup"]').exists()).toBe(true);
    expect(names(wrapper)).toEqual(['Deutsch', 'English', 'Français', 'Italiano', 'Español']);
    const list = wrapper.find('[role="radiogroup"]');
    expect(list.attributes('aria-labelledby')).toBe(wrapper.find('button.lang-toggle').attributes('id'));
    expect(list.element.children[1].classList.contains('lang-rule')).toBe(true);
    expect(wrapper.findAll('[role="radio"]').map((radio) => radio.attributes('lang'))).toEqual(['de', 'en', 'fr', 'it', 'es']);
    expect(checked(wrapper)).toEqual(['Deutsch']);
  });

  it('starts with English when the phone speaks a language the app does not', async () => {
    setLanguages(['pt-BR']);
    initLocale();
    const wrapper = await mountOpen();
    expect(names(wrapper)).toEqual(['English', 'Français', 'Italiano', 'Deutsch', 'Español']);
    expect(checked(wrapper)).toEqual(['English']);
  });

  it('applies a pick at once, with no button to save, and says so to the locale service', async () => {
    const wrapper = await mountOpen();
    await wrapper.findAll('[role="radio"]')[2].trigger('click');
    expect(getLocale()).toBe('fr');
    expect(getLocaleSource()).toBe(SOURCE_EXPLICIT);
    expect(document.documentElement.getAttribute('lang')).toBe('fr');
    expect(changes).toEqual([{ locale: 'fr', source: SOURCE_EXPLICIT }]);
    expect(checked(wrapper)).toEqual(['Français']);
    expect(wrapper.find('button.lang-toggle').text()).toBe('LangueFrançais');
    expect(wrapper.text()).not.toMatch(/save|enregistrer/i);
  });

  it('goes back to the phone with the first line, announcing "phone" so the server forgets the choice', async () => {
    const wrapper = await mountOpen();
    await wrapper.findAll('[role="radio"]')[3].trigger('click');
    changes.length = 0;
    await wrapper.findAll('[role="radio"]')[0].trigger('click');
    expect(getLocale()).toBe('de');
    expect(getLocaleSource()).toBe(SOURCE_PHONE);
    expect(changes).toEqual([{ locale: 'de', source: SOURCE_PHONE }]);
    expect(checked(wrapper)).toEqual(['Deutsch']);
  });

  it('does nothing when the checked line is picked again', async () => {
    const wrapper = await mountOpen();
    await wrapper.findAll('[role="radio"]')[0].trigger('click');
    await wrapper.findAll('[role="radio"]')[1].trigger('click');
    changes.length = 0;
    await wrapper.findAll('[role="radio"]')[1].trigger('click');
    expect(changes).toEqual([]);
  });

  it('checks the phone\'s line when a saved choice is the phone\'s own language, and lets it go back to following', async () => {
    initLocale({ saved: 'de' });
    expect(getLocaleSource()).toBe(SOURCE_EXPLICIT);
    const wrapper = await mountOpen();
    expect(checked(wrapper)).toEqual(['Deutsch']);
    await wrapper.findAll('[role="radio"]')[0].trigger('click');
    expect(getLocaleSource()).toBe(SOURCE_PHONE);
    expect(changes).toEqual([{ locale: 'de', source: SOURCE_PHONE }]);
  });

  it('is a radio group for the keyboard: one tab stop, arrows move and pick, wrapping at the ends', async () => {
    const wrapper = await mountOpen();
    const radios = () => wrapper.findAll('[role="radio"]');
    expect(radios().map((radio) => radio.attributes('tabindex'))).toEqual(['0', '-1', '-1', '-1', '-1']);

    await wrapper.find('[role="radiogroup"]').trigger('keydown', { key: 'ArrowDown' });
    expect(getLocale()).toBe('en');
    expect(radios().map((radio) => radio.attributes('tabindex'))).toEqual(['-1', '0', '-1', '-1', '-1']);
    await wrapper.find('[role="radiogroup"]').trigger('keydown', { key: 'ArrowUp' });
    expect(getLocaleSource()).toBe(SOURCE_PHONE);
    await wrapper.find('[role="radiogroup"]').trigger('keydown', { key: 'ArrowUp' });
    expect(getLocale()).toBe('es');
    await wrapper.find('[role="radiogroup"]').trigger('keydown', { key: 'ArrowRight' });
    expect(getLocaleSource()).toBe(SOURCE_PHONE);
    await wrapper.find('[role="radiogroup"]').trigger('keydown', { key: 'End' });
    expect(getLocale()).toBe('es');
    await wrapper.find('[role="radiogroup"]').trigger('keydown', { key: 'Home' });
    expect(getLocaleSource()).toBe(SOURCE_PHONE);
    expect(document.activeElement).toBe(radios()[0].element);
  });

  it('closes on Escape and gives the focus back to the row', async () => {
    const wrapper = await mountOpen();
    await wrapper.find('[role="radiogroup"]').trigger('keydown', { key: 'Escape' });
    expect(wrapper.find('[role="radiogroup"]').exists()).toBe(false);
    expect(document.activeElement).toBe(wrapper.find('button.lang-toggle').element);
  });

  it('follows the phone when it was picked while another language was chosen elsewhere', async () => {
    const wrapper = await mountOpen();
    await wrapper.findAll('[role="radio"]')[4].trigger('click');
    followPhone();
    await wrapper.vm.$nextTick();
    expect(checked(wrapper)).toEqual(['Deutsch']);
  });
});

// The selector next to the app shell's toast, as the Account page has them.
const Shell = {
  components: { LanguageSelectorV2, LocaleSaveErrorToast },
  template: '<div><LanguageSelectorV2 /><LocaleSaveErrorToast /></div>',
};

const mountOpenInShell = async () => {
  const wrapper = mount(Shell, { attachTo: document.body });
  await wrapper.find('button.lang-toggle').trigger('click');
  return wrapper;
};

describe('LanguageSelectorV2 when the save fails', () => {
  let stop;
  const toastText = (wrapper) => wrapper.find('.toast-message').text();

  beforeEach(() => {
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    api.updateMyLocale.mockReset().mockResolvedValue(null);
    setLanguages(['de-DE', 'en']);
    initLocale();
    stop = startLocaleSync();
  });

  afterEach(() => {
    stop();
    vi.restoreAllMocks();
    delete window.navigator.languages;
    initLocale();
    document.body.innerHTML = '';
  });

  it('puts the checkmark and the texts back, and shows the error toast in the previous language', async () => {
    api.updateMyLocale.mockRejectedValueOnce(new Error('503'));
    const wrapper = await mountOpenInShell();

    // Not awaited: the save's answer may already be in by the next tick.
    wrapper.findAll('[role="radio"]')[2].trigger('click');
    expect(getLocale()).toBe('fr');
    await flush();
    await wrapper.vm.$nextTick();

    expect(getLocale()).toBe('de');
    expect(document.documentElement.getAttribute('lang')).toBe('de');
    expect(checked(wrapper)).toEqual(['Deutsch']);
    expect(wrapper.find('button.lang-toggle').text()).toBe('SpracheDeutsch');
    expect(toastText(wrapper)).toContain('Sprache konnte nicht gespeichert werden. Versuch es noch mal.');
    expect(wrapper.find('.toast-message').classes()).toContain('error');
    expect(wrapper.find('.toast-message').attributes('data-label')).toBe('Fehler');
    // Once: the selector has no toast of its own.
    expect(wrapper.findAll('.toast-message')).toHaveLength(1);
  });

  it('leaves the toast to the app shell: the selector alone shows none', async () => {
    api.updateMyLocale.mockRejectedValueOnce(new Error('503'));
    const wrapper = await mountOpen();
    await wrapper.findAll('[role="radio"]')[2].trigger('click');
    await flush();
    await wrapper.vm.$nextTick();
    expect(getLocale()).toBe('de');
    expect(wrapper.find('.toast-message').exists()).toBe(false);
  });

  it('shows no toast when the save succeeds', async () => {
    const wrapper = await mountOpenInShell();
    await wrapper.findAll('[role="radio"]')[2].trigger('click');
    await flush();
    await wrapper.vm.$nextTick();
    expect(getLocale()).toBe('fr');
    expect(wrapper.find('.toast-message').exists()).toBe(false);
  });

  it('says it in every language: the previous one speaks', async () => {
    const expected = {
      en: "Couldn't save your language. Try again.",
      fr: 'Impossible d’enregistrer la langue. Réessaie.',
      de: 'Sprache konnte nicht gespeichert werden. Versuch es noch mal.',
      it: 'Impossibile salvare la lingua. Riprova.',
      es: 'No se pudo guardar el idioma. Inténtalo de nuevo.',
    };
    for (const [code, text] of Object.entries(expected)) {
      setLanguages([code]);
      initLocale();
      expect(getLocaleSource()).toBe(SOURCE_PHONE);
      api.updateMyLocale.mockRejectedValueOnce(new Error('503'));
      const wrapper = await mountOpenInShell();
      // Any language but the phone's.
      const other = wrapper.findAll('[role="radio"]')[1];
      await other.trigger('click');
      await flush();
      await wrapper.vm.$nextTick();
      expect(getLocale()).toBe(code);
      expect(toastText(wrapper).replace(/\u00a0/g, ' ')).toContain(text);
      wrapper.unmount();
    }
  });
});

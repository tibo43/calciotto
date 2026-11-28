// The API's error codes (app/internal/handlers/testdata/error_codes.json, which
// the Go tests keep in step with errorcodes.go) against the apiError messages:
// every code has a text in every language, English says what the API says, and
// the other languages keep its placeholders.
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { messages } from '@/locales';

const catalogue = JSON.parse(readFileSync(resolve(__dirname, '../../../../app/internal/handlers/testdata/error_codes.json'), 'utf8'));
const codes = Object.keys(catalogue).sort();
// The v2 rule for a message: a capital first letter and a full stop. The API's
// English text is lower case; the English bundle is that text as a sentence.
const sentence = (text) => {
  const capital = text.charAt(0).toUpperCase() + text.slice(1);
  return /[.!?]$/.test(capital) ? capital : `${capital}.`;
};
const placeholders = (text) => [...new Set([...text.matchAll(/\{(\w+)\}/g)].map((m) => m[1]))].sort();

describe('apiError messages', () => {
  it('has exactly the API codes in English, with the text the API sends as a sentence', () => {
    expect(messages.en.apiError).toEqual(Object.fromEntries(Object.entries(catalogue).map(([code, text]) => [code, sentence(text)])));
  });

  describe.each(['fr', 'it', 'de', 'es'])('in %s', (language) => {
    const bundle = messages[language].apiError;

    it('has a text for every code and none for a code the API does not send', () => {
      expect(Object.keys(bundle).sort()).toEqual(codes);
    });

    it('keeps the placeholders of the English text, and ends each text like a sentence', () => {
      for (const code of codes) {
        expect(placeholders(bundle[code]), `${language} ${code}`).toEqual(placeholders(catalogue[code]));
        expect(bundle[code], `${language} ${code}`).toMatch(/[.!?]$/);
        expect(bundle[code], `${language} ${code}`).not.toBe(sentence(catalogue[code]));
      }
    });
  });

  it('keeps MVP and Agoranking as the English words', () => {
    for (const language of ['fr', 'it', 'de', 'es']) {
      for (const code of ['cannot_vote_for_self', 'voter_not_on_roster', 'voting_closed']) {
        expect(messages[language].apiError[code], `${language} ${code}`).toContain('MVP');
      }
      expect(messages[language].apiError.agoranking_not_enabled).toContain('Agoranking');
    }
  });
});

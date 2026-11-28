// Guards for the English copy living in src/locales/en.json:
//  - no raw English left in the templates of the app's components,
//  - every message key the code asks for exists in en.json,
//  - every message renders, with its linked glossary terms resolved.
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';
import { describe, expect, it } from 'vitest';
import { parse } from '@vue/compiler-sfc';
import en from '@/locales/en.json';
import { i18n } from '@/i18n';

const SRC = join(__dirname, '../../../src');

function filesUnder(dir, ext) {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) return filesUnder(path, ext);
    return path.endsWith(ext) ? [path] : [];
  });
}

const SFCS = filesUnder(SRC, '.vue');
const SOURCES = [...SFCS, ...filesUnder(SRC, '.js')];

// Text that is not copy: symbols, separators, sample values.
const ALLOWED_TEXT = new Set(['https://example.com', 'AB2N7TQR']);
// Attributes a person reads or hears.
const READ_ATTRIBUTES = new Set(['aria-label', 'aria-description', 'placeholder', 'title', 'alt', 'label', 'summary', 'text']);
const NODE = { ELEMENT: 1, TEXT: 2, INTERPOLATION: 5, ATTRIBUTE: 6, DIRECTIVE: 7 };

// A string literal inside a template expression that looks like copy (a
// capital letter first, or words with spaces), not a key, a class or a value.
function copyLiterals(expression) {
  const found = [];
  const literal = /'((?:[^'\\]|\\.)*)'|"((?:[^"\\]|\\.)*)"/g;
  let match;
  while ((match = literal.exec(expression))) {
    const value = match[1] ?? match[2];
    if (!/[A-Za-z]/.test(value)) continue;
    if (/^[a-z][\w-]*(\.[\w-]+)+$/.test(value)) continue; // a message key
    if (/name:\s*$/.test(expression.slice(0, match.index))) continue; // a route name
    if (/^[A-Z]/.test(value) || /[A-Za-z]+\s+[A-Za-z]+/.test(value)) found.push(value);
  }
  return found;
}

function rawTextIn(file) {
  const { descriptor } = parse(readFileSync(file, 'utf8'), { filename: file });
  if (!descriptor.template) return [];
  const problems = [];
  const visit = (node) => {
    if (node.type === NODE.ELEMENT && node.tag === 'svg') return;
    if (node.type === NODE.TEXT) {
      const text = node.content.trim();
      if (/[A-Za-z]/.test(text) && !ALLOWED_TEXT.has(text)) problems.push(`text: ${text}`);
    }
    if (node.type === NODE.INTERPOLATION) {
      copyLiterals(node.content.content || '').forEach((value) => problems.push(`expression: ${value}`));
    }
    (node.props || []).forEach((prop) => {
      if (prop.type === NODE.ATTRIBUTE && READ_ATTRIBUTES.has(prop.name) && prop.value
        && /[A-Za-z]/.test(prop.value.content) && !ALLOWED_TEXT.has(prop.value.content)) {
        problems.push(`${prop.name}="${prop.value.content}"`);
      }
      if (prop.type === NODE.DIRECTIVE && prop.exp && prop.exp.content) {
        copyLiterals(prop.exp.content).forEach((value) => problems.push(`:${prop.arg?.content || prop.name}: ${value}`));
      }
    });
    (node.children || []).forEach(visit);
    (node.branches || []).forEach(visit);
  };
  visit(descriptor.template.ast);
  return problems.map((problem) => `${relative(SRC, file)}: ${problem}`);
}

function hasKey(key) {
  let node = en;
  for (const part of key.split('.')) {
    if (!node || typeof node !== 'object' || !(part in node)) return false;
    node = node[part];
  }
  return typeof node === 'string';
}

function leaves(node, prefix = '') {
  return Object.entries(node).flatMap(([key, value]) => (
    typeof value === 'string' ? [[`${prefix}${key}`, value]] : leaves(value, `${prefix}${key}.`)
  ));
}

describe('i18n extraction', () => {
  it('leaves no raw English in the component templates', () => {
    expect(SFCS.length).toBeGreaterThan(15);
    expect(SFCS.flatMap(rawTextIn)).toEqual([]);
  });

  it('finds raw English when there is some (the check itself works)', () => {
    const probe = (template) => {
      const { descriptor } = parse(`<template>${template}</template>`);
      return descriptor.template.ast.children.length;
    };
    expect(probe('<p>x</p>')).toBe(1);
    expect(copyLiterals("ok ? 'Copied' : $t('invite.copy')")).toEqual(['Copied']);
    expect(copyLiterals("$t('a.b', { n: 'is-on' })")).toEqual([]);
  });

  it('has every message key the code uses', () => {
    const used = new Set();
    const patterns = [
      /\$t\(\s*'([^']+)'/g,
      /keypath="([^"]+)"/g,
      /labelKey: '([^']+)'/g,
      /_KEY = '([a-z]\w*(?:\.\w+)+)'/g,
    ];
    SOURCES.forEach((file) => {
      const source = readFileSync(file, 'utf8');
      // The plain t() only where it is the one from @/i18n.
      const own = /import \{[^}]*\bt\b[^}]*\} from '@\/i18n'/.test(source) ? [/(?<![\w.$])t\(\s*'([^']+)'/g] : [];
      [...patterns, ...own].forEach((pattern) => {
        for (const match of source.matchAll(pattern)) used.add(match[1]);
      });
    });
    // Keys built at run time, listed out.
    ['GK', 'DEF', 'MID', 'FWD'].forEach((position) => used.add(`positions.${position}`));
    ['match', 'predictions', 'reminder', 'admin'].forEach((id) => used.add(`profile.notifications.groups.${id}`));
    Object.keys(en.guide).filter((id) => typeof en.guide[id] === 'object')
      .forEach((id) => ['title', 'description'].forEach((part) => used.add(`guide.${id}.${part}`)));

    expect(used.size).toBeGreaterThan(300);
    expect([...used].filter((key) => !hasKey(key))).toEqual([]);
  });

  it('renders every message, with MVP and Agoranking resolved', () => {
    const params = new Proxy({}, { get: (_, name) => (typeof name === 'string' ? `<${name}>` : undefined), has: () => true });
    const broken = leaves(en).filter(([key]) => {
      const text = i18n.global.t(key, params, 2);
      return text === key || text.includes('glossary.') || text.includes('@:') || /\{\w+\}/.test(text);
    });
    expect(broken).toEqual([]);
    expect(i18n.global.t('award.mvpOfTheMatch')).toBe('MVP of the Match');
    expect(i18n.global.t('notifications.motmAnnounced')).toBe('MVP announced');
  });
});

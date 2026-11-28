// Every component test mounts with vue-i18n installed, as main.js does for
// the app, so templates can call $t. The tests run in English.
import { config } from '@vue/test-utils';
import { i18n } from '@/i18n';

config.global.plugins = [...(config.global.plugins || []), i18n];

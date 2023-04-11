import {Joi} from '@docusaurus/utils-validation';
import type {
  LoadContext,
  Plugin,
  OptionValidationContext,
  ThemeConfig,
  ThemeConfigValidationContext,
} from '@docusaurus/types';
import type {PluginOptions, Options} from './options';
import {script} from './script';

export default function pluginGoogleGtag(
  context: LoadContext,
  opts: PluginOptions,
): Plugin {
  const {anonymizeIP, trackingID} = opts;
  const isProd = process.env.NODE_ENV === 'production';

  return {
    name: 'docusaurus-plugin-gtag-with-cookie-consent',

    contentLoaded({actions}) {
      actions.setGlobalData(opts);
    },

    getClientModules() {
      return isProd ? ['./gtag'] : [];
    },

    injectHtmlTags() {
      if (!isProd) {
        return {};
      }
      return {
        headTags: [
          {
            tagName: 'link',
            attributes: {
              rel: 'preconnect',
              href: 'https://www.google-analytics.com',
            },
          },
          {
            tagName: 'link',
            attributes: {
              rel: 'preconnect',
              href: 'https://www.googletagmanager.com',
            },
          },
          {
            tagName: 'link',
            attributes: {
              rel: 'preconnect',
              href: 'https://cdn.jsdelivr.net',
            },
          },
          {
            tagName: 'link',
            attributes: {
              rel: 'stylesheet',
              type: 'text/css',
              href: 'https://cdn.jsdelivr.net/npm/cookieconsent@3/build/cookieconsent.min.css'
            }
          },
          {
            tagName: 'script',
            attributes: {
              src: 'https://cdn.jsdelivr.net/npm/cookieconsent@3/build/cookieconsent.min.js',
              'data-cfasync': 'false'
            }
          },
          {
            tagName: 'script',
            attributes: {
              src: `https://www.googletagmanager.com/gtag/js?id=${opts.trackingID}`,
              async: true
            }
          },
          {
            tagName: 'script',
            innerHTML: script(opts)
          },
        ],
      };
    },
  };
}

const pluginOptionsSchema = Joi.object<PluginOptions>({
  trackingID: Joi.string().required(),
  anonymizeIP: Joi.boolean().default(false),
});

export function validateOptions({
  validate,
  options,
}: OptionValidationContext<Options, PluginOptions>): PluginOptions {
  return validate(pluginOptionsSchema, options);
}

export function validateThemeConfig({
  themeConfig,
}: ThemeConfigValidationContext<ThemeConfig>): ThemeConfig {
  if ('gtag' in themeConfig) {
    throw new Error(
      'The "gtag" field in themeConfig should now be specified as option for plugin-google-gtag. More information at https://github.com/facebook/docusaurus/pull/5832.',
    );
  }
  return themeConfig;
}

export type {PluginOptions, Options};

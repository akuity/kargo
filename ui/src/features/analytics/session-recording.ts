import type { PostHogConfig } from 'posthog-js';

const mask = '*';

// Attributes whose values are shown to users, name what an element is, or
// point at a resource: tooltips, labels, link targets, form values, and the
// IDs that component libraries derive from row keys and tab names.
const maskedAttributes = new Set([
  'alt',
  'content',
  'download',
  'for',
  'href',
  'id',
  'label',
  'name',
  'placeholder',
  'title',
  'value',
  'xlink:href'
]);

// The data attributes the UI's own styles select on. A recording replayed
// without them would not look like what the user saw.
const keptDataAttributes = new Set(['data-theme']);

// Elements whose href is needed to replay a recording and never names a
// resource: stylesheets, and the document's base URL.
const keptHrefElements = new Set(['base', 'link']);

/**
 * maskAttribute decides what a session recording keeps of an element's
 * attribute. Attributes that only affect how the element is drawn (class,
 * style, an SVG path) are kept. Those that can carry the name of a Kargo
 * resource are replaced.
 */
export const maskAttribute = (name: string, value: string, element?: Element): string => {
  const attribute = name.toLowerCase();
  if (attribute.startsWith('data-')) {
    return keptDataAttributes.has(attribute) ? value : mask;
  }
  if (attribute.startsWith('aria-')) {
    return mask;
  }
  if (!maskedAttributes.has(attribute)) {
    return value;
  }
  if (attribute === 'id') {
    // SVG markers, gradients, and clip paths are referenced by ID from the
    // shapes that use them.
    return element?.closest('defs') ? value : mask;
  }
  if (attribute === 'href' || attribute === 'xlink:href') {
    const references = value.startsWith('#');
    const kept = keptHrefElements.has(element?.tagName.toLowerCase() ?? '');
    return references || kept ? value : mask;
  }
  return mask;
};

/**
 * sessionRecordingOptions has PostHog mask a session recording as it is made,
 * so that what leaves the browser shows how the UI was laid out and used but
 * not what it said.
 */
export const sessionRecordingOptions: PostHogConfig['session_recording'] = {
  // Every text node and every form field, whatever it holds.
  maskTextSelector: '*',
  maskAllInputs: true,
  maskAttributeFn: maskAttribute,
  // Requests to the API name resources in their paths and carry them in
  // their bodies.
  recordHeaders: false,
  recordBody: false
};

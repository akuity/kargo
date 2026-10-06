// Detects console output in a promotion step's output.
//
// There is no dedicated logs field on a step result, so a step that shells out
// surfaces its console output as an ordinary output value: an array of lines,
// or one multi-line string. Both read badly as YAML.

// Keys that name console output. Matching is by name, since values are far too
// easy to mistake by shape -- plenty of ordinary outputs are string arrays or
// multi-line text. Deliberately narrow: a step whose output is console output
// under any other key keeps its YAML view.
const logKeys = ['lines', 'logs', 'log'];

// A plain string only reads as a log once it spans a few lines. Below this, it
// is more likely an ordinary scalar output (a commit SHA, a URL, ...).
const minTextLines = 3;

// Rendered elsewhere already: the deep-link plugin draws `links` beside the
// step's name, so it does not count as accompanying data.
const externallyRenderedKeys = ['links'];

// toLines interprets one output value as log lines.
const toLines = (value: unknown): string[] | undefined => {
  if (Array.isArray(value)) {
    return value.length > 0 && value.every((line) => typeof line === 'string')
      ? (value as string[])
      : undefined;
  }

  if (typeof value === 'string') {
    // Normalize line endings and drop a single trailing newline so the panel
    // does not render a phantom empty last line.
    const lines = value.replace(/\r\n/g, '\n').replace(/\n$/, '').split('\n');

    return lines.length >= minTextLines ? lines : undefined;
  }

  return undefined;
};

// getStepLogs reads a step's output as console output lines, or returns
// undefined when it is anything else -- including when something log-shaped
// sits next to other values, which reads as a data output rather than a log.
export const getStepLogs = (output?: object): string[] | undefined => {
  if (!output) {
    return undefined;
  }

  const entries = Object.entries(output).filter(([key]) => !externallyRenderedKeys.includes(key));

  if (entries.length !== 1) {
    return undefined;
  }

  const [key, value] = entries[0];

  if (!logKeys.includes(key)) {
    return undefined;
  }

  return toLines(value);
};

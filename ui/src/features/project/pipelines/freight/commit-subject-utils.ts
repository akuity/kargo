const MAX_SUBJECT_LENGTH = 100;

// trailing PR/MR reference: GitHub "(#123)" or GitLab "(!123)"
const TRAILING_REF = /\s*\([#!]\d+\)$/;

// first line of a commit message, truncated with "..." but keeping a trailing PR/MR reference
export const commitSubject = (message: string = '', maxLength = MAX_SUBJECT_LENGTH) => {
  const subject = message.split('\n')[0].trim();

  if (subject.length <= maxLength) {
    return subject;
  }

  const ref = subject.match(TRAILING_REF)?.[0] || '';
  const head = subject.slice(0, maxLength - ref.length - 3).trimEnd();

  return `${head}...${ref}`;
};

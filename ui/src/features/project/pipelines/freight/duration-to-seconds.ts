import { Duration } from 'date-fns';

export const durationToSeconds = (duration: Duration): number => {
  const {
    years = 0,
    months = 0,
    weeks = 0,
    days = 0,
    hours = 0,
    minutes = 0,
    seconds = 0
  } = duration;

  const totalSeconds =
    years * 365.25 * 24 * 60 * 60 +
    months * 30.44 * 24 * 60 * 60 +
    weeks * 7 * 24 * 60 * 60 +
    days * 24 * 60 * 60 +
    hours * 60 * 60 +
    minutes * 60 +
    seconds;

  return Math.floor(totalSeconds);
};

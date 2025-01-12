import { describe, expect, test } from "vitest";
import { calculatePageForSelectedRow } from "./pagination";

describe("calculatePageForSelectedRow", () => {
    test.each([
        [undefined, ["a", "b", "c"], (option: string) => option, 1],
        ["a", ["a", "b", "c"], (option: string) => option, 1],
        ["j", ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"], (option: string) => option, 1],
        ["k", ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"], (option: string) => option, 2],
    ])('selectedOption: %s, options: %s, key: %s, expectedPage: %s', (selectedOption, options, key, expectedPage) => {
        const page = calculatePageForSelectedRow(selectedOption, options, key);
        expect(page).toBe(expectedPage);
    });
});

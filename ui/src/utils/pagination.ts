export function calculatePageForSelectedRow<T>(selectedOption: T | undefined, options: T[], key: (option: T) => string): number {
    const pageSize = 10;

    if (selectedOption) {
        const index = options.findIndex((option) => key(option) === key(selectedOption));
        if (index >= 0) {
            const page = Math.floor(index / pageSize) + 1;
            return page;
        }
    }
    return 1;
}

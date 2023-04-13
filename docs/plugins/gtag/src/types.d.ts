interface Window {
  gtag: (
    command: string,
    fields: string,
    params: {
      page_title?: string;
      page_location?: string;
      page_path?: string;
    },
  ) => void;
}

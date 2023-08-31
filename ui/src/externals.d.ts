declare module '*.less' {
  const resource: { [key: string]: string };
  export = resource;
}

declare module '*.svg' {
  import Vue, { VueConstructor } from 'vue';
  const content: VueConstructor<Vue>;
  export default content;
}

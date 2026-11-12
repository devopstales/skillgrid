declare module "js-yaml" {
  export function load(input: string): unknown;
  export function dump(obj: unknown): string;
  const yaml: { load: (input: string) => unknown; dump: (obj: unknown) => string };
  export default yaml;
}
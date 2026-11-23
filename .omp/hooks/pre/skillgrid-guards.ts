// Thin re-export (three levels up to the repo root) so ambient omp discovery
// (<cwd>/.omp/hooks/pre/*.ts) loads the guard factory from the plugins/ tree
// (single source of truth). See plugins/omp/skillgrid-guards/README.md.
export { default } from "../../../plugins/omp/skillgrid-guards/hooks/skillgrid-guards.ts";

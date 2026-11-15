/** Today's date as YYYY-MM-DD (UTC, matching the CLI's date-only convention). */
export function today(): string {
  return new Date().toISOString().slice(0, 10);
}

export function normalizeStatus(s: string): string {
  return s.trim();
}

export function isTerminal(status: string, doneStatuses: string[]): boolean {
  return doneStatuses.includes(status);
}
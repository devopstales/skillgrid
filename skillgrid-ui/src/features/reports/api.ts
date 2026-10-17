import { apiFetch } from '../../lib/api'

export interface Finding {
  tool?: string
  severity: string
  title: string
  cve_id?: string
  rule_id?: string
  package?: string
  version?: string
  fixed_version?: string
  file?: string
  line: number
  message?: string
  links?: string[]
}

export interface SourceResult {
  tool: string
  status: string
  findings: Finding[]
}

export interface CoverageRow {
  file: string
  pkg: string
  pct_line: number
}

export interface ReportJSON {
  static: SourceResult[]
  dynamic: SourceResult[]
  coverage: CoverageRow[]
  generated_at: string
}

export async function fetchReport(): Promise<ReportJSON> {
  return apiFetch<ReportJSON>('/report/json')
}
